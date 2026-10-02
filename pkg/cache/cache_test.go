package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/brad/gh-pr-pro/pkg/metrics"
)

func TestMergePRs(t *testing.T) {
	t1 := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 12, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name              string
		cached            []metrics.ProcessedPR
		incoming          []metrics.ProcessedPR
		expectedCount     int
		expectedPRNumbers []int
		validateResult    func(t *testing.T, merged []metrics.ProcessedPR)
	}{
		{
			name: "update existing PR state and append new PR with sorting",
			cached: []metrics.ProcessedPR{
				{Number: 1, Title: "PR 1 (old)", State: "OPEN", CreatedAt: t1},
				{Number: 2, Title: "PR 2", State: "MERGED", CreatedAt: t2},
			},
			incoming: []metrics.ProcessedPR{
				{Number: 1, Title: "PR 1 (updated)", State: "MERGED", CreatedAt: t1},
				{Number: 3, Title: "PR 3 (new)", State: "OPEN", CreatedAt: t3},
			},
			expectedCount:     3,
			expectedPRNumbers: []int{3, 2, 1},
			validateResult: func(t *testing.T, merged []metrics.ProcessedPR) {
				if merged[2].Title != "PR 1 (updated)" || merged[2].State != "MERGED" {
					t.Errorf("expected PR 1 to be updated to MERGED, got %s / %s", merged[2].Title, merged[2].State)
				}
			},
		},
		{
			name:              "empty cached and incoming slices",
			cached:            nil,
			incoming:          nil,
			expectedCount:     0,
			expectedPRNumbers: []int{},
		},
		{
			name:   "incoming PRs only",
			cached: nil,
			incoming: []metrics.ProcessedPR{
				{Number: 5, Title: "PR 5", State: "OPEN", CreatedAt: t1},
			},
			expectedCount:     1,
			expectedPRNumbers: []int{5},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			merged := MergePRs(tc.cached, tc.incoming)
			if len(merged) != tc.expectedCount {
				t.Fatalf("expected %d merged PRs, got %d", tc.expectedCount, len(merged))
			}

			for i, num := range tc.expectedPRNumbers {
				if merged[i].Number != num {
					t.Errorf("at index %d: expected PR %d, got %d", i, num, merged[i].Number)
				}
			}

			if tc.validateResult != nil {
				tc.validateResult(t, merged)
			}
		})
	}
}

func TestCacheManagerLoadAndSave(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManagerWithDir(tempDir)

	tests := []struct {
		name        string
		repo        string
		savePRs     []metrics.ProcessedPR
		isFirstLoad bool
		expectCount int
	}{
		{
			name:        "load non-existent cache returns empty without error",
			repo:        "cli/cli",
			isFirstLoad: true,
			expectCount: 0,
		},
		{
			name: "save and reload cached PRs for standard repo",
			repo: "cli/cli",
			savePRs: []metrics.ProcessedPR{
				{
					Number:    100,
					Title:     "test PR",
					State:     "MERGED",
					CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			isFirstLoad: false,
			expectCount: 1,
		},
		{
			name: "save and reload cached PRs for another repo with isolation",
			repo: "owner/another-repo",
			savePRs: []metrics.ProcessedPR{
				{
					Number:    200,
					Title:     "another PR",
					State:     "OPEN",
					CreatedAt: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Number:    201,
					Title:     "second PR",
					State:     "MERGED",
					CreatedAt: time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC),
				},
			},
			isFirstLoad: false,
			expectCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.isFirstLoad {
				loaded, lastFetched, err := cm.Load(tc.repo)
				if err != nil {
					t.Fatalf("expected no error loading non-existent cache, got: %v", err)
				}
				if len(loaded) != 0 || !lastFetched.IsZero() {
					t.Errorf("expected empty PRs and zero time, got %d PRs and %v", len(loaded), lastFetched)
				}
				return
			}

			// Save
			if err := cm.Save(tc.repo, tc.savePRs); err != nil {
				t.Fatalf("failed to save cache for %s: %v", tc.repo, err)
			}

			// Reload
			reloaded, fetchTime, err := cm.Load(tc.repo)
			if err != nil {
				t.Fatalf("failed to reload cache for %s: %v", tc.repo, err)
			}
			if len(reloaded) != tc.expectCount {
				t.Errorf("expected %d reloaded PRs, got %d", tc.expectCount, len(reloaded))
			}
			if fetchTime.IsZero() {
				t.Errorf("expected non-zero fetchTime")
			}
		})
	}
}

