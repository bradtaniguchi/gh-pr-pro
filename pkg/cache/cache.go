// Package cache provides local disk-based persistence and incremental delta synchronization
// for analyzed pull request records.
//
// By caching historical PR datasets per repository under ~/.cache/gh-pr-pro/, gh-pr-pro
// accelerates repeated metric queries and prevents rate limit exhaustion on large codebases.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/brad/gh-pr-pro/pkg/metrics"
	"github.com/brad/gh-pr-pro/pkg/version"
)

var (
	// CurrentVersion defines the active cache data format version, aligned with package release version.
	CurrentVersion = version.Version

	// LegacyVersion represents unversioned cache files created prior to versioning.
	LegacyVersion = "v0.0.0"
)

// CacheManager manages thread-safe local file storage of processed PR data.
// It serializes historical metrics to JSON files in the specified base directory.
type CacheManager struct {
	baseDir  string
	mu       sync.RWMutex
	migrator *Migrator
}

// CachedData represents the root envelope structure persisted to JSON on disk.
type CachedData struct {
	// Version indicates the semantic package version of this cache file.
	Version string `json:"version"`
	// Repo is the full repository name in "owner/name" format.
	Repo string `json:"repo"`
	// LastFetched is the timestamp when the cache file was last refreshed from the GitHub API.
	LastFetched time.Time `json:"last_fetched"`
	// PRs is the slice of processed PR records stored in this cache entry.
	PRs []metrics.ProcessedPR `json:"prs"`
}

// NewCacheManager creates and initializes a CacheManager targeting the user's default cache directory
// ($HOME/.cache/gh-pr-pro). Returns an error if the user home directory cannot be resolved.
func NewCacheManager() (*CacheManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	cacheDir := filepath.Join(home, ".cache", "gh-pr-pro")
	return NewCacheManagerWithDir(cacheDir), nil
}

// NewCacheManagerWithDir creates a CacheManager rooted at the specified custom filesystem directory path.
func NewCacheManagerWithDir(dir string) *CacheManager {
	return &CacheManager{
		baseDir:  dir,
		migrator: NewMigrator(),
	}
}

// getFilePath returns the sanitized JSON filepath for a given repository identifier.
func (cm *CacheManager) getFilePath(repo string) string {
	sanitized := strings.ReplaceAll(repo, "/", "_")
	return filepath.Join(cm.baseDir, fmt.Sprintf("%s.json", sanitized))
}

// Load reads and deserializes cached PR data for the given repository.
//
// If the cache file does not exist, it returns (nil, zeroTime, nil) without error.
// If the file is outdated and migratable, it automatically migrates and updates the file on disk.
// If the file is corrupted, requires an API refresh, or was created by a future version,
// an appropriate sentinel error is returned.
func (cm *CacheManager) Load(repo string) ([]metrics.ProcessedPR, time.Time, error) {
	cm.mu.RLock()
	filePath := cm.getFilePath(repo)
	data, err := os.ReadFile(filePath)
	cm.mu.RUnlock()

	if err != nil {
		if os.IsNotExist(err) {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, err
	}

	ver, _, status, err := DetectVersion(data)
	if err != nil {
		return nil, time.Time{}, err
	}

	switch status {
	case StatusCurrent:
		var cached CachedData
		if err := json.Unmarshal(data, &cached); err != nil {
			return nil, time.Time{}, fmt.Errorf("%w: %v", ErrCacheCorrupt, err)
		}
		return cached.PRs, cached.LastFetched, nil

	case StatusFuture:
		return nil, time.Time{}, fmt.Errorf("%w (file version %s is newer than current %s)", ErrCacheFutureVersion, ver, CurrentVersion)

	case StatusOutdated:
		if !cm.migrator.CanMigrate(ver) {
			return nil, time.Time{}, fmt.Errorf("%w (version %s cannot be migrated to %s)", ErrCacheRequiresRefresh, ver, CurrentVersion)
		}

		// Perform automatic migration under write lock
		cm.mu.Lock()
		defer cm.mu.Unlock()

		// Re-read file under write lock to avoid race conditions
		freshData, err := os.ReadFile(filePath)
		if err != nil {
			return nil, time.Time{}, err
		}

		freshVer, _, _, err := DetectVersion(freshData)
		if err != nil {
			return nil, time.Time{}, err
		}

		migratedData, err := cm.migrator.Migrate(freshData, freshVer, CurrentVersion)
		if err != nil {
			return nil, time.Time{}, err
		}

		var cached CachedData
		if err := json.Unmarshal(migratedData, &cached); err != nil {
			return nil, time.Time{}, fmt.Errorf("%w after migration: %v", ErrCacheCorrupt, err)
		}

		// Save migrated file back to disk
		//nolint:gosec // filePath is constrained to baseDir via getFilePath
		if err := os.WriteFile(filePath, migratedData, 0o600); err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to save migrated cache: %w", err)
		}

		return cached.PRs, cached.LastFetched, nil

	default:
		return nil, time.Time{}, ErrCacheCorrupt
	}
}

