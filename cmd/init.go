package cmd

import (
	"fmt"
	"os"

	"github.com/snowmerak/aloy/internal/scaffold"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new aloy project",
	Long:  "Creates project.yaml, src/, include/ directories, and a starter main.cpp.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		return scaffold.Init(dir)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
