package cmd

import (
	"strings"
	"testing"
)

func TestExec_RequiresCommand(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", true)

	// exec without command should fail
	err := runExec(nil, []string{"dev1"})
	if err == nil {
		t.Fatal("expected error when no command provided")
	}
	if !strings.Contains(err.Error(), "command required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExec_ContainerNotExists(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerNotExists("dev1")

	err := runExec(nil, []string{"dev1", "whoami"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExec_ContainerNotRunning(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", false) // stopped

	err := runExec(nil, []string{"dev1", "whoami"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExec_GetStatusFails(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.mock.SetOutput("info dev1", "Name: dev1")
	env.mock.SetError("list dev1 -cs -f csv", "permission denied")

	err := runExec(nil, []string{"dev1", "whoami"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExec_ContainerWithDifferentStatuses(t *testing.T) {
	// Only test non-running statuses since RUNNING case would call syscall.Exec
	// which replaces the test process. The RUNNING path is tested via e2e tests.
	tests := []struct {
		name   string
		status string
	}{
		{"stopped", "STOPPED"},
		{"frozen", "FROZEN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnv(t)
			env.writeConfigWithContainer("dev1", "ubuntu:24.04")
			env.mock.SetOutput("info dev1", "Name: dev1")
			env.mock.SetOutput("list dev1 -cs -f csv", tt.status)

			err := runExec(nil, []string{"dev1", "whoami"})
			if err == nil {
				t.Fatal("expected error for non-running container")
			}
			if !strings.Contains(err.Error(), "not running") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Note: TestExec_Success would require mocking syscall.Exec
// which is complex. The actual exec functionality is tested via e2e tests.

func TestBuildExecArgs(t *testing.T) {
	tests := []struct {
		name      string
		container string
		user      string
		cmdArgs   []string
		expected  []string
	}{
		{
			name:      "simple command no user",
			container: "dev",
			user:      "",
			cmdArgs:   []string{"whoami"},
			expected:  []string{"exec", "dev", "--", "whoami"},
		},
		{
			name:      "simple command with user",
			container: "dev",
			user:      "dev",
			cmdArgs:   []string{"whoami"},
			expected:  []string{"exec", "dev", "--", "su", "-l", "dev", "whoami"},
		},
		{
			name:      "command with flags",
			container: "dev",
			user:      "dev",
			cmdArgs:   []string{"ls", "-la", "/tmp"},
			expected:  []string{"exec", "dev", "--", "su", "-l", "dev", "ls", "-la", "/tmp"},
		},
		{
			name:      "nested double dash",
			container: "dev",
			user:      "dev",
			cmdArgs:   []string{"zellij", "run", "--", "ls"},
			expected:  []string{"exec", "dev", "--", "su", "-l", "dev", "zellij", "run", "--", "ls"},
		},
		{
			name:      "multi-arg command",
			container: "dev",
			user:      "dev",
			cmdArgs:   []string{"npm", "run", "dev"},
			expected:  []string{"exec", "dev", "--", "su", "-l", "dev", "npm", "run", "dev"},
		},
		{
			name:      "explicit bash shell",
			container: "dev",
			user:      "dev",
			cmdArgs:   []string{"bash"},
			expected:  []string{"exec", "dev", "--", "su", "-l", "dev", "bash"},
		},
		{
			name:      "root user command",
			container: "dev",
			user:      "root",
			cmdArgs:   []string{"apt", "update"},
			expected:  []string{"exec", "dev", "--", "su", "-l", "root", "apt", "update"},
		},
		{
			name:      "command with complex nested flags",
			container: "mycontainer",
			user:      "dev",
			cmdArgs:   []string{"zellij", "run", "--", "ls", "-la", "--color=auto"},
			expected:  []string{"exec", "mycontainer", "--", "su", "-l", "dev", "zellij", "run", "--", "ls", "-la", "--color=auto"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildExecArgs(tt.container, tt.user, tt.cmdArgs)
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

func TestBuildExecArgs_DifferentUsers(t *testing.T) {
	tests := []struct {
		user     string
		expected []string
	}{
		{"dev", []string{"exec", "test-container", "--", "su", "-l", "dev", "htop"}},
		{"root", []string{"exec", "test-container", "--", "su", "-l", "root", "htop"}},
		{"ubuntu", []string{"exec", "test-container", "--", "su", "-l", "ubuntu", "htop"}},
		{"", []string{"exec", "test-container", "--", "htop"}},
	}

	for _, tt := range tests {
		name := tt.user
		if name == "" {
			name = "no-user"
		}
		t.Run(name, func(t *testing.T) {
			args := buildExecArgs("test-container", tt.user, []string{"htop"})
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
