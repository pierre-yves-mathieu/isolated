package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync <container>",
	Short: "Sync configured files to a container",
	Long: `Copy all files configured in the sync section of containers.yaml to a container.

Source paths are resolved relative to the containers.yaml directory.

Examples:
  lxc-dev-manager sync dev1
  lxc-dev-manager sync dev1 --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: runSync,
}

var syncVerbose bool

var syncAddCmd = &cobra.Command{
	Use:   "add <container> <source> <dest>",
	Short: "Add a file sync entry",
	Long: `Add a file to sync from host to container.

Source is relative to the containers.yaml directory.
Dest is the absolute path inside the container.

Examples:
  lxc-dev-manager sync add dev1 .env /home/dev/project/.env
  lxc-dev-manager sync add dev1 config/secrets.json /home/dev/project/config/secrets.json`,
	Args: cobra.ExactArgs(3),
	RunE: runSyncAdd,
}

var syncRmCmd = &cobra.Command{
	Use:   "rm <container> <source>",
	Short: "Remove a file sync entry",
	Long: `Remove a sync entry by its source path.

Examples:
  lxc-dev-manager sync rm dev1 .env`,
	Args: cobra.ExactArgs(2),
	RunE: runSyncRm,
}

var syncListCmd = &cobra.Command{
	Use:   "list <container>",
	Short: "List sync entries for a container",
	Args:  cobra.ExactArgs(1),
	RunE:  runSyncList,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVarP(&syncVerbose, "verbose", "v", false, "Show detailed output")
	syncCmd.AddCommand(syncAddCmd)
	syncCmd.AddCommand(syncRmCmd)
	syncCmd.AddCommand(syncListCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	containerName := args[0]

	cfg, _, err := requireContainer(containerName)
	if err != nil {
		return err
	}

	entries := cfg.GetSyncEntries(containerName)
	if len(entries) == 0 {
		fmt.Println("No sync entries configured")
		return nil
	}

	if syncVerbose {
		fmt.Printf("Syncing %d files to %s...\n", len(entries), containerName)
		for _, e := range entries {
			fmt.Printf("  %s -> %s\n", e.Source, e.Dest)
		}
	}

	if err := operations.SyncFiles(cfg, containerName, cfg.Dir); err != nil {
		return err
	}

	fmt.Printf("Synced %d files to %s\n", len(entries), containerName)
	return nil
}

func runSyncAdd(cmd *cobra.Command, args []string) error {
	containerName := args[0]
	source := args[1]
	dest := args[2]

	cfg, _, lock, err := requireContainerWithLock(containerName)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	cfg.AddSyncEntry(containerName, config.SyncEntry{
		Source: source,
		Dest:   dest,
	})

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Added sync: %s -> %s\n", source, dest)
	return nil
}

func runSyncRm(cmd *cobra.Command, args []string) error {
	containerName := args[0]
	source := args[1]

	cfg, _, lock, err := requireContainerWithLock(containerName)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	cfg.RemoveSyncEntry(containerName, source)

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Removed sync entry: %s\n", source)
	return nil
}

func runSyncList(cmd *cobra.Command, args []string) error {
	containerName := args[0]

	cfg, _, err := requireContainer(containerName)
	if err != nil {
		return err
	}

	entries := cfg.GetSyncEntries(containerName)
	if len(entries) == 0 {
		fmt.Println("No sync entries configured")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SOURCE\tDEST")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\n", e.Source, e.Dest)
	}
	return w.Flush()
}
