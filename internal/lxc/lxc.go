package lxc

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Launch creates and starts a new container
func Launch(name, image string) error {
	output, err := DefaultExecutor.RunCombined("launch", image, name)
	if err != nil {
		return fmt.Errorf("failed to launch container: %s", string(output))
	}
	return nil
}

// ConfigSet sets a config key on a container
func ConfigSet(name, key, value string) error {
	output, err := DefaultExecutor.RunCombined("config", "set", name, key, value)
	if err != nil {
		return fmt.Errorf("failed to set config %s: %s", key, string(output))
	}
	return nil
}

// EnableNesting enables Docker-in-LXC support
func EnableNesting(name string) error {
	configs := map[string]string{
		"security.nesting":                     "true",
		"security.syscalls.intercept.mknod":    "true",
		"security.syscalls.intercept.setxattr": "true",
	}

	for key, value := range configs {
		if err := ConfigSet(name, key, value); err != nil {
			return err
		}
	}
	return nil
}

// Exec runs a command inside a container
func Exec(name string, args ...string) error {
	cmdArgs := append([]string{"exec", name, "--"}, args...)
	output, err := DefaultExecutor.RunCombined(cmdArgs...)
	if err != nil {
		return fmt.Errorf("exec failed: %s", string(output))
	}
	return nil
}

// ExecScript runs a shell script inside a container
func ExecScript(name, script string) error {
	return Exec(name, "bash", "-c", script)
}

// SetupUser creates a user with password and sudo access
func SetupUser(containerName, username, password string) error {
	script := fmt.Sprintf(`
		# Create user if not exists
		id %s &>/dev/null || useradd -m -s /bin/bash %s

		# Set password
		echo '%s:%s' | chpasswd

		# Add to sudo group
		usermod -aG sudo %s 2>/dev/null || usermod -aG wheel %s 2>/dev/null || true

		# Enable passwordless sudo
		echo '%s ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/%s
		chmod 440 /etc/sudoers.d/%s
	`, username, username, username, password, username, username, username, username, username)
	return ExecScript(containerName, script)
}

// EnableSSH ensures SSH is installed and running
func EnableSSH(name string) error {
	script := `
		# Install openssh-server if not present
		which sshd &>/dev/null || {
			apt-get update -qq
			apt-get install -y -qq openssh-server
		}

		# Ensure SSH is enabled and started
		systemctl enable ssh 2>/dev/null || systemctl enable sshd 2>/dev/null || true
		systemctl start ssh 2>/dev/null || systemctl start sshd 2>/dev/null || true
	`
	return ExecScript(name, script)
}

// WaitForReady waits for container to be ready (cloud-init complete)
func WaitForReady(name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check if cloud-init is done
		output, err := DefaultExecutor.RunCombined("exec", name, "--", "cloud-init", "status")
		if err == nil && strings.Contains(string(output), "done") {
			return nil
		}

		// Also check if it's just running (no cloud-init)
		if strings.Contains(string(output), "not found") {
			// No cloud-init, assume ready
			time.Sleep(2 * time.Second)
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for container to be ready")
}

// Start starts a stopped container
func Start(name string) error {
	output, err := DefaultExecutor.RunCombined("start", name)
	if err != nil {
		return fmt.Errorf("failed to start container: %s", string(output))
	}
	return nil
}

// Stop stops a running container
func Stop(name string) error {
	// Use a short timeout to avoid long waits for graceful shutdown
	output, err := DefaultExecutor.RunCombined("stop", name, "--timeout=5")
	if err != nil {
		return fmt.Errorf("failed to stop container: %s", string(output))
	}
	return nil
}

// Delete removes a container
func Delete(name string) error {
	output, err := DefaultExecutor.RunCombined("delete", name, "--force")
	if err != nil {
		return fmt.Errorf("failed to delete container: %s", string(output))
	}
	return nil
}

// Publish creates an image from a container
func Publish(name, alias string) error {
	output, err := DefaultExecutor.RunCombined("publish", name, "--alias", alias)
	if err != nil {
		return fmt.Errorf("failed to publish container: %s", string(output))
	}
	return nil
}

// Snapshot creates a named snapshot of a container
func Snapshot(container, snapshotName string) error {
	output, err := DefaultExecutor.RunCombined("snapshot", container, snapshotName)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %s", string(output))
	}
	return nil
}

// DeleteSnapshot deletes a named snapshot
func DeleteSnapshot(container, snapshotName string) error {
	output, err := DefaultExecutor.RunCombined("delete", container+"/"+snapshotName)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %s", string(output))
	}
	return nil
}

// Restore restores a container from a snapshot
func Restore(container, snapshotName string) error {
	output, err := DefaultExecutor.RunCombined("restore", container, snapshotName)
	if err != nil {
		return fmt.Errorf("failed to restore snapshot: %s", string(output))
	}
	return nil
}

// SnapshotExists checks if a snapshot exists
func SnapshotExists(container, snapshotName string) bool {
	_, err := DefaultExecutor.Run("info", container+"/"+snapshotName)
	return err == nil
}

