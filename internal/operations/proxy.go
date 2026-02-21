package operations

import (
	"fmt"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
	"lxc-dev-manager/internal/proxy"
)

// StartProxy starts proxying ports for a container
func StartProxy(cfg *config.Config, name string) (*proxy.Manager, string, []int, error) {
	if !cfg.HasContainer(name) {
		return nil, "", nil, fmt.Errorf("container '%s' not found in config", name)
	}

	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return nil, "", nil, fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	// Check if running
	status, err := lxc.GetStatus(lxcName)
	if err != nil {
		return nil, "", nil, err
	}
	if status != "RUNNING" {
		return nil, "", nil, fmt.Errorf("container '%s' is not running", name)
	}

	// Get container IP
	ip, err := lxc.GetIP(lxcName)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to get container IP: %w", err)
	}

	// Get ports from config
	ports := cfg.GetPorts(name)
	if len(ports) == 0 {
		return nil, "", nil, fmt.Errorf("no ports configured for container '%s'", name)
	}

	// Start proxies
	manager := proxy.NewManager()

	for _, port := range ports {
		if err := manager.Add(port, ip, port); err != nil {
			manager.StopAll()
			return nil, "", nil, fmt.Errorf("failed to start proxy for port %d: %w", port, err)
		}
	}

	return manager, ip, ports, nil
}
