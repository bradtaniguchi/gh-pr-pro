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
)

// CacheManager manages thread-safe local file storage of processed PR data.
// It serializes historical metrics to JSON files in the specified base directory.
type CacheManager struct {
	baseDir string
	mu      sync.RWMutex
}

// CachedData represents the root envelope structure persisted to JSON on disk.
type CachedData struct {
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
		baseDir: dir,
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
// If the file cannot be read or unmarshaled, an error is returned.
func (cm *CacheManager) Load(repo string) ([]metrics.ProcessedPR, time.Time, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	filePath := cm.getFilePath(repo)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, err
	}

	var cached CachedData
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, time.Time{}, err
	}

	return cached.PRs, cached.LastFetched, nil
}

// Save writes and serializes the provided slice of processed PRs to disk for the given repository.
// It automatically creates any missing parent directories and records the current timestamp as LastFetched.
func (cm *CacheManager) Save(repo string, prs []metrics.ProcessedPR) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if err := os.MkdirAll(cm.baseDir, 0755); err != nil {
		return err
	}

	filePath := cm.getFilePath(repo)
	cached := CachedData{
		Repo:        repo,
		LastFetched: time.Now(),
		PRs:         prs,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// GetBaseDir returns the base cache directory path.
func (cm *CacheManager) GetBaseDir() string {
	return cm.baseDir
}

// CacheEntryInfo summarizes a single cached repository entry stored on disk.
type CacheEntryInfo struct {
	Repo        string    `json:"repo"`
	FilePath    string    `json:"file_path"`
	SizeBytes   int64     `json:"size_bytes"`
	PRCount     int       `json:"pr_count"`
	LastFetched time.Time `json:"last_fetched"`
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

		var cached CachedData
		if err := json.Unmarshal(data, &cached); err != nil {
			continue
		}

		repoName := cached.Repo
		if repoName == "" {
			repoName = strings.TrimSuffix(entry.Name(), ".json")
		}

		results = append(results, CacheEntryInfo{
			Repo:        repoName,
			FilePath:    fullPath,
			SizeBytes:   info.Size(),
			PRCount:     len(cached.PRs),
			LastFetched: cached.LastFetched,
		})
	}

	return results, nil
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
