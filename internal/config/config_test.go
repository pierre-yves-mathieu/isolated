package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper to run tests in a temp directory
func withTempDir(t *testing.T, fn func(dir string)) {
	t.Helper()
	dir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)
	fn(dir)
}

func TestLoad_FileNotExists(t *testing.T) {
	withTempDir(t, func(dir string) {
		cfg, err := Load("")
		if err != ErrNoProject {
			t.Fatalf("expected ErrNoProject, got %v", err)
		}
		if cfg != nil {
			t.Fatal("expected nil config when file doesn't exist")
		}
	})
}

func TestLoad_ValidYAML(t *testing.T) {
	withTempDir(t, func(dir string) {
		yaml := `defaults:
  ports:
    - 3000
    - 8080
containers:
  dev1:
    image: ubuntu:24.04
  dev2:
    image: my-image
    ports:
      - 5000
`
		if err := os.WriteFile(ConfigFile, []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load("")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(cfg.Defaults.Ports) != 2 {
			t.Errorf("expected 2 default ports, got %d", len(cfg.Defaults.Ports))
		}
		if cfg.Defaults.Ports[0] != 3000 {
			t.Errorf("expected port 3000, got %d", cfg.Defaults.Ports[0])
		}

		if len(cfg.Containers) != 2 {
			t.Errorf("expected 2 containers, got %d", len(cfg.Containers))
		}
		if cfg.Containers["dev1"].Image != "ubuntu:24.04" {
			t.Errorf("expected ubuntu:24.04, got %s", cfg.Containers["dev1"].Image)
		}
		if len(cfg.Containers["dev2"].Ports) != 1 {
			t.Errorf("expected 1 port for dev2, got %d", len(cfg.Containers["dev2"].Ports))
		}
	})
}

func TestLoad_InvalidYAML(t *testing.T) {
	withTempDir(t, func(dir string) {
		invalidYAML := `defaults:
  ports: [not valid yaml
    - broken
`
		if err := os.WriteFile(ConfigFile, []byte(invalidYAML), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := Load("")
		if err == nil {
			t.Fatal("expected error for invalid YAML, got nil")
		}
	})
}

func TestLoad_EmptyFile(t *testing.T) {
	withTempDir(t, func(dir string) {
		if err := os.WriteFile(ConfigFile, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load("")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Empty file should result in empty config with initialized map
		if cfg.Containers == nil {
			t.Error("expected Containers map to be initialized")
		}
	})
}

func TestSave_CreatesFile(t *testing.T) {
	withTempDir(t, func(dir string) {
		cfg := &Config{
			Defaults: Defaults{
				Ports: []int{5173, 8000},
			},
			Containers: map[string]Container{
				"test1": {Image: "ubuntu:24.04"},
			},
		}

		if err := cfg.Save(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(ConfigFile); os.IsNotExist(err) {
			t.Fatal("expected config file to be created")
		}

		// Verify content can be loaded back
		loaded, err := Load("")
		if err != nil {
			t.Fatalf("failed to load saved config: %v", err)
		}
		if loaded.Containers["test1"].Image != "ubuntu:24.04" {
			t.Errorf("expected ubuntu:24.04, got %s", loaded.Containers["test1"].Image)
		}
	})
}

func TestSave_OverwritesFile(t *testing.T) {
	withTempDir(t, func(dir string) {
		// Create initial config
		cfg1 := &Config{
			Defaults:   Defaults{Ports: []int{3000}},
			Containers: map[string]Container{"old": {Image: "old-image"}},
		}
		if err := cfg1.Save(); err != nil {
			t.Fatal(err)
		}

		// Overwrite with new config
		cfg2 := &Config{
			Defaults:   Defaults{Ports: []int{8000}},
			Containers: map[string]Container{"new": {Image: "new-image"}},
		}
		if err := cfg2.Save(); err != nil {
			t.Fatal(err)
		}

		// Verify new content
		loaded, err := Load("")
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := loaded.Containers["old"]; ok {
			t.Error("old container should not exist")
		}
		if loaded.Containers["new"].Image != "new-image" {
			t.Errorf("expected new-image, got %s", loaded.Containers["new"].Image)
		}
	})
}

func TestAddContainer(t *testing.T) {
	cfg := &Config{
		Containers: make(map[string]Container),
	}

	cfg.AddContainer("dev1", "ubuntu:24.04")

	if !cfg.HasContainer("dev1") {
		t.Error("expected dev1 to exist")
	}
	if cfg.Containers["dev1"].Image != "ubuntu:24.04" {
		t.Errorf("expected ubuntu:24.04, got %s", cfg.Containers["dev1"].Image)
	}
}

func TestAddContainer_Duplicate(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "old-image"},
		},
	}

	cfg.AddContainer("dev1", "new-image")

	if cfg.Containers["dev1"].Image != "new-image" {
		t.Errorf("expected new-image, got %s", cfg.Containers["dev1"].Image)
	}
}

func TestRemoveContainer(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"},
			"dev2": {Image: "debian"},
		},
	}

	cfg.RemoveContainer("dev1")

	if cfg.HasContainer("dev1") {
		t.Error("dev1 should be removed")
	}
	if !cfg.HasContainer("dev2") {
		t.Error("dev2 should still exist")
	}
}

