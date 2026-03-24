package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/snowmerak/aloy/internal/cmake"
	"github.com/snowmerak/aloy/internal/parser"
	"github.com/snowmerak/aloy/internal/resolver"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Resolve dependencies, generate CMake, and configure the build",
	Long:  "Clones/updates dependencies, generates CMakeLists.txt files, and runs cmake -B build.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		return runSync(dir)
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(dir string) error {
	// 1. Load project config
	cfg, err := parser.LoadProject(dir)
	if err != nil {
		return fmt.Errorf("failed to load project.yaml: %w", err)
	}

	// 2. Resolve dependency graph
	fmt.Println("Resolving dependencies...")
	resolvedDeps, err := resolver.ResolveGraph(dir, cfg)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}
	fmt.Printf("  Resolved %d dependencies\n", len(resolvedDeps))

	// 3. Generate lock file
	lf := resolver.BuildLockFile(resolvedDeps)
	if err := parser.SaveLockFile(dir, lf); err != nil {
		return fmt.Errorf("failed to save lock file: %w", err)
	}

	// 4. Generate CMakeLists.txt for aloy sub-packages
	for _, dep := range resolvedDeps {
		if dep.IsSystem || !dep.IsAloyPackage {
			continue
		}
		modulePath := filepath.Join(dir, resolver.ModulesDir, dep.ModuleDir)
		fmt.Printf("  Generating CMake for %s...\n", dep.Name)
		if err := cmake.GenerateForModule(modulePath); err != nil {
			return fmt.Errorf("failed to generate CMake for %s: %w", dep.Name, err)
		}
	}

	// 5. Generate master CMakeLists.txt
	fmt.Println("Generating root CMakeLists.txt...")
	if err := cmake.GenerateMaster(dir, cfg, resolvedDeps); err != nil {
		return fmt.Errorf("failed to generate root CMakeLists.txt: %w", err)
	}

	// 6. Run cmake -B build
	fmt.Println("Configuring build...")
	cmakeCmd := exec.Command("cmake", "-B", "build", "-S", ".")
	cmakeCmd.Dir = dir
	cmakeCmd.Stdout = os.Stdout
	cmakeCmd.Stderr = os.Stderr
	if err := cmakeCmd.Run(); err != nil {
		return fmt.Errorf("cmake configuration failed: %w", err)
	}

	fmt.Println("Sync complete!")
	return nil
}
