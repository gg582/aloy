package resolver

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/snowmerak/aloy/internal/git"
	"github.com/snowmerak/aloy/internal/model"
	"github.com/snowmerak/aloy/internal/parser"
)

var cmakeProjectNameRe = regexp.MustCompile(`(?i)^\s*project\s*\(\s*(\S+)`)

// extractCMakeProjectName reads a CMakeLists.txt and returns the project() name.
func extractCMakeProjectName(cmakeListsPath string) string {
	f, err := os.Open(cmakeListsPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		if m := cmakeProjectNameRe.FindStringSubmatch(s.Text()); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

const ModulesDir = ".my_modules"

// ResolvedDep holds the final resolved information for a dependency.
type ResolvedDep struct {
	Name            string
	GitURL          string
	ResolvedVersion string
	CommitSHA       string
	ModuleDir       string // path relative to project root
	CMakeTarget     string // detected or overridden CMake project name
	IsAloyPackage   bool   // has project.yaml
	IsSystem        bool   // type: system
}

// ResolveGraph performs BFS dependency resolution starting from the root project.
// It clones repositories, resolves SemVer constraints, and returns a flat list of resolved deps.
func ResolveGraph(projectRoot string, cfg *model.ProjectConfig) ([]ResolvedDep, error) {
	modulesBase := filepath.Join(projectRoot, ModulesDir)
	if err := os.MkdirAll(modulesBase, 0755); err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", modulesBase, err)
	}

	// Collect all dependencies from all targets
	var allDeps []model.Dependency
	for _, target := range cfg.Targets {
		allDeps = append(allDeps, target.Dependencies...)
	}

	// Track resolved packages to detect duplicates
	resolved := make(map[string]*ResolvedDep)
	// BFS queue
	type queueItem struct {
		dep     model.Dependency
		fromPkg string
	}
	var queue []queueItem
	for _, d := range allDeps {
		queue = append(queue, queueItem{dep: d, fromPkg: "root"})
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		dep := item.dep

		dirName := dep.ModuleDir()

		// System dependencies don't need git
		if dep.Type == "system" {
			if _, exists := resolved[dirName]; !exists {
				resolved[dirName] = &ResolvedDep{
					Name:     dep.Name,
					IsSystem: true,
				}
			}
			continue
		}

		if dep.Git == "" {
			return nil, fmt.Errorf("dependency %q from %s: git URL is required for non-system packages", dep.Name, item.fromPkg)
		}

		destPath := filepath.Join(modulesBase, dirName)

		// Check for existing resolution
		if existing, exists := resolved[dirName]; exists {
			// Both want the same package; validate version compatibility
			if dep.Version != "" && existing.ResolvedVersion != "" {
				if IsMajorConflict(dep.Version, existing.ResolvedVersion) {
					return nil, fmt.Errorf(
						"major version conflict for %q: %s (from %s) vs %s (previously resolved)\n"+
							"  Hint: use 'alias' field to import both versions under different names",
						dep.Name, dep.Version, item.fromPkg, existing.ResolvedVersion,
					)
				}
			}
			continue
		}

		// Clone the repository
		fmt.Printf("  Fetching %s ...\n", dep.Name)
		if err := git.CloneFull(dep.Git, destPath); err != nil {
			return nil, fmt.Errorf("failed to clone %s: %w", dep.Name, err)
		}

		// Resolve version
		var resolvedVersion string
		var commitSHA string

		if dep.Version != "" {
			// Find best matching tag
			if err := git.FetchTags(destPath); err != nil {
				return nil, fmt.Errorf("failed to fetch tags for %s: %w", dep.Name, err)
			}
			tag, ver, err := git.FindBestTag(destPath, dep.Version)
			if err != nil {
				// Fallback to default branch if no tags
				branch, brErr := git.DefaultBranch(destPath)
				if brErr != nil {
					return nil, fmt.Errorf("no matching version for %s (%s) and no default branch: %w", dep.Name, dep.Version, err)
				}
				fmt.Printf("  Warning: no semver tags for %s, using branch %s\n", dep.Name, branch)
				if err := git.Checkout(destPath, branch); err != nil {
					return nil, err
				}
				resolvedVersion = branch
			} else {
				resolvedVersion = ver.String()
				if err := git.Checkout(destPath, tag); err != nil {
					return nil, err
				}
			}
		} else {
			// No version constraint — use default branch
			branch, err := git.DefaultBranch(destPath)
			if err != nil {
				return nil, fmt.Errorf("no default branch for %s: %w", dep.Name, err)
			}
			resolvedVersion = branch
		}

		sha, err := git.GetHeadSHA(destPath)
		if err != nil {
			return nil, err
		}
		commitSHA = sha

		// Check if this is an aloy package
		isAloy := false
		projectYamlPath := filepath.Join(destPath, parser.ProjectFileName)
		if _, err := os.Stat(projectYamlPath); err == nil {
			isAloy = true
		}

		// Detect CMake project name
		cmakeTarget := dep.Name
		if dep.CMakeTarget != "" {
			cmakeTarget = dep.CMakeTarget
		} else {
			cmakeListsPath := filepath.Join(destPath, "CMakeLists.txt")
			if detected := extractCMakeProjectName(cmakeListsPath); detected != "" {
				cmakeTarget = detected
			}
		}

		rd := &ResolvedDep{
			Name:            dep.Name,
			GitURL:          dep.Git,
			ResolvedVersion: resolvedVersion,
			CommitSHA:       commitSHA,
			ModuleDir:       dirName,
			CMakeTarget:     cmakeTarget,
			IsAloyPackage:   isAloy,
			IsSystem:        false,
		}
		resolved[dirName] = rd

		// If aloy package, recurse into its dependencies
		if isAloy {
			subCfg, err := parser.LoadProject(destPath)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s/project.yaml: %w", dep.Name, err)
			}
			for _, target := range subCfg.Targets {
				for _, subDep := range target.Dependencies {
					queue = append(queue, queueItem{dep: subDep, fromPkg: dep.Name})
				}
			}
		}
	}

	// Convert map to slice, sorted by name for deterministic output
	var result []ResolvedDep
	for _, rd := range resolved {
		result = append(result, *rd)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// BuildLockFile creates a LockFile from resolved dependencies.
func BuildLockFile(deps []ResolvedDep) *model.LockFile {
	lf := &model.LockFile{Version: 1}
	for _, d := range deps {
		lf.Packages = append(lf.Packages, model.LockedPackage{
			Name:            d.Name,
			GitURL:          d.GitURL,
			ResolvedVersion: d.ResolvedVersion,
			CommitSHA:       d.CommitSHA,
			CMakeTarget:     d.CMakeTarget,
			IsAloyPackage:   d.IsAloyPackage,
			IsSystem:        d.IsSystem,
		})
	}
	return lf
}
