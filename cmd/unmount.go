package cmd

import (
	"fmt"

	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var unmountForce bool

var unmountCmd = &cobra.Command{
	Use:   "unmount <container> <name-or-path>",
	Short: "Unmount a disk from a container",
	Long: `Unmount a disk device from a container.

The device can be specified by its name or by its container path.

Examples:
  lxc-dev-manager unmount dev1 repo
  lxc-dev-manager unmount dev1 /repo.git
  lxc-dev-manager unmount dev1 /workspace --force`,
	Args: cobra.ExactArgs(2),
	RunE: runUnmount,
}

func init() {
	rootCmd.AddCommand(unmountCmd)

	unmountCmd.Flags().BoolVarP(&unmountForce, "force", "f", false, "Force unmount (no confirmation)")
}

func runUnmount(cmd *cobra.Command, args []string) error {
	containerName := args[0]
	nameOrPath := args[1]

	// Load config with lock to prevent race conditions
	cfg, _, lock, err := requireContainerWithLock(containerName)
	if err != nil {
		return err
	}
	defer lock.Release()

	fmt.Printf("Unmounting '%s' from container '%s'...\n", nameOrPath, containerName)

	// Use operations package for core logic
	if err := operations.Unmount(cfg, containerName, nameOrPath); err != nil {
		return err
	}

	fmt.Printf("Device unmounted successfully.\n")
	return nil
}
