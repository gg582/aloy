package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/snowmerak/aloy/internal/model"
	"github.com/snowmerak/aloy/internal/parser"
	"github.com/snowmerak/aloy/internal/resolver"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a dependency from the project",
	Long:  "Removes a dependency from project.yaml and deletes its source from .my_modules/.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := parser.LoadProject(dir)
		if err != nil {
			return fmt.Errorf("failed to load project.yaml: %w", err)
		}

		name := args[0]
		found := false

		for targetName, target := range cfg.Targets {
			// Filter out the dependency
			var newDepList []model.Dependency
			for _, dep := range target.Dependencies {
				if dep.Name == name || dep.Alias == name {
					found = true
					continue
				}
				newDepList = append(newDepList, dep)
			}

			if len(newDepList) != len(target.Dependencies) {
				target.Dependencies = newDepList
				cfg.Targets[targetName] = target
			}
		}

		if !found {
			return fmt.Errorf("dependency %q not found in any target", name)
		}

		// Save updated config
		if err := parser.SaveProject(dir, cfg); err != nil {
			return err
		}

		// Remove module directory
		modulePath := filepath.Join(dir, resolver.ModulesDir, name)
		if _, err := os.Stat(modulePath); err == nil {
			if err := os.RemoveAll(modulePath); err != nil {
				return fmt.Errorf("failed to remove %s: %w", modulePath, err)
			}
			fmt.Printf("Removed %s/%s/\n", resolver.ModulesDir, name)
		}

		// Regenerate lock file
		resolvedDeps, err := resolver.ResolveGraph(dir, cfg)
		if err != nil {
			return fmt.Errorf("failed to re-resolve dependencies: %w", err)
		}
		lf := resolver.BuildLockFile(resolvedDeps)
		if err := parser.SaveLockFile(dir, lf); err != nil {
			return err
		}

		fmt.Printf("Removed dependency %q\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
