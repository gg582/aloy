package cmake

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// ScanSources resolves glob patterns (including **) relative to basePath
// and returns a sorted list of matched files as relative paths from basePath.
func ScanSources(patterns []string, basePath string) ([]string, error) {
	seen := make(map[string]bool)
	var result []string

	for _, pattern := range patterns {
		fullPattern := filepath.Join(basePath, filepath.FromSlash(pattern))
		// Use doublestar for ** support
		matches, err := doublestar.FilepathGlob(fullPattern)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			// Only include regular files
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			rel, err := filepath.Rel(basePath, match)
			if err != nil {
				continue
			}
			// Normalize to forward slashes for CMake
			rel = filepath.ToSlash(rel)
			if !seen[rel] {
				seen[rel] = true
				result = append(result, rel)
			}
		}
	}

	sort.Strings(result)
	return result, nil
}

// IsCppSource returns true if the file has a C++ source extension.
func IsCppSource(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".cpp", ".cxx", ".cc", ".c++", ".c":
		return true
	}
	return false
}