// Save writes and serializes the provided slice of processed PRs to disk for the given repository.
// It automatically creates any missing parent directories, tags the file with CurrentVersion,
// and records the current timestamp as LastFetched.
func (cm *CacheManager) Save(repo string, prs []metrics.ProcessedPR) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if err := os.MkdirAll(cm.baseDir, 0o755); err != nil {
		return err
	}

	filePath := cm.getFilePath(repo)
	cached := CachedData{
		Version:     CurrentVersion,
		Repo:        repo,
		LastFetched: time.Now(),
		PRs:         prs,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0o600)
}

// GetBaseDir returns the base cache directory path.
func (cm *CacheManager) GetBaseDir() string {
	return cm.baseDir
}

// CacheEntryInfo summarizes a single cached repository entry stored on disk.
type CacheEntryInfo struct {
	Repo         string      `json:"repo"`
	FilePath     string      `json:"file_path"`
	SizeBytes    int64       `json:"size_bytes"`
	PRCount      int         `json:"pr_count"`
	LastFetched  time.Time   `json:"last_fetched"`
	Version      string      `json:"version"`
	Status       CacheStatus `json:"status"`
	IsMigratable bool        `json:"is_migratable"`
}

// ListEntries scans the cache directory and returns metadata for all cached repository entries.
func (cm *CacheManager) ListEntries() ([]CacheEntryInfo, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	entries, err := os.ReadDir(cm.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []CacheEntryInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		fullPath := filepath.Join(cm.baseDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		ver, repoName, status, detectErr := DetectVersion(data)
		if detectErr != nil {
			status = StatusCorrupt
			repoName = strings.TrimSuffix(entry.Name(), ".json")
		}

		var prCount int
		var lastFetched time.Time
		if status != StatusCorrupt {
			var cached CachedData
			if err := json.Unmarshal(data, &cached); err == nil {
				prCount = len(cached.PRs)
				lastFetched = cached.LastFetched
				if repoName == "" {
					repoName = cached.Repo
				}
			}
		}

		if repoName == "" {
			repoName = strings.TrimSuffix(entry.Name(), ".json")
		}

		results = append(results, CacheEntryInfo{
			Repo:         repoName,
			FilePath:     fullPath,
			SizeBytes:    info.Size(),
			PRCount:      prCount,
			LastFetched:  lastFetched,
			Version:      ver,
			Status:       status,
			IsMigratable: cm.migrator.CanMigrate(ver),
		})
	}

	return results, nil
}

// Migrate inspects and upgrades the cache file for a specific repository to CurrentVersion.
// If dryRun is true, it verifies migratability without modifying the file on disk.
func (cm *CacheManager) Migrate(repo string, dryRun bool) (fromVer, toVer string, err error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	filePath := cm.getFilePath(repo)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}

	ver, _, status, err := DetectVersion(data)
	if err != nil {
		return "", "", err
	}

	if status == StatusCurrent {
		return ver, ver, nil
	}

	if status == StatusFuture {
		return ver, ver, fmt.Errorf("%w: file version %s is newer than current %s", ErrCacheFutureVersion, ver, CurrentVersion)
	}

	if !cm.migrator.CanMigrate(ver) {
		return ver, ver, fmt.Errorf("%w: version %s cannot be migrated to %s", ErrCacheRequiresRefresh, ver, CurrentVersion)
	}

	migratedData, err := cm.migrator.Migrate(data, ver, CurrentVersion)
	if err != nil {
		return ver, ver, err
	}

	if !dryRun {
		//nolint:gosec // filePath is constrained to baseDir via getFilePath
		if err := os.WriteFile(filePath, migratedData, 0o600); err != nil {
			return ver, CurrentVersion, fmt.Errorf("failed to save migrated cache: %w", err)
		}
	}

	return ver, CurrentVersion, nil
}

