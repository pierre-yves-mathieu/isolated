package cmd

import (
	"strings"
	"testing"
)

func TestDown_Success(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", true) // Running
	env.mock.SetOutput("stop dev1 --timeout=5", "")

	err := runDown(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.mock.HasCall("stop", "dev1", "--timeout=5") {
		t.Error("expected stop command")
	}
}

func TestDown_AlreadyStopped(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", false) // Stopped

	err := runDown(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not call stop
	if env.mock.HasCall("stop", "dev1", "--timeout=5") {
		t.Error("should not stop already stopped container")
	}
}

func TestDown_NotExists(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerNotExists("dev1")

	err := runDown(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDown_StopFails(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", true)
	env.mock.SetError("stop dev1 --timeout=5", "failed to stop")

	err := runDown(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to stop") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDown_GetStatusFails(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.mock.SetOutput("info dev1", "Name: dev1")
	env.mock.SetError("list dev1 -cs -f csv", "error getting status")

	err := runDown(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
}
