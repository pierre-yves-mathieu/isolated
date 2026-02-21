package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <name> -- <command> [args...]",
	Short: "Execute a command in a container",
	Long: `Execute a command in a container.

Requires a command after --. For an interactive shell, use 'ssh' instead.

Examples:
  lxc-dev-manager exec dev -- htop
  lxc-dev-manager exec dev -- zellij
  lxc-dev-manager exec dev -u root -- apt update
  lxc-dev-manager exec dev -- npm run dev
  lxc-dev-manager exec dev -- zellij run -- ls    # nested -- works
  lxc-dev-manager exec dev -- bash                # explicit shell`,
	Args: cobra.MinimumNArgs(2), // container + at least one command arg
	RunE: runExec,
}

var execUser string

func init() {
	rootCmd.AddCommand(execCmd)
	execCmd.Flags().StringVarP(&execUser, "user", "u", "", "Run as user (default: configured user)")
}

// buildExecArgs constructs the lxc exec arguments for running a command
func buildExecArgs(lxcName, user string, cmdArgs []string) []string {
	args := []string{"exec", lxcName, "--"}

	if user != "" {
		// Run command as specified user via su -l
		args = append(args, "su", "-l", user)
		args = append(args, cmdArgs...)
	} else {
		// Run command directly as root
		args = append(args, cmdArgs...)
	}

	return args
}

func runExec(cmd *cobra.Command, args []string) error {
	name := args[0]
	cmdArgs := args[1:] // Everything after container name

	if len(cmdArgs) == 0 {
		return fmt.Errorf("command required after --\nFor interactive shell, use: %s ssh %s", os.Args[0], name)
	}

	cfg, lxcName, err := requireRunningContainer(name)
	if err != nil {
		return err
	}

	// Determine which user to use
	user := execUser
	if cmd == nil || !cmd.Flags().Changed("user") {
		// No -u flag provided, use config user
		user = cfg.GetUser(name).Name
	}

	// Build lxc exec command
	lxcArgs := buildExecArgs(lxcName, user, cmdArgs)

	// Replace current process with lxc exec (for proper TTY handling)
	lxcPath, err := exec.LookPath("lxc")
	if err != nil {
		return fmt.Errorf("lxc command not found: %w", err)
	}

	return syscall.Exec(lxcPath, append([]string{"lxc"}, lxcArgs...), os.Environ())
}
