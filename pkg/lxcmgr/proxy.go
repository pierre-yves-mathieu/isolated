package lxcmgr

import (
	"lxc-dev-manager/internal/operations"
	"lxc-dev-manager/internal/proxy"
)

// ProxyManager manages TCP proxies for port forwarding
type ProxyManager struct {
	manager *proxy.Manager
	IP      string
	Ports   []int
}

// StartProxy starts proxying ports for a container
func (c *Client) StartProxy(name string) (*ProxyManager, error) {
	manager, ip, ports, err := operations.StartProxy(c.cfg, name)
	if err != nil {
		return nil, wrapContainerErr("proxy", name, err)
	}
	return &ProxyManager{
		manager: manager,
		IP:      ip,
		Ports:   ports,
	}, nil
}

// Stop stops all proxies
func (pm *ProxyManager) Stop() {
	if pm.manager != nil {
		pm.manager.StopAll()
	}
}