func TestRemoveContainer_NotExists(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{},
	}

	// Should not panic
	cfg.RemoveContainer("nonexistent")

	if len(cfg.Containers) != 0 {
		t.Error("containers should still be empty")
	}
}

func TestGetPorts_ContainerSpecific(t *testing.T) {
	cfg := &Config{
		Defaults: Defaults{Ports: []int{3000, 8000}},
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu",
				Ports: []int{5000, 6000, 7000},
			},
		},
	}

	ports := cfg.GetPorts("dev1")

	if len(ports) != 3 {
		t.Errorf("expected 3 ports, got %d", len(ports))
	}
	if ports[0] != 5000 {
		t.Errorf("expected 5000, got %d", ports[0])
	}
}

func TestGetPorts_DefaultFallback(t *testing.T) {
	cfg := &Config{
		Defaults: Defaults{Ports: []int{3000, 8000}},
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu"}, // No ports specified
		},
	}

	ports := cfg.GetPorts("dev1")

	if len(ports) != 2 {
		t.Errorf("expected 2 default ports, got %d", len(ports))
	}
	if ports[0] != 3000 {
		t.Errorf("expected 3000, got %d", ports[0])
	}
}

func TestGetPorts_EmptyDefaults(t *testing.T) {
	cfg := &Config{
		Defaults: Defaults{Ports: []int{}},
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu"},
		},
	}

	ports := cfg.GetPorts("dev1")

	if len(ports) != 0 {
		t.Errorf("expected 0 ports, got %d", len(ports))
	}
}

func TestGetPorts_NonexistentContainer(t *testing.T) {
	cfg := &Config{
		Defaults:   Defaults{Ports: []int{3000}},
		Containers: map[string]Container{},
	}

	ports := cfg.GetPorts("nonexistent")

	// Should return defaults
	if len(ports) != 1 {
		t.Errorf("expected 1 port, got %d", len(ports))
	}
}

func TestHasContainer_Exists(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu"},
		},
	}

	if !cfg.HasContainer("dev1") {
		t.Error("expected HasContainer to return true")
	}
}

func TestHasContainer_NotExists(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{},
	}

	if cfg.HasContainer("dev1") {
		t.Error("expected HasContainer to return false")
	}
}

func TestLoad_PermissionError(t *testing.T) {
	withTempDir(t, func(dir string) {
		// Create unreadable file
		path := filepath.Join(dir, ConfigFile)
		if err := os.WriteFile(path, []byte("test"), 0000); err != nil {
			t.Skip("cannot create unreadable file")
		}
		defer os.Chmod(path, 0644) // Cleanup

		_, err := Load("")
		if err == nil {
			t.Error("expected permission error")
		}
	})
}

// Snapshot tests

func TestAddSnapshot(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"},
		},
	}

	cfg.AddSnapshot("dev1", "snap1", "Test snapshot")

	if !cfg.HasSnapshot("dev1", "snap1") {
		t.Error("expected snapshot to exist")
	}
	snap := cfg.Containers["dev1"].Snapshots["snap1"]
	if snap.Description != "Test snapshot" {
		t.Errorf("expected description 'Test snapshot', got '%s'", snap.Description)
	}
	if snap.CreatedAt == "" {
		t.Error("expected CreatedAt to be set")
	}
}

