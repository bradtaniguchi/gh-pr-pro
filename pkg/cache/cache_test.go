package cache

import (
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
