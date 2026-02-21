package operations

import (
	"io"
	"time"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
)

// Executor is the interface for running LXC commands
type Executor = lxc.Executor

// CreateContainerOpts holds options for container creation
type CreateContainerOpts struct {
	Ports    []int
	User     string
	Password string
}

// CloneOpts holds options for container cloning
type CloneOpts struct {
	FromSnapshot string
}

// MountOpts holds options for mounting
type MountOpts struct {
	Name           string
	ReadWrite      bool
	Shift          bool
	AllowRiskyPath bool
}

// CopyOpts holds options for file copy operations
type CopyOpts struct {
	AutoCreateDir bool
}

// ShellOpts holds options for shell access
type ShellOpts struct {
	User string
}

// MountInfo holds combined mount information
type MountInfo struct {
	Name   string
	Source string
	Path   string
	Mode   string // "ro" or "rw"
	Status string // "ok", "untracked", "missing"
}

// SnapshotInfo holds snapshot information
type SnapshotInfo struct {
	Name        string
	Description string
	CreatedAt   time.Time
}

// ContainerInfo holds container status information
type ContainerInfo struct {
	Name   string
	Image  string
	Status string
	IP     string
	Ports  []int
}

// ImageInfo holds image information
type ImageInfo struct {
	Alias       string
	Fingerprint string
	Size        string
	Description string
}

// CreateProjectOpts holds options for project creation
type CreateProjectOpts struct {
	Name  string
	Ports []int
}

// ImageCreateWriter wraps stdout/stderr for image creation progress
type ImageCreateWriter struct {
	Stdout io.Writer
	Stderr io.Writer
}

// GetMode returns "ro" or "rw" based on device config
func GetMode(deviceConfig map[string]string) string {
	if deviceConfig["readonly"] == "true" {
		return "ro"
	}
	return "rw"
}

// ConfigToContainerInfo converts config data to ContainerInfo
func ConfigToContainerInfo(name string, container config.Container, status, ip string, ports []int) ContainerInfo {
	return ContainerInfo{
		Name:   name,
		Image:  container.Image,
		Status: status,
		IP:     ip,
		Ports:  ports,
	}
}
