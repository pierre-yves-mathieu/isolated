package cmd

import (
	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var sshCmd = &cobra.Command{
	Use:   "ssh <name>",
	Short: "Open a shell in a container",
	Long: `Open an interactive bash shell in a container using lxc exec.

By default, logs in as the user defined in containers.yaml (defaults to 'dev').
Use -u to override with a different user, or -u root for root shell.

This is simpler than SSH and doesn't require network access.

Example:
  lxc-dev-manager ssh dev1          # Login as configured user
  lxc-dev-manager ssh dev1 -u root  # Login as root`,
	Args: cobra.ExactArgs(1),
	RunE: runSSH,
}

var sshUser string

func init() {
	rootCmd.AddCommand(sshCmd)
	sshCmd.Flags().StringVarP(&sshUser, "user", "u", "", "Override user (e.g., -u root for root shell)")
}

func runSSH(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, _, err := requireRunningContainer(name)
	if err != nil {
		return err
	}

	// Determine which user to use
	user := sshUser
	if cmd == nil || !cmd.Flags().Changed("user") {
		// No -u flag provided, use config user
		user = cfg.GetUser(name).Name
	}

	// Use operations package for shell access
	return operations.Shell(cfg, name, operations.ShellOpts{
		User: user,
	})
}