func TestAddSnapshot_InitializesMap(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"}, // No Snapshots map
		},
	}

	cfg.AddSnapshot("dev1", "snap1", "")

	if cfg.Containers["dev1"].Snapshots == nil {
		t.Error("expected Snapshots map to be initialized")
	}
}

func TestAddSnapshot_EmptyDescription(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"},
		},
	}

	cfg.AddSnapshot("dev1", "snap1", "")

	snap := cfg.Containers["dev1"].Snapshots["snap1"]
	if snap.Description != "" {
		t.Errorf("expected empty description, got '%s'", snap.Description)
	}
}

func TestRemoveSnapshot(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Snapshots: map[string]Snapshot{
					"snap1": {Description: "test"},
					"snap2": {Description: "test2"},
				},
			},
		},
	}

	cfg.RemoveSnapshot("dev1", "snap1")

	if cfg.HasSnapshot("dev1", "snap1") {
		t.Error("snap1 should be removed")
	}
	if !cfg.HasSnapshot("dev1", "snap2") {
		t.Error("snap2 should still exist")
	}
}

func TestRemoveSnapshot_NotExists(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"},
		},
	}

	// Should not panic
	cfg.RemoveSnapshot("dev1", "nonexistent")
}

func TestRemoveSnapshot_ContainerNotExists(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{},
	}

	// Should not panic
	cfg.RemoveSnapshot("nonexistent", "snap1")
}

func TestGetSnapshots(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Snapshots: map[string]Snapshot{
					"snap1": {Description: "test1"},
					"snap2": {Description: "test2"},
				},
			},
		},
	}

	snapshots := cfg.GetSnapshots("dev1")

	if len(snapshots) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(snapshots))
	}
}

func TestGetSnapshots_ContainerNotExists(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{},
	}

	snapshots := cfg.GetSnapshots("nonexistent")

	if snapshots != nil {
		t.Error("expected nil for nonexistent container")
	}
}

func TestGetSnapshots_NoSnapshots(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"},
		},
	}

	snapshots := cfg.GetSnapshots("dev1")

	if snapshots != nil {
		t.Error("expected nil when no snapshots")
	}
}

func TestHasSnapshot_True(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Snapshots: map[string]Snapshot{
					"snap1": {Description: "test"},
				},
			},
		},
	}

	if !cfg.HasSnapshot("dev1", "snap1") {
		t.Error("expected HasSnapshot to return true")
	}
}

func TestHasSnapshot_False(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"},
		},
	}

	if cfg.HasSnapshot("dev1", "snap1") {
		t.Error("expected HasSnapshot to return false")
	}
}

func TestHasSnapshot_ContainerNotExists(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{},
	}

	if cfg.HasSnapshot("nonexistent", "snap1") {
		t.Error("expected HasSnapshot to return false")
	}
}

// User config tests

func TestGetUser_ContainerSpecific(t *testing.T) {
	cfg := &Config{
		Defaults: Defaults{User: User{Name: "default", Password: "defaultpass"}},
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu", User: User{Name: "alice", Password: "alicepass"}},
		},
	}

	user := cfg.GetUser("dev1")

	if user.Name != "alice" {
		t.Errorf("expected alice, got %s", user.Name)
	}
	if user.Password != "alicepass" {
		t.Errorf("expected alicepass, got %s", user.Password)
	}
}

func TestGetUser_DefaultFallback(t *testing.T) {
	cfg := &Config{
		Defaults: Defaults{User: User{Name: "default", Password: "defaultpass"}},
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu"}, // No user specified
		},
	}

	user := cfg.GetUser("dev1")

	if user.Name != "default" {
		t.Errorf("expected default, got %s", user.Name)
	}
	if user.Password != "defaultpass" {
		t.Errorf("expected defaultpass, got %s", user.Password)
	}
}

