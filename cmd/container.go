package cmd

import (
	"fmt"
	"os"

	"lxc-dev-manager/internal/lxc"
	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var containerCmd = &cobra.Command{
	Use:     "container",
	Aliases: []string{"c"},
	Short:   "Manage containers within the project",
	Long: `Commands for managing containers within the current project.

All container names are prefixed with the project name in LXC.
For example, if the project is "webapp" and you create container "dev1",
the actual LXC container will be named "webapp-dev1".`,
}

var containerCreateCmd = &cobra.Command{
	Use:   "create <name> <image>",
	Short: "Create a new container in the current project",
	Long: `Create a new container from an image and configure it for development.

The container will be set up with:
  - Nesting enabled (Docker support)
  - User with passwordless sudo (configurable in containers.yaml, default: dev/dev)
  - SSH enabled

The container name will be prefixed with the project name in LXC.

Examples:
  lxc-dev-manager container create dev1 ubuntu:24.04
  lxc-dev-manager c create myapp my-custom-base`,
	Args: cobra.ExactArgs(2),
	RunE: runContainerCreate,
}

var containerResetCmd = &cobra.Command{
	Use:   "reset <container> [snapshot]",
	Short: "Reset container to a snapshot",
	Long: `Reset a container to a snapshot state.

If no snapshot is specified, resets to 'initial-state'.
Uses ZFS snapshots - the operation is instant.

Examples:
  lxc-dev-manager container reset dev1                    # reset to initial-state
  lxc-dev-manager container reset dev1 before-refactor    # reset to named snapshot`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runContainerReset,
}

var containerCloneCmd = &cobra.Command{
	Use:   "clone <source> <new-name>",
	Short: "Clone a container",
	Long: `Clone an existing container to create a new one.

By default, clones the current state of the container. Use --snapshot to clone
from a specific snapshot instead.

The cloned container will:
  - Have all the same data as the source
  - Get a new 'initial-state' snapshot
  - Be registered in the project config

Examples:
  lxc-dev-manager container clone dev dev2                     # clone current state
  lxc-dev-manager container clone dev dev2 --snapshot checkpoint  # clone from snapshot`,
	Args: cobra.ExactArgs(2),
	RunE: runContainerClone,
}

var cloneSnapshot string

func init() {
	rootCmd.AddCommand(containerCmd)
	containerCmd.AddCommand(containerCreateCmd)
	containerCmd.AddCommand(containerResetCmd)
	containerCmd.AddCommand(containerCloneCmd)

	// Clone flags
	containerCloneCmd.Flags().StringVarP(&cloneSnapshot, "snapshot", "s", "", "Clone from a specific snapshot instead of current state")
}

func runContainerCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	image := args[1]

	// Load config with lock to prevent race conditions
	cfg, lock, err := requireProjectWithLock()
	if err != nil {
		return err
	}
	defer lock.Release()

	lxcName := cfg.GetLXCName(name)

	fmt.Printf("Creating container '%s' (LXC: %s) from image '%s'...\n", name, lxcName, image)

	// Use operations package for core logic
	if err := operations.CreateContainer(cfg, name, image, operations.CreateContainerOpts{}); err != nil {
		return err
	}

	// Get IP for display
	ip, err := lxc.GetIP(lxcName)
	if err != nil {
		ip = "(pending)"
	}

	// Get user config for display
	user := cfg.GetUser(name)

	fmt.Printf("\nContainer '%s' created successfully!\n", name)
	fmt.Printf("  LXC name: %s\n", lxcName)
	fmt.Printf("  IP: %s\n", ip)
	fmt.Printf("  User: %s / Password: %s\n", user.Name, user.Password)
	fmt.Printf("\nConnect with: %s ssh %s\n", os.Args[0], name)

	return nil
}

func runContainerReset(cmd *cobra.Command, args []string) error {
	name := args[0]
	snapshotName := "initial-state"
	if len(args) > 1 {
		snapshotName = args[1]
	}

	cfg, lxcName, err := requireContainer(name)
	if err != nil {
		return err
	}

	// Check status before reset for display purposes
	status, _ := lxc.GetStatus(lxcName)
	wasRunning := status == "RUNNING"

	fmt.Printf("Restoring container '%s' to snapshot '%s'...\n", name, snapshotName)

	// Use operations package for core logic
	if err := operations.Reset(cfg, name, snapshotName); err != nil {
		return err
	}

	// Display result
	if wasRunning {
		ip, _ := lxc.GetIP(lxcName)
		if ip != "" {
			fmt.Printf("\nContainer '%s' reset to '%s' successfully! IP: %s\n", name, snapshotName, ip)
		} else {
			fmt.Printf("\nContainer '%s' reset to '%s' successfully!\n", name, snapshotName)
		}
	} else {
		fmt.Printf("\nContainer '%s' reset to '%s' successfully! (kept stopped)\n", name, snapshotName)
	}

	return nil
}

func runContainerClone(cmd *cobra.Command, args []string) error {
	sourceName := args[0]
	newName := args[1]

	// Load config with lock to prevent race conditions
	cfg, _, lock, err := requireContainerWithLock(sourceName)
	if err != nil {
		return err
	}
	defer lock.Release()

	if cloneSnapshot != "" {
		fmt.Printf("Cloning container '%s' (snapshot: %s) to '%s'...\n", sourceName, cloneSnapshot, newName)
	} else {
		fmt.Printf("Cloning container '%s' to '%s'...\n", sourceName, newName)
	}

	// Use operations package for core logic
	if err := operations.Clone(cfg, sourceName, newName, operations.CloneOpts{
		FromSnapshot: cloneSnapshot,
	}); err != nil {
		return err
	}

	newLXC := cfg.GetLXCName(newName)

	// Get IP for display
	ip, _ := lxc.GetIP(newLXC)
	if ip == "" {
		ip = "(pending)"
	}

	// Get user config for display
	user := cfg.GetUser(newName)

	fmt.Printf("\nContainer '%s' cloned successfully!\n", newName)
	fmt.Printf("  LXC name: %s\n", newLXC)
	fmt.Printf("  Source: %s", sourceName)
	if cloneSnapshot != "" {
		fmt.Printf(" (snapshot: %s)", cloneSnapshot)
	}
	fmt.Println()
	fmt.Printf("  IP: %s\n", ip)
	fmt.Printf("  User: %s\n", user.Name)
	fmt.Printf("  SSH: ssh %s@%s\n", user.Name, ip)

	return nil
}