func TestCacheManagerManagement(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManagerWithDir(tempDir)

	if cm.GetBaseDir() != tempDir {
		t.Errorf("expected baseDir %s, got %s", tempDir, cm.GetBaseDir())
	}

	// 1. Initially empty
	entries, err := cm.ListEntries()
	if err != nil {
		t.Fatalf("failed to list empty cache: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}

	// 2. Save two repositories
	repo1PRs := []metrics.ProcessedPR{
		{Number: 1, Title: "PR 1", State: "MERGED", CreatedAt: time.Now()},
		{Number: 2, Title: "PR 2", State: "OPEN", CreatedAt: time.Now()},
	}
	repo2PRs := []metrics.ProcessedPR{
		{Number: 10, Title: "PR 10", State: "CLOSED", CreatedAt: time.Now()},
	}

	if err := cm.Save("owner/repo1", repo1PRs); err != nil {
		t.Fatalf("failed to save repo1: %v", err)
	}
	if err := cm.Save("owner/repo2", repo2PRs); err != nil {
		t.Fatalf("failed to save repo2: %v", err)
	}

	// 3. List entries
	entries, err = cm.ListEntries()
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// 4. Delete one repo
	deleted, err := cm.Delete("owner/repo1")
	if err != nil {
		t.Fatalf("failed to delete repo1: %v", err)
	}
	if !deleted {
		t.Errorf("expected repo1 to be deleted")
	}

	// Delete non-existent
	deleted, err = cm.Delete("owner/non-existent")
	if err != nil {
		t.Fatalf("unexpected error deleting non-existent repo: %v", err)
	}
	if deleted {
		t.Errorf("expected false deleting non-existent repo")
	}

	// List again
	entries, err = cm.ListEntries()
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}
	if len(entries) != 1 || entries[0].Repo != "owner/repo2" {
		t.Errorf("expected only repo2 in cache, got %+v", entries)
	}

	// 5. Clear all
	cleared, err := cm.ClearAll()
	if err != nil {
		t.Fatalf("failed to clear all: %v", err)
	}
	if cleared != 1 {
		t.Errorf("expected 1 cleared file, got %d", cleared)
	}

	entries, err = cm.ListEntries()
	if err != nil {
		t.Fatalf("failed to list entries after clear: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(entries))
	}
}

func TestAtomicCacheSave(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManagerWithDir(tempDir)

	prs := []metrics.ProcessedPR{
		{Number: 1, Title: "Atomic PR", State: "OPEN", CreatedAt: time.Now()},
	}

	if err := cm.Save("owner/repo", prs); err != nil {
		t.Fatalf("failed to save cache atomically: %v", err)
	}

	// Verify destination file exists and is valid
	filePath := filepath.Join(tempDir, "owner_repo.json")
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("expected cache file to exist at %s: %v", filePath, err)
	}

	// Check file permissions (on Unix systems)
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("expected file mode 0600, got %o", perm)
		}
	}

	// Verify no temporary files remain in directory
	dirEntries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read tempDir: %v", err)
	}
	if len(dirEntries) != 1 {
		t.Errorf("expected exactly 1 file in directory, got %d", len(dirEntries))
	}
	for _, entry := range dirEntries {
		if entry.Name() != "owner_repo.json" {
			t.Errorf("unexpected residual file in cache dir: %s", entry.Name())
		}
	}
}

func TestConcurrentCacheAccess(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManagerWithDir(tempDir)

	repo := "concurrent/test-repo"
	numWriters := 10
	numReaders := 20
	iterations := 25

	var wg sync.WaitGroup
	errCh := make(chan error, numWriters*iterations+numReaders*iterations)

	// Launch concurrent writers
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				prs := make([]metrics.ProcessedPR, 50)
				for p := 0; p < 50; p++ {
					prs[p] = metrics.ProcessedPR{
						Number:    p + 1,
						Title:     fmt.Sprintf("PR from writer %d iter %d", writerID, i),
						State:     "OPEN",
						CreatedAt: time.Now(),
					}
				}
				if err := cm.Save(repo, prs); err != nil {
					errCh <- fmt.Errorf("writer %d iter %d failed: %w", writerID, i, err)
					return
				}
			}
		}(w)
	}

	// Launch concurrent readers
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				loaded, _, err := cm.Load(repo)
				if err != nil {
					errCh <- fmt.Errorf("reader %d iter %d failed to load: %w", readerID, i, err)
					return
				}
				// If loaded, ensure it is not a partial/corrupted slice
				if loaded != nil && len(loaded) != 50 {
					errCh <- fmt.Errorf("reader %d iter %d read partial data: len=%d", readerID, i, len(loaded))
					return
				}
			}
		}(r)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent cache access error: %v", err)
	}
}

