package operations

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
)

// CreateProject creates a new project in the specified directory.
// If dir is empty, it uses the current working directory.
func CreateProject(dir string, opts CreateProjectOpts) (*config.Config, error) {
	// Check if project already exists
	cfg, err := config.Load(dir)
	if err != nil && !errors.Is(err, config.ErrNoProject) {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if cfg != nil {
		return nil, fmt.Errorf("project already exists: %s", cfg.Project)
	}

	// Determine project name
	projectName := opts.Name
	if projectName == "" {
		projectName, err = config.GetProjectFromFolder(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to get folder name: %w", err)
		}
	}

	// Validate project name
	if !config.IsValidProjectName(projectName) {
		return nil, fmt.Errorf("invalid project name %q: must contain only letters, numbers, hyphens, and underscores", projectName)
	}

	// Resolve dir for the config
	cfgDir := dir
	if cfgDir == "" {
		cfgDir = "."
	}

	// Create config
	cfg = &config.Config{
		Dir:     cfgDir,
		Project: projectName,
		Defaults: config.Defaults{
			Ports: opts.Ports,
		},
		Containers: make(map[string]config.Container),
	}

	if err := cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return cfg, nil
}

// DeleteProject deletes a project and all its containers.
// If dir is empty, it uses the current working directory.
func DeleteProject(dir string, force bool) error {
	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Delete all containers
	var deleteErrors []error
	for name := range cfg.Containers {
		lxcName := cfg.GetLXCName(name)
		if lxc.Exists(lxcName) {
			if err := lxc.Delete(lxcName); err != nil {
				if !force {
					return fmt.Errorf("failed to delete container %s: %w", name, err)
				}
				deleteErrors = append(deleteErrors, fmt.Errorf("%s: %w", name, err))
			}
		}
	}

	// Remove config file
	cfgDir := dir
	if cfgDir == "" {
		cfgDir = "."
	}
	configPath := filepath.Join(cfgDir, config.ConfigFile)
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to remove config: %w", err)
	}

	if len(deleteErrors) > 0 {
		return fmt.Errorf("some containers failed to delete: %v", deleteErrors)
	}

	return nil
}

// LoadProject loads an existing project configuration.
// If dir is empty, it uses the current working directory.
func LoadProject(dir string) (*config.Config, error) {
	cfg, err := config.Load(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// LoadProjectWithLock loads an existing project configuration with lock.
// If dir is empty, it uses the current working directory.
func LoadProjectWithLock(dir string) (*config.Config, *config.ConfigLock, error) {
	cfg, lock, err := config.LoadWithLock(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, lock, nil
}
