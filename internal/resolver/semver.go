package resolver

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// IntersectConstraints computes the intersection of two SemVer constraints
// against a list of candidate versions. It returns the highest version that
// satisfies both constraints, or an error if no version satisfies both.
func IntersectConstraints(c1, c2 string, candidates []*semver.Version) (*semver.Version, error) {
	con1, err := semver.NewConstraint(c1)
	if err != nil {
		return nil, fmt.Errorf("invalid constraint %q: %w", c1, err)
	}
	con2, err := semver.NewConstraint(c2)
	if err != nil {
		return nil, fmt.Errorf("invalid constraint %q: %w", c2, err)
	}

	var best *semver.Version
	for _, v := range candidates {
		if con1.Check(v) && con2.Check(v) {
			if best == nil || v.GreaterThan(best) {
				best = v
			}
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no version satisfies both %q and %q", c1, c2)
	}
	return best, nil
}

// MergeConstraints takes multiple SemVer constraint strings and returns the
// highest version from candidates that satisfies all of them.
func MergeConstraints(constraints []string, candidates []*semver.Version) (*semver.Version, error) {
	if len(constraints) == 0 {
		return nil, fmt.Errorf("no constraints provided")
	}

	var parsed []*semver.Constraints
	for _, c := range constraints {
		con, err := semver.NewConstraint(c)
		if err != nil {
			return nil, fmt.Errorf("invalid constraint %q: %w", c, err)
		}
		parsed = append(parsed, con)
	}

	var best *semver.Version
	for _, v := range candidates {
		ok := true
		for _, con := range parsed {
			if !con.Check(v) {
				ok = false
				break
			}
		}
		if ok {
			if best == nil || v.GreaterThan(best) {
				best = v
			}
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no version satisfies all constraints %v", constraints)
	}
	return best, nil
}

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
