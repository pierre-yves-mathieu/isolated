package operations

import (
	"fmt"
	"sort"
	"time"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
)

// CreateSnapshot creates a snapshot of a container
func CreateSnapshot(cfg *config.Config, containerName, snapshotName, description string) error {
	if !cfg.HasContainer(containerName) {
		return fmt.Errorf("container '%s' not found in config", containerName)
	}

	lxcName := cfg.GetLXCName(containerName)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Check if snapshot already exists
	if lxc.SnapshotExists(lxcName, snapshotName) {
		return fmt.Errorf("snapshot '%s' already exists", snapshotName)
	}

	if err := lxc.Snapshot(lxcName, snapshotName); err != nil {
		return err
	}

	// Register in config
	cfg.AddSnapshot(containerName, snapshotName, description)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// ListSnapshots lists all snapshots for a container
func ListSnapshots(cfg *config.Config, containerName string) ([]SnapshotInfo, error) {
	if !cfg.HasContainer(containerName) {
		return nil, fmt.Errorf("container '%s' not found in config", containerName)
	}

	lxcName := cfg.GetLXCName(containerName)
	if !lxc.Exists(lxcName) {
		return nil, fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Get snapshots from LXC
	lxcSnapshots, err := lxc.ListSnapshots(lxcName)
	if err != nil {
		return nil, err
	}

	if len(lxcSnapshots) == 0 {
		return nil, nil
	}

	// Get metadata from config
	configSnapshots := cfg.GetSnapshots(containerName)

	// Sort snapshots by name
	sort.Strings(lxcSnapshots)

	var result []SnapshotInfo
	for _, name := range lxcSnapshots {
		info := SnapshotInfo{
			Name: name,
		}

		if configSnapshots != nil {
			if meta, ok := configSnapshots[name]; ok {
				info.Description = meta.Description
				if meta.CreatedAt != "" {
					t, err := time.Parse(time.RFC3339, meta.CreatedAt)
					if err == nil {
						info.CreatedAt = t
					}
				}
			}
		}

		result = append(result, info)
	}

	return result, nil
}

// DeleteSnapshot deletes a snapshot from a container
func DeleteSnapshot(cfg *config.Config, containerName, snapshotName string) error {
	if !cfg.HasContainer(containerName) {
		return fmt.Errorf("container '%s' not found in config", containerName)
	}

	lxcName := cfg.GetLXCName(containerName)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Prevent deleting initial-state
	if snapshotName == "initial-state" {
		return fmt.Errorf("cannot delete 'initial-state' snapshot")
	}

	if !lxc.SnapshotExists(lxcName, snapshotName) {
		return fmt.Errorf("snapshot '%s' does not exist", snapshotName)
	}

	if err := lxc.DeleteSnapshot(lxcName, snapshotName); err != nil {
		return err
	}

	// Remove from config
	cfg.RemoveSnapshot(containerName, snapshotName)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