func TestListEntriesIgnoresTempAndHiddenFiles(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManagerWithDir(tempDir)

	// Create valid cache entry
	prs := []metrics.ProcessedPR{
		{Number: 1, Title: "PR 1", State: "OPEN", CreatedAt: time.Now()},
	}
	if err := cm.Save("owner/valid-repo", prs); err != nil {
		t.Fatalf("failed to save valid repo: %v", err)
	}

	// Create dummy hidden/temporary files
	hiddenFile := filepath.Join(tempDir, ".hidden.json")
	if err := os.WriteFile(hiddenFile, []byte(`{"invalid": true}`), 0o600); err != nil {
		t.Fatalf("failed to write hidden file: %v", err)
	}

	tmpFile := filepath.Join(tempDir, ".tmp-owner_repo-12345.json")
	if err := os.WriteFile(tmpFile, []byte(`{"partial": true}`), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	entries, err := cm.ListEntries()
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 valid entry, got %d", len(entries))
	}
	if entries[0].Repo != "owner/valid-repo" {
		t.Errorf("expected repo owner/valid-repo, got %s", entries[0].Repo)
	}

	// ClearAll should remove all files including .tmp- files
	cleared, err := cm.ClearAll()
	if err != nil {
		t.Fatalf("failed to clear all: %v", err)
	}
	if cleared < 2 {
		t.Errorf("expected at least 2 files cleared (valid + temp), got %d", cleared)
	}
}

func TestCacheVersioningAndMigration(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManagerWithDir(tempDir)

	// 1. Write an unversioned legacy (v0.0.0) cache file
	legacyJSON := `{
  "repo": "owner/legacy",
  "last_fetched": "2025-01-01T00:00:00Z",
  "prs": [
    {
      "number": 1,
      "title": "Legacy PR",
      "state": "MERGED",
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}`
	legacyPath := cm.getFilePath("owner/legacy")
	if err := os.WriteFile(legacyPath, []byte(legacyJSON), 0o600); err != nil {
		t.Fatalf("failed writing legacy cache: %v", err)
	}

	// 2. Inspect entries before load - should be detected as outdated and migratable
	entries, err := cm.ListEntries()
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Version != LegacyVersion || entries[0].Status != StatusOutdated || !entries[0].IsMigratable {
		t.Errorf("expected legacy outdated migratable, got %+v", entries[0])
	}

	// 3. Test explicit dry-run migration
	fromVer, toVer, err := cm.Migrate("owner/legacy", true)
	if err != nil {
		t.Fatalf("unexpected error in dry-run migration: %v", err)
	}
	if fromVer != LegacyVersion || toVer != CurrentVersion {
		t.Errorf("expected migration from %s to %s, got %s to %s", LegacyVersion, CurrentVersion, fromVer, toVer)
	}

	// Verify file is still legacy after dry-run
	entries, _ = cm.ListEntries()
	if entries[0].Version != LegacyVersion {
		t.Errorf("expected file to remain %s after dry-run, got %s", LegacyVersion, entries[0].Version)
	}

	// 4. Test Load automatic migration
	prs, lastFetched, err := cm.Load("owner/legacy")
	if err != nil {
		t.Fatalf("failed to load and auto-migrate legacy cache: %v", err)
	}
	if len(prs) != 1 || prs[0].Number != 1 {
		t.Errorf("expected 1 loaded PR, got %+v", prs)
	}
	if lastFetched.IsZero() {
		t.Errorf("expected non-zero lastFetched")
	}

	// 5. Verify the file on disk was upgraded to CurrentVersion
	entries, _ = cm.ListEntries()
	if len(entries) != 1 || entries[0].Version != CurrentVersion || entries[0].Status != StatusCurrent {
		t.Errorf("expected entry to be upgraded to current, got %+v", entries[0])
	}

	// 6. Test already current file migration (no-op)
	fromVer, toVer, err = cm.Migrate("owner/legacy", false)
	if err != nil {
		t.Fatalf("unexpected error migrating current file: %v", err)
	}
	if fromVer != CurrentVersion || toVer != CurrentVersion {
		t.Errorf("expected no-op migration, got %s to %s", fromVer, toVer)
	}
}

