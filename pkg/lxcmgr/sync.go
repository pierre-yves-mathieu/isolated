package lxcmgr

import (
	"fmt"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/operations"
)

// SyncFiles copies all configured sync entries from host to container.
func (c *Client) SyncFiles(container string) error {
	return operations.SyncFiles(c.cfg, container, c.dir)
}

// AddSyncEntry adds a file sync entry to a container's configuration.
// If an entry with the same source already exists, it is overwritten.
func (c *Client) AddSyncEntry(container, source, dest string) error {
	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	cfg.AddSyncEntry(container, config.SyncEntry{
		Source: source,
		Dest:   dest,
	})
	if err := cfg.Save(); err != nil {
		return err
	}
	c.cfg = cfg
	return nil
}

// RemoveSyncEntry removes a sync entry by source path.
func (c *Client) RemoveSyncEntry(container, source string) error {
	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	cfg.RemoveSyncEntry(container, source)
	if err := cfg.Save(); err != nil {
		return err
	}
	c.cfg = cfg
	return nil
}

// ListSyncEntries returns all sync entries for a container.
func (c *Client) ListSyncEntries(container string) ([]config.SyncEntry, error) {
	if c.cfg == nil {
		return nil, ErrProjectNotFound
	}
	if !c.cfg.HasContainer(container) {
		return nil, fmt.Errorf("container '%s' not found in config", container)
	}
	return c.cfg.GetSyncEntries(container), nil
}
