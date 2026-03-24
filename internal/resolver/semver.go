package resolver

import (
	"strings"

	"github.com/Masterminds/semver/v3"
)

// IsMajorConflict checks if two constraint strings require different major versions
// by parsing the major version directly from the constraint prefix.
func IsMajorConflict(c1, c2 string) bool {
	m1, ok1 := extractMajor(c1)
	m2, ok2 := extractMajor(c2)
	if !ok1 || !ok2 {
		return false // can't determine, assume no conflict
	}
	return m1 != m2
}

// extractMajor parses the major version number from a SemVer constraint string.
// It trims common prefixes (^, ~, >=, >, =, v) and returns the major component.
func extractMajor(constraint string) (uint64, bool) {
	s := strings.TrimSpace(constraint)
	// Strip constraint operators
	for _, prefix := range []string{"^", "~", ">=", ">", "<=", "<", "="} {
		s = strings.TrimPrefix(s, prefix)
	}
	s = strings.TrimSpace(s)
	// Strip optional "v" prefix
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	if s == "" {
		return 0, false
	}
	v, err := semver.StrictNewVersion(s)
	if err != nil {
		// Try parsing just the major number
		parts := strings.SplitN(s, ".", 2)
		v, err = semver.StrictNewVersion(parts[0] + ".0.0")
		if err != nil {
			return 0, false
		}
	}
	return v.Major(), true
}