// Copy creates a clone of an existing container
func Copy(source, dest string) error {
	output, err := DefaultExecutor.RunCombined("copy", source, dest)
	if err != nil {
		return fmt.Errorf("failed to copy container: %s", string(output))
	}
	return nil
}

// CopySnapshot creates a container from a snapshot of another container
func CopySnapshot(source, snapshotName, dest string) error {
	snapshotPath := source + "/" + snapshotName
	output, err := DefaultExecutor.RunCombined("copy", snapshotPath, dest)
	if err != nil {
		return fmt.Errorf("failed to copy from snapshot: %s", string(output))
	}
	return nil
}

// DirExists checks if a directory exists in a container
func DirExists(container, path string) bool {
	err := Exec(container, "test", "-d", path)
	return err == nil
}

// FilePush copies a file or directory from host to container
func FilePush(container, localPath, remotePath string, recursive bool) error {
	args := []string{"file", "push"}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, localPath, container+"/"+remotePath)
	output, err := DefaultExecutor.RunCombined(args...)
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if strings.Contains(errMsg, "Not Found") || strings.Contains(errMsg, "not found") {
			return fmt.Errorf("destination path '%s' not found in container (does the directory exist?)", remotePath)
		}
		return fmt.Errorf("failed to copy to container: %s", errMsg)
	}
	return nil
}

// FilePull copies a file or directory from container to host
func FilePull(container, remotePath, localPath string, recursive bool) error {
	args := []string{"file", "pull"}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, container+"/"+remotePath, localPath)
	output, err := DefaultExecutor.RunCombined(args...)
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if strings.Contains(errMsg, "Not Found") || strings.Contains(errMsg, "not found") {
			return fmt.Errorf("source path '%s' not found in container", remotePath)
		}
		return fmt.Errorf("failed to copy from container: %s", errMsg)
	}
	return nil
}

// FileExists checks if a file exists in a container
func FileExists(container, path string) bool {
	err := Exec(container, "test", "-e", path)
	return err == nil
}

// IsDir checks if a path is a directory in a container
func IsDir(container, path string) bool {
	err := Exec(container, "test", "-d", path)
	return err == nil
}

// ListSnapshots returns all snapshot names for a container
func ListSnapshots(container string) ([]string, error) {
	output, err := DefaultExecutor.Run("query", "/1.0/instances/"+container+"/snapshots")
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %v", err)
	}

	// Parse JSON array of snapshot paths like ["/1.0/instances/foo/snapshots/snap1"]
	var paths []string
	if err := json.Unmarshal(output, &paths); err != nil {
		return nil, fmt.Errorf("failed to parse snapshots: %v", err)
	}

	// Extract snapshot names from paths
	var names []string
	for _, path := range paths {
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			names = append(names, parts[len(parts)-1])
		}
	}
	return names, nil
}

// PublishSnapshotWithProgress publishes a container snapshot as an image,
// streaming progress output to the provided writers
func PublishSnapshotWithProgress(container, snapshotName, alias string, stdout, stderr io.Writer) error {
	source := container
	if snapshotName != "" {
		source = container + "/" + snapshotName
	}

	cmd := exec.Command("lxc", "publish", source, "--alias", alias)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to publish image: %w", err)
	}
	return nil
}

// ImageInfo holds information about an image
type ImageInfo struct {
	Alias       string
	Fingerprint string
	Size        string
	Description string
	CreatedAt   string
}

// ListImages returns all local images
func ListImages(all bool) ([]ImageInfo, error) {
	// Format: l=alias, f=fingerprint, s=size, d=description
	output, err := DefaultExecutor.Run("image", "list", "--format=csv", "-c", "lfsd")
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %v", err)
	}

	var images []ImageInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ",", 4)
		if len(parts) >= 3 {
			info := ImageInfo{
				Alias:       parts[0],
				Fingerprint: parts[1],
				Size:        parts[2],
			}
			if len(parts) >= 4 {
				info.Description = parts[3]
			}

			// Skip non-aliased images unless all is true
			if !all && info.Alias == "" {
				continue
			}

			images = append(images, info)
		}
	}

	return images, nil
}

// DeleteImage deletes an image by alias or fingerprint
func DeleteImage(alias string) error {
	output, err := DefaultExecutor.RunCombined("image", "delete", alias)
	if err != nil {
		return fmt.Errorf("failed to delete image: %s", string(output))
	}
	return nil
}

// GetImageFingerprint returns the fingerprint for an image alias
func GetImageFingerprint(alias string) (string, error) {
	output, err := DefaultExecutor.Run("image", "list", alias, "--format=csv", "-c", "f")
	if err != nil {
		return "", fmt.Errorf("failed to get image fingerprint: %v", err)
	}

	fp := strings.TrimSpace(string(output))
	if fp == "" {
		return "", fmt.Errorf("image '%s' not found", alias)
	}

	// May have multiple lines, take first
	if idx := strings.Index(fp, "\n"); idx > 0 {
		fp = fp[:idx]
	}

	return fp, nil
}

