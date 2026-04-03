package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/snowmerak/aloy/internal/parser"
	"github.com/spf13/cobra"
)

var (
	buildConfig   string
	buildParallel int
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the project",
	Long:  "Runs sync if needed, then builds using cmake --build.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		// Check if sync is needed
		if needsSync(dir) {
			fmt.Println("Running sync first...")
			if err := runSync(dir); err != nil {
				return err
			}
		}

		// Build
		cfg, err := parser.LoadProject(dir)
		if err != nil {
			return fmt.Errorf("failed to load project.yaml: %w", err)
		}
		buildSystem := cfg.BuildSystem
		if buildSystem == "" {
			buildSystem = "cmake"
		}

		fmt.Printf("Building (%s)...\n", buildConfig)
		if buildSystem == "makefile" {
			makeArgs := []string{}
			if buildParallel > 0 {
				makeArgs = []string{fmt.Sprintf("-j%d", buildParallel)}
			}
			makeCmd := exec.Command("make", makeArgs...)
			makeCmd.Dir = dir
			makeCmd.Stdout = os.Stdout
			makeCmd.Stderr = os.Stderr
			if err := makeCmd.Run(); err != nil {
				return fmt.Errorf("build failed: %w", err)
			}
		} else {
			buildArgs := []string{"--build", "build", "--config", buildConfig}
			if buildParallel > 0 {
				buildArgs = append(buildArgs, "--parallel", fmt.Sprintf("%d", buildParallel))
			}
			cmakeCmd := exec.Command("cmake", buildArgs...)
			cmakeCmd.Dir = dir
			cmakeCmd.Stdout = os.Stdout
			cmakeCmd.Stderr = os.Stderr
			if err := cmakeCmd.Run(); err != nil {
				return fmt.Errorf("build failed: %w", err)
			}
		}

		fmt.Println("Build complete!")
		return nil
	},
}

func init() {
	buildCmd.Flags().StringVar(&buildConfig, "config", "Release", "Build configuration (Debug, Release, RelWithDebInfo, MinSizeRel)")
	buildCmd.Flags().IntVarP(&buildParallel, "parallel", "j", 0, "Number of parallel build jobs")
	rootCmd.AddCommand(buildCmd)
}

// needsSync checks whether sync needs to be run before building.
func needsSync(dir string) bool {
	buildDir := filepath.Join(dir, "build")
	cmakeLists := filepath.Join(dir, "CMakeLists.txt")
	makefilePath := filepath.Join(dir, "Makefile")
	projectYaml := filepath.Join(dir, "project.yaml")

	buildSystem := "cmake"
	if cfg, err := parser.LoadProject(dir); err == nil && cfg.BuildSystem != "" {
		buildSystem = cfg.BuildSystem
	}

	// Missing generated build files → need sync
	switch buildSystem {
	case "makefile":
		if _, err := os.Stat(makefilePath); os.IsNotExist(err) {
			return true
		}
	default:
		// No build dir or no CMakeLists.txt → need sync
		if _, err := os.Stat(buildDir); os.IsNotExist(err) {
			return true
		}
		if _, err := os.Stat(cmakeLists); os.IsNotExist(err) {
			return true
		}
	}

	// If project.yaml is newer than generated file → need sync
	yamlInfo, err := os.Stat(projectYaml)
	if err != nil {
		return true
	}
	generatedPath := cmakeLists
	if buildSystem == "makefile" {
		generatedPath = makefilePath
	}
	generatedInfo, err := os.Stat(generatedPath)
	if err != nil {
		return true
	}
	return yamlInfo.ModTime().After(generatedInfo.ModTime())
}