func TestGetUser_HardcodedFallback(t *testing.T) {
	cfg := &Config{
		Defaults:   Defaults{},
		Containers: map[string]Container{"dev1": {Image: "ubuntu"}},
	}

	user := cfg.GetUser("dev1")

	if user.Name != "dev" {
		t.Errorf("expected dev, got %s", user.Name)
	}
	if user.Password != "dev" {
		t.Errorf("expected dev, got %s", user.Password)
	}
}

func TestGetUser_NonexistentContainer(t *testing.T) {
	cfg := &Config{
		Defaults:   Defaults{User: User{Name: "default", Password: "pass"}},
		Containers: map[string]Container{},
	}

	user := cfg.GetUser("nonexistent")

	if user.Name != "default" {
		t.Errorf("expected default, got %s", user.Name)
	}
	if user.Password != "pass" {
		t.Errorf("expected pass, got %s", user.Password)
	}
}

func TestGetUser_PartialContainerConfig_PasswordFromDefaults(t *testing.T) {
	cfg := &Config{
		Defaults: Defaults{User: User{Name: "ignored", Password: "defaultpass"}},
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu", User: User{Name: "alice"}}, // No password
		},
	}

	user := cfg.GetUser("dev1")

	if user.Name != "alice" {
		t.Errorf("expected alice, got %s", user.Name)
	}
	if user.Password != "defaultpass" {
		t.Errorf("expected defaultpass, got %s", user.Password)
	}
}

func TestGetUser_PartialContainerConfig_PasswordHardcoded(t *testing.T) {
	cfg := &Config{
		Defaults: Defaults{User: User{Name: "ignored"}}, // No password in defaults
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu", User: User{Name: "alice"}},
		},
	}

	user := cfg.GetUser("dev1")

	if user.Name != "alice" {
		t.Errorf("expected alice, got %s", user.Name)
	}
	if user.Password != "dev" {
		t.Errorf("expected dev (hardcoded), got %s", user.Password)
	}
}

func TestGetUser_DefaultsPartialConfig(t *testing.T) {
	cfg := &Config{
		Defaults:   Defaults{User: User{Name: "default"}}, // No password
		Containers: map[string]Container{"dev1": {Image: "ubuntu"}},
	}

	user := cfg.GetUser("dev1")

	if user.Name != "default" {
		t.Errorf("expected default, got %s", user.Name)
	}
	if user.Password != "dev" {
		t.Errorf("expected dev (hardcoded), got %s", user.Password)
	}
}

