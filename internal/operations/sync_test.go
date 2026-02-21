package operations

import (
	"os"
	"path/filepath"
	"testing"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
)

func setupSyncMock(t *testing.T) *lxc.MockExecutor {
	t.Helper()
	mock := lxc.NewMockExecutor()
	lxc.SetExecutor(mock)
	t.Cleanup(func() {
		lxc.ResetExecutor()
	})
	return mock
}

func setupSyncTest(t *testing.T, entries []config.SyncEntry) (*config.Config, string) {
	t.Helper()

	dir := t.TempDir()

	cfg := &config.Config{
		Project: "test",
		Containers: map[string]config.Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Sync:  entries,
				User:  config.User{Name: "dev", Password: "dev"},
			},
		},
	}

	return cfg, dir
}

func mockContainerRunning(mock *lxc.MockExecutor, lxcName string) {
	mock.SetOutput("info "+lxcName, "Name: "+lxcName)
	mock.SetOutput("list "+lxcName+" -cs -f csv", "RUNNING")
}

func TestSyncFiles_Success(t *testing.T) {
	mock := setupSyncMock(t)

	dir := t.TempDir()
	// Create source file
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=value"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _ := setupSyncTest(t, []config.SyncEntry{
		{Source: ".env", Dest: "/home/dev/project/.env"},
	})

	mockContainerRunning(mock, "test-dev1")
	// Mock dir exists check
	mock.SetOutput("exec test-dev1", "")
	// Mock file push
	mock.SetOutput("file push", "")

	err := SyncFiles(cfg, "dev1", dir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify file push was called
	if !mock.HasCallPrefix("file", "push") {
		t.Error("expected file push to be called")
	}
}

func TestSyncFiles_SourceNotFound(t *testing.T) {
	mock := setupSyncMock(t)

	dir := t.TempDir()
	// Don't create the source file

	cfg, _ := setupSyncTest(t, []config.SyncEntry{
		{Source: ".env", Dest: "/home/dev/project/.env"},
		{Source: "exists.txt", Dest: "/home/dev/project/exists.txt"},
	})

	// Create the second file so it succeeds
	if err := os.WriteFile(filepath.Join(dir, "exists.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	mockContainerRunning(mock, "test-dev1")
	mock.SetOutput("exec test-dev1", "")
	mock.SetOutput("file push", "")

	err := SyncFiles(cfg, "dev1", dir)
	if err == nil {
		t.Fatal("expected error for missing source")
	}
	// Should report the missing file but still attempt the second
	if !mock.HasCallPrefix("file", "push") {
		t.Error("expected file push for the existing file")
	}
}

func TestSyncFiles_RelativePath(t *testing.T) {
	mock := setupSyncMock(t)

	dir := t.TempDir()
	subdir := filepath.Join(dir, "config")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "secrets.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _ := setupSyncTest(t, []config.SyncEntry{
		{Source: "config/secrets.json", Dest: "/home/dev/project/config/secrets.json"},
	})

	mockContainerRunning(mock, "test-dev1")
	mock.SetOutput("exec test-dev1", "")
	mock.SetOutput("file push", "")

	err := SyncFiles(cfg, "dev1", dir)
	if err != nil {
		t.Fatalf("expected no error for relative path, got: %v", err)
	}
}

func TestSyncFiles_AbsolutePath(t *testing.T) {
	mock := setupSyncMock(t)

	// Create a temp file with an absolute path
	tmpFile := filepath.Join(t.TempDir(), "abs.env")
	if err := os.WriteFile(tmpFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Project: "test",
		Containers: map[string]config.Container{
			"dev1": {
				Image: "ubuntu:24.04",
				Sync:  []config.SyncEntry{{Source: tmpFile, Dest: "/app/.env"}},
				User:  config.User{Name: "dev", Password: "dev"},
			},
		},
	}

	mockContainerRunning(mock, "test-dev1")
	mock.SetOutput("exec test-dev1", "")
	mock.SetOutput("file push", "")

	// baseDir shouldn't matter for absolute paths
	err := SyncFiles(cfg, "dev1", "/irrelevant")
	if err != nil {
		t.Fatalf("expected no error for absolute source path, got: %v", err)
	}
}

func TestSyncFiles_EmptyConfig(t *testing.T) {
	cfg, _ := setupSyncTest(t, nil)

	err := SyncFiles(cfg, "dev1", "/some/dir")
	if err != nil {
		t.Fatalf("expected no error for empty sync config, got: %v", err)
	}
}

func TestSyncFiles_ContainerNotRunning(t *testing.T) {
	mock := setupSyncMock(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _ := setupSyncTest(t, []config.SyncEntry{
		{Source: ".env", Dest: "/app/.env"},
	})

	mock.SetOutput("info test-dev1", "Name: test-dev1")
	mock.SetOutput("list test-dev1 -cs -f csv", "STOPPED")

	err := SyncFiles(cfg, "dev1", dir)
	if err == nil {
		t.Fatal("expected error for stopped container")
	}
}

func TestSyncFiles_ContainerNotFound(t *testing.T) {
	cfg := &config.Config{
		Project:    "test",
		Containers: map[string]config.Container{},
	}

	err := SyncFiles(cfg, "nonexistent", "/some/dir")
	if err == nil {
		t.Fatal("expected error for unknown container")
	}
}