func TestCacheFutureVersionProtection(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManagerWithDir(tempDir)

	futureJSON := `{
  "version": "v99.0.0",
  "repo": "owner/future",
  "last_fetched": "2026-01-01T00:00:00Z",
  "prs": []
}`
	futurePath := cm.getFilePath("owner/future")
	if err := os.WriteFile(futurePath, []byte(futureJSON), 0o600); err != nil {
		t.Fatalf("failed writing future cache: %v", err)
	}

	entries, err := cm.ListEntries()
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}
	if len(entries) != 1 || entries[0].Status != StatusFuture || entries[0].Version != "v99.0.0" {
		t.Errorf("expected future entry, got %+v", entries[0])
	}

	// Loading should return ErrCacheFutureVersion
	_, _, err = cm.Load("owner/future")
	if err == nil || !errors.Is(err, ErrCacheFutureVersion) {
		t.Errorf("expected ErrCacheFutureVersion, got: %v", err)
	}

	// Migrating should error
	_, _, err = cm.Migrate("owner/future", false)
	if err == nil || !errors.Is(err, ErrCacheFutureVersion) {
		t.Errorf("expected ErrCacheFutureVersion on migrate, got: %v", err)
	}
}

func TestCacheCorruptAndSelectiveClearing(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManagerWithDir(tempDir)

	// Valid entry
	if err := cm.Save("owner/valid", []metrics.ProcessedPR{{Number: 1, Title: "Valid", State: "OPEN", CreatedAt: time.Now()}}); err != nil {
		t.Fatalf("failed to save valid repo: %v", err)
	}

	// Outdated entry (v0)
	legacyPath := cm.getFilePath("owner/outdated")
	if err := os.WriteFile(legacyPath, []byte(`{"repo":"owner/outdated","prs":[]}`), 0o600); err != nil {
		t.Fatalf("failed writing outdated cache: %v", err)
	}

	// Corrupt entry
	corruptPath := filepath.Join(tempDir, "owner_corrupt.json")
	if err := os.WriteFile(corruptPath, []byte(`{not valid json}`), 0o600); err != nil {
		t.Fatalf("failed writing corrupt cache: %v", err)
	}

	entries, err := cm.ListEntries()
	if err != nil {
		t.Fatalf("failed listing: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Loading corrupt file should return ErrCacheCorrupt
	_, _, err = cm.Load("owner/corrupt")
	if err == nil || !errors.Is(err, ErrCacheCorrupt) {
		t.Errorf("expected ErrCacheCorrupt, got: %v", err)
	}

	// Clear corrupt files only
	clearedCorrupt, err := cm.ClearCorrupt()
	if err != nil {
		t.Fatalf("failed clearing corrupt: %v", err)
	}
	if clearedCorrupt != 1 {
		t.Errorf("expected 1 corrupt file cleared, got %d", clearedCorrupt)
	}

	entries, _ = cm.ListEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries remaining, got %d", len(entries))
	}

	// Clear outdated files only
	clearedOutdated, err := cm.ClearOutdated()
	if err != nil {
		t.Fatalf("failed clearing outdated: %v", err)
	}
	if clearedOutdated != 1 {
		t.Errorf("expected 1 outdated file cleared, got %d", clearedOutdated)
	}

	entries, _ = cm.ListEntries()
	if len(entries) != 1 || entries[0].Repo != "owner/valid" {
		t.Errorf("expected only valid repo to remain, got %+v", entries)
	}
}

func TestMigratorPipeline(t *testing.T) {
	m := NewMigrator()

	// Register dummy step v0.2.0 -> v0.3.0
	m.Register("v0.2.0", func(raw []byte) ([]byte, error) {
		var d map[string]interface{}
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, err
		}
		d["version"] = "v0.3.0"
		d["migrated_to_v0_3"] = true
		return json.Marshal(d)
	})

	input := []byte(`{"version":"v0.2.0","repo":"test/repo"}`)
	out, err := m.Migrate(input, "v0.2.0", "v0.3.0")
	if err != nil {
		t.Fatalf("failed migrating v0.2.0 -> v0.3.0: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("failed unmarshaling migrated json: %v", err)
	}
	if parsed["version"].(string) != "v0.3.0" || parsed["migrated_to_v0_3"] != true {
		t.Errorf("expected migrated fields, got %+v", parsed)
	}

	// Downgrade error check
	_, err = m.Migrate(out, "v0.3.0", "v0.2.0")
	if err == nil {
		t.Errorf("expected error attempting downgrade")
	}

	// Missing step check
	_, err = m.Migrate(out, "v0.3.0", "v1.0.0")
	if err == nil {
		t.Errorf("expected error for missing migration step")
	}
}
