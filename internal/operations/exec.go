package operations

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
)

// Exec runs a command inside a container and returns the output
func Exec(cfg *config.Config, name string, cmd []string) ([]byte, error) {
	if !cfg.HasContainer(name) {
		return nil, fmt.Errorf("container '%s' not found in config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return nil, fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Check if running
	status, err := lxc.GetStatus(lxcName)
	if err != nil {
		return nil, err
	}
	if status != "RUNNING" {
		return nil, fmt.Errorf("container '%s' is not running", name)
	}

	// Build command
	args := append([]string{"exec", lxcName, "--"}, cmd...)
	execCmd := exec.Command("lxc", args...)
	return execCmd.CombinedOutput()
}

// ExecInteractive runs an interactive command inside a container
func ExecInteractive(cfg *config.Config, name string, cmd []string) error {
	if !cfg.HasContainer(name) {
		return fmt.Errorf("container '%s' not found in config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Check if running
	status, err := lxc.GetStatus(lxcName)
	if err != nil {
		return err
	}
	if status != "RUNNING" {
		return fmt.Errorf("container '%s' is not running", name)
	}

	// Build command
	args := append([]string{"exec", lxcName, "--"}, cmd...)

	lxcPath, err := exec.LookPath("lxc")
	if err != nil {
		return fmt.Errorf("lxc command not found: %w", err)
	}

	// Use syscall.Exec to replace the process for proper TTY handling
	return syscall.Exec(lxcPath, append([]string{"lxc"}, args...), os.Environ())
}

// Shell opens an interactive shell in a container
func Shell(cfg *config.Config, name string, opts ShellOpts) error {
	if !cfg.HasContainer(name) {
		return fmt.Errorf("container '%s' not found in config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Check if running
	status, err := lxc.GetStatus(lxcName)
	if err != nil {
		return err
	}
	if status != "RUNNING" {
		return fmt.Errorf("container '%s' is not running", name)
	}

	// Determine which user to use
	user := opts.User
	if user == "" {
		user = cfg.GetUser(name).Name
	}

	// Build lxc exec command
	args := []string{"exec", lxcName, "--"}
	if user != "" && user != "root" {
		// Use su -l to get a proper login shell with all supplementary groups loaded
		args = append(args, "su", "-l", user)
	} else {
		// Root shell
		args = append(args, "bash", "-l")
	}

	lxcPath, err := exec.LookPath("lxc")
	if err != nil {
		return fmt.Errorf("lxc command not found: %w", err)
	}

	// Use syscall.Exec to replace the process for proper TTY handling
	return syscall.Exec(lxcPath, append([]string{"lxc"}, args...), os.Environ())
}

// BuildShellArgs constructs the lxc exec arguments for Shell
func BuildShellArgs(lxcName, user string) []string {
	args := []string{"exec", lxcName, "--"}

	if user != "" && user != "root" {
		args = append(args, "su", "-l", user)
	} else {
		args = append(args, "bash", "-l")
	}

	return args
}
