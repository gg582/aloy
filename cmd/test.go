package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var testConfig string

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tests using CTest",
	Long:  "Builds the project and runs integration tests via CTest.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		if needsSync(dir) {
			fmt.Println("Running sync first...")
			if err := runSync(dir); err != nil {
				return err
			}
		}

		fmt.Printf("Building tests (%s)...\n", testConfig)
		cmakeCmd := exec.Command("cmake", "--build", "build", "--config", testConfig)
		cmakeCmd.Dir = dir
		cmakeCmd.Stdout = os.Stdout
		cmakeCmd.Stderr = os.Stderr
		if err := cmakeCmd.Run(); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}

		fmt.Println("Running CTest...")
		ctestCmd := exec.Command("ctest", "--test-dir", "build", "-C", testConfig, "--output-on-failure")
		ctestCmd.Dir = dir
		ctestCmd.Stdout = os.Stdout
		ctestCmd.Stderr = os.Stderr
		if err := ctestCmd.Run(); err != nil {
			return fmt.Errorf("tests failed: %w", err)
		}

		return nil
	},
}

func init() {
	testCmd.Flags().StringVar(&testConfig, "config", "Release", "Build configuration for tests")
	rootCmd.AddCommand(testCmd)
}
