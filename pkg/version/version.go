// Package version defines the semantic release version of the gh-pr-pro extension
// and provides lightweight Semantic Versioning (SemVer) parsing and comparison utilities.
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is the current semantic release version of the gh-pr-pro package.
// For official distribution builds, this value is injected at compile time via -ldflags.
var Version = "v0.3.0"

// SemVer represents a parsed semantic version components (MAJOR.MINOR.PATCH[-PRERELEASE]).
type SemVer struct {
	Major int
	Minor int
	Patch int
	Pre   string
}

// Normalize ensures a version string starts with a lowercase 'v'.
func Normalize(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "v0.0.0"
	}
	if !strings.HasPrefix(v, "v") && !strings.HasPrefix(v, "V") {
		return "v" + v
	}
	return "v" + strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
}

// Parse converts a SemVer string like "v0.2.0", "0.2.0", or "v1.0.0-rc.1" into a typed SemVer struct.
func Parse(s string) (SemVer, error) {
	norm := strings.TrimPrefix(Normalize(s), "v")

	// Split pre-release metadata if present
	var pre string
	if idx := strings.Index(norm, "-"); idx != -1 {
		pre = norm[idx+1:]
		norm = norm[:idx]
	}

	parts := strings.Split(norm, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return SemVer{}, fmt.Errorf("invalid semantic version: %q", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid major version in %q: %w", s, err)
	}

	minor := 0
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid minor version in %q: %w", s, err)
		}
	}

	patch := 0
	if len(parts) > 2 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid patch version in %q: %w", s, err)
		}
	}

	return SemVer{
		Major: major,
		Minor: minor,
		Patch: patch,
		Pre:   pre,
	}, nil
}

// String returns the canonical normalized SemVer string with leading 'v'.
func (s SemVer) String() string {
	res := fmt.Sprintf("v%d.%d.%d", s.Major, s.Minor, s.Patch)
	if s.Pre != "" {
		res += "-" + s.Pre
	}
	return res
}

// Compare compares two semantic version strings.
// Returns:
//
//	-1 if v1 < v2
//	 0 if v1 == v2
//	 1 if v1 > v2
func Compare(v1, v2 string) int {
	sv1, err1 := Parse(v1)
	sv2, err2 := Parse(v2)

	// If parsing fails for either, fall back to string comparison
	if err1 != nil || err2 != nil {
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	}

	if sv1.Major != sv2.Major {
		if sv1.Major < sv2.Major {
			return -1
		}
		return 1
	}

	if sv1.Minor != sv2.Minor {
		if sv1.Minor < sv2.Minor {
			return -1
		}
		return 1
	}

	if sv1.Patch != sv2.Patch {
		if sv1.Patch < sv2.Patch {
			return -1
		}
		return 1
	}

	// SemVer specification: a version with pre-release has lower precedence than one without.
	if sv1.Pre != sv2.Pre {
		if sv1.Pre == "" && sv2.Pre != "" {
			return 1 // release > pre-release
		}
		if sv1.Pre != "" && sv2.Pre == "" {
			return -1 // pre-release < release
		}
		// Both have pre-release tags, compare lexicographically
		if sv1.Pre < sv2.Pre {
			return -1
		}
		return 1
	}

	return 0
}
