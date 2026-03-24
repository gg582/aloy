package resolver

import (
	"fmt"

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

// IsMajorConflict checks if two constraint strings require different major versions.
func IsMajorConflict(c1, c2 string) bool {
	con1, err1 := semver.NewConstraint(c1)
	con2, err2 := semver.NewConstraint(c2)
	if err1 != nil || err2 != nil {
		return false
	}

	// Test a range of major versions to detect conflict
	for major := 0; major < 100; major++ {
		v, _ := semver.NewVersion(fmt.Sprintf("%d.0.0", major))
		c1Match := con1.Check(v)
		c2Match := con2.Check(v)
		if c1Match && c2Match {
			return false // found a common major
		}
	}
	return true
}
