package lxcmgr

import (
	"errors"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/operations"
)

// CreateSnapshot creates a snapshot of a container
func (c *Client) CreateSnapshot(container, name, description string) error {
	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		if errors.Is(err, config.ErrNoProject) {
			return ErrProjectNotFound
		}
		return wrapSnapshotErr("create", container, name, err)
	}
	defer lock.Release()

	if err := operations.CreateSnapshot(cfg, container, name, description); err != nil {
		return wrapSnapshotErr("create", container, name, err)
	}

	c.cfg = cfg
	return nil
}

// ListSnapshots returns all snapshots for a container
func (c *Client) ListSnapshots(container string) ([]SnapshotInfo, error) {
	snapshots, err := operations.ListSnapshots(c.cfg, container)
	if err != nil {
		return nil, wrapSnapshotErr("list", container, "", err)
	}

	var result []SnapshotInfo
	for _, s := range snapshots {
		result = append(result, SnapshotInfo{
			Name:        s.Name,
			Description: s.Description,
			CreatedAt:   s.CreatedAt,
		})
	}
	return result, nil
}

// DeleteSnapshot deletes a snapshot from a container
func (c *Client) DeleteSnapshot(container, name string) error {
	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		if errors.Is(err, config.ErrNoProject) {
			return ErrProjectNotFound
		}
		return wrapSnapshotErr("delete", container, name, err)
	}
	defer lock.Release()

	if err := operations.DeleteSnapshot(cfg, container, name); err != nil {
		return wrapSnapshotErr("delete", container, name, err)
	}

	c.cfg = cfg
	return nil
}
