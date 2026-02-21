package lxcmgr

import (
	"lxc-dev-manager/internal/operations"
)

// Exec runs a command inside a container and returns the output
func (c *Client) Exec(name string, cmd []string) ([]byte, error) {
	output, err := operations.Exec(c.cfg, name, cmd)
	return output, wrapContainerErr("exec", name, err)
}

// ExecInteractive runs an interactive command inside a container.
// This replaces the current process with the container shell.
func (c *Client) ExecInteractive(name string, cmd []string) error {
	return wrapContainerErr("exec", name, operations.ExecInteractive(c.cfg, name, cmd))
}

// Shell opens an interactive shell in a container.
// This replaces the current process with the container shell.
func (c *Client) Shell(name string, opts ...ShellOption) error {
	o := &shellOpts{}
	for _, opt := range opts {
		opt(o)
	}

	return wrapContainerErr("shell", name, operations.Shell(c.cfg, name, operations.ShellOpts{
		User: o.user,
	}))
}
