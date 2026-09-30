package cache

import "errors"

var (
	// ErrCacheRequiresRefresh indicates that a cache file has an older schema version
	// that cannot be migrated locally and requires a fresh re-synchronization from the GitHub API.
	ErrCacheRequiresRefresh = errors.New("cache requires fresh re-fetch from GitHub API")

	// ErrCacheFutureVersion indicates that a cache file was written by a newer version
	// of gh-pr-pro with a schema version greater than CurrentSchemaVersion.
	ErrCacheFutureVersion = errors.New("cache was created by a newer version of gh-pr-pro")

	// ErrCacheCorrupt indicates that a cache file exists but cannot be parsed as valid JSON.
	ErrCacheCorrupt = errors.New("cache file is corrupted or contains invalid JSON")
)
