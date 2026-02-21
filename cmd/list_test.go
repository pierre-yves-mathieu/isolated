package cmd

import (
	"testing"

	"lxc-dev-manager/internal/config"
)

func TestList_Empty(t *testing.T) {
	env := setupTestEnv(t)
	env.writeMinimalConfig()
	env.setListAllContainers("")

	// Should not error with empty config
	err := runList(nil, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestList_WithContainers(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`defaults:
  ports: [5173, 8000]
containers:
  dev1:
    image: ubuntu:24.04
  dev2:
    image: my-image
`)
	env.setListAllContainers(`dev1,RUNNING,10.10.10.45 (eth0)
dev2,STOPPED,`)

	err := runList(nil, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestList_MixedStatus(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`containers:
  dev1:
    image: ubuntu
  dev2:
    image: ubuntu
  dev3:
    image: ubuntu
`)
	env.setListAllContainers(`dev1,RUNNING,10.10.10.1 (eth0)
dev2,STOPPED,
dev3,RUNNING,10.10.10.3 (eth0)`)

	err := runList(nil, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestList_ContainerNotInLXC(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`containers:
  dev1:
    image: ubuntu:24.04
  dev2:
    image: my-image
`)
	// Only dev1 exists in LXC
	env.setListAllContainers(`dev1,RUNNING,10.10.10.45 (eth0)`)

	err := runList(nil, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// dev2 should show as NOT FOUND (verified by visual inspection)
}

func TestList_CustomPorts(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`defaults:
  ports: [3000]
containers:
  dev1:
    image: ubuntu
  dev2:
    image: ubuntu
    ports: [5000, 6000, 7000]
`)
	env.setListAllContainers(`dev1,RUNNING,10.10.10.1 (eth0)
dev2,RUNNING,10.10.10.2 (eth0)`)

	err := runList(nil, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify ports are correctly loaded
	cfg, _ := config.Load("")
	dev1Ports := cfg.GetPorts("dev1")
	dev2Ports := cfg.GetPorts("dev2")

	if len(dev1Ports) != 1 || dev1Ports[0] != 3000 {
		t.Errorf("dev1 should have default ports, got %v", dev1Ports)
	}
	if len(dev2Ports) != 3 || dev2Ports[0] != 5000 {
		t.Errorf("dev2 should have custom ports, got %v", dev2Ports)
	}
}

func TestList_LXCError(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`containers:
  dev1:
    image: ubuntu
`)
	env.mock.SetError("list -c ns4 -f csv", "permission denied")

	err := runList(nil, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
}
