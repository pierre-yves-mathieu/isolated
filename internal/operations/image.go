package operations

import (
	"fmt"
	"io"
	"time"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
)

// ListImages returns all local images
func ListImages(all bool) ([]ImageInfo, error) {
	images, err := lxc.ListImages(all)
	if err != nil {
		return nil, err
	}

	var result []ImageInfo
	for _, img := range images {
		result = append(result, ImageInfo{
			Alias:       img.Alias,
			Fingerprint: img.Fingerprint,
			Size:        img.Size,
			Description: img.Description,
		})
	}

	return result, nil
}

// CreateImage creates an image from a container
func CreateImage(cfg *config.Config, containerName, imageName string, stdout, stderr io.Writer) error {
	if !cfg.HasContainer(containerName) {
		return fmt.Errorf("container '%s' not found in config", containerName)
	}

	lxcName := cfg.GetLXCName(containerName)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	snapshotName := fmt.Sprintf("snapshot-%d", time.Now().Unix())

	// Check if running, stop if so
	status, err := lxc.GetStatus(lxcName)
	if err != nil {
		return err
	}

	wasRunning := status == "RUNNING"

	// Stop container if running
	if wasRunning {
		if err := lxc.Stop(lxcName); err != nil {
			return err
		}
	}

	// Create snapshot (instant with ZFS/btrfs)
	if err := lxc.Snapshot(lxcName, snapshotName); err != nil {
		// Try to restart if it was running
		if wasRunning {
			lxc.Start(lxcName)
		}
		return err
	}

	// Publish snapshot as image
	err = lxc.PublishSnapshotWithProgress(lxcName, snapshotName, imageName, stdout, stderr)

	// Clean up snapshot regardless of publish result
	lxc.DeleteSnapshot(lxcName, snapshotName)

	if err != nil {
		// Try to restart if it was running
		if wasRunning {
			lxc.Start(lxcName)
		}
		return err
	}

	// Restart if was running
	if wasRunning {
		if err := lxc.Start(lxcName); err != nil {
			return fmt.Errorf("failed to restart container: %w", err)
		}
	}

	return nil
}

// DeleteImage deletes an image by alias
func DeleteImage(name string) error {
	if !lxc.ImageExists(name) {
		return fmt.Errorf("image '%s' not found", name)
	}

	return lxc.DeleteImage(name)
}

// RenameImage renames an image
func RenameImage(oldName, newName string) error {
	if !lxc.ImageExists(oldName) {
		return fmt.Errorf("image '%s' not found", oldName)
	}

	if lxc.ImageExists(newName) {
		return fmt.Errorf("image '%s' already exists", newName)
	}

	return lxc.RenameImage(oldName, newName)
}

// ImageExists checks if an image exists
func ImageExists(name string) bool {
	return lxc.ImageExists(name)
}
