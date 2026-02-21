package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var projectDir string

var rootCmd = &cobra.Command{
	Use:   "lxc-dev-manager",
	Short: "Manage LXC containers for local development",
	Long: `lxc-dev-manager is a CLI tool to manage LXC containers for local development.

It provides easy container lifecycle management and port proxying to make
containers feel like local services.`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&projectDir, "project-dir", "C", "",
		"path to project directory (default: current directory)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
