package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage lxc-dev-manager projects",
	Long: `Commands for managing lxc-dev-manager projects.

A project groups containers under a common prefix and stores configuration
in a containers.yaml file in the current directory.`,
}

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Initialize a new project in the current directory",
	Long: `Creates a containers.yaml file with the project name.

The project name defaults to the current folder name, or can be
specified with --name. All containers will be prefixed with
the project name in LXC.

Default ports for proxying can be specified with --ports as a
comma-separated list. If not specified, no default ports are set.

Examples:
  lxc-dev-manager project create
  lxc-dev-manager project create --name my-app
  lxc-dev-manager project create --ports 5173,8000,5432
  lxc-dev-manager create  # alias for project create`,
	Args: cobra.NoArgs,
	RunE: runProjectCreate,
}

// Root-level create command as alias for project create
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Initialize a new project (alias for 'project create')",
	Long: `Creates a containers.yaml file with the project name.

The project name defaults to the current folder name, or can be
specified with --name. All containers will be prefixed with
the project name in LXC.

Default ports for proxying can be specified with --ports as a
comma-separated list. If not specified, no default ports are set.

This is an alias for 'lxc-dev-manager project create'.

Examples:
  lxc-dev-manager create
  lxc-dev-manager create --name my-app
  lxc-dev-manager create --ports 5173,8000,5432`,
	Args: cobra.NoArgs,
	RunE: runProjectCreate,
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete the project and all its containers",
	Long: `Deletes all containers belonging to this project and removes
the containers.yaml file. This action is destructive and irreversible.

Examples:
  lxc-dev-manager project delete
  lxc-dev-manager project delete --force`,
	Args: cobra.NoArgs,
	RunE: runProjectDelete,
}

var (
	projectNameFlag    string
	projectPortsFlag   string
	projectDeleteForce bool
)

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectDeleteCmd)

	// Add --name flag to project create
	projectCreateCmd.Flags().StringVarP(&projectNameFlag, "name", "n", "", "Project name (defaults to folder name)")
	projectCreateCmd.Flags().StringVarP(&projectPortsFlag, "ports", "p", "", "Default ports to proxy (comma-separated, e.g., 5173,8000,5432)")

	// Add --force flag to project delete
	projectDeleteCmd.Flags().BoolVarP(&projectDeleteForce, "force", "f", false, "Skip confirmation prompt")

	// Add root-level create alias
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&projectNameFlag, "name", "n", "", "Project name (defaults to folder name)")
	createCmd.Flags().StringVarP(&projectPortsFlag, "ports", "p", "", "Default ports to proxy (comma-separated, e.g., 5173,8000,5432)")
}

func runProjectCreate(cmd *cobra.Command, args []string) error {
	// Parse ports flag
	var ports []int
	if projectPortsFlag != "" {
		portStrs := strings.Split(projectPortsFlag, ",")
		for _, ps := range portStrs {
			ps = strings.TrimSpace(ps)
			if ps == "" {
				continue
			}
			port, err := strconv.Atoi(ps)
			if err != nil {
				return fmt.Errorf("invalid port %q: %w", ps, err)
			}
			if port < 1 || port > 65535 {
				return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
			}
			ports = append(ports, port)
		}
	}

	// Use operations package for project creation
	cfg, err := operations.CreateProject(projectDir, operations.CreateProjectOpts{
		Name:  projectNameFlag,
		Ports: ports,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Project '%s' created\n", cfg.Project)
	fmt.Printf("  Config: %s\n", config.ConfigFile)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  %s container create dev1 ubuntu:24.04\n", os.Args[0])

	return nil
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// List containers to be deleted
	fmt.Printf("Project: %s\n", cfg.Project)
	fmt.Printf("Config:  %s\n\n", config.ConfigFile)

	if len(cfg.Containers) > 0 {
		fmt.Println("Containers to be deleted:")
		for name := range cfg.Containers {
			lxcName := cfg.GetLXCName(name)
			status := "NOT FOUND"
			if lxc.Exists(lxcName) {
				s, _ := lxc.GetStatus(lxcName)
				status = s
			}
			fmt.Printf("  - %s (%s) [%s]\n", name, lxcName, status)
		}
		fmt.Println()
	} else {
		fmt.Println("No containers defined.")
	}

	// Confirm deletion
	if !projectDeleteForce {
		if !confirmPrompt("Are you sure you want to delete this project?") {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Delete all containers
	var deleteErrors []string
	for name := range cfg.Containers {
		lxcName := cfg.GetLXCName(name)
		fmt.Printf("Deleting container '%s'... ", name)

		if lxc.Exists(lxcName) {
			if err := lxc.Delete(lxcName); err != nil {
				fmt.Printf("FAILED: %v\n", err)
				deleteErrors = append(deleteErrors, fmt.Sprintf("%s: %v", name, err))
				continue
			}
		}
		fmt.Println("done")
	}

	// Remove config file
	cfgDir := projectDir
	if cfgDir == "" {
		cfgDir = "."
	}
	configPath := filepath.Join(cfgDir, config.ConfigFile)
	fmt.Printf("Removing %s... ", configPath)
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to remove config: %w", err)
	}
	fmt.Println("done")

	if len(deleteErrors) > 0 {
		fmt.Printf("\nWarning: Some containers failed to delete:\n")
		for _, e := range deleteErrors {
			fmt.Printf("  - %s\n", e)
		}
	}

	fmt.Printf("\nProject '%s' deleted\n", cfg.Project)
	return nil
}
