package cmd

import (
	"fmt"

	"lxc-dev-manager/internal/operations"
	"lxc-dev-manager/internal/validation"

	"github.com/spf13/cobra"
)

var (
	mountName      string
	mountReadWrite bool
	mountShift     bool
	mountAllowRisky bool
	mountYes       bool
)

var mountCmd = &cobra.Command{
	Use:   "mount <container> <source> <path>",
	Short: "Mount a host directory into a container",
	Long: `Mount a host directory into a container as a disk device.

By default, mounts are read-only for safety. Use --rw for read-write access.

Examples:
  lxc-dev-manager mount dev1 ~/project /workspace
  lxc-dev-manager mount dev1 ~/.isollm/repo.git /repo.git --rw
  lxc-dev-manager mount dev1 /data /mnt/data --name data-mount
  lxc-dev-manager mount dev1 /home /mnt/home --allow-risky`,
	Args: cobra.ExactArgs(3),
	RunE: runMount,
}

func init() {
	rootCmd.AddCommand(mountCmd)
	mountCmd.Flags().StringVarP(&mountName, "name", "n", "", "Device name (default: auto-generated from path)")
	mountCmd.Flags().BoolVar(&mountReadWrite, "rw", false, "Mount read-write (default: read-only)")
	mountCmd.Flags().BoolVar(&mountShift, "shift", false, "Enable UID/GID shifting")
	mountCmd.Flags().BoolVar(&mountAllowRisky, "allow-risky", false, "Allow mounting risky paths (e.g., /home)")
	mountCmd.Flags().BoolVarP(&mountYes, "yes", "y", false, "Skip confirmation prompts")
}

func runMount(cmd *cobra.Command, args []string) error {
	containerName := args[0]
	sourcePath := args[1]
	containerPath := args[2]

	// Load config with lock and verify container
	cfg, _, lock, err := requireContainerWithLock(containerName)
	if err != nil {
		return err
	}
	defer lock.Release()

	// Handle risky path warning interactively (CLI-specific)
	resolvedSource, warning, err := validation.ValidateSourcePath(sourcePath)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}

	allowRiskyPath := mountAllowRisky
	if warning != "" && !mountAllowRisky && !mountYes {
		fmt.Printf("Warning: %s\n", warning)
		if confirmPrompt("Do you want to continue?") {
			allowRiskyPath = true
		} else {
			fmt.Println("Cancelled")
			return nil
		}
	}

	// Use operations package for core logic
	deviceName, err := operations.Mount(cfg, containerName, sourcePath, containerPath, operations.MountOpts{
		Name:           mountName,
		ReadWrite:      mountReadWrite,
		Shift:          mountShift,
		AllowRiskyPath: allowRiskyPath,
	})
	if err != nil {
		return err
	}

	// Print success message
	mode := "ro"
	if mountReadWrite {
		mode = "rw"
	}
	fmt.Printf("Mounted '%s' -> '%s' (%s) as device '%s'\n", resolvedSource, containerPath, mode, deviceName)
	return nil
}
