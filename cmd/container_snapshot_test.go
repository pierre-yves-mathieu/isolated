package cmd

import (
	"strings"
	"testing"
)

func TestSnapshotCreate_Success(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetError("info test-dev1/checkpoint", "not found") // Snapshot doesn't exist yet
	env.mock.SetOutput("snapshot test-dev1 checkpoint", "")

	err := runSnapshotCreate(nil, []string{"dev1", "checkpoint"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify snapshot command was called
	if !env.mock.HasCall("snapshot", "test-dev1", "checkpoint") {
		t.Error("expected snapshot command")
	}

	// Verify config was updated
	cfg := env.readConfig()
	if !strings.Contains(cfg, "checkpoint") {
		t.Error("expected snapshot to be added to config")
	}
}

func TestSnapshotCreate_WithDescription(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetError("info test-dev1/checkpoint", "not found")
	env.mock.SetOutput("snapshot test-dev1 checkpoint", "")

	snapshotDescription = "Before major refactoring"
	defer func() { snapshotDescription = "" }()

	err := runSnapshotCreate(nil, []string{"dev1", "checkpoint"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := env.readConfig()
	if !strings.Contains(cfg, "Before major refactoring") {
		t.Error("expected description in config")
	}
}

func TestSnapshotCreate_ContainerNotFound(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers: {}
`)

	err := runSnapshotCreate(nil, []string{"dev1", "checkpoint"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSnapshotCreate_AlreadyExists(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetOutput("info test-dev1/checkpoint", "Name: checkpoint") // Snapshot exists

	err := runSnapshotCreate(nil, []string{"dev1", "checkpoint"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSnapshotCreate_NoProject(t *testing.T) {
	_ = setupTestEnv(t)
	// No config file

	err := runSnapshotCreate(nil, []string{"dev1", "checkpoint"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no project") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSnapshotList_Success(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
    snapshots:
      initial-state:
        description: Initial state
        created_at: "2024-01-15T10:30:00Z"
`)
	env.mock.SetOutput("query /1.0/instances/test-dev1/snapshots",
		`["/1.0/instances/test-dev1/snapshots/initial-state"]`)

	err := runSnapshotList(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSnapshotList_Empty(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.mock.SetOutput("query /1.0/instances/test-dev1/snapshots", "[]")

	err := runSnapshotList(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSnapshotList_ContainerNotFound(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers: {}
`)

	err := runSnapshotList(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSnapshotDelete_Success(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
    snapshots:
      checkpoint:
        description: test
`)
	env.mock.SetOutput("info test-dev1/checkpoint", "Name: checkpoint")
	env.mock.SetOutput("delete test-dev1/checkpoint", "")

	err := runSnapshotDelete(nil, []string{"dev1", "checkpoint"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.mock.HasCall("delete", "test-dev1/checkpoint") {
		t.Error("expected delete command")
	}
}

func TestSnapshotDelete_InitialState(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)

	err := runSnapshotDelete(nil, []string{"dev1", "initial-state"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot delete 'initial-state'") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSnapshotDelete_NotExists(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.mock.SetError("info test-dev1/checkpoint", "not found")

	err := runSnapshotDelete(nil, []string{"dev1", "checkpoint"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSnapshotDelete_ContainerNotFound(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers: {}
`)

	err := runSnapshotDelete(nil, []string{"dev1", "checkpoint"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}
