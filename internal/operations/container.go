package operations

import (
	"fmt"
	"time"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
	"lxc-dev-manager/internal/validation"
)

// CreateContainer creates a new container
func CreateContainer(cfg *config.Config, name, image string, opts CreateContainerOpts) error {
	// Validate container name
	if err := validation.ValidateContainerName(name); err != nil {
		return fmt.Errorf("invalid container name: %w", err)
	}

	// Validate combined name (project + container)
	if err := validation.ValidateFullContainerName(cfg.Project, name); err != nil {
		return err
	}

	// Check if already exists in config
	if cfg.HasContainer(name) {
		return fmt.Errorf("container '%s' already exists in config", name)
	}

	// Get full LXC name with prefix
	lxcName := cfg.GetLXCName(name)

	// Check if already exists in LXC
	if lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' already exists in LXC", lxcName)
	}

	// Launch container
	if err := lxc.Launch(lxcName, image); err != nil {
		return err
	}

	// Enable nesting for Docker support
	if err := lxc.EnableNesting(lxcName); err != nil {
		// Non-fatal, container created but nesting not enabled
	}

	// Wait for container to be ready
	if err := lxc.WaitForReady(lxcName, 60*time.Second); err != nil {
		return err
	}

	// Get user config
	user := cfg.GetUser(name)
	if opts.User != "" {
		user.Name = opts.User
	}
	if opts.Password != "" {
		user.Password = opts.Password
	}

	// Set up user
	if err := lxc.SetupUser(lxcName, user.Name, user.Password); err != nil {
		return fmt.Errorf("failed to set up user: %w", err)
	}

	// Enable SSH
	if err := lxc.EnableSSH(lxcName); err != nil {
		return fmt.Errorf("failed to enable SSH: %w", err)
	}

	// Add to config with short name
	cfg.AddContainer(name, image)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Create initial snapshot for reset
	if err := lxc.Snapshot(lxcName, "initial-state"); err == nil {
		cfg.AddSnapshot(name, "initial-state", "Initial state after setup")
		cfg.Save()
	}

	return nil
}

