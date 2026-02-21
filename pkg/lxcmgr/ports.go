package lxcmgr

import "lxc-dev-manager/internal/config"

// GetDefaultPorts returns the default ports from containers.yaml.
func (c *Client) GetDefaultPorts() []int {
	if c.cfg == nil {
		return nil
	}
	return c.cfg.Defaults.Ports
}

// SetDefaultPorts updates the default ports in containers.yaml.
func (c *Client) SetDefaultPorts(ports []int) error {
	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	cfg.Defaults.Ports = ports
	if err := cfg.Save(); err != nil {
		return err
	}
	c.cfg = cfg
	return nil
}