func TestLoad_WithUserConfig(t *testing.T) {
	withTempDir(t, func(dir string) {
		yaml := `project: test
defaults:
  user:
    name: devuser
    password: devpass
containers:
  dev1:
    image: ubuntu:24.04
  dev2:
    image: ubuntu:24.04
    user:
      name: customuser
      password: custompass
`
		if err := os.WriteFile(ConfigFile, []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check defaults
		if cfg.Defaults.User.Name != "devuser" {
			t.Errorf("expected default user devuser, got %s", cfg.Defaults.User.Name)
		}
		if cfg.Defaults.User.Password != "devpass" {
			t.Errorf("expected default password devpass, got %s", cfg.Defaults.User.Password)
		}

		// Check dev1 has no user config
		if cfg.Containers["dev1"].User.Name != "" {
			t.Errorf("expected dev1 to have no user name, got %s", cfg.Containers["dev1"].User.Name)
		}

		// Check dev2 has custom user
		if cfg.Containers["dev2"].User.Name != "customuser" {
			t.Errorf("expected customuser, got %s", cfg.Containers["dev2"].User.Name)
		}
		if cfg.Containers["dev2"].User.Password != "custompass" {
			t.Errorf("expected custompass, got %s", cfg.Containers["dev2"].User.Password)
		}
	})
}

func TestSave_WithUserConfig(t *testing.T) {
	withTempDir(t, func(dir string) {
		cfg := &Config{
			Project:  "test",
			Defaults: Defaults{User: User{Name: "default", Password: "pass"}},
			Containers: map[string]Container{
				"dev1": {Image: "ubuntu", User: User{Name: "alice", Password: "secret"}},
			},
		}

		if err := cfg.Save(); err != nil {
			t.Fatalf("failed to save: %v", err)
		}

		loaded, err := Load("")
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		// Verify defaults preserved
		if loaded.Defaults.User.Name != "default" {
			t.Errorf("expected default user name, got %s", loaded.Defaults.User.Name)
		}
		if loaded.Defaults.User.Password != "pass" {
			t.Errorf("expected default password, got %s", loaded.Defaults.User.Password)
		}

		// Verify container user preserved
		if loaded.Containers["dev1"].User.Name != "alice" {
			t.Errorf("expected alice, got %s", loaded.Containers["dev1"].User.Name)
		}
		if loaded.Containers["dev1"].User.Password != "secret" {
			t.Errorf("expected secret, got %s", loaded.Containers["dev1"].User.Password)
		}
	})
}

func TestLoad_WithSnapshots(t *testing.T) {
	withTempDir(t, func(dir string) {
		yaml := `project: test
containers:
  dev1:
    image: ubuntu:24.04
    snapshots:
      initial-state:
        description: Initial state after setup
        created_at: "2024-01-15T10:30:00Z"
      checkpoint:
        description: Before refactoring
        created_at: "2024-01-15T14:00:00Z"
`
		if err := os.WriteFile(ConfigFile, []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !cfg.HasSnapshot("dev1", "initial-state") {
			t.Error("expected initial-state snapshot")
		}
		if !cfg.HasSnapshot("dev1", "checkpoint") {
			t.Error("expected checkpoint snapshot")
		}
		if cfg.Containers["dev1"].Snapshots["checkpoint"].Description != "Before refactoring" {
			t.Error("unexpected description")
		}
	})
}

// Device tests

func TestAddDevice(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"},
		},
	}

	device := Device{
		Type: "disk",
		Config: map[string]string{
			"source": "/host/path",
			"path":   "/container/path",
		},
	}

	cfg.AddDevice("dev1", "mydevice", device)

	if !cfg.HasDevice("dev1", "mydevice") {
		t.Error("expected device to exist")
	}
	if cfg.Containers["dev1"].Devices["mydevice"].Type != "disk" {
		t.Errorf("expected type disk, got %s", cfg.Containers["dev1"].Devices["mydevice"].Type)
	}
	if cfg.Containers["dev1"].Devices["mydevice"].Config["source"] != "/host/path" {
		t.Errorf("expected source /host/path, got %s", cfg.Containers["dev1"].Devices["mydevice"].Config["source"])
	}
}

func TestAddDevice_ContainerNotExists(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{},
	}

	device := Device{
		Type: "disk",
		Config: map[string]string{
			"source": "/host/path",
			"path":   "/container/path",
		},
	}

	// Should not panic
	cfg.AddDevice("nonexistent", "mydevice", device)

	// Should still have no containers
	if len(cfg.Containers) != 0 {
		t.Error("expected no containers")
	}
}

func TestRemoveDevice(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Devices: map[string]Device{
					"device1": {Type: "disk", Config: map[string]string{"source": "/a", "path": "/b"}},
					"device2": {Type: "disk", Config: map[string]string{"source": "/c", "path": "/d"}},
				},
			},
		},
	}

	cfg.RemoveDevice("dev1", "device1")

	if cfg.HasDevice("dev1", "device1") {
		t.Error("device1 should be removed")
	}
	if !cfg.HasDevice("dev1", "device2") {
		t.Error("device2 should still exist")
	}
}

func TestRemoveDevice_NotExists(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"},
		},
	}

	// Should not panic
	cfg.RemoveDevice("dev1", "nonexistent")
}

func TestGetDevices(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Devices: map[string]Device{
					"device1": {Type: "disk", Config: map[string]string{"source": "/a", "path": "/b"}},
					"device2": {Type: "disk", Config: map[string]string{"source": "/c", "path": "/d"}},
				},
			},
		},
	}

	devices := cfg.GetDevices("dev1")

	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
	if devices["device1"].Type != "disk" {
		t.Errorf("expected disk type, got %s", devices["device1"].Type)
	}
}

func TestGetDevices_ContainerNotExists(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{},
	}

	devices := cfg.GetDevices("nonexistent")

	if devices != nil {
		t.Error("expected nil for nonexistent container")
	}
}

