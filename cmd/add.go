package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/snowmerak/aloy/internal/model"
	"github.com/snowmerak/aloy/internal/parser"
	"github.com/spf13/cobra"
)

var (
	addVersion   string
	addAlias     string
	addSystem    bool
	addCMakeOpts []string
	addTarget    string
)

var addCmd = &cobra.Command{
	Use:   "add <git_url>",
	Short: "Add a dependency to the project",
	Long:  "Adds a git-based or system dependency to project.yaml.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := parser.LoadProject(dir)
		if err != nil {
			return fmt.Errorf("failed to load project.yaml (run 'aloy init' first): %w", err)
		}

		gitURL := args[0]
		name := inferPackageName(gitURL)

		dep := model.Dependency{
			Name:    name,
			Version: addVersion,
			Alias:   addAlias,
		}

		if addSystem {
			dep.Type = "system"
			dep.Name = gitURL // for system, the arg is the package name, not a URL
			dep.Git = ""
		} else {
			dep.Git = gitURL
		}

		// Parse cmake options (key=value)
		if len(addCMakeOpts) > 0 {
			dep.CMakeOptions = make(map[string]string)
			for _, opt := range addCMakeOpts {
				parts := strings.SplitN(opt, "=", 2)
				if len(parts) == 2 {
					dep.CMakeOptions[parts[0]] = parts[1]
				}
			}
		}

		// Determine target to add dep to
		targetName := addTarget
		if targetName == "" {
			// Use the first target
			for k := range cfg.Targets {
				targetName = k
				break
			}
		}

		if targetName == "" {
			return fmt.Errorf("no targets defined in project.yaml; add a target first")
		}

		target, ok := cfg.Targets[targetName]
		if !ok {
			return fmt.Errorf("target %q not found in project.yaml", targetName)
		}

		// Check for duplicates
		for _, existing := range target.Dependencies {
			if existing.Name == dep.Name || (dep.Alias != "" && existing.Alias == dep.Alias) {
				return fmt.Errorf("dependency %q already exists in target %q", dep.Name, targetName)
			}
		}

		target.Dependencies = append(target.Dependencies, dep)
		cfg.Targets[targetName] = target

		if err := parser.SaveProject(dir, cfg); err != nil {
			return err
		}

		fmt.Printf("Added %s to target %s\n", dep.Name, targetName)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addVersion, "version", "v", "", "SemVer constraint (e.g., ^1.2.0)")
	addCmd.Flags().StringVarP(&addAlias, "alias", "a", "", "Alias name for the dependency")
	addCmd.Flags().BoolVar(&addSystem, "system", false, "Mark as system dependency (find_package)")
	addCmd.Flags().StringArrayVar(&addCMakeOpts, "cmake-option", nil, "CMake option in key=value format")
	addCmd.Flags().StringVarP(&addTarget, "target", "t", "", "Target to add the dependency to (default: first target)")
	rootCmd.AddCommand(addCmd)
}

func inferPackageName(gitURL string) string {
	// Handle SSH URLs: git@github.com:user/repo.git
	if idx := strings.LastIndex(gitURL, ":"); idx != -1 && !strings.Contains(gitURL, "://") {
		gitURL = gitURL[idx+1:]
	}

	name := filepath.Base(gitURL)
	name = strings.TrimSuffix(name, ".git")
	return name
}
