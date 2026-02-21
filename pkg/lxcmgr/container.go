package lxcmgr

import (
	"errors"
	"time"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
	"lxc-dev-manager/internal/operations"
)

// CreateContainer creates a new container in the project
func (c *Client) CreateContainer(name, image string, opts ...CreateOption) error {
	o := &createOpts{}
	for _, opt := range opts {
		opt(o)
	}

	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		if errors.Is(err, config.ErrNoProject) {
			return ErrProjectNotFound
		}
		return wrapContainerErr("create", name, err)
	}
	defer lock.Release()

	if err := operations.CreateContainer(cfg, name, image, operations.CreateContainerOpts{
		Ports:    o.ports,
		User:     o.user,
		Password: o.password,
	}); err != nil {
		return wrapContainerErr("create", name, err)
	}

	c.cfg = cfg
	return nil
}

// Start starts a stopped container
func (c *Client) Start(name string) error {
	return wrapContainerErr("start", name, operations.Start(c.cfg, name))
}

// Stop stops a running container
func (c *Client) Stop(name string) error {
	return wrapContainerErr("stop", name, operations.Stop(c.cfg, name))
}

// Remove removes a container from the project
func (c *Client) Remove(name string, force bool) error {
	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		if errors.Is(err, config.ErrNoProject) {
			return ErrProjectNotFound
		}
		return wrapContainerErr("remove", name, err)
	}
	defer lock.Release()

	if err := operations.Remove(cfg, name, force); err != nil {
		return wrapContainerErr("remove", name, err)
	}

	c.cfg = cfg
	return nil
}

// Destroy deletes the LXC container but keeps its entry in containers.yaml.
// Snapshot entries are cleared since they no longer exist.
// This is useful when you want to recreate a container with the same config.
func (c *Client) Destroy(name string) error {
	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		if errors.Is(err, config.ErrNoProject) {
			return ErrProjectNotFound
		}
		return wrapContainerErr("destroy", name, err)
	}
	defer func() { _ = lock.Release() }()

	if !cfg.HasContainer(name) {
		return wrapContainerErr("destroy", name, errors.New("not found in config"))
	}

	lxcName := cfg.GetLXCName(name)

	// Delete the LXC container (--force stops it first if running)
	if lxc.Exists(lxcName) {
		if err := lxc.Delete(lxcName); err != nil {
			return wrapContainerErr("destroy", name, err)
		}
	}

	// Clear snapshots from config (they no longer exist)
	container := cfg.Containers[name]
	container.Snapshots = nil
	cfg.Containers[name] = container

	if err := cfg.Save(); err != nil {
		return wrapContainerErr("destroy", name, err)
	}

	c.cfg = cfg
	return nil
}

// Reset resets a container to a snapshot state
func (c *Client) Reset(name, snapshot string) error {
	return wrapContainerErr("reset", name, operations.Reset(c.cfg, name, snapshot))
}

// Clone clones a container to create a new one
func (c *Client) Clone(source, dest string, opts ...CloneOption) error {
	o := &cloneOpts{}
	for _, opt := range opts {
		opt(o)
	}

	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		if errors.Is(err, config.ErrNoProject) {
			return ErrProjectNotFound
		}
		return wrapContainerErr("clone", source, err)
	}
	defer lock.Release()

	if err := operations.Clone(cfg, source, dest, operations.CloneOpts{
		FromSnapshot: o.fromSnapshot,
	}); err != nil {
		return wrapContainerErr("clone", source, err)
	}

	c.cfg = cfg
	return nil
}

// List returns all containers in the project
func (c *Client) List() ([]ContainerInfo, error) {
	containers, err := operations.List(c.cfg)
	if err != nil {
		return nil, err
	}

	var result []ContainerInfo
	for _, info := range containers {
		result = append(result, ContainerInfo{
			Name:   info.Name,
			Image:  info.Image,
			Status: ContainerStatus(info.Status),
			IP:     info.IP,
			Ports:  info.Ports,
		})
	}
	return result, nil
}

// Status returns the status of a container
func (c *Client) Status(name string) (ContainerStatus, error) {
	status, err := operations.Status(c.cfg, name)
	return ContainerStatus(status), wrapContainerErr("status", name, err)
}

// IP returns the IP address of a container
func (c *Client) IP(name string) (string, error) {
	ip, err := operations.IP(c.cfg, name)
	return ip, wrapContainerErr("ip", name, err)
}

// Exists checks if a container exists in the project (both config and LXC)
func (c *Client) Exists(name string) bool {
	return operations.Exists(c.cfg, name)
}

// HasContainer checks if a container exists in the project config (regardless of LXC state)
func (c *Client) HasContainer(name string) bool {
	return c.cfg.HasContainer(name)
}

// SetContainerImage updates the image for a container in the config
func (c *Client) SetContainerImage(name, image string) error {
	cfg, lock, err := config.LoadWithLock(c.dir)
	if err != nil {
		return wrapContainerErr("set-image", name, err)
	}
	defer func() { _ = lock.Release() }()

	if !cfg.SetContainerImage(name, image) {
		return wrapContainerErr("set-image", name, errors.New("not found in config"))
	}

	if err := cfg.Save(); err != nil {
		return wrapContainerErr("set-image", name, err)
	}

	c.cfg = cfg
	return nil
}

// ListContainerNames returns the names of all containers in the config
func (c *Client) ListContainerNames() []string {
	names := make([]string, 0, len(c.cfg.Containers))
	for name := range c.cfg.Containers {
		names = append(names, name)
	}
	return names
}

// GetContainerImage returns the image for a container from the config
func (c *Client) GetContainerImage(name string) (string, bool) {
	container, ok := c.cfg.Containers[name]
	if !ok {
		return "", false
	}
	return container.Image, true
}

// WaitForReady waits for a container to be ready
func (c *Client) WaitForReady(name string, timeout time.Duration) error {
	return wrapContainerErr("wait", name, operations.WaitForReady(c.cfg, name, timeout))
}
