package operations

import (
	"fmt"
	"sort"
	"strings"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
	"lxc-dev-manager/internal/validation"
)

// Mount mounts a host directory into a container
func Mount(cfg *config.Config, containerName, sourcePath, containerPath string, opts MountOpts) (string, error) {
	if !cfg.HasContainer(containerName) {
		return "", fmt.Errorf("container '%s' not found in config", containerName)
	}

	lxcName := cfg.GetLXCName(containerName)
	if !lxc.Exists(lxcName) {
		return "", fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Validate source path
	resolvedSource, warning, err := validation.ValidateSourcePath(sourcePath)
	if err != nil {
		return "", fmt.Errorf("invalid source path: %w", err)
	}

	// Check risky path
	if warning != "" && !opts.AllowRiskyPath {
		return "", fmt.Errorf("risky path: %s", warning)
	}

	// Validate container path
	if err := validation.ValidateContainerPath(containerPath); err != nil {
		return "", fmt.Errorf("invalid container path: %w", err)
	}

	// Generate mount name if not provided
	deviceName := opts.Name
	if deviceName == "" {
		deviceName = validation.GenerateMountName(resolvedSource)
	}

	// Validate mount name
	if err := validation.ValidateMountName(deviceName); err != nil {
		return "", fmt.Errorf("invalid device name: %w", err)
	}

	// Check for name conflict
	if cfg.HasDevice(containerName, deviceName) {
		return "", fmt.Errorf("device '%s' already exists on container '%s'", deviceName, containerName)
	}

	// Check for path conflict
	if existingName, found := cfg.FindDeviceByPath(containerName, containerPath); found {
		return "", fmt.Errorf("container path '%s' is already mounted by device '%s'", containerPath, existingName)
	}

	// Check privileged container restrictions
	privileged, err := lxc.IsPrivileged(lxcName)
	if err != nil {
		return "", fmt.Errorf("failed to check container privilege status: %w", err)
	}

	if privileged {
		if opts.ReadWrite {
			return "", fmt.Errorf("read-write mounts are disabled for privileged containers")
		}
		if strings.HasPrefix(resolvedSource, "/home") {
			return "", fmt.Errorf("mounting /home to privileged containers is blocked for security reasons")
		}
	}

	// Build config map
	deviceConfig := map[string]string{
		"source": resolvedSource,
		"path":   containerPath,
	}
	if !opts.ReadWrite {
		deviceConfig["readonly"] = "true"
	}
	if opts.Shift {
		deviceConfig["shift"] = "true"
	}

	// Add device to LXC
	if err := lxc.DeviceAdd(lxcName, deviceName, "disk", deviceConfig); err != nil {
		return "", fmt.Errorf("failed to add device to container: %w", err)
	}

	// Add device to config
	cfg.AddDevice(containerName, deviceName, config.Device{
		Type:   "disk",
		Config: deviceConfig,
	})

	// Save config
	if err := cfg.Save(); err != nil {
		// Try to rollback LXC device if config save fails
		lxc.DeviceRemove(lxcName, deviceName)
		return "", fmt.Errorf("failed to save config: %w", err)
	}

	return deviceName, nil
}

// Unmount removes a mount from a container
func Unmount(cfg *config.Config, containerName, nameOrPath string) error {
	if !cfg.HasContainer(containerName) {
		return fmt.Errorf("container '%s' not found in config", containerName)
	}

	lxcName := cfg.GetLXCName(containerName)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Determine if the argument is a path or a device name
	var deviceName string
	if strings.HasPrefix(nameOrPath, "/") {
		// It's a path, look up the device name
		var found bool
		deviceName, found = cfg.FindDeviceByPath(containerName, nameOrPath)
		if !found {
			return fmt.Errorf("no device found with path '%s' in container '%s'", nameOrPath, containerName)
		}
	} else {
		deviceName = nameOrPath
	}

	// Verify device exists in config
	if !cfg.HasDevice(containerName, deviceName) {
		return fmt.Errorf("device '%s' not found in container '%s'", deviceName, containerName)
	}

	// Remove device from LXC
	if err := lxc.DeviceRemove(lxcName, deviceName); err != nil {
		return fmt.Errorf("failed to remove device from LXC: %w", err)
	}

	// Remove device from config
	cfg.RemoveDevice(containerName, deviceName)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// ListMounts lists all mounts for a container
func ListMounts(cfg *config.Config, containerName string) ([]MountInfo, error) {
	if !cfg.HasContainer(containerName) {
		return nil, fmt.Errorf("container '%s' not found in config", containerName)
	}

	lxcName := cfg.GetLXCName(containerName)
	if !lxc.Exists(lxcName) {
		return nil, fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Get devices from config
	configDevices := cfg.GetDevices(containerName)
	if configDevices == nil {
		configDevices = make(map[string]config.Device)
	}

	// Get devices from LXC (filter to disk type only)
	lxcDevices, err := lxc.DeviceList(lxcName)
	if err != nil {
		return nil, err
	}

	// Build a map of LXC disk devices
	lxcDiskDevices := make(map[string]lxc.DeviceInfo)
	for _, dev := range lxcDevices {
		if dev.Type == "disk" {
			lxcDiskDevices[dev.Name] = dev
		}
	}

	// Build combined mount info list
	var mounts []MountInfo
	seenNames := make(map[string]bool)

	// Process config devices first
	for name, device := range configDevices {
		if device.Type != "disk" {
			continue
		}
		seenNames[name] = true

		info := MountInfo{
			Name:   name,
			Source: device.Config["source"],
			Path:   device.Config["path"],
			Mode:   GetMode(device.Config),
		}

		if _, existsInLXC := lxcDiskDevices[name]; existsInLXC {
			info.Status = "ok"
		} else {
			info.Status = "missing"
		}

		mounts = append(mounts, info)
	}

	// Process LXC devices not in config (untracked)
	for name, dev := range lxcDiskDevices {
		if seenNames[name] {
			continue
		}

		mounts = append(mounts, MountInfo{
			Name:   name,
			Source: dev.Config["source"],
			Path:   dev.Config["path"],
			Mode:   GetMode(dev.Config),
			Status: "untracked",
		})
	}

	// Sort by name for consistent output
	sort.Slice(mounts, func(i, j int) bool {
		return mounts[i].Name < mounts[j].Name
	})

	return mounts, nil
}

// SyncMounts synchronizes mounts between config and LXC
func SyncMounts(cfg *config.Config, containerName string) error {
	if !cfg.HasContainer(containerName) {
		return fmt.Errorf("container '%s' not found in config", containerName)
	}

	lxcName := cfg.GetLXCName(containerName)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	mounts, err := ListMounts(cfg, containerName)
	if err != nil {
		return err
	}

	configDevices := cfg.GetDevices(containerName)
	if configDevices == nil {
		configDevices = make(map[string]config.Device)
	}

	for _, m := range mounts {
		switch m.Status {
		case "untracked":
			// Add to config
			deviceConfig := map[string]string{
				"source": m.Source,
				"path":   m.Path,
			}
			if m.Mode == "ro" {
				deviceConfig["readonly"] = "true"
			}
			cfg.AddDevice(containerName, m.Name, config.Device{
				Type:   "disk",
				Config: deviceConfig,
			})

		case "missing":
			// Re-add to LXC
			device := configDevices[m.Name]
			if err := lxc.DeviceAdd(lxcName, m.Name, device.Type, device.Config); err != nil {
				return fmt.Errorf("failed to re-add device '%s': %w", m.Name, err)
			}
		}
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
