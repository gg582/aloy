package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/snowmerak/aloy/internal/git"
	"github.com/snowmerak/aloy/internal/parser"
	"github.com/snowmerak/aloy/internal/resolver"
	"github.com/spf13/cobra"
)

var updateDryRun bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update dependencies to latest versions within SemVer range",
	Long:  "Fetches latest tags and updates aloy.lock to the newest compatible versions.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := parser.LoadProject(dir)
		if err != nil {
			return fmt.Errorf("failed to load project.yaml: %w", err)
		}

		modulesBase := filepath.Join(dir, resolver.ModulesDir)
		var updated int

		for _, target := range cfg.Targets {
			for _, dep := range target.Dependencies {
				if dep.Type == "system" || dep.Git == "" {
					continue
				}

				dirName := dep.ModuleDir()
				destPath := filepath.Join(modulesBase, dirName)
				if _, err := os.Stat(destPath); os.IsNotExist(err) {
					fmt.Printf("  %s: not cloned, skipping (run sync first)\n", dep.Name)
					continue
				}

				if err := git.FetchTags(destPath); err != nil {
					fmt.Printf("  %s: failed to fetch tags: %v\n", dep.Name, err)
					continue
				}

				constraint := dep.Version
				tag, ver, err := git.FindBestTag(destPath, constraint)
				if err != nil {
					fmt.Printf("  %s: no matching tag: %v\n", dep.Name, err)
					continue
				}

				currentSHA, _ := git.GetHeadSHA(destPath)
				tagSHA, _ := git.GetTagSHA(destPath, tag)

				if currentSHA == tagSHA {
					fmt.Printf("  %s: already at %s\n", dep.Name, ver)
					continue
				}

				if updateDryRun {
					fmt.Printf("  %s: would update to %s (%s)\n", dep.Name, ver, tag)
				} else {
					if err := git.Checkout(destPath, tag); err != nil {
						fmt.Printf("  %s: failed to checkout %s: %v\n", dep.Name, tag, err)
						continue
					}
					fmt.Printf("  %s: updated to %s (%s)\n", dep.Name, ver, tag)
				}
				updated++
			}
		}

		if !updateDryRun && updated > 0 {
			// Re-run resolution to update lock file
			resolvedDeps, err := resolver.ResolveGraph(dir, cfg)
			if err != nil {
				return err
			}
			lf := resolver.BuildLockFile(resolvedDeps)
			if err := parser.SaveLockFile(dir, lf); err != nil {
				return err
			}
			fmt.Printf("Updated lock file with %d changes.\n", updated)
		} else if updateDryRun {
			fmt.Printf("Dry run: %d packages would be updated.\n", updated)
		} else {
			fmt.Println("All dependencies are up to date.")
		}

		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Show what would be updated without making changes")
	rootCmd.AddCommand(updateCmd)
}