func TestHasDevice(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Devices: map[string]Device{
					"mydevice": {Type: "disk", Config: map[string]string{"source": "/a", "path": "/b"}},
				},
			},
		},
	}

	if !cfg.HasDevice("dev1", "mydevice") {
		t.Error("expected HasDevice to return true")
	}
}

func TestHasDevice_NotExists(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"},
		},
	}

	if cfg.HasDevice("dev1", "nonexistent") {
		t.Error("expected HasDevice to return false")
	}
}

func TestFindDeviceByPath(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Devices: map[string]Device{
					"mount1": {Type: "disk", Config: map[string]string{"source": "/host/a", "path": "/container/a"}},
					"mount2": {Type: "disk", Config: map[string]string{"source": "/host/b", "path": "/container/b"}},
				},
			},
		},
	}

	name, found := cfg.FindDeviceByPath("dev1", "/container/b")

	if !found {
		t.Error("expected to find device")
	}
	if name != "mount2" {
		t.Errorf("expected mount2, got %s", name)
	}
}

func TestFindDeviceByPath_NotFound(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Devices: map[string]Device{
					"mount1": {Type: "disk", Config: map[string]string{"source": "/host/a", "path": "/container/a"}},
				},
			},
		},
	}

	_, found := cfg.FindDeviceByPath("dev1", "/nonexistent/path")

	if found {
		t.Error("expected not to find device")
	}
}

func TestValidate_DeviceTypeEmpty(t *testing.T) {
	cfg := &Config{
		Project: "test",
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Devices: map[string]Device{
					"baddevice": {Type: ""},
				},
			},
		},
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("expected validation error for empty device type")
	}
	if !strings.Contains(err.Error(), "device type must not be empty") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_DiskDeviceMissingSource(t *testing.T) {
	cfg := &Config{
		Project: "test",
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Devices: map[string]Device{
					"baddevice": {
						Type:   "disk",
						Config: map[string]string{"path": "/container/path"},
					},
				},
			},
		},
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("expected validation error for missing source")
	}
	if !strings.Contains(err.Error(), "requires 'source' config key") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_DiskDeviceMissingPath(t *testing.T) {
	cfg := &Config{
		Project: "test",
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Devices: map[string]Device{
					"baddevice": {
						Type:   "disk",
						Config: map[string]string{"source": "/host/path"},
					},
				},
			},
		},
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("expected validation error for missing path")
	}
	if !strings.Contains(err.Error(), "requires 'path' config key") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_DevicePathControlChars(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		errMsg string
	}{
		{
			name: "control char in source",
			device: Device{
				Type:   "disk",
				Config: map[string]string{"source": "/host/path\x00bad", "path": "/container/path"},
			},
			errMsg: "source path contains control characters",
		},
		{
			name: "control char in path",
			device: Device{
				Type:   "disk",
				Config: map[string]string{"source": "/host/path", "path": "/container/path\t"},
			},
			errMsg: "path contains control characters",
		},
		{
			name: "newline in source",
			device: Device{
				Type:   "disk",
				Config: map[string]string{"source": "/host/path\n", "path": "/container/path"},
			},
			errMsg: "source path contains control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Project: "test",
				Containers: map[string]Container{
					"dev1": {
						Image: "ubuntu:24.04",
						Devices: map[string]Device{
							"baddevice": tt.device,
						},
					},
				},
			}

			err := cfg.Validate()

			if err == nil {
				t.Fatal("expected validation error for control characters")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_DeviceValid(t *testing.T) {
	cfg := &Config{
		Project: "test",
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Devices: map[string]Device{
					"validdevice": {
						Type: "disk",
						Config: map[string]string{
							"source": "/host/valid/path",
							"path":   "/container/valid/path",
						},
					},
				},
			},
		},
	}

	err := cfg.Validate()

	if err != nil {
		t.Errorf("expected no validation error, got %v", err)
	}
}

// --- Sync Entry Tests ---

