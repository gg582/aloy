package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/snowmerak/aloy/internal/resolver"
	"github.com/spf13/cobra"
)

var cleanAll bool

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove build artifacts",
	Long:  "Deletes the build/ directory. With --all, also removes .my_modules/.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		buildDir := filepath.Join(dir, "build")
		if err := os.RemoveAll(buildDir); err != nil {
			return fmt.Errorf("failed to remove build/: %w", err)
		}
		fmt.Println("Removed build/")

		if cleanAll {
			modulesDir := filepath.Join(dir, resolver.ModulesDir)
			if err := os.RemoveAll(modulesDir); err != nil {
				return fmt.Errorf("failed to remove %s: %w", resolver.ModulesDir, err)
			}
			fmt.Printf("Removed %s/\n", resolver.ModulesDir)

			cmakeLists := filepath.Join(dir, "CMakeLists.txt")
			os.Remove(cmakeLists)
			fmt.Println("Removed CMakeLists.txt")
		}

		fmt.Println("Clean complete!")
		return nil
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Also remove .my_modules/ and generated CMakeLists.txt")
	rootCmd.AddCommand(cleanCmd)
}
