package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var mountsSync bool

var mountsCmd = &cobra.Command{
	Use:   "mounts [container]",
	Short: "List mounted directories for a container",
	Long: `List all disk mounts for a container, showing their status.

Status values:
  ok        - Mount exists in both config and LXC
  untracked - Mount exists in LXC but not in config (manually added)
  missing   - Mount exists in config but not in LXC (needs re-add)

Use --sync to reconcile config with LXC state:
  - untracked mounts will be added to config
  - missing mounts will be re-added to LXC

Examples:
  lxc-dev-manager mounts dev1
  lxc-dev-manager mounts dev1 --sync`,
	Args: cobra.ExactArgs(1),
	RunE: runMounts,
}

func init() {
	rootCmd.AddCommand(mountsCmd)
	mountsCmd.Flags().BoolVar(&mountsSync, "sync", false, "Reconcile config with LXC state")
}

func runMounts(cmd *cobra.Command, args []string) error {
	containerName := args[0]

	var cfg *config.Config
	var lock *config.ConfigLock
	var err error

	// Use lock if we're syncing (will modify config)
	if mountsSync {
		cfg, _, lock, err = requireContainerWithLock(containerName)
		if err != nil {
			return err
		}
		defer lock.Release()
	} else {
		cfg, _, err = requireContainer(containerName)
		if err != nil {
			return err
		}
	}

	// Handle sync if requested
	if mountsSync {
		if err := operations.SyncMounts(cfg, containerName); err != nil {
			return err
		}
		fmt.Println("Mounts synchronized.")
		fmt.Println()
	}

	// Use operations package to get mount list
	mounts, err := operations.ListMounts(cfg, containerName)
	if err != nil {
		return err
	}

	// Print table
	if len(mounts) == 0 {
		fmt.Println("No mounts found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tPATH\tMODE\tSTATUS")

	for _, m := range mounts {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", m.Name, m.Source, m.Path, m.Mode, m.Status)
	}
	w.Flush()

	return nil
}
