package cmd

import (
	"fmt"

	"lxc-dev-manager/internal/lxc"
	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down <name>",
	Short: "Stop a container",
	Long: `Stop a running container.

Example:
  lxc-dev-manager down dev1`,
	Args: cobra.ExactArgs(1),
	RunE: runDown,
}

func init() {
	rootCmd.AddCommand(downCmd)
}

func runDown(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, lxcName, err := requireContainer(name)
	if err != nil {
		return err
	}

	// Check current status for user feedback
	status, err := lxc.GetStatus(lxcName)
	if err != nil {
		return err
	}

	if status == "STOPPED" {
		fmt.Printf("Container '%s' is already stopped\n", name)
		return nil
	}

	fmt.Printf("Stopping container '%s'...\n", name)

	// Use operations package for core logic
	if err := operations.Stop(cfg, name); err != nil {
		return err
	}

	fmt.Printf("Container '%s' stopped\n", name)
	return nil
}
