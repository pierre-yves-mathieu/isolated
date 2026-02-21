package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestMount_Success(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetOutput("config get test-dev1 security.privileged", "")
	env.mock.SetOutput("config device add test-dev1 myrepo disk", "")

	// Create a real temp directory for source path validation
	sourceDir := t.TempDir()

	// Reset flags
	mountName = "myrepo"
	mountReadWrite = false
	mountShift = false
	mountAllowRisky = false
	mountYes = false
	defer func() {
		mountName = ""
		mountReadWrite = false
		mountShift = false
		mountAllowRisky = false
		mountYes = false
	}()

	err := runMount(nil, []string{"dev1", sourceDir, "/workspace"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify device add command was called
	if !env.mock.HasCallPrefix("config", "device", "add", "test-dev1", "myrepo", "disk") {
		t.Error("expected device add command")
	}

	// Verify config was updated
	cfg := env.readConfig()
	if !strings.Contains(cfg, "myrepo") {
		t.Error("expected device to be added to config")
	}
	if !strings.Contains(cfg, "type: disk") {
		t.Error("expected device type disk in config")
	}
}

func TestMount_ReadOnlyDefault(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetOutput("config get test-dev1 security.privileged", "")
	env.mock.SetOutput("config device add test-dev1", "")

	sourceDir := t.TempDir()

	// Reset flags - explicitly not setting mountReadWrite (default false = readonly)
	mountName = "myrepo"
	mountReadWrite = false
	mountShift = false
	mountAllowRisky = false
	mountYes = false
	defer func() {
		mountName = ""
		mountReadWrite = false
		mountShift = false
		mountAllowRisky = false
		mountYes = false
	}()

	err := runMount(nil, []string{"dev1", sourceDir, "/workspace"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify config has readonly: "true"
	cfg := env.readConfig()
	if !strings.Contains(cfg, "readonly:") {
		t.Error("expected readonly setting in config")
	}
}

func TestMount_ReadWrite(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetOutput("config get test-dev1 security.privileged", "")
	env.mock.SetOutput("config device add test-dev1", "")

	sourceDir := t.TempDir()

	// Set --rw flag
	mountName = "myrepo"
	mountReadWrite = true
	mountShift = false
	mountAllowRisky = false
	mountYes = false
	defer func() {
		mountName = ""
		mountReadWrite = false
		mountShift = false
		mountAllowRisky = false
		mountYes = false
	}()

	err := runMount(nil, []string{"dev1", sourceDir, "/workspace"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify config does NOT have readonly: "true" (rw mode = no readonly key)
	cfg := env.readConfig()
	if strings.Contains(cfg, "readonly:") {
		t.Error("expected no readonly setting in config for rw mount")
	}
}

func TestMount_ContainerNotFound(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers: {}
`)

	sourceDir := t.TempDir()

	err := runMount(nil, []string{"dev1", sourceDir, "/workspace"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMount_NoProject(t *testing.T) {
	_ = setupTestEnv(t)
	// No config file

	err := runMount(nil, []string{"dev1", "/tmp", "/workspace"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no project") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMount_NameConflict(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
    devices:
      myrepo:
        type: disk
        config:
          source: /existing/path
          path: /existing
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetOutput("config get test-dev1 security.privileged", "")

	sourceDir := t.TempDir()

	// Try to add device with existing name
	mountName = "myrepo"
	mountReadWrite = false
	mountShift = false
	mountAllowRisky = false
	mountYes = false
	defer func() {
		mountName = ""
		mountReadWrite = false
		mountShift = false
		mountAllowRisky = false
		mountYes = false
	}()

	err := runMount(nil, []string{"dev1", sourceDir, "/workspace"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMount_PathConflict(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
    devices:
      existing:
        type: disk
        config:
          source: /existing/path
          path: /workspace
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetOutput("config get test-dev1 security.privileged", "")

	sourceDir := t.TempDir()

	// Try to mount to same container path
	mountName = "newmount"
	mountReadWrite = false
	mountShift = false
	mountAllowRisky = false
	mountYes = false
	defer func() {
		mountName = ""
		mountReadWrite = false
		mountShift = false
		mountAllowRisky = false
		mountYes = false
	}()

	err := runMount(nil, []string{"dev1", sourceDir, "/workspace"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "already mounted") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMount_AutoGeneratesName(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetOutput("config get test-dev1 security.privileged", "")
	env.mock.SetOutput("config device add test-dev1", "")

	// Create a temp directory with a specific name
	sourceDir := t.TempDir() + "/myproject"
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	// Don't set mountName - should auto-generate from path
	mountName = ""
	mountReadWrite = false
	mountShift = false
	mountAllowRisky = false
	mountYes = false
	defer func() {
		mountName = ""
		mountReadWrite = false
		mountShift = false
		mountAllowRisky = false
		mountYes = false
	}()

	err := runMount(nil, []string{"dev1", sourceDir, "/workspace"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify config was updated with auto-generated name
	cfg := env.readConfig()
	if !strings.Contains(cfg, "myproject:") {
		t.Error("expected auto-generated device name 'myproject' in config")
	}
}

// TestUnmount tests

func TestUnmount_ByName(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
    devices:
      myrepo:
        type: disk
        config:
          source: /host/path
          path: /container/path
          readonly: "true"
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetOutput("config device remove test-dev1 myrepo", "")

	err := runUnmount(nil, []string{"dev1", "myrepo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify device remove command was called
	if !env.mock.HasCall("config", "device", "remove", "test-dev1", "myrepo") {
		t.Error("expected device remove command")
	}

	// Verify config was updated
	cfg := env.readConfig()
	if strings.Contains(cfg, "myrepo:") {
		t.Error("expected device to be removed from config")
	}
}

func TestUnmount_ByPath(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
    devices:
      myrepo:
        type: disk
        config:
          source: /host/path
          path: /container/path
          readonly: "true"
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetOutput("config device remove test-dev1 myrepo", "")

	// Unmount by path (starts with /)
	err := runUnmount(nil, []string{"dev1", "/container/path"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify device remove command was called with the device name
	if !env.mock.HasCall("config", "device", "remove", "test-dev1", "myrepo") {
		t.Error("expected device remove command")
	}
}

func TestUnmount_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
    devices: {}
`)
	env.setContainerExists("test-dev1", true)

	err := runUnmount(nil, []string{"dev1", "nonexistent"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUnmount_ContainerNotFound(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers: {}
`)

	err := runUnmount(nil, []string{"dev1", "myrepo"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestMounts tests

func TestMounts_List(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
    devices:
      repo:
        type: disk
        config:
          source: /host/path
          path: /container/path
          readonly: "true"
`)
	env.setContainerExists("test-dev1", true)

	// Mock device list response (YAML format)
	env.mock.SetOutput("config device show test-dev1", `repo:
  type: disk
  source: /host/path
  path: /container/path
  readonly: "true"
`)

	mountsSync = false
	defer func() { mountsSync = false }()

	err := runMounts(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMounts_Empty(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)

	// Mock empty device list
	env.mock.SetOutput("config device show test-dev1", "")

	mountsSync = false
	defer func() { mountsSync = false }()

	err := runMounts(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMounts_ContainerNotFound(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers: {}
`)

	mountsSync = false
	defer func() { mountsSync = false }()

	err := runMounts(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}
