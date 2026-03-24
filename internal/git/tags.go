package git

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// TagVersion pairs a git tag string with its parsed semver.
type TagVersion struct {
	Tag     string
	Version *semver.Version
}

// ListSemVerTags returns all semver-compatible tags sorted in ascending order.
func ListSemVerTags(repoPath string) ([]TagVersion, error) {
	cmd := exec.Command("git", "tag", "-l")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git tag -l in %s: %w", repoPath, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var tags []TagVersion
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip leading "v" or "V" prefix
		raw := line
		if strings.HasPrefix(raw, "v") || strings.HasPrefix(raw, "V") {
			raw = raw[1:]
		}
		v, err := semver.StrictNewVersion(raw)
		if err != nil {
			continue // skip non-semver tags
		}
		tags = append(tags, TagVersion{Tag: line, Version: v})
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Version.LessThan(tags[j].Version)
	})

	return tags, nil
}

// FindBestTag returns the highest version tag that satisfies the given SemVer constraint.
// If constraint is empty, it returns the highest tag overall.
// Returns the tag name and the parsed version, or an error if no matching tag is found.
func FindBestTag(repoPath, constraint string) (string, *semver.Version, error) {
	tags, err := ListSemVerTags(repoPath)
	if err != nil {
		return "", nil, err
	}
	if len(tags) == 0 {
		return "", nil, fmt.Errorf("no semver tags found in %s", repoPath)
	}

	if constraint == "" {
		best := tags[len(tags)-1]
		return best.Tag, best.Version, nil
	}

	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return "", nil, fmt.Errorf("invalid semver constraint %q: %w", constraint, err)
	}

	// Iterate from highest to lowest
	for i := len(tags) - 1; i >= 0; i-- {
		if c.Check(tags[i].Version) {
			return tags[i].Tag, tags[i].Version, nil
		}
	}

	return "", nil, fmt.Errorf("no tag satisfies constraint %q in %s", constraint, repoPath)
}

// GetTagSHA returns the commit SHA for a given tag.
func GetTagSHA(repoPath, tag string) (string, error) {
	// Dereference annotated tags to get the commit SHA
	cmd := exec.Command("git", "rev-list", "-n", "1", tag)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-list %s in %s: %w", tag, repoPath, err)
	}
	return strings.TrimSpace(string(out)), nil
}
