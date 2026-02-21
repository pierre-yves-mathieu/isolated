package lxcmgr

import (
	"errors"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/operations"
)

// Mount mounts a host directory into a container
func (c *Client) Mount(container, source, path string, opts ...MountOption) error {
	o := &mountOpts{}
	for _, opt := range opts {
		opt(o)
	}

	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		if errors.Is(err, config.ErrNoProject) {
			return ErrProjectNotFound
		}
		return wrapMountErr("mount", container, o.name, err)
	}
	defer lock.Release()

	if _, err := operations.Mount(cfg, container, source, path, operations.MountOpts{
		Name:           o.name,
		ReadWrite:      o.readWrite,
		Shift:          o.shift,
		AllowRiskyPath: o.allowRiskyPath,
	}); err != nil {
		return wrapMountErr("mount", container, o.name, err)
	}

	c.cfg = cfg
	return nil
}

// Unmount removes a mount from a container
func (c *Client) Unmount(container, nameOrPath string) error {
	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		if errors.Is(err, config.ErrNoProject) {
			return ErrProjectNotFound
		}
		return wrapMountErr("unmount", container, nameOrPath, err)
	}
	defer lock.Release()

	if err := operations.Unmount(cfg, container, nameOrPath); err != nil {
		return wrapMountErr("unmount", container, nameOrPath, err)
	}

	c.cfg = cfg
	return nil
}

// ListMounts returns all mounts for a container
func (c *Client) ListMounts(container string) ([]MountInfo, error) {
	mounts, err := operations.ListMounts(c.cfg, container)
	if err != nil {
		return nil, wrapMountErr("list", container, "", err)
	}

	var result []MountInfo
	for _, m := range mounts {
		result = append(result, MountInfo{
			Name:     m.Name,
			Source:   m.Source,
			Path:     m.Path,
			ReadOnly: m.Mode == "ro",
			Status:   MountStatus(m.Status),
		})
	}
	return result, nil
}

// SyncMounts synchronizes mounts between config and LXC
func (c *Client) SyncMounts(container string) error {
	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		if errors.Is(err, config.ErrNoProject) {
			return ErrProjectNotFound
		}
		return wrapMountErr("sync", container, "", err)
	}
	defer lock.Release()

	if err := operations.SyncMounts(cfg, container); err != nil {
		return wrapMountErr("sync", container, "", err)
	}

	c.cfg = cfg
	return nil
}
