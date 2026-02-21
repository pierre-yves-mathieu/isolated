package lxcmgr

import (
	"os"
	"path/filepath"
	"testing"

	"lxc-dev-manager/internal/lxc"
)

// setupTestProject creates a temporary test project directory
func setupTestProject(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "lxcmgr-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a basic config file
	configContent := `project: test-project
defaults:
  ports: [8080, 9000]
containers:
  dev1:
    image: ubuntu:24.04
  dev2:
    image: ubuntu:24.04
`
	configPath := filepath.Join(tmpDir, "containers.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create config file: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// setupMockExecutor sets up a mock executor for testing
func setupMockExecutor(t *testing.T) (*lxc.MockExecutor, func()) {
	t.Helper()

	mock := lxc.NewMockExecutor()
	lxc.SetExecutor(mock)

	cleanup := func() {
		lxc.ResetExecutor()
	}

	return mock, cleanup
}

func TestNew(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	mock, mockCleanup := setupMockExecutor(t)
	defer mockCleanup()

	// Mock container existence checks
	mock.SetOutput("info test-project-dev1", "")
	mock.SetOutput("info test-project-dev2", "")

	client, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if client.ProjectName() != "test-project" {
		t.Errorf("Expected project name 'test-project', got '%s'", client.ProjectName())
	}

	if client.Dir() != tmpDir {
		t.Errorf("Expected dir '%s', got '%s'", tmpDir, client.Dir())
	}
}

func TestNew_ProjectNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lxcmgr-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = New(tmpDir)
	if err == nil {
		t.Error("Expected error for non-existent project")
	}
}

func TestNewProject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lxcmgr-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	client, err := NewProject(tmpDir,
		WithProjectName("my-project"),
		WithDefaultPorts(8080, 9000),
	)
	if err != nil {
		t.Fatalf("NewProject() failed: %v", err)
	}

	if client.ProjectName() != "my-project" {
		t.Errorf("Expected project name 'my-project', got '%s'", client.ProjectName())
	}

	// Verify config file was created
	configPath := filepath.Join(tmpDir, "containers.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestClient_List(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	mock, mockCleanup := setupMockExecutor(t)
	defer mockCleanup()

	// Mock container existence checks
	mock.SetOutput("info test-project-dev1", "")
	mock.SetOutput("info test-project-dev2", "")

	// Mock list output
	mock.SetOutput("list -c ns4 -f csv", "test-project-dev1,RUNNING,10.0.0.1\ntest-project-dev2,STOPPED,")

	client, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	containers, err := client.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(containers) != 2 {
		t.Errorf("Expected 2 containers, got %d", len(containers))
	}

	// Find dev1 in the list
	var foundDev1 bool
	for _, c := range containers {
		if c.Name == "dev1" {
			foundDev1 = true
			if c.Status != StatusRunning {
				t.Errorf("Expected dev1 status RUNNING, got %s", c.Status)
			}
			if c.IP != "10.0.0.1" {
				t.Errorf("Expected dev1 IP '10.0.0.1', got '%s'", c.IP)
			}
		}
	}

	if !foundDev1 {
		t.Error("dev1 not found in container list")
	}
}

func TestClient_Exists(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	mock, mockCleanup := setupMockExecutor(t)
	defer mockCleanup()

	// Mock container existence - dev1 exists, dev2 exists
	mock.SetOutput("info test-project-dev1", "")
	mock.SetOutput("info test-project-dev2", "")

	client, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if !client.Exists("dev1") {
		t.Error("Expected dev1 to exist")
	}

	if client.Exists("nonexistent") {
		t.Error("Expected nonexistent to not exist")
	}
}

func TestClient_Start(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	mock, mockCleanup := setupMockExecutor(t)
	defer mockCleanup()

	// Mock container existence
	mock.SetOutput("info test-project-dev1", "")
	mock.SetOutput("info test-project-dev2", "")

	// Mock status check (STOPPED)
	mock.SetOutput("list test-project-dev1 -cs -f csv", "STOPPED")

	// Mock start success
	mock.SetOutput("start test-project-dev1", "")

	client, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = client.Start("dev1")
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Verify start was called
	if !mock.HasCallPrefix("start", "test-project-dev1") {
		t.Error("Expected start command to be called")
	}
}

func TestClient_Stop(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	mock, mockCleanup := setupMockExecutor(t)
	defer mockCleanup()

	// Mock container existence
	mock.SetOutput("info test-project-dev1", "")
	mock.SetOutput("info test-project-dev2", "")

	// Mock status check (RUNNING)
	mock.SetOutput("list test-project-dev1 -cs -f csv", "RUNNING")

	// Mock stop success
	mock.SetOutput("stop test-project-dev1 --timeout=5", "")

	client, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = client.Stop("dev1")
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Verify stop was called
	if !mock.HasCallPrefix("stop", "test-project-dev1") {
		t.Error("Expected stop command to be called")
	}
}

func TestClient_Status(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	mock, mockCleanup := setupMockExecutor(t)
	defer mockCleanup()

	// Mock container existence
	mock.SetOutput("info test-project-dev1", "")
	mock.SetOutput("info test-project-dev2", "")

	// Mock status check
	mock.SetOutput("list test-project-dev1 -cs -f csv", "RUNNING")

	client, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	status, err := client.Status("dev1")
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	if status != StatusRunning {
		t.Errorf("Expected status RUNNING, got %s", status)
	}
}

func TestClient_IP(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	mock, mockCleanup := setupMockExecutor(t)
	defer mockCleanup()

	// Mock container existence
	mock.SetOutput("info test-project-dev1", "")
	mock.SetOutput("info test-project-dev2", "")

	// Mock IP lookup
	mock.SetOutput("list test-project-dev1 -c4 -f csv", "10.0.0.5 (eth0)")

	client, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ip, err := client.IP("dev1")
	if err != nil {
		t.Fatalf("IP() failed: %v", err)
	}

	if ip != "10.0.0.5" {
		t.Errorf("Expected IP '10.0.0.5', got '%s'", ip)
	}
}

func TestContainerError_Unwrap(t *testing.T) {
	innerErr := ErrContainerNotFound
	err := &ContainerError{
		Container: "test",
		Op:        "start",
		Err:       innerErr,
	}

	if err.Error() != "start test: container not found" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}

	if err.Unwrap() != innerErr {
		t.Error("Unwrap() did not return inner error")
	}
}

func TestMountError_Unwrap(t *testing.T) {
	innerErr := ErrMountNotFound
	err := &MountError{
		Container: "test",
		Mount:     "data",
		Op:        "unmount",
		Err:       innerErr,
	}

	if err.Unwrap() != innerErr {
		t.Error("Unwrap() did not return inner error")
	}
}
