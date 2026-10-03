package cmd

import (
	"testing"

	"github.com/brad/gh-pr-pro/pkg/cache"
)

// setupIsolatedTestHome configures an isolated temporary home directory across
// POSIX (HOME) and Windows (USERPROFILE) environments, returning the path and an
// initialized CacheManager targeting the isolated directory.
func setupIsolatedTestHome(t *testing.T) (string, *cache.CacheManager) {
	t.Helper()
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	cm, err := cache.NewCacheManager()
	if err != nil {
		t.Fatalf("failed to create isolated cache manager: %v", err)
	}
	return tempHome, cm
}