// MigrateAll scans all cached repositories and upgrades any outdated entries to CurrentVersion.
// If dryRun is true, it calculates the number of eligible files without modifying disk.
func (cm *CacheManager) MigrateAll(dryRun bool) (int, error) {
	entries, err := cm.ListEntries()
	if err != nil {
		return 0, err
	}

	migratedCount := 0
	for _, entry := range entries {
		if entry.Status == StatusOutdated && entry.IsMigratable {
			if _, _, err := cm.Migrate(entry.Repo, dryRun); err != nil {
				return migratedCount, fmt.Errorf("failed migrating %s: %w", entry.Repo, err)
			}
			migratedCount++
		}
	}

	return migratedCount, nil
}

// ClearOutdated deletes all cache files that are outdated or require an API refresh.
func (cm *CacheManager) ClearOutdated() (int, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	entries, err := os.ReadDir(cm.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	deletedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		fullPath := filepath.Join(cm.baseDir, entry.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		ver, _, status, _ := DetectVersion(data)
		if status == StatusOutdated || (status != StatusCurrent && status != StatusFuture && version.Compare(ver, CurrentVersion) < 0) {
			if err := os.Remove(fullPath); err == nil {
				deletedCount++
			}
		}
	}

	return deletedCount, nil
}

// ClearCorrupt deletes all cache files in the cache directory that contain invalid or corrupted JSON.
func (cm *CacheManager) ClearCorrupt() (int, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	entries, err := os.ReadDir(cm.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	deletedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		fullPath := filepath.Join(cm.baseDir, entry.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		_, _, status, detectErr := DetectVersion(data)
		if detectErr != nil || status == StatusCorrupt {
			if err := os.Remove(fullPath); err == nil {
				deletedCount++
			}
		}
	}

	return deletedCount, nil
}

// Delete removes the cache file for a specific repository.
// Returns true if a file was found and removed, false if no cache entry existed.
func (cm *CacheManager) Delete(repo string) (bool, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	filePath := cm.getFilePath(repo)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if err := os.Remove(filePath); err != nil {
		return false, err
	}
	return true, nil
}

// ClearAll removes all cached JSON entries from the cache directory and returns the number of deleted files.
func (cm *CacheManager) ClearAll() (int, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	entries, err := os.ReadDir(cm.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	deletedCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			fullPath := filepath.Join(cm.baseDir, entry.Name())
			if err := os.Remove(fullPath); err == nil {
				deletedCount++
			}
		}
	}

	return deletedCount, nil
}

// MergePRs combines an existing slice of cached PRs with newly fetched incoming PRs.
// It deduplicates records by PR number (incoming takes precedence on collision) and
// sorts the resulting dataset descending by PR CreatedAt timestamp (most recent first).
func MergePRs(existing, incoming []metrics.ProcessedPR) []metrics.ProcessedPR {
	prMap := make(map[int]metrics.ProcessedPR)
	for _, pr := range existing {
		prMap[pr.Number] = pr
	}
	// Overwrite/insert incoming PRs
	for _, pr := range incoming {
		prMap[pr.Number] = pr
	}

	var result []metrics.ProcessedPR
	for _, pr := range prMap {
		result = append(result, pr)
	}

	// Sort descending by CreatedAt (most recent first)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].CreatedAt.Before(result[j].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}
