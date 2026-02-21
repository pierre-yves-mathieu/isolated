package cmd

import (
	"fmt"
	"os"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
)

// requireProject loads config and ensures a project exists.
// Returns the config or an error if no project is found.
func requireProject() (*config.Config, error) {
	cfg, err := config.Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

// requireContainer ensures a container exists in both config and LXC.
// Returns the config, LXC name, and any error.
func requireContainer(name string) (*config.Config, string, error) {
	cfg, err := requireProject()
	if err != nil {
		return nil, "", err
	}

	if !cfg.HasContainer(name) {
		return nil, "", fmt.Errorf("container '%s' not found in project config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return nil, "", fmt.Errorf("container '%s' does not exist in LXC (expected: %s)", name, lxcName)
	}

	return cfg, lxcName, nil
}

// requireRunningContainer ensures a container exists and is running.
// Returns the config, LXC name, and any error.
func requireRunningContainer(name string) (*config.Config, string, error) {
	cfg, lxcName, err := requireContainer(name)
	if err != nil {
		return nil, "", err
	}

	status, err := lxc.GetStatus(lxcName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get container status: %w", err)
	}
	if status != "RUNNING" {
		return nil, "", fmt.Errorf("container '%s' is not running (status: %s). Start it with: %s up %s", name, status, os.Args[0], name)
	}

	return cfg, lxcName, nil
}

// requireProjectWithLock loads config with exclusive lock and ensures a project exists.
// The caller must call lock.Release() when done.
func requireProjectWithLock() (*config.Config, *config.ConfigLock, error) {
	cfg, lock, err := config.LoadWithLock(projectDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, lock, nil
}

// requireContainerWithLock ensures a container exists in both config and LXC, with lock held.
// The caller must call lock.Release() when done.
func requireContainerWithLock(name string) (*config.Config, string, *config.ConfigLock, error) {
	cfg, lock, err := requireProjectWithLock()
	if err != nil {
		return nil, "", nil, err
	}

	if !cfg.HasContainer(name) {
		lock.Release()
		return nil, "", nil, fmt.Errorf("container '%s' not found in project config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		lock.Release()
		return nil, "", nil, fmt.Errorf("container '%s' does not exist in LXC (expected: %s)", name, lxcName)
	}

	return cfg, lxcName, lock, nil
}