// Start starts a stopped container
func Start(cfg *config.Config, name string) error {
	if !cfg.HasContainer(name) {
		return fmt.Errorf("container '%s' not found in config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	status, err := lxc.GetStatus(lxcName)
	if err != nil {
		return err
	}

	if status == "RUNNING" {
		return nil // Already running
	}

	return lxc.Start(lxcName)
}

// Stop stops a running container
func Stop(cfg *config.Config, name string) error {
	if !cfg.HasContainer(name) {
		return fmt.Errorf("container '%s' not found in config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	status, err := lxc.GetStatus(lxcName)
	if err != nil {
		return err
	}

	if status == "STOPPED" {
		return nil // Already stopped
	}

	return lxc.Stop(lxcName)
}

// Remove removes a container
func Remove(cfg *config.Config, name string, force bool) error {
	lxcName := cfg.GetLXCName(name)

	existsInLXC := lxc.Exists(lxcName)
	existsInConfig := cfg.HasContainer(name)

	if !existsInLXC && !existsInConfig {
		return fmt.Errorf("container '%s' not found", name)
	}

	// Delete from LXC if exists
	if existsInLXC {
		if err := lxc.Delete(lxcName); err != nil {
			return err
		}
	}

	// Remove from config if exists
	if existsInConfig {
		cfg.RemoveContainer(name)
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	return nil
}

// Reset resets a container to a snapshot
func Reset(cfg *config.Config, name, snapshotName string) error {
	if !cfg.HasContainer(name) {
		return fmt.Errorf("container '%s' not found in config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	if snapshotName == "" {
		snapshotName = "initial-state"
	}

	// Check if snapshot exists
	if !lxc.SnapshotExists(lxcName, snapshotName) {
		if snapshotName == "initial-state" {
			return fmt.Errorf("container '%s' has no initial-state snapshot (created before this feature was added)", name)
		}
		return fmt.Errorf("snapshot '%s' does not exist", snapshotName)
	}

	// Check if running
	status, err := lxc.GetStatus(lxcName)
	if err != nil {
		return err
	}
	wasRunning := status == "RUNNING"

	// Stop if running
	if wasRunning {
		if err := lxc.Stop(lxcName); err != nil {
			return err
		}
	}

	// Restore from snapshot
	if err := lxc.Restore(lxcName, snapshotName); err != nil {
		return err
	}

	// Restart if was running
	if wasRunning {
		if err := lxc.Start(lxcName); err != nil {
			return err
		}
	}

	return nil
}

// Clone clones a container
func Clone(cfg *config.Config, sourceName, newName string, opts CloneOpts) error {
	// Validate new container name
	if err := validation.ValidateContainerName(newName); err != nil {
		return fmt.Errorf("invalid container name: %w", err)
	}

	if err := validation.ValidateFullContainerName(cfg.Project, newName); err != nil {
		return err
	}

	// Check source exists
	if !cfg.HasContainer(sourceName) {
		return fmt.Errorf("source container '%s' not found in config", sourceName)
	}

	sourceLXC := cfg.GetLXCName(sourceName)
	if !lxc.Exists(sourceLXC) {
		return fmt.Errorf("source container '%s' does not exist in LXC", sourceLXC)
	}

	// Check if new name already exists
	if cfg.HasContainer(newName) {
		return fmt.Errorf("container '%s' already exists in config", newName)
	}

	newLXC := cfg.GetLXCName(newName)
	if lxc.Exists(newLXC) {
		return fmt.Errorf("container '%s' already exists in LXC", newLXC)
	}

	// If cloning from snapshot, verify it exists
	if opts.FromSnapshot != "" {
		if !lxc.SnapshotExists(sourceLXC, opts.FromSnapshot) {
			return fmt.Errorf("snapshot '%s' does not exist on container '%s'", opts.FromSnapshot, sourceName)
		}
	}

	// Perform the clone
	if opts.FromSnapshot != "" {
		if err := lxc.CopySnapshot(sourceLXC, opts.FromSnapshot, newLXC); err != nil {
			return err
		}
	} else {
		if err := lxc.Copy(sourceLXC, newLXC); err != nil {
			return err
		}
	}

	// Get source container config to copy image info
	sourceImage := "cloned"
	if sourceContainer, ok := cfg.Containers[sourceName]; ok {
		sourceImage = sourceContainer.Image
	}

	// Add to config
	cfg.AddContainer(newName, sourceImage+":cloned-from-"+sourceName)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Create initial snapshot for reset
	if err := lxc.Snapshot(newLXC, "initial-state"); err == nil {
		cfg.AddSnapshot(newName, "initial-state", "Initial state after clone")
		cfg.Save()
	}

	// Start the cloned container
	lxc.Start(newLXC)

	return nil
}

// List returns all containers in the project
func List(cfg *config.Config) ([]ContainerInfo, error) {
	if len(cfg.Containers) == 0 {
		return nil, nil
	}

	// Get all LXC container info
	lxcContainers, err := lxc.ListAll()
	if err != nil {
		return nil, err
	}

	// Build lookup map
	lxcInfo := make(map[string]lxc.ContainerInfo)
	for _, c := range lxcContainers {
		lxcInfo[c.Name] = c
	}

	var result []ContainerInfo
	for name, container := range cfg.Containers {
		lxcName := cfg.GetLXCName(name)

		status := "NOT FOUND"
		ip := ""

		if info, ok := lxcInfo[lxcName]; ok {
			status = info.Status
			ip = info.IP
		}

		ports := cfg.GetPorts(name)

		result = append(result, ContainerInfo{
			Name:   name,
			Image:  container.Image,
			Status: status,
			IP:     ip,
			Ports:  ports,
		})
	}

	return result, nil
}

// Status returns the status of a container
func Status(cfg *config.Config, name string) (string, error) {
	if !cfg.HasContainer(name) {
		return "", fmt.Errorf("container '%s' not found in config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return "", fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	return lxc.GetStatus(lxcName)
}

// IP returns the IP address of a container
func IP(cfg *config.Config, name string) (string, error) {
	if !cfg.HasContainer(name) {
		return "", fmt.Errorf("container '%s' not found in config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return "", fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	return lxc.GetIP(lxcName)
}

// Exists checks if a container exists
func Exists(cfg *config.Config, name string) bool {
	if !cfg.HasContainer(name) {
		return false
	}

	lxcName := cfg.GetLXCName(name)
	return lxc.Exists(lxcName)
}

// WaitForReady waits for a container to be ready
func WaitForReady(cfg *config.Config, name string, timeout time.Duration) error {
	if !cfg.HasContainer(name) {
		return fmt.Errorf("container '%s' not found in config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	return lxc.WaitForReady(lxcName, timeout)
}
