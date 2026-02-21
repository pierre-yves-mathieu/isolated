package lxcmgr

import (
	"lxc-dev-manager/internal/operations"
)

// CopyToContainer copies a file or directory from host to container
func (c *Client) CopyToContainer(container, localPath, remotePath string, opts ...CopyOption) error {
	o := &copyOpts{}
	for _, opt := range opts {
		opt(o)
	}

	return operations.CopyToContainer(c.cfg, container, localPath, remotePath, operations.CopyOpts{
		AutoCreateDir: o.autoCreateDir,
	})
}

// CopyFromContainer copies a file or directory from container to host
func (c *Client) CopyFromContainer(container, remotePath, localPath string, opts ...CopyOption) error {
	return operations.CopyFromContainer(c.cfg, container, remotePath, localPath)
}

// CopyBetweenContainers copies a file or directory from one container to another
func (c *Client) CopyBetweenContainers(srcContainer, srcPath, destContainer, destPath string, opts ...CopyOption) error {
	o := &copyOpts{}
	for _, opt := range opts {
		opt(o)
	}

	return operations.CopyBetweenContainers(c.cfg, srcContainer, srcPath, destContainer, destPath, operations.CopyOpts{
		AutoCreateDir: o.autoCreateDir,
	})
}
