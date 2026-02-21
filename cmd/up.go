package cmd

import (
	"fmt"
	"time"

	"lxc-dev-manager/internal/lxc"
	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up <name>",
	Short: "Start a container",
	Long: `Start a stopped container.

Example:
  lxc-dev-manager up dev1`,
	Args: cobra.ExactArgs(1),
	RunE: runUp,
}

func init() {
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
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

	if status == "RUNNING" {
		fmt.Printf("Container '%s' is already running\n", name)
		ip, _ := lxc.GetIP(lxcName)
		if ip != "" {
			fmt.Printf("  IP: %s\n", ip)
		}
		return nil
	}

	fmt.Printf("Starting container '%s'...\n", name)

	// Use operations package for core logic
	if err := operations.Start(cfg, name); err != nil {
		return err
	}

	// Wait a moment for network
	time.Sleep(2 * time.Second)

	// Get IP for display
	ip, err := lxc.GetIP(lxcName)
	if err != nil {
		ip = "(pending)"
	}

	fmt.Printf("Container '%s' started\n", name)
	fmt.Printf("  IP: %s\n", ip)

	return nil
}
