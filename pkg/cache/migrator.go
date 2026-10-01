package cache

import (
	"encoding/json"
	"fmt"

	"github.com/brad/gh-pr-pro/pkg/version"
)

// CacheStatus represents the lifecycle and compatibility state of a cache entry on disk.
type CacheStatus string

const (
	// StatusCurrent indicates the cache file matches the package CurrentVersion.
	StatusCurrent CacheStatus = "current"
	// StatusOutdated indicates the cache file has an older version that can be migrated or refreshed.
	StatusOutdated CacheStatus = "outdated"
	// StatusFuture indicates the cache file was written by a newer version of gh-pr-pro.
	StatusFuture CacheStatus = "future"
	// StatusCorrupt indicates the cache file cannot be deserialized as valid JSON.
	StatusCorrupt CacheStatus = "corrupt"
)

// MigrationFunc transforms raw JSON bytes of a cache file from a source version to a target version.
type MigrationFunc func(raw []byte) ([]byte, error)

// Migrator coordinates schema inspection and sequential migration routines between cache versions.
type Migrator struct {
	migrations map[string]MigrationFunc
}

// NewMigrator initializes a Migrator pre-populated with standard version migration routines.
func NewMigrator() *Migrator {
	m := &Migrator{
		migrations: make(map[string]MigrationFunc),
	}
	// Register legacy v0.0.0 (unversioned) -> CurrentVersion migration
	m.Register(LegacyVersion, migrateLegacyToCurrent)
	return m
}

// Register adds a migration function for transitioning from a specific source version to the target.
func (m *Migrator) Register(fromVersion string, fn MigrationFunc) {
	norm := version.Normalize(fromVersion)
	m.migrations[norm] = fn
}

// CanMigrate reports whether a migration pathway exists from fromVersion to CurrentVersion.
func (m *Migrator) CanMigrate(fromVersion string) bool {
	norm := version.Normalize(fromVersion)
	if version.Compare(norm, CurrentVersion) >= 0 {
		return false
	}
	_, exists := m.migrations[norm]
	return exists
}

// Migrate executes registered migration logic to upgrade data from fromVersion to toVersion.
func (m *Migrator) Migrate(raw []byte, fromVersion, toVersion string) ([]byte, error) {
	normFrom := version.Normalize(fromVersion)
	normTo := version.Normalize(toVersion)

	cmp := version.Compare(normFrom, normTo)
	if cmp == 0 {
		return raw, nil
	}
	if cmp > 0 {
		return nil, fmt.Errorf("cannot downgrade cache version from %s to %s", normFrom, normTo)
	}

	fn, exists := m.migrations[normFrom]
	if !exists {
		return nil, fmt.Errorf("missing migration step from version %s to %s: %w", normFrom, normTo, ErrCacheRequiresRefresh)
	}

	upgraded, err := fn(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate cache from version %s to %s: %w", normFrom, normTo, err)
	}

	return upgraded, nil
}

// rawEnvelope is used for lightweight version sniffing without decoding entire PR collections.
type rawEnvelope struct {
	Version       string `json:"version,omitempty"`
	SchemaVersion *int   `json:"schema_version,omitempty"`
	Repo          string `json:"repo"`
}

// DetectVersion inspects raw cache JSON to determine the semantic version and repository name.
// Unversioned legacy cache files return LegacyVersion ("v0.0.0").
func DetectVersion(data []byte) (ver string, repo string, status CacheStatus, err error) {
	var env rawEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return "", "", StatusCorrupt, fmt.Errorf("%w: %v", ErrCacheCorrupt, err)
	}

	ver = LegacyVersion
	if env.Version != "" {
		ver = version.Normalize(env.Version)
	} else if env.SchemaVersion != nil {
		// Map previous integer schema_version to semantic versions
		if *env.SchemaVersion >= 1 {
			ver = CurrentVersion
		} else {
			ver = LegacyVersion
		}
	}

	cmp := version.Compare(ver, CurrentVersion)
	switch {
	case cmp == 0:
		return ver, env.Repo, StatusCurrent, nil
	case cmp < 0:
		return ver, env.Repo, StatusOutdated, nil
	default:
		return ver, env.Repo, StatusFuture, nil
	}
}

// migrateLegacyToCurrent upgrades a legacy or older cache payload to the current package version.
func migrateLegacyToCurrent(raw []byte) ([]byte, error) {
	var cached CachedData
	if err := json.Unmarshal(raw, &cached); err != nil {
		return nil, err
	}

	cached.Version = CurrentVersion
	return json.MarshalIndent(cached, "", "  ")
}
