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

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download dependencies from lock file",
	Long:  "Clones exact versions from aloy.lock without generating CMake files. Useful for CI.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		lf, err := parser.LoadLockFile(dir)
		if err != nil {
			return fmt.Errorf("failed to load aloy.lock: %w", err)
		}
		if len(lf.Packages) == 0 {
			fmt.Println("No packages in lock file. Run 'aloy sync' first.")
			return nil
		}

		modulesBase := filepath.Join(dir, resolver.ModulesDir)
		if err := os.MkdirAll(modulesBase, 0755); err != nil {
			return err
		}

		for _, pkg := range lf.Packages {
			dest := filepath.Join(modulesBase, pkg.Name)
			fmt.Printf("  Downloading %s @ %s ...\n", pkg.Name, pkg.CommitSHA[:8])
			if err := git.CloneFull(pkg.GitURL, dest); err != nil {
				return fmt.Errorf("failed to clone %s: %w", pkg.Name, err)
			}
			if err := git.Checkout(dest, pkg.CommitSHA); err != nil {
				return fmt.Errorf("failed to checkout %s @ %s: %w", pkg.Name, pkg.CommitSHA, err)
			}
		}

		fmt.Printf("Downloaded %d packages.\n", len(lf.Packages))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}
