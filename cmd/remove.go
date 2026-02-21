package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"lxc-dev-manager/internal/lxc"
	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a container",
	Long: `Remove a container and delete it from the config.

This will forcefully delete the container even if it's running.
By default, asks for confirmation. Use --force to skip.

Example:
  lxc-dev-manager remove dev1
  lxc-dev-manager remove dev1 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

var removeForce bool

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Skip confirmation prompt")
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Load config with lock to prevent race conditions
	cfg, lock, err := requireProjectWithLock()
	if err != nil {
		return err
	}
	defer lock.Release()

	// Get full LXC name with prefix
	lxcName := cfg.GetLXCName(name)

	// Check if container exists anywhere
	existsInLXC := lxc.Exists(lxcName)
	existsInConfig := cfg.HasContainer(name)

	if !existsInLXC && !existsInConfig {
		return fmt.Errorf("container '%s' not found", name)
	}

	// Show what will be deleted
	if existsInLXC {
		status, _ := lxc.GetStatus(lxcName)
		ip, _ := lxc.GetIP(lxcName)

		fmt.Printf("\nContainer: %s (LXC: %s)\n", name, lxcName)
		fmt.Printf("  Status: %s\n", status)
		if ip != "" {
			fmt.Printf("  IP: %s\n", ip)
		}
		if existsInConfig {
			fmt.Printf("  In config: yes\n")
		}
		fmt.Println()
	}

	// Ask for confirmation unless --force
	if !removeForce {
		if !confirmPrompt(fmt.Sprintf("Are you sure you want to delete container '%s'?", name)) {
			fmt.Println("Cancelled")
			return nil
		}
	}

	fmt.Printf("Deleting container '%s'...\n", name)

	// Use operations package for core logic
	if err := operations.Remove(cfg, name, removeForce); err != nil {
		return err
	}

	fmt.Printf("Container '%s' removed\n", name)
	return nil
}

// confirmPrompt asks user for yes/no confirmation
func confirmPrompt(question string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s [y/N]: ", question)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
