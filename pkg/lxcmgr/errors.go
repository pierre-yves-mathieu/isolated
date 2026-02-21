// Package lxcmgr provides a Go SDK for managing LXC containers using lxc-dev-manager.
package lxcmgr

import (
	"errors"
	"fmt"
)

// Sentinel errors for programmatic handling
var (
	// Project errors
	ErrProjectNotFound    = errors.New("project not found")
	ErrProjectExists      = errors.New("project already exists")
	ErrInvalidProjectName = errors.New("invalid project name")

	// Container errors
	ErrContainerNotFound    = errors.New("container not found")
	ErrContainerExists      = errors.New("container already exists")
	ErrContainerRunning     = errors.New("container is running")
	ErrContainerStopped     = errors.New("container is stopped")
	ErrInvalidContainerName = errors.New("invalid container name")

	// Snapshot errors
	ErrSnapshotNotFound   = errors.New("snapshot not found")
	ErrSnapshotExists     = errors.New("snapshot already exists")
	ErrSnapshotProtected  = errors.New("snapshot is protected") // initial-state

	// Mount errors
	ErrMountNotFound        = errors.New("mount not found")
	ErrMountExists          = errors.New("mount already exists")
	ErrMountPathConflict    = errors.New("mount path already in use")
	ErrInvalidSourcePath    = errors.New("invalid source path")
	ErrInvalidContainerPath = errors.New("invalid container path")
	ErrBlockedPath          = errors.New("path is blocked for security")
	ErrPrivilegedMount      = errors.New("operation not allowed on privileged container")
	ErrRiskyPath            = errors.New("path is risky and requires explicit permission")

	// Image errors
	ErrImageNotFound = errors.New("image not found")
	ErrImageExists   = errors.New("image already exists")

	// Validation errors
	ErrValidation = errors.New("validation failed")
)

// ContainerError wraps errors with container context
type ContainerError struct {
	Container string
	Op        string // "create", "start", "stop", etc.
	Err       error
}

func (e *ContainerError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.Container, e.Err)
}

func (e *ContainerError) Unwrap() error {
	return e.Err
}

// ProjectError wraps errors with project context
type ProjectError struct {
	Project string
	Op      string
	Err     error
}

func (e *ProjectError) Error() string {
	return fmt.Sprintf("%s project %s: %v", e.Op, e.Project, e.Err)
}

func (e *ProjectError) Unwrap() error {
	return e.Err
}

// MountError wraps errors with mount context
type MountError struct {
	Container string
	Mount     string
	Op        string
	Err       error
}

func (e *MountError) Error() string {
	if e.Mount != "" {
		return fmt.Sprintf("%s mount %s on %s: %v", e.Op, e.Mount, e.Container, e.Err)
	}
	return fmt.Sprintf("%s mount on %s: %v", e.Op, e.Container, e.Err)
}

func (e *MountError) Unwrap() error {
	return e.Err
}

// SnapshotError wraps errors with snapshot context
type SnapshotError struct {
	Container string
	Snapshot  string
	Op        string
	Err       error
}

func (e *SnapshotError) Error() string {
	return fmt.Sprintf("%s snapshot %s on %s: %v", e.Op, e.Snapshot, e.Container, e.Err)
}

func (e *SnapshotError) Unwrap() error {
	return e.Err
}

// wrapContainerErr wraps an error with container context
func wrapContainerErr(op, container string, err error) error {
	if err == nil {
		return nil
	}
	return &ContainerError{
		Container: container,
		Op:        op,
		Err:       err,
	}
}

// wrapMountErr wraps an error with mount context
func wrapMountErr(op, container, mount string, err error) error {
	if err == nil {
		return nil
	}
	return &MountError{
		Container: container,
		Mount:     mount,
		Op:        op,
		Err:       err,
	}
}

// wrapSnapshotErr wraps an error with snapshot context
func wrapSnapshotErr(op, container, snapshot string, err error) error {
	if err == nil {
		return nil
	}
	return &SnapshotError{
		Container: container,
		Snapshot:  snapshot,
		Op:        op,
		Err:       err,
	}
}
