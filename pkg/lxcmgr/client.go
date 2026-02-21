package lxcmgr

import (
	"os"
	"path/filepath"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
	"lxc-dev-manager/internal/operations"
)

// Client manages containers within an lxc-dev-manager project
type Client struct {
	dir      string
	cfg      *config.Config
	executor lxc.Executor
}

// New opens an existing project
func New(projectDir string) (*Client, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}

	// Verify directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return nil, ErrProjectNotFound
	}

	// Load config
	cfg, err := operations.LoadProject(absDir)
	if err != nil {
		return nil, err
	}

	return &Client{
		dir:      absDir,
		cfg:      cfg,
		executor: lxc.DefaultExecutor,
	}, nil
}

// NewWithExecutor creates a client with a custom executor (for testing)
func NewWithExecutor(projectDir string, executor lxc.Executor) (*Client, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}

	// Verify directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return nil, ErrProjectNotFound
	}

	// Load config
	cfg, err := operations.LoadProject(absDir)
	if err != nil {
		return nil, err
	}

	return &Client{
		dir:      absDir,
		cfg:      cfg,
		executor: executor,
	}, nil
}

// NewProject creates a new project and returns a client
func NewProject(dir string, opts ...ProjectOption) (*Client, error) {
	o := &projectOpts{}
	for _, opt := range opts {
		opt(o)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// Ensure directory exists
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, err
	}

	cfg, err := operations.CreateProject(absDir, operations.CreateProjectOpts{
		Name:  o.name,
		Ports: o.ports,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		dir:      absDir,
		cfg:      cfg,
		executor: lxc.DefaultExecutor,
	}, nil
}

// ProjectName returns the project name
func (c *Client) ProjectName() string {
	return c.cfg.Project
}

// Dir returns the project directory
func (c *Client) Dir() string {
	return c.dir
}

// DeleteProject deletes the project and all its containers
func (c *Client) DeleteProject(force bool) error {
	return operations.DeleteProject(c.dir, force)
}

// Reload reloads the configuration from disk
func (c *Client) Reload() error {
	cfg, err := operations.LoadProject(c.dir)
	if err != nil {
		return err
	}
	c.cfg = cfg
	return nil
}

