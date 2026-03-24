package resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/snowmerak/aloy/internal/git"
	"github.com/snowmerak/aloy/internal/model"
	"github.com/snowmerak/aloy/internal/parser"
)

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

// resolveSingle clones a single dependency, resolves its version, and returns the result.
// This function is safe to call from multiple goroutines (operates on its own directory).
func resolveSingle(dep model.Dependency, modulesBase string) (*ResolvedDep, error) {
	dirName := dep.ModuleDir()
	destPath := filepath.Join(modulesBase, dirName)

	fmt.Printf("  Fetching %s ...\n", dep.Name)
	if err := git.CloneFull(dep.Git, destPath); err != nil {
		return nil, fmt.Errorf("failed to clone %s: %w", dep.Name, err)
	}

	var resolvedVersion string

	if dep.Version != "" {
		if err := git.FetchTags(destPath); err != nil {
			return nil, fmt.Errorf("failed to fetch tags for %s: %w", dep.Name, err)
		}
		tag, ver, err := git.FindBestTag(destPath, dep.Version)
		if err != nil {
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

	isAloy := false
	projectYamlPath := filepath.Join(destPath, parser.ProjectFileName)
	if _, err := os.Stat(projectYamlPath); err == nil {
		isAloy = true
	}

	cmakeTarget := dep.Name
	if dep.CMakeTarget != "" {
		cmakeTarget = dep.CMakeTarget
	}

	return &ResolvedDep{
		Name:            dep.Name,
		GitURL:          dep.Git,
		ResolvedVersion: resolvedVersion,
		CommitSHA:       sha,
		ModuleDir:       dirName,
		CMakeTarget:     cmakeTarget,
		IsAloyPackage:   isAloy,
		IsSystem:        false,
	}, nil
}

// ResolveGraph performs BFS dependency resolution starting from the root project.
// Dependencies at the same BFS level are cloned in parallel.
func ResolveGraph(projectRoot string, cfg *model.ProjectConfig) ([]ResolvedDep, error) {
	modulesBase := filepath.Join(projectRoot, ModulesDir)
	if err := os.MkdirAll(modulesBase, 0755); err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", modulesBase, err)
	}

	// Collect all dependencies from all targets
	type queueItem struct {
		dep     model.Dependency
		fromPkg string
	}
	var queue []queueItem
	for _, target := range cfg.Targets {
		for _, d := range target.Dependencies {
			queue = append(queue, queueItem{dep: d, fromPkg: "root"})
		}
	}

	// Track resolved packages to detect duplicates
	resolved := make(map[string]*ResolvedDep)

	// BFS level-by-level: each level is processed in parallel
	for len(queue) > 0 {
		currentLevel := queue
		queue = nil

		// 1. Filter: handle system deps and duplicates synchronously, collect git deps to fetch
		type fetchJob struct {
			item    queueItem
			dirName string
		}
		var toFetch []fetchJob
		seen := make(map[string]bool) // dedup within same level

		for _, item := range currentLevel {
			dep := item.dep
			dirName := dep.ModuleDir()

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

			if existing, exists := resolved[dirName]; exists {
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

			if seen[dirName] {
				continue
			}
			seen[dirName] = true
			toFetch = append(toFetch, fetchJob{item: item, dirName: dirName})
		}

		if len(toFetch) == 0 {
			continue
		}

		// 2. Clone & resolve in parallel
		type fetchResult struct {
			rd  *ResolvedDep
			dep model.Dependency
			err error
		}
		results := make([]fetchResult, len(toFetch))
		var wg sync.WaitGroup
		wg.Add(len(toFetch))

		for i, job := range toFetch {
			go func(idx int, j fetchJob) {
				defer wg.Done()
				rd, err := resolveSingle(j.item.dep, modulesBase)
				results[idx] = fetchResult{rd: rd, dep: j.item.dep, err: err}
			}(i, job)
		}
		wg.Wait()

		// 3. Collect results and enqueue sub-dependencies
		for _, r := range results {
			if r.err != nil {
				return nil, r.err
			}
			resolved[r.rd.ModuleDir] = r.rd

			if r.rd.IsAloyPackage {
				destPath := filepath.Join(modulesBase, r.rd.ModuleDir)
				subCfg, err := parser.LoadProject(destPath)
				if err != nil {
					return nil, fmt.Errorf("failed to parse %s/project.yaml: %w", r.dep.Name, err)
				}
				for _, target := range subCfg.Targets {
					for _, subDep := range target.Dependencies {
						queue = append(queue, queueItem{dep: subDep, fromPkg: r.dep.Name})
					}
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
