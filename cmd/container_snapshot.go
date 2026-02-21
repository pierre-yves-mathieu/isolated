package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var snapshotDescription string

var containerSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage container snapshots",
}

var containerSnapshotCreateCmd = &cobra.Command{
	Use:   "create <container> <name>",
	Short: "Create a named snapshot",
	Long: `Create a named snapshot of a container.

The snapshot is instant with ZFS storage.

Examples:
  lxc-dev-manager container snapshot create dev1 before-refactor
  lxc-dev-manager container snapshot create dev1 checkpoint -d "Before database migration"`,
	Args: cobra.ExactArgs(2),
	RunE: runSnapshotCreate,
}

var containerSnapshotListCmd = &cobra.Command{
	Use:   "list <container>",
	Short: "List snapshots for a container",
	Args:  cobra.ExactArgs(1),
	RunE:  runSnapshotList,
}

var containerSnapshotDeleteCmd = &cobra.Command{
	Use:   "delete <container> <name>",
	Short: "Delete a snapshot",
	Args:  cobra.ExactArgs(2),
	RunE:  runSnapshotDelete,
}

func init() {
	containerCmd.AddCommand(containerSnapshotCmd)
	containerSnapshotCmd.AddCommand(containerSnapshotCreateCmd)
	containerSnapshotCmd.AddCommand(containerSnapshotListCmd)
	containerSnapshotCmd.AddCommand(containerSnapshotDeleteCmd)

	containerSnapshotCreateCmd.Flags().StringVarP(&snapshotDescription, "description", "d", "", "Snapshot description")
}

func runSnapshotCreate(cmd *cobra.Command, args []string) error {
	containerName := args[0]
	snapshotName := args[1]

	// Load config with lock to prevent race conditions
	cfg, _, lock, err := requireContainerWithLock(containerName)
	if err != nil {
		return err
	}
	defer lock.Release()

	fmt.Printf("Creating snapshot '%s'...\n", snapshotName)

	// Use operations package for core logic
	if err := operations.CreateSnapshot(cfg, containerName, snapshotName, snapshotDescription); err != nil {
		return err
	}

	fmt.Printf("Snapshot '%s' created successfully!\n", snapshotName)
	return nil
}

func runSnapshotList(cmd *cobra.Command, args []string) error {
	containerName := args[0]

	cfg, _, err := requireContainer(containerName)
	if err != nil {
		return err
	}

	// Use operations package to get snapshot list
	snapshots, err := operations.ListSnapshots(cfg, containerName)
	if err != nil {
		return err
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots found.")
		return nil
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCREATED\tDESCRIPTION")

	for _, s := range snapshots {
		created := "-"
		description := "-"
		if !s.CreatedAt.IsZero() {
			created = s.CreatedAt.Format("2006-01-02 15:04")
		}
		if s.Description != "" {
			description = s.Description
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, created, description)
	}
	w.Flush()

	return nil
}

func runSnapshotDelete(cmd *cobra.Command, args []string) error {
	containerName := args[0]
	snapshotName := args[1]

	// Load config with lock to prevent race conditions
	cfg, _, lock, err := requireContainerWithLock(containerName)
	if err != nil {
		return err
	}
	defer lock.Release()

	fmt.Printf("Deleting snapshot '%s'...\n", snapshotName)

	// Use operations package for core logic
	if err := operations.DeleteSnapshot(cfg, containerName, snapshotName); err != nil {
		return err
	}

	fmt.Printf("Snapshot '%s' deleted.\n", snapshotName)
	return nil
}
