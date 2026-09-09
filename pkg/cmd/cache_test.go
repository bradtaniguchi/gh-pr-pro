package cmd

import (
	"bytes"
	"testing"
	"time"

	"github.com/brad/gh-pr-pro/pkg/cache"
	"github.com/brad/gh-pr-pro/pkg/metrics"
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
