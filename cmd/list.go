package cmd

import (
	"fmt"
	"os"
	"strings"

	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all containers",
	Long: `List all containers defined in the config with their status.

Example:
  lxc-dev-manager list`,
	Args: cobra.NoArgs,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := requireProject()
	if err != nil {
		return err
	}

	// Show project header
	fmt.Printf("Project: %s\n\n", cfg.Project)

	if len(cfg.Containers) == 0 {
		fmt.Println("No containers defined in config")
		fmt.Printf("Create one with: %s container create <name> <image>\n", os.Args[0])
		return nil
	}

	// Use operations package to get container list
	containers, err := operations.List(cfg)
	if err != nil {
		return err
	}

	// Print header
	fmt.Printf("%-15s %-20s %-10s %-15s %s\n", "NAME", "IMAGE", "STATUS", "IP", "PORTS")
	fmt.Println(strings.Repeat("-", 75))

	// Print each container
	for _, c := range containers {
		ip := c.IP
		if ip == "" {
			ip = "-"
		}

		portStr := formatPorts(c.Ports)

		fmt.Printf("%-15s %-20s %-10s %-15s %s\n", c.Name, c.Image, c.Status, ip, portStr)
	}

	return nil
}

func formatPorts(ports []int) string {
	if len(ports) == 0 {
		return "-"
	}

	strs := make([]string, len(ports))
	for i, p := range ports {
		strs[i] = fmt.Sprintf("%d", p)
	}
	return strings.Join(strs, ",")
}
