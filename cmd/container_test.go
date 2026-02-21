package cmd

import (
	"strings"
	"testing"
)

func TestContainerReset_DefaultSnapshot(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetOutput("info test-dev1/initial-state", "Name: initial-state")
	env.mock.SetOutput("stop test-dev1 --timeout=5", "")
	env.mock.SetOutput("restore test-dev1 initial-state", "")
	env.mock.SetOutput("start test-dev1", "")

	err := runContainerReset(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.mock.HasCall("restore", "test-dev1", "initial-state") {
		t.Error("expected restore to initial-state")
	}
}

func TestContainerReset_NamedSnapshot(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetOutput("info test-dev1/checkpoint", "Name: checkpoint")
	env.mock.SetOutput("stop test-dev1 --timeout=5", "")
	env.mock.SetOutput("restore test-dev1 checkpoint", "")
	env.mock.SetOutput("start test-dev1", "")

	err := runContainerReset(nil, []string{"dev1", "checkpoint"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.mock.HasCall("restore", "test-dev1", "checkpoint") {
		t.Error("expected restore to checkpoint")
	}
}

func TestContainerReset_StoppedContainer(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", false) // Stopped
	env.mock.SetOutput("info test-dev1/initial-state", "Name: initial-state")
	env.mock.SetOutput("restore test-dev1 initial-state", "")

	err := runContainerReset(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not try to stop or start
	if env.mock.HasCall("stop", "test-dev1") {
		t.Error("should not stop already stopped container")
	}
	if env.mock.HasCall("start", "test-dev1") {
		t.Error("should not start container that was stopped")
	}
}

func TestContainerReset_SnapshotNotExists(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetError("info test-dev1/nonexistent", "not found")

	err := runContainerReset(nil, []string{"dev1", "nonexistent"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContainerReset_NoInitialState(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", true)
	env.mock.SetError("info test-dev1/initial-state", "not found")

	err := runContainerReset(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "created before this feature") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContainerReset_ContainerNotFound(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers: {}
`)

	err := runContainerReset(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContainerReset_NoProject(t *testing.T) {
	_ = setupTestEnv(t)
	// No config file

	err := runContainerReset(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no project") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContainerReset_ContainerNotInLXC(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerNotExists("test-dev1")

	err := runContainerReset(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does not exist in LXC") {
		t.Errorf("unexpected error: %v", err)
	}
}

// Clone tests

func TestContainerClone_Success(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", false) // Source exists (stopped)
	env.setContainerNotExists("test-dev2")     // Dest doesn't exist
	env.mock.SetOutput("copy test-dev1 test-dev2", "")
	env.mock.SetOutput("snapshot test-dev2 initial-state", "")
	env.mock.SetOutput("start test-dev2", "")

	cloneSnapshot = "" // Reset flag
	err := runContainerClone(nil, []string{"dev1", "dev2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.mock.HasCall("copy", "test-dev1", "test-dev2") {
		t.Error("expected copy command")
	}
}

func TestContainerClone_FromSnapshot(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", false)
	env.setContainerNotExists("test-dev2")
	env.mock.SetOutput("info test-dev1/checkpoint", "Name: checkpoint")
	env.mock.SetOutput("copy test-dev1/checkpoint test-dev2", "")
	env.mock.SetOutput("snapshot test-dev2 initial-state", "")
	env.mock.SetOutput("start test-dev2", "")

	cloneSnapshot = "checkpoint"
	err := runContainerClone(nil, []string{"dev1", "dev2"})
	cloneSnapshot = "" // Reset

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.mock.HasCall("copy", "test-dev1/checkpoint", "test-dev2") {
		t.Error("expected copy from snapshot")
	}
}

func TestContainerClone_SourceNotFound(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers: {}
`)

	cloneSnapshot = ""
	err := runContainerClone(nil, []string{"dev1", "dev2"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found in project config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContainerClone_DestExists(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
  dev2:
    image: ubuntu:24.04
`)

	cloneSnapshot = ""
	err := runContainerClone(nil, []string{"dev1", "dev2"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContainerClone_SnapshotNotExists(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
	env.setContainerExists("test-dev1", false)
	env.setContainerNotExists("test-dev2")
	env.mock.SetError("info test-dev1/nonexistent", "not found")

	cloneSnapshot = "nonexistent"
	err := runContainerClone(nil, []string{"dev1", "dev2"})
	cloneSnapshot = ""

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContainerClone_NoProject(t *testing.T) {
	_ = setupTestEnv(t)
	// No config file

	cloneSnapshot = ""
	err := runContainerClone(nil, []string{"dev1", "dev2"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no project") {
		t.Errorf("unexpected error: %v", err)
	}
}
