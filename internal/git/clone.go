package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Clone performs a git clone of the given URL into dest.
// If dest already exists, it does nothing.
func Clone(url, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return nil // already cloned
	}
	cmd := exec.Command("git", "clone", "--depth", "1", "--recurse-submodules", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", url, err)
	}
	return nil
}

// CloneFull performs a full (non-shallow) clone to allow tag enumeration.
func CloneFull(url, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	cmd := exec.Command("git", "clone", "--recurse-submodules", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", url, err)
	}
	return nil
}

// FetchTags fetches all tags for a repository.
func FetchTags(repoPath string) error {
	cmd := exec.Command("git", "fetch", "--tags")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch --tags in %s: %w", repoPath, err)
	}
	return nil
}

// Checkout performs a detached HEAD checkout to the given ref (tag, SHA, branch).
func Checkout(repoPath, ref string) error {
	cmd := exec.Command("git", "checkout", "--detach", ref)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout %s in %s: %w", ref, repoPath, err)
	}
	return nil
}

// GetHeadSHA returns the current HEAD commit SHA.
func GetHeadSHA(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD in %s: %w", repoPath, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// DefaultBranch returns the default branch name (main or master).
func DefaultBranch(repoPath string) (string, error) {
	for _, branch := range []string{"main", "master"} {
		cmd := exec.Command("git", "rev-parse", "--verify", branch)
		cmd.Dir = repoPath
		if err := cmd.Run(); err == nil {
			return branch, nil
		}
	}
	return "", fmt.Errorf("no default branch (main/master) found in %s", repoPath)
}

// Unshallow converts a shallow clone to a full clone (needed for tag listing).
func Unshallow(repoPath string) error {
	cmd := exec.Command("git", "fetch", "--unshallow")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// --unshallow fails on non-shallow repos; ignore that error
	_ = cmd.Run()
	return nil
}