// RenameImage renames an image by creating a new alias and deleting the old one
func RenameImage(oldAlias, newAlias string) error {
	// Get fingerprint of old alias
	fp, err := GetImageFingerprint(oldAlias)
	if err != nil {
		return err
	}

	// Create new alias
	output, err := DefaultExecutor.RunCombined("image", "alias", "create", newAlias, fp)
	if err != nil {
		return fmt.Errorf("failed to create new alias: %s", string(output))
	}

	// Delete old alias
	output, err = DefaultExecutor.RunCombined("image", "alias", "delete", oldAlias)
	if err != nil {
		// Try to clean up new alias
		DefaultExecutor.RunCombined("image", "alias", "delete", newAlias)
		return fmt.Errorf("failed to delete old alias: %s", string(output))
	}

	return nil
}

// ImageExists checks if an image exists by alias
func ImageExists(alias string) bool {
	_, err := GetImageFingerprint(alias)
	return err == nil
}

// GetIP returns the container's IP address (prefers eth0)
func GetIP(name string) (string, error) {
	output, err := DefaultExecutor.Run("list", name, "-c4", "-f", "csv")
	if err != nil {
		return "", fmt.Errorf("failed to get IP: %v", err)
	}

	// Output format: "IP1 (iface1)\nIP2 (iface2)\n..." with surrounding quotes
	content := strings.TrimSpace(string(output))
	content = strings.Trim(content, "\"")

	// Parse each line looking for eth0 first, then fall back to first IP
	lines := strings.Split(content, "\n")
	var firstIP string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract IP (before the space)
		ip := line
		if idx := strings.Index(line, " "); idx > 0 {
			ip = line[:idx]
		}

		// Prefer eth0
		if strings.Contains(line, "(eth0)") {
			return ip, nil
		}

		// Save first valid IP as fallback
		if firstIP == "" && ip != "" {
			firstIP = ip
		}
	}

	if firstIP == "" {
		return "", fmt.Errorf("container has no IP address")
	}

	return firstIP, nil
}

// GetStatus returns the container status
func GetStatus(name string) (string, error) {
	output, err := DefaultExecutor.Run("list", name, "-cs", "-f", "csv")
	if err != nil {
		return "", fmt.Errorf("failed to get status: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Exists checks if a container exists
func Exists(name string) bool {
	_, err := DefaultExecutor.Run("info", name)
	return err == nil
}

// ContainerInfo holds container information
type ContainerInfo struct {
	Name   string
	Status string
	IP     string
}

// ListAll returns all containers with their status and IP
func ListAll() ([]ContainerInfo, error) {
	output, err := DefaultExecutor.Run("list", "-c", "ns4", "-f", "csv")
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %v", err)
	}

	var containers []ContainerInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			info := ContainerInfo{
				Name:   parts[0],
				Status: parts[1],
			}
			if len(parts) >= 3 {
				ip := parts[2]
				if idx := strings.Index(ip, " "); idx > 0 {
					ip = ip[:idx]
				}
				info.IP = ip
			}
			containers = append(containers, info)
		}
	}

	return containers, nil
}

// DeviceInfo holds information about a device attached to a container
type DeviceInfo struct {
	Name   string
	Type   string
	Config map[string]string
}

// DeviceAdd adds a device to a container
func DeviceAdd(container, name, deviceType string, config map[string]string) error {
	args := []string{"config", "device", "add", container, name, deviceType}
	for key, value := range config {
		args = append(args, key+"="+value)
	}
	output, err := DefaultExecutor.RunCombined(args...)
	if err != nil {
		return fmt.Errorf("failed to add device: %s", string(output))
	}
	return nil
}

// DeviceRemove removes a device from a container
func DeviceRemove(container, name string) error {
	output, err := DefaultExecutor.RunCombined("config", "device", "remove", container, name)
	if err != nil {
		return fmt.Errorf("failed to remove device: %s", string(output))
	}
	return nil
}

// DeviceList returns all devices attached to a container
func DeviceList(container string) ([]DeviceInfo, error) {
	output, err := DefaultExecutor.RunCombined("config", "device", "show", container)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %s", string(output))
	}

	// Parse YAML output
	// Format:
	// devicename:
	//   type: disk
	//   key: value
	//   ...
	var rawDevices map[string]map[string]string
	if err := yaml.Unmarshal(output, &rawDevices); err != nil {
		return nil, fmt.Errorf("failed to parse device list: %v", err)
	}

	var devices []DeviceInfo
	for name, props := range rawDevices {
		device := DeviceInfo{
			Name:   name,
			Config: make(map[string]string),
		}
		for key, value := range props {
			if key == "type" {
				device.Type = value
			} else {
				device.Config[key] = value
			}
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// DeviceExists checks if a device exists on a container
func DeviceExists(container, name string) (bool, error) {
	devices, err := DeviceList(container)
	if err != nil {
		return false, err
	}
	for _, device := range devices {
		if device.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// IsPrivileged checks if a container is running in privileged mode
func IsPrivileged(container string) (bool, error) {
	output, err := DefaultExecutor.RunCombined("config", "get", container, "security.privileged")
	if err != nil {
		return false, fmt.Errorf("failed to get privileged status: %s", string(output))
	}
	return strings.TrimSpace(string(output)) == "true", nil
}
