package lxcmgr

import (
	"time"
)

// ContainerStatus represents the status of a container
type ContainerStatus string

const (
	StatusRunning  ContainerStatus = "RUNNING"
	StatusStopped  ContainerStatus = "STOPPED"
	StatusNotFound ContainerStatus = "NOT FOUND"
)

// ContainerInfo holds container information
type ContainerInfo struct {
	Name   string
	Image  string
	Status ContainerStatus
	IP     string
	Ports  []int
}

// SnapshotInfo holds snapshot information
type SnapshotInfo struct {
	Name        string
	Description string
	CreatedAt   time.Time
}

// MountInfo holds mount information
type MountInfo struct {
	Name     string
	Source   string
	Path     string
	ReadOnly bool
	Shift    bool
	Status   MountStatus
}

// MountStatus represents the status of a mount
type MountStatus string

const (
	MountOK        MountStatus = "ok"
	MountUntracked MountStatus = "untracked"
	MountMissing   MountStatus = "missing"
)

// ImageInfo holds image information
type ImageInfo struct {
	Alias       string
	Fingerprint string
	Size        string
	Description string
}

// UserConfig holds user configuration
type UserConfig struct {
	Name     string
	Password string
}
