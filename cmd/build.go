package cmd

import (
	"fmt"
	"os"
	"os/exec"

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
		fmt.Printf("Building (%s)...\n", buildConfig)
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
	buildDir := dir + "/build"
	cmakeLists := dir + "/CMakeLists.txt"
	projectYaml := dir + "/project.yaml"

	// No build dir or no CMakeLists.txt → need sync
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		return true
	}
	if _, err := os.Stat(cmakeLists); os.IsNotExist(err) {
		return true
	}

	// If project.yaml is newer than CMakeLists.txt → need sync
	yamlInfo, err := os.Stat(projectYaml)
	if err != nil {
		return true
	}
	cmakeInfo, err := os.Stat(cmakeLists)
	if err != nil {
		return true
	}
	return yamlInfo.ModTime().After(cmakeInfo.ModTime())
}
