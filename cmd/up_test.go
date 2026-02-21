package cmd

import (
	"strings"
	"testing"
)

func TestUp_Success(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", false) // Stopped
	env.mock.SetOutput("start dev1", "")
	env.mock.SetOutput("list dev1 -c4 -f csv", "10.10.10.100 (eth0)")

	err := runUp(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.mock.HasCall("start", "dev1") {
		t.Error("expected start command")
	}
}

func TestUp_AlreadyRunning(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", true) // Running

	err := runUp(nil, []string{"dev1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not call start
	if env.mock.HasCall("start", "dev1") {
		t.Error("should not start already running container")
	}
}

func TestUp_NotExists(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerNotExists("dev1")

	err := runUp(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUp_StartFails(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", false)
	env.mock.SetError("start dev1", "failed to start")

	err := runUp(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to start") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUp_GetStatusFails(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.mock.SetOutput("info dev1", "Name: dev1")
	env.mock.SetError("list dev1 -cs -f csv", "error getting status")

	err := runUp(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
}