func TestLoad_WithSyncEntries(t *testing.T) {
	withTempDir(t, func(dir string) {
		yaml := `project: test
containers:
  dev1:
    image: ubuntu:24.04
    sync:
      - source: .env
        dest: /home/dev/project/.env
      - source: config/secrets.json
        dest: /home/dev/project/config/secrets.json
`
		if err := os.WriteFile(ConfigFile, []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load("")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		entries := cfg.GetSyncEntries("dev1")
		if len(entries) != 2 {
			t.Fatalf("expected 2 sync entries, got %d", len(entries))
		}
		if entries[0].Source != ".env" {
			t.Errorf("expected source '.env', got %q", entries[0].Source)
		}
		if entries[0].Dest != "/home/dev/project/.env" {
			t.Errorf("expected dest '/home/dev/project/.env', got %q", entries[0].Dest)
		}
		if entries[1].Source != "config/secrets.json" {
			t.Errorf("expected source 'config/secrets.json', got %q", entries[1].Source)
		}
	})
}

func TestLoad_SyncEmptyList(t *testing.T) {
	withTempDir(t, func(dir string) {
		yaml := `project: test
containers:
  dev1:
    image: ubuntu:24.04
    sync: []
`
		if err := os.WriteFile(ConfigFile, []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load("")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		entries := cfg.GetSyncEntries("dev1")
		if len(entries) != 0 {
			t.Errorf("expected 0 sync entries, got %d", len(entries))
		}
	})
}

func TestAddSyncEntry(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"},
		},
	}

	cfg.AddSyncEntry("dev1", SyncEntry{Source: ".env", Dest: "/app/.env"})

	entries := cfg.GetSyncEntries("dev1")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Source != ".env" || entries[0].Dest != "/app/.env" {
		t.Errorf("unexpected entry: %+v", entries[0])
	}
}

func TestAddSyncEntry_DuplicateSource(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {Image: "ubuntu:24.04"},
		},
	}

	cfg.AddSyncEntry("dev1", SyncEntry{Source: ".env", Dest: "/old/.env"})
	cfg.AddSyncEntry("dev1", SyncEntry{Source: ".env", Dest: "/new/.env"})

	entries := cfg.GetSyncEntries("dev1")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (overwritten), got %d", len(entries))
	}
	if entries[0].Dest != "/new/.env" {
		t.Errorf("expected dest to be overwritten to '/new/.env', got %q", entries[0].Dest)
	}
}

func TestRemoveSyncEntry(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Sync: []SyncEntry{
					{Source: ".env", Dest: "/app/.env"},
					{Source: "secrets.json", Dest: "/app/secrets.json"},
				},
			},
		},
	}

	cfg.RemoveSyncEntry("dev1", ".env")

	entries := cfg.GetSyncEntries("dev1")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after removal, got %d", len(entries))
	}
	if entries[0].Source != "secrets.json" {
		t.Errorf("expected remaining entry to be 'secrets.json', got %q", entries[0].Source)
	}
}

func TestRemoveSyncEntry_NotFound(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Sync:  []SyncEntry{{Source: ".env", Dest: "/app/.env"}},
			},
		},
	}

	// Should be a no-op
	cfg.RemoveSyncEntry("dev1", "nonexistent")

	entries := cfg.GetSyncEntries("dev1")
	if len(entries) != 1 {
		t.Errorf("expected entry to remain, got %d entries", len(entries))
	}
}

func TestGetSyncEntries_NoContainer(t *testing.T) {
	cfg := &Config{
		Containers: map[string]Container{},
	}

	entries := cfg.GetSyncEntries("nonexistent")
	if entries != nil {
		t.Errorf("expected nil for unknown container, got %v", entries)
	}
}

func TestSave_WithSyncEntries(t *testing.T) {
	withTempDir(t, func(dir string) {
		cfg := &Config{
			Project: "test",
			Containers: map[string]Container{
				"dev1": {
					Image: "ubuntu:24.04",
					Sync: []SyncEntry{
						{Source: ".env", Dest: "/home/dev/project/.env"},
					},
				},
			},
		}

		if err := cfg.Save(); err != nil {
			t.Fatalf("save failed: %v", err)
		}

		loaded, err := Load("")
		if err != nil {
			t.Fatalf("reload failed: %v", err)
		}

		entries := loaded.GetSyncEntries("dev1")
		if len(entries) != 1 {
			t.Fatalf("expected 1 sync entry after reload, got %d", len(entries))
		}
		if entries[0].Source != ".env" {
			t.Errorf("expected source '.env', got %q", entries[0].Source)
		}
	})
}
