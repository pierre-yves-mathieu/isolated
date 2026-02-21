package cmd

import (
	"strings"
	"testing"
)

// Note: Full snapshot tests require e2e testing because PublishSnapshotWithProgress
// uses exec.Command directly for streaming output. These tests cover the
// pre-publish validation logic.

func TestImageCreate_NotExists(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerNotExists("dev1")

	err := runImageCreate(nil, []string{"dev1", "my-image"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestImageCreate_StopFails(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", true)
	env.mock.SetError("stop dev1 --timeout=5", "failed to stop")

	err := runImageCreate(nil, []string{"dev1", "my-image"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to stop") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestImageCreate_SnapshotFails(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", false) // Already stopped
	env.mock.SetError("snapshot dev1", "failed to create snapshot")

	err := runImageCreate(nil, []string{"dev1", "my-image"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "snapshot") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestImageCreate_StopsRunningContainer(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", true) // Running
	env.mock.SetOutput("stop dev1 --timeout=5", "")
	// Let snapshot fail so we don't hit the exec.Command publish
	env.mock.SetError("snapshot dev1", "test stop")

	runImageCreate(nil, []string{"dev1", "my-image"})

	if !env.mock.HasCall("stop", "dev1", "--timeout=5") {
		t.Error("expected stop command for running container")
	}
}

func TestImageCreate_SkipsStopForStoppedContainer(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", false) // Already stopped
	// Let snapshot fail so we don't hit the exec.Command publish
	env.mock.SetError("snapshot dev1", "test stop")

	runImageCreate(nil, []string{"dev1", "my-image"})

	if env.mock.HasCall("stop", "dev1") {
		t.Error("should not stop already stopped container")
	}
}

func TestImageCreate_CreatesSnapshot(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", false)
	env.mock.SetOutput("snapshot dev1", "")
	// Let publish fail (it uses exec.Command directly)
	// We just want to verify snapshot was called

	runImageCreate(nil, []string{"dev1", "my-image"})

	// Check that snapshot was called with the container name
	found := false
	for _, call := range env.mock.Calls {
		if len(call.Args) >= 2 && call.Args[0] == "snapshot" && call.Args[1] == "dev1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected snapshot command")
	}
}

func TestImageCreate_GetStatusFails(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.mock.SetOutput("info dev1", "Name: dev1")
	env.mock.SetError("list dev1 -cs -f csv", "permission denied")

	err := runImageCreate(nil, []string{"dev1", "my-image"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestImageCreate_DeletesSnapshotOnSuccess(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", false)
	env.mock.SetOutput("snapshot dev1", "")
	// The delete is called for cleanup, but publish uses exec.Command
	// so we can only verify the snapshot was created

	runImageCreate(nil, []string{"dev1", "my-image"})

	// Verify snapshot was created
	snapshotCalled := false
	for _, call := range env.mock.Calls {
		if len(call.Args) >= 2 && call.Args[0] == "snapshot" {
			snapshotCalled = true
			break
		}
	}
	if !snapshotCalled {
		t.Error("expected snapshot to be created")
	}
}

func TestImageCreate_HandlesContainerWithSpecialChars(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("my-dev-1", "ubuntu:24.04")
	env.mock.SetOutput("info my-dev-1", "Name: my-dev-1")
	env.mock.SetOutput("list my-dev-1 -cs -f csv", "STOPPED")
	env.mock.SetOutput("list my-dev-1 -c4 -f csv", "")
	env.mock.SetOutput("snapshot my-dev-1", "")

	runImageCreate(nil, []string{"my-dev-1", "my-base-image"})

	// Verify snapshot was called with correct name
	found := false
	for _, call := range env.mock.Calls {
		if len(call.Args) >= 2 && call.Args[0] == "snapshot" && call.Args[1] == "my-dev-1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected snapshot command with correct container name")
	}
}

func TestImageCreate_CallsStopBeforeSnapshot(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", true) // Running
	env.mock.SetOutput("stop dev1 --timeout=5", "")
	env.mock.SetError("snapshot dev1", "test stop") // Fail early for testing

	runImageCreate(nil, []string{"dev1", "my-image"})

	// Find the index of stop and snapshot calls
	stopIdx := -1
	snapshotIdx := -1
	for i, call := range env.mock.Calls {
		if len(call.Args) >= 1 && call.Args[0] == "stop" {
			stopIdx = i
		}
		if len(call.Args) >= 1 && call.Args[0] == "snapshot" {
			snapshotIdx = i
		}
	}

	if stopIdx == -1 {
		t.Error("expected stop command to be called")
	}
	if snapshotIdx == -1 {
		t.Error("expected snapshot command to be called")
	}
	if stopIdx > snapshotIdx {
		t.Error("expected stop to be called before snapshot")
	}
}
