package git

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GlobalCacheDir returns the root directory for aloy's global git cache.
func GlobalCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aloy", "cache"), nil
}

// GetCachePath returns the unique path for a bare cloned repository based on its URL.
func GetCachePath(gitURL string) (string, error) {
	base, err := GlobalCacheDir()
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	hasher.Write([]byte(gitURL))
	hashStr := hex.EncodeToString(hasher.Sum(nil))[:16]

	parts := strings.Split(strings.TrimSuffix(gitURL, "/"), "/")
	namePart := "repo"
	if len(parts) > 0 {
		namePart = strings.TrimSuffix(parts[len(parts)-1], ".git")
	}

	dirName := fmt.Sprintf("%s-%s.git", namePart, hashStr)
	return filepath.Join(base, dirName), nil
}

// FetchCache ensures the bare repository exists and fetches latest tags/branches.
func FetchCache(gitURL string) (string, error) {
	cachePath, err := GetCachePath(gitURL)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Dir(cachePath), 0755)
		if err != nil {
			return "", err
		}

		cmd := exec.Command("git", "clone", "--bare", gitURL, cachePath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to clone bare repo: %s\n%v", string(out), err)
		}
	} else {
		cmd := exec.Command("git", "fetch", "--all", "--tags", "--force")
		cmd.Dir = cachePath
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to fetch bare repo: %s\n%v", string(out), err)
		}
	}

	return cachePath, nil
}

// CloneFromCache creates a quick local clone referencing the global bare cache.
func CloneFromCache(cachePath, gitURL, destPath string) error {
	if _, err := os.Stat(destPath); err == nil {
		return nil // already exists
	}

	cmd := exec.Command("git", "clone", "--reference", cachePath, gitURL, destPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone from cache: %s\n%v", string(out), err)
	}
	return nil
}
