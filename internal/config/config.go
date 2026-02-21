package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"lxc-dev-manager/internal/validation"

	"gopkg.in/yaml.v3"
)

// ErrNoProject is returned when no project config file exists
var ErrNoProject = errors.New("no project found in current directory")

const (
	ConfigFile  = "containers.yaml"
	lockFile    = "containers.yaml.lock"
	lockTimeout = 5 * time.Second
)

type Config struct {
	Dir        string               `yaml:"-"` // directory containing this config file (not serialized)
	Project    string               `yaml:"project"`
	Defaults   Defaults             `yaml:"defaults"`
	Containers map[string]Container `yaml:"containers"`
}

type User struct {
	Name     string `yaml:"name,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type Defaults struct {
	Ports []int `yaml:"ports"`
	User  User  `yaml:"user,omitempty"`
}

type Snapshot struct {
	Description string `yaml:"description,omitempty"`
	CreatedAt   string `yaml:"created_at"`
}

type Device struct {
	Type   string            `yaml:"type"`
	Config map[string]string `yaml:"config,omitempty"`
}

type SyncEntry struct {
	Source string `yaml:"source"` // Host path (relative to containers.yaml dir or absolute)
	Dest   string `yaml:"dest"`   // Container path
}

type Container struct {
	Image     string              `yaml:"image"`
	Ports     []int               `yaml:"ports,omitempty"`
	User      User                `yaml:"user,omitempty"`
	Sync      []SyncEntry         `yaml:"sync,omitempty"`
	Snapshots map[string]Snapshot `yaml:"snapshots,omitempty"`
	Devices   map[string]Device   `yaml:"devices,omitempty"`
}

// Load reads the config from the given directory.
// If dir is empty, it uses the current working directory.
func Load(dir string) (*Config, error) {
	if dir == "" {
		dir = "."
	}
	configPath := filepath.Join(dir, ConfigFile)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoProject
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML in %s: %w", ConfigFile, err)
	}

	cfg.Dir = dir

	if cfg.Containers == nil {
		cfg.Containers = make(map[string]Container)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks all configuration values for correctness
func (c *Config) Validate() error {
	// Validate project name
	if c.Project != "" && !IsValidProjectName(c.Project) {
		return fmt.Errorf("invalid project name %q", c.Project)
	}

	// Validate default ports
	if err := validation.ValidatePorts(c.Defaults.Ports); err != nil {
		return fmt.Errorf("invalid default ports: %w", err)
	}

	// Validate each container
	for name, container := range c.Containers {
		if err := validation.ValidateFullContainerName(c.Project, name); err != nil {
			return fmt.Errorf("container '%s': %w", name, err)
		}

		if len(container.Ports) > 0 {
			if err := validation.ValidatePorts(container.Ports); err != nil {
				return fmt.Errorf("container '%s': %w", name, err)
			}
		}

		// Validate devices
		for deviceName, device := range container.Devices {
			if err := validateDevice(deviceName, device); err != nil {
				return fmt.Errorf("container '%s' device '%s': %w", name, deviceName, err)
			}
		}
	}

	return nil
}

// validateDevice validates a single device configuration
func validateDevice(name string, device Device) error {
	// Device type must not be empty
	if device.Type == "" {
		return fmt.Errorf("device type must not be empty")
	}

	// For disk devices, validate required fields
	if device.Type == "disk" {
		if device.Config == nil {
			return fmt.Errorf("disk device requires 'source' config key")
		}
		source, hasSource := device.Config["source"]
		path, hasPath := device.Config["path"]

		if !hasSource || source == "" {
			return fmt.Errorf("disk device requires 'source' config key")
		}
		if !hasPath || path == "" {
			return fmt.Errorf("disk device requires 'path' config key")
		}

		// Validate paths don't contain control characters
		if containsControlChars(source) {
			return fmt.Errorf("source path contains control characters")
		}
		if containsControlChars(path) {
			return fmt.Errorf("path contains control characters")
		}
	}

	return nil
}

// containsControlChars checks if a string contains control characters
func containsControlChars(s string) bool {
	for _, r := range s {
		if r < 32 || r == 127 {
			return true
		}
	}
	return false
}

// GetLXCName returns the full LXC container name with project prefix
func (c *Config) GetLXCName(shortName string) string {
	if c.Project == "" {
		return shortName
	}
	return c.Project + "-" + shortName
}

// GetShortName extracts short name from LXC name by stripping project prefix
func (c *Config) GetShortName(lxcName string) string {
	if c.Project == "" {
		return lxcName
	}
	prefix := c.Project + "-"
	if strings.HasPrefix(lxcName, prefix) {
		return strings.TrimPrefix(lxcName, prefix)
	}
	return lxcName
}

// HasProject returns true if project is initialized
func (c *Config) HasProject() bool {
	return c.Project != ""
}

// GetProjectFromFolder returns the directory name to use as a project name.
// If dir is non-empty, it uses the base name of that path.
// If dir is empty, it uses the current working directory name.
func GetProjectFromFolder(dir string) (string, error) {
	if dir != "" {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		return filepath.Base(absDir), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Base(cwd), nil
}

// IsValidProjectName validates project name (alphanumeric, hyphens, underscores only)
func IsValidProjectName(name string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return re.MatchString(name)
}

func (c *Config) Save() error {
	dir := c.Dir
	if dir == "" {
		dir = "."
	}
	configPath := filepath.Join(dir, ConfigFile)

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return atomicWriteFile(configPath, data, 0644)
}

// atomicWriteFile writes data to a file atomically using temp file + rename.
// This prevents corruption from partial writes if the process is interrupted.
func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	if dir == "" {
		dir = "."
	}

	tmp, err := os.CreateTemp(dir, ".containers.yaml.tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()

	success := false
	defer func() {
		if !success {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}

// ConfigLock represents an exclusive lock on the config file.
// Use this when performing Load→Modify→Save operations to prevent race conditions.
type ConfigLock struct {
	file *os.File
}

// AcquireLock acquires an exclusive lock on the config file with timeout.
// If dir is empty, it uses the current working directory.
func AcquireLock(dir string) (*ConfigLock, error) {
	if dir == "" {
		dir = "."
	}
	lockPath := filepath.Join(dir, lockFile)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	deadline := time.Now().Add(lockTimeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			f.Close()
			return nil, fmt.Errorf("timeout waiting for config lock (another instance may be running)")
		}
		time.Sleep(100 * time.Millisecond)
	}

	return &ConfigLock{file: f}, nil
}

// Release releases the config lock.
func (l *ConfigLock) Release() error {
	if l.file == nil {
		return nil
	}
	syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	err := l.file.Close()
	l.file = nil
	return err
}

// LoadWithLock loads the config while holding an exclusive lock.
// If dir is empty, it uses the current working directory.
// The caller must call Release() on the returned lock when done.
func LoadWithLock(dir string) (*Config, *ConfigLock, error) {
	lock, err := AcquireLock(dir)
	if err != nil {
		return nil, nil, err
	}

	cfg, err := Load(dir)
	if err != nil {
		lock.Release()
		return nil, nil, err
	}

	return cfg, lock, nil
}

func (c *Config) AddContainer(name, image string) {
	c.Containers[name] = Container{
		Image: image,
	}
}

func (c *Config) RemoveContainer(name string) {
	delete(c.Containers, name)
}

// SetContainerImage updates the image for a container
func (c *Config) SetContainerImage(name, image string) bool {
	container, ok := c.Containers[name]
	if !ok {
		return false
	}
	container.Image = image
	c.Containers[name] = container
	return true
}

func (c *Config) GetPorts(name string) []int {
	if container, ok := c.Containers[name]; ok && len(container.Ports) > 0 {
		return container.Ports
	}
	return c.Defaults.Ports
}

// GetUser returns the user config for a container (per-container > defaults > hardcoded)
func (c *Config) GetUser(name string) User {
	// Check per-container first
	if container, ok := c.Containers[name]; ok && container.User.Name != "" {
		user := container.User
		// Fill in missing password from defaults or hardcoded
		if user.Password == "" {
			user.Password = c.Defaults.User.Password
		}
		if user.Password == "" {
			user.Password = "dev"
		}
		return user
	}
	// Fall back to defaults
	if c.Defaults.User.Name != "" {
		user := c.Defaults.User
		if user.Password == "" {
			user.Password = "dev"
		}
		return user
	}
	// Hardcoded fallback
	return User{Name: "dev", Password: "dev"}
}

func (c *Config) HasContainer(name string) bool {
	_, ok := c.Containers[name]
	return ok
}

func (c *Config) AddSnapshot(containerName, snapshotName, description string) {
	container := c.Containers[containerName]
	if container.Snapshots == nil {
		container.Snapshots = make(map[string]Snapshot)
	}
	container.Snapshots[snapshotName] = Snapshot{
		Description: description,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	c.Containers[containerName] = container
}

func (c *Config) RemoveSnapshot(containerName, snapshotName string) {
	if container, ok := c.Containers[containerName]; ok {
		delete(container.Snapshots, snapshotName)
		c.Containers[containerName] = container
	}
}

func (c *Config) GetSnapshots(containerName string) map[string]Snapshot {
	if container, ok := c.Containers[containerName]; ok {
		return container.Snapshots
	}
	return nil
}

func (c *Config) HasSnapshot(containerName, snapshotName string) bool {
	if container, ok := c.Containers[containerName]; ok {
		_, exists := container.Snapshots[snapshotName]
		return exists
	}
	return false
}

// AddDevice adds a device to a container
func (c *Config) AddDevice(containerName, deviceName string, device Device) {
	container, ok := c.Containers[containerName]
	if !ok {
		return
	}
	if container.Devices == nil {
		container.Devices = make(map[string]Device)
	}
	container.Devices[deviceName] = device
	c.Containers[containerName] = container
}

// RemoveDevice removes a device from a container
func (c *Config) RemoveDevice(containerName, deviceName string) {
	if container, ok := c.Containers[containerName]; ok {
		delete(container.Devices, deviceName)
		c.Containers[containerName] = container
	}
}

// GetDevices returns all devices for a container
func (c *Config) GetDevices(containerName string) map[string]Device {
	if container, ok := c.Containers[containerName]; ok {
		return container.Devices
	}
	return nil
}

// HasDevice checks if a device exists on a container
func (c *Config) HasDevice(containerName, deviceName string) bool {
	if container, ok := c.Containers[containerName]; ok {
		_, exists := container.Devices[deviceName]
		return exists
	}
	return false
}

// AddSyncEntry adds a sync entry to a container. If an entry with the same source
// already exists, it is overwritten.
func (c *Config) AddSyncEntry(containerName string, entry SyncEntry) {
	container, ok := c.Containers[containerName]
	if !ok {
		return
	}
	// Overwrite if source already exists
	for i, e := range container.Sync {
		if e.Source == entry.Source {
			container.Sync[i] = entry
			c.Containers[containerName] = container
			return
		}
	}
	container.Sync = append(container.Sync, entry)
	c.Containers[containerName] = container
}

// RemoveSyncEntry removes a sync entry by source path
func (c *Config) RemoveSyncEntry(containerName, source string) {
	container, ok := c.Containers[containerName]
	if !ok {
		return
	}
	for i, e := range container.Sync {
		if e.Source == source {
			container.Sync = append(container.Sync[:i], container.Sync[i+1:]...)
			c.Containers[containerName] = container
			return
		}
	}
}

// GetSyncEntries returns all sync entries for a container
func (c *Config) GetSyncEntries(containerName string) []SyncEntry {
	if container, ok := c.Containers[containerName]; ok {
		return container.Sync
	}
	return nil
}

// FindDeviceByPath finds a device name by its container path (for unmount by path)
func (c *Config) FindDeviceByPath(containerName, path string) (string, bool) {
	container, ok := c.Containers[containerName]
	if !ok {
		return "", false
	}
	for name, device := range container.Devices {
		if device.Config != nil && device.Config["path"] == path {
			return name, true
		}
	}
	return "", false
}
