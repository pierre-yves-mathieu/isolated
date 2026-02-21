package cmd

import (
	"strings"
	"testing"

	"lxc-dev-manager/internal/config"
)

// Helper to set force flag for tests
func withForceFlag(t *testing.T) {
	t.Helper()
	removeForce = true
	t.Cleanup(func() { removeForce = false })
}

func TestRemove_Success(t *testing.T) {
	env := setupTestEnv(t)
	withForceFlag(t)

	env.writeConfig(`containers:
  dev1:
    image: ubuntu:24.04
  dev2:
    image: other
`)
	env.setContainerExists("dev1", true)
	env.mock.SetOutput("delete dev1 --force", "")

	err := runRemove(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.mock.HasCall("delete", "dev1", "--force") {
		t.Error("expected delete command")
	}

	// Verify removed from config
	cfg, _ := config.Load("")
	if cfg.HasContainer("dev1") {
		t.Error("dev1 should be removed from config")
	}
	if !cfg.HasContainer("dev2") {
		t.Error("dev2 should still exist")
	}
}

func TestRemove_OnlyInConfig(t *testing.T) {
	env := setupTestEnv(t)
	withForceFlag(t)

	env.writeConfig(`containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerNotExists("dev1")

	err := runRemove(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should remove from config even if not in LXC
	cfg, _ := config.Load("")
	if cfg.HasContainer("dev1") {
		t.Error("dev1 should be removed from config")
	}

	// Should not call delete since container doesn't exist in LXC
	if env.mock.HasCall("delete", "dev1", "--force") {
		t.Error("should not delete non-existent container")
	}
}

func TestRemove_OnlyInLXC(t *testing.T) {
	env := setupTestEnv(t)
	withForceFlag(t)
	env.writeMinimalConfig()

	// Container exists in LXC but not in config
	env.setContainerExists("dev1", false)
	env.mock.SetOutput("delete dev1 --force", "")

	err := runRemove(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.mock.HasCall("delete", "dev1", "--force") {
		t.Error("expected delete command")
	}
}

func TestRemove_NotExists(t *testing.T) {
	env := setupTestEnv(t)
	withForceFlag(t)
	env.writeMinimalConfig()

	env.setContainerNotExists("dev1")

	err := runRemove(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error for non-existent container")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRemove_DeleteFails(t *testing.T) {
	env := setupTestEnv(t)
	withForceFlag(t)
	env.writeMinimalConfig()

	env.setContainerExists("dev1", true)
	env.mock.SetError("delete dev1 --force", "failed to delete")

	err := runRemove(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to delete") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRemove_BothConfigAndLXC(t *testing.T) {
	env := setupTestEnv(t)
	withForceFlag(t)

	env.writeConfig(`containers:
  dev1:
    image: ubuntu
`)
	env.setContainerExists("dev1", true)
	env.mock.SetOutput("delete dev1 --force", "")

	err := runRemove(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should delete from both LXC and config
	if !env.mock.HasCall("delete", "dev1", "--force") {
		t.Error("expected delete command")
	}

	cfg, _ := config.Load("")
	if cfg.HasContainer("dev1") {
		t.Error("dev1 should be removed from config")
	}
}
