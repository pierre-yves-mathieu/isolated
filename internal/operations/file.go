package operations

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
)

// CopyToContainer copies a file or directory from host to container
func CopyToContainer(cfg *config.Config, containerName, localPath, remotePath string, opts CopyOpts) error {
	if !cfg.HasContainer(containerName) {
		return fmt.Errorf("container '%s' not found in config", containerName)
	}

	lxcName := cfg.GetLXCName(containerName)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Validate source exists on host
	info, err := os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source '%s' does not exist", localPath)
		}
		return fmt.Errorf("cannot access source '%s': %w", localPath, err)
	}

	if remotePath == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	// Expand ~ to user's home directory
	if strings.HasPrefix(remotePath, "~/") {
		user := cfg.GetUser(containerName)
		remotePath = "/home/" + user.Name + remotePath[1:]
	} else if remotePath == "~" {
		user := cfg.GetUser(containerName)
		remotePath = "/home/" + user.Name
	}

	// Determine if recursive (directory)
	recursive := info.IsDir()

	// Get the destination directory to check/create
	destDir := path.Dir(remotePath)

	// Check if destination directory exists
	user := cfg.GetUser(containerName)
	if !lxc.DirExists(lxcName, destDir) {
		if !opts.AutoCreateDir {
			return fmt.Errorf("destination directory '%s' does not exist", destDir)
		}
		if err := lxc.Exec(lxcName, "mkdir", "-p", destDir); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		lxc.Exec(lxcName, "chown", user.Name+":"+user.Name, destDir)
	}

	// Push the file
	pushPath := remotePath
	if recursive {
		pushPath = path.Dir(remotePath)
	}

	if err := lxc.FilePush(lxcName, localPath, pushPath, recursive); err != nil {
		return err
	}

	// Fix ownership
	if recursive {
		if err := lxc.Exec(lxcName, "chown", "-R", user.Name+":"+user.Name, remotePath); err != nil {
			return fmt.Errorf("could not set ownership: %w", err)
		}
	} else {
		if err := lxc.Exec(lxcName, "chown", user.Name+":"+user.Name, remotePath); err != nil {
			return fmt.Errorf("could not set ownership: %w", err)
		}
	}

	return nil
}

// CopyFromContainer copies a file or directory from container to host
func CopyFromContainer(cfg *config.Config, containerName, remotePath, localPath string) error {
	if !cfg.HasContainer(containerName) {
		return fmt.Errorf("container '%s' not found in config", containerName)
	}

	lxcName := cfg.GetLXCName(containerName)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Expand ~ to user's home directory
	if strings.HasPrefix(remotePath, "~/") {
		user := cfg.GetUser(containerName)
		remotePath = "/home/" + user.Name + remotePath[1:]
	} else if remotePath == "~" {
		user := cfg.GetUser(containerName)
		remotePath = "/home/" + user.Name
	}

	// Check if source exists in container
	if !lxc.FileExists(lxcName, remotePath) {
		return fmt.Errorf("source '%s' does not exist in container %s", remotePath, containerName)
	}

	// Determine if recursive (directory)
	recursive := lxc.IsDir(lxcName, remotePath)

	// Ensure local destination directory exists
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Pull the file
	if err := lxc.FilePull(lxcName, remotePath, localPath, recursive); err != nil {
		return err
	}

	return nil
}

// CopyBetweenContainers copies a file or directory from one container to another
func CopyBetweenContainers(cfg *config.Config, srcContainer, srcPath, destContainer, destPath string, opts CopyOpts) error {
	// Create temp directory for intermediate storage
	tempDir, err := os.MkdirTemp("", "lxc-copy-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Pull from source container to temp
	tempPath := filepath.Join(tempDir, filepath.Base(srcPath))
	if err := CopyFromContainer(cfg, srcContainer, srcPath, tempPath); err != nil {
		return fmt.Errorf("failed to pull from source: %w", err)
	}

	// Push to destination container
	if err := CopyToContainer(cfg, destContainer, tempPath, destPath, opts); err != nil {
		return fmt.Errorf("failed to push to destination: %w", err)
	}

	return nil
}
