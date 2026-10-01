package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brad/gh-pr-pro/pkg/cache"
	"github.com/brad/gh-pr-pro/pkg/metrics"
	"github.com/spf13/cobra"
)

func TestFormatByteSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tc := range tests {
		res := formatByteSize(tc.bytes)
		if res != tc.expected {
			t.Errorf("for %d bytes: expected %s, got %s", tc.bytes, tc.expected, res)
		}
	}
}

func TestCacheCommandsFlagScopingInvariant(t *testing.T) {
	// AGENTS.md invariant: Cache commands must NEVER accept metric or filtering flags
	metricFlags := []string{"past", "since", "until", "group-by", "author", "reviewer", "label", "base", "percentiles", "detailed"}

	var checkCmd func(c *cobra.Command)
	checkCmd = func(c *cobra.Command) {
		for _, f := range metricFlags {
			if c.Flags().Lookup(f) != nil {
				t.Errorf("command '%s' must not have metric flag '--%s'", c.CommandPath(), f)
			}
		}
		for _, sub := range c.Commands() {
			checkCmd(sub)
		}
	}

	checkCmd(cacheCmd)
}

func TestCacheCommands(t *testing.T) {
	tempDir := t.TempDir()
	cm := cache.NewCacheManagerWithDir(tempDir)

	prs := []metrics.ProcessedPR{
		{Number: 1, Title: "PR 1", State: "MERGED", CreatedAt: time.Now()},
	}
	if err := cm.Save("testowner/testrepo", prs); err != nil {
		t.Fatalf("failed to save test repo: %v", err)
	}

	entries, err := cm.ListEntries()
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Repo != "testowner/testrepo" {
		t.Errorf("expected repo testowner/testrepo, got %s", entries[0].Repo)
	}
	if entries[0].Version != cache.CurrentVersion {
		t.Errorf("expected version %s, got %s", cache.CurrentVersion, entries[0].Version)
	}

	// Test cache path
	buf := new(bytes.Buffer)
	cachePathCmd.SetOut(buf)
	cachePathCmd.Run(cachePathCmd, []string{})

	// Test cache clean
	deleted, err := cm.Delete("testowner/testrepo")
	if err != nil || !deleted {
		t.Errorf("failed to delete testowner/testrepo: %v", err)
	}
}

func TestCacheStatusAndMigrateExecution(t *testing.T) {
	tempDir := t.TempDir()
	cm := cache.NewCacheManagerWithDir(tempDir)

	// Create legacy v0 file
	legacyJSON := `{"repo":"old/repo","prs":[{"number":10,"title":"Old","state":"OPEN"}]}`
	legacyPath := filepath.Join(tempDir, "old_repo.json")
	if err := os.WriteFile(legacyPath, []byte(legacyJSON), 0o600); err != nil {
		t.Fatalf("failed to write legacy cache: %v", err)
	}

	// Create corrupt file
	corruptPath := filepath.Join(tempDir, "corrupt_repo.json")
	if err := os.WriteFile(corruptPath, []byte(`bad-json`), 0o600); err != nil {
		t.Fatalf("failed to write corrupt cache: %v", err)
	}

	entries, err := cm.ListEntries()
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Test dry-run migration
	migratedCount, err := cm.MigrateAll(true)
	if err != nil {
		t.Fatalf("dry-run MigrateAll failed: %v", err)
	}
	if migratedCount != 1 {
		t.Errorf("expected 1 migratable entry in dry run, got %d", migratedCount)
	}

	// Test actual migration
	migratedCount, err = cm.MigrateAll(false)
	if err != nil {
		t.Fatalf("actual MigrateAll failed: %v", err)
	}
	if migratedCount != 1 {
		t.Errorf("expected 1 migrated entry, got %d", migratedCount)
	}

	// Verify migrated file is now current
	entries, _ = cm.ListEntries()
	var oldRepoEntry *cache.CacheEntryInfo
	for _, e := range entries {
		if e.Repo == "old/repo" {
			oldRepoEntry = &e
			break
		}
	}
	if oldRepoEntry == nil || oldRepoEntry.Version != cache.CurrentVersion || oldRepoEntry.Status != cache.StatusCurrent {
		t.Errorf("expected old/repo to be current %s, got %+v", cache.CurrentVersion, oldRepoEntry)
	}

	// Test ClearCorrupt
	clearedCorrupt, err := cm.ClearCorrupt()
	if err != nil {
		t.Fatalf("failed ClearCorrupt: %v", err)
	}
	if clearedCorrupt != 1 {
		t.Errorf("expected 1 corrupt file cleared, got %d", clearedCorrupt)
	}
}
