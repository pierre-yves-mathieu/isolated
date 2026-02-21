package cmd

import (
	"strings"
	"testing"

	"lxc-dev-manager/internal/operations"
)

func TestSSH_ContainerNotExists(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerNotExists("dev1")

	err := runSSH(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSSH_ContainerNotRunning(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", false) // stopped

	err := runSSH(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSSH_GetStatusFails(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.mock.SetOutput("info dev1", "Name: dev1")
	env.mock.SetError("list dev1 -cs -f csv", "permission denied")

	err := runSSH(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSSH_ChecksContainerIsRunning(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.mock.SetOutput("info dev1", "Name: dev1")
	env.mock.SetOutput("list dev1 -cs -f csv", "STOPPED")

	err := runSSH(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error for stopped container")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSSH_ContainerWithDifferentStatuses(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		expectErr bool
	}{
		// Skip RUNNING - can't test without mocking syscall.Exec
		{"stopped", "STOPPED", true},
		{"frozen", "FROZEN", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnv(t)
			env.writeConfigWithContainer("dev1", "ubuntu:24.04")
			env.mock.SetOutput("info dev1", "Name: dev1")
			env.mock.SetOutput("list dev1 -cs -f csv", tt.status)

			err := runSSH(nil, []string{"dev1"})
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error for non-running container")
				}
				if !strings.Contains(err.Error(), "not running") {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Note: TestSSH_Success would require mocking syscall.Exec
// which is complex. The actual shell functionality is tested via e2e tests.

func TestBuildShellArgs_WithUser(t *testing.T) {
	// When user is specified, should use "su -l <user>" to get proper login shell
	// This ensures PAM is triggered and supplementary groups (like docker) are loaded
	args := operations.BuildShellArgs("mycontainer", "dev")

	expected := []string{"exec", "mycontainer", "--", "su", "-l", "dev"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, expected[i], arg)
		}
	}
}

func TestBuildShellArgs_WithoutUser(t *testing.T) {
	// When no user specified, should use root bash shell
	args := operations.BuildShellArgs("mycontainer", "")

	expected := []string{"exec", "mycontainer", "--", "bash", "-l"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, expected[i], arg)
		}
	}
}

func TestBuildShellArgs_DifferentUsers(t *testing.T) {
	tests := []struct {
		user     string
		expected []string
	}{
		{"dev", []string{"exec", "test-container", "--", "su", "-l", "dev"}},
		{"root", []string{"exec", "test-container", "--", "bash", "-l"}}, // root uses bash directly
		{"ubuntu", []string{"exec", "test-container", "--", "su", "-l", "ubuntu"}},
		{"", []string{"exec", "test-container", "--", "bash", "-l"}},
	}

	for _, tt := range tests {
		name := tt.user
		if name == "" {
			name = "no-user"
		}
		t.Run(name, func(t *testing.T) {
			args := operations.BuildShellArgs("test-container", tt.user)
			if len(args) != len(tt.expected) {
				t.Fatalf("expected %d args, got %d: %v", len(tt.expected), len(args), args)
			}
			for i, arg := range args {
				if arg != tt.expected[i] {
					t.Errorf("arg[%d]: expected %q, got %q", i, tt.expected[i], arg)
				}
			}
		})
	}
}
