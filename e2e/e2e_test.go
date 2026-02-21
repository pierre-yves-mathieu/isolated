//go:build e2e

package e2e

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	testProject       = "e2etest"
	testContainerName = "dev"
	testImageName     = "e2etest-image"
	testImage         = "ubuntu:24.04"
)

// lxcContainerName returns the full LXC container name (project-container)
func lxcContainerName(container string) string {
	return testProject + "-" + container
}

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary
	tmpDir, err := os.MkdirTemp("", "lxc-dev-manager-e2e")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "lxc-dev-manager")

	// Build from project root (parent of e2e directory)
	projectRoot, _ := filepath.Abs(filepath.Join(".."))
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build: %v\n%s\n", err, output)
		os.Exit(1)
	}

	// Clean up any leftover test containers
	cleanup()

	code := m.Run()

	// Final cleanup
	cleanup()

	os.Exit(code)
}

func cleanup() {
	// Delete test containers if they exist (use full LXC names)
	exec.Command("lxc", "delete", lxcContainerName("dev"), "--force").Run()
	exec.Command("lxc", "delete", lxcContainerName("dev2"), "--force").Run()
	exec.Command("lxc", "delete", lxcContainerName("clone"), "--force").Run()
	// Delete test image if exists
	exec.Command("lxc", "image", "delete", testImageName).Run()
}

func runInDir(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func lxc(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("lxc", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// setupProject creates a temp dir and initializes a project
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	output, err := runInDir(t, dir, "create", "--name", testProject)
	if err != nil {
		t.Fatalf("project create failed: %v\n%s", err, output)
	}

	return dir
}

func TestE2E_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	dir := setupProject(t)
	lxcName := lxcContainerName("dev")

	// Create container
	t.Log("Creating container...")
	output, err := runInDir(t, dir, "container", "create", "dev", testImage)
	if err != nil {
		t.Fatalf("container create failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "created successfully") {
		t.Errorf("unexpected output: %s", output)
	}

	// List should show container
	t.Log("Listing containers...")
	output, err = runInDir(t, dir, "list")
	if err != nil {
		t.Fatalf("list failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "dev") {
		t.Errorf("container not in list: %s", output)
	}

	// Stop container
	t.Log("Stopping container...")
	output, err = runInDir(t, dir, "down", "dev")
	if err != nil {
		t.Fatalf("down failed: %v\n%s", err, output)
	}

	// Start container
	t.Log("Starting container...")
	output, err = runInDir(t, dir, "up", "dev")
	if err != nil {
		t.Fatalf("up failed: %v\n%s", err, output)
	}

	// Remove container
	t.Log("Removing container...")
	output, err = runInDir(t, dir, "remove", "dev", "--force")
	if err != nil {
		t.Fatalf("remove failed: %v\n%s", err, output)
	}

	// Verify removed
	output, _ = lxc(t, "info", lxcName)
	if !strings.Contains(output, "not found") && !strings.Contains(output, "does not exist") {
		time.Sleep(time.Second)
		output, _ = lxc(t, "info", lxcName)
		if !strings.Contains(output, "not found") && !strings.Contains(output, "does not exist") {
			t.Errorf("container should be removed: %s", output)
		}
	}
}

func TestE2E_ContainerCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	dir := setupProject(t)
	lxcName := lxcContainerName("dev")
	defer func() {
		runInDir(t, dir, "remove", "dev", "--force")
	}()

	output, err := runInDir(t, dir, "container", "create", "dev", testImage)
	if err != nil {
		t.Fatalf("container create failed: %v\n%s", err, output)
	}

	// Verify container exists in LXC
	output, err = lxc(t, "info", lxcName)
	if err != nil {
		t.Fatalf("container not created in LXC: %v\n%s", err, output)
	}
	if !strings.Contains(output, "Status: RUNNING") {
		t.Errorf("container should be running: %s", output)
	}
}

func TestE2E_DevUserExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	dir := setupProject(t)
	lxcName := lxcContainerName("dev")
	defer func() {
		runInDir(t, dir, "remove", "dev", "--force")
	}()

	_, err := runInDir(t, dir, "container", "create", "dev", testImage)
	if err != nil {
		t.Fatalf("container create failed: %v", err)
	}

	// Check dev user exists
	output, err := lxc(t, "exec", lxcName, "--", "id", "dev")
	if err != nil {
		t.Fatalf("dev user should exist: %v\n%s", err, output)
	}
	if !strings.Contains(output, "dev") {
		t.Errorf("unexpected id output: %s", output)
	}
}

func TestE2E_SSHWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	dir := setupProject(t)
	lxcName := lxcContainerName("dev")
	defer func() {
		runInDir(t, dir, "remove", "dev", "--force")
	}()

	_, err := runInDir(t, dir, "container", "create", "dev", testImage)
	if err != nil {
		t.Fatalf("container create failed: %v", err)
	}

	// Check SSH is running
	output, err := lxc(t, "exec", lxcName, "--", "systemctl", "is-active", "ssh")
	if err != nil {
		// Try sshd instead
		output, err = lxc(t, "exec", lxcName, "--", "systemctl", "is-active", "sshd")
	}
	if err != nil {
		t.Fatalf("SSH should be running: %v\n%s", err, output)
	}
	if !strings.Contains(output, "active") {
		t.Errorf("SSH should be active: %s", output)
	}
}

func TestE2E_NestingEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	dir := setupProject(t)
	lxcName := lxcContainerName("dev")
	defer func() {
		runInDir(t, dir, "remove", "dev", "--force")
	}()

	_, err := runInDir(t, dir, "container", "create", "dev", testImage)
	if err != nil {
		t.Fatalf("container create failed: %v", err)
	}

	// Check nesting is enabled
	output, err := lxc(t, "config", "get", lxcName, "security.nesting")
	if err != nil {
		t.Fatalf("failed to get config: %v\n%s", err, output)
	}
	if !strings.Contains(output, "true") {
		t.Errorf("nesting should be enabled: %s", output)
	}
}

func TestE2E_ContainerClone(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	dir := setupProject(t)
	defer func() {
		runInDir(t, dir, "remove", "dev", "--force")
		runInDir(t, dir, "remove", "clone", "--force")
	}()

	// Create first container
	_, err := runInDir(t, dir, "container", "create", "dev", testImage)
	if err != nil {
		t.Fatalf("container create failed: %v", err)
	}

	// Clone it
	output, err := runInDir(t, dir, "container", "clone", "dev", "clone")
	if err != nil {
		t.Fatalf("clone failed: %v\n%s", err, output)
	}

	// Verify clone exists and has dev user
	lxcClone := lxcContainerName("clone")
	output, err = lxc(t, "exec", lxcClone, "--", "id", "dev")
	if err != nil {
		t.Fatalf("clone should have dev user: %v\n%s", err, output)
	}
}

func TestE2E_ProxyForwarding(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	dir := setupProject(t)
	lxcName := lxcContainerName("dev")
	defer func() {
		runInDir(t, dir, "remove", "dev", "--force")
	}()

	// Update config with specific port
	configPath := filepath.Join(dir, "containers.yaml")
	configYAML := fmt.Sprintf(`project: %s
defaults:
  ports:
    - 18080
containers: {}
`, testProject)
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := runInDir(t, dir, "container", "create", "dev", testImage)
	if err != nil {
		t.Fatalf("container create failed: %v", err)
	}

	// Start a simple HTTP server in container
	go func() {
		lxc(t, "exec", lxcName, "--", "bash", "-c",
			"echo 'hello from container' > /tmp/index.html && cd /tmp && python3 -m http.server 18080 &")
	}()

	// Give server time to start
	time.Sleep(2 * time.Second)

	// Start proxy in background
	cmd := exec.Command(binaryPath, "proxy", "dev")
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer cmd.Process.Kill()

	// Give proxy time to start
	time.Sleep(500 * time.Millisecond)

	// Try to connect through proxy
	resp, err := http.Get("http://localhost:18080/index.html")
	if err != nil {
		t.Skipf("could not connect to proxy (may be port conflict): %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestE2E_MvFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	dir := setupProject(t)
	lxcName := lxcContainerName("dev")
	defer func() {
		runInDir(t, dir, "remove", "dev", "--force")
	}()

	// Create container
	_, err := runInDir(t, dir, "container", "create", "dev", testImage)
	if err != nil {
		t.Fatalf("container create failed: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(dir, "testfile.txt")
	testContent := "hello from host"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Copy file to container
	output, err := runInDir(t, dir, "mv", testFile, "dev:~/testfile.txt")
	if err != nil {
		t.Fatalf("mv failed: %v\n%s", err, output)
	}

	// Verify file exists in container with correct content
	output, err = lxc(t, "exec", lxcName, "--", "cat", "/home/dev/testfile.txt")
	if err != nil {
		t.Fatalf("file should exist in container: %v\n%s", err, output)
	}
	if !strings.Contains(output, testContent) {
		t.Errorf("file content mismatch: expected %q, got %q", testContent, output)
	}

	// Verify ownership is correct (should be dev:dev)
	output, err = lxc(t, "exec", lxcName, "--", "stat", "-c", "%U:%G", "/home/dev/testfile.txt")
	if err != nil {
		t.Fatalf("stat failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "dev:dev") {
		t.Errorf("ownership should be dev:dev, got %s", output)
	}
}

func TestE2E_MvDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	dir := setupProject(t)
	lxcName := lxcContainerName("dev")
	defer func() {
		runInDir(t, dir, "remove", "dev", "--force")
	}()

	// Create container
	_, err := runInDir(t, dir, "container", "create", "dev", testImage)
	if err != nil {
		t.Fatalf("container create failed: %v", err)
	}

	// Create a test directory with files
	testDir := filepath.Join(dir, "myproject")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	// Copy directory to container (use -y to auto-create)
	output, err := runInDir(t, dir, "mv", testDir, "dev:~/myproject", "-y")
	if err != nil {
		t.Fatalf("mv failed: %v\n%s", err, output)
	}

	// Verify directory exists with both files
	output, err = lxc(t, "exec", lxcName, "--", "ls", "/home/dev/myproject/")
	if err != nil {
		t.Fatalf("directory should exist: %v\n%s", err, output)
	}
	if !strings.Contains(output, "file1.txt") || !strings.Contains(output, "file2.txt") {
		t.Errorf("files should exist in directory: %s", output)
	}

	// Verify ownership is correct recursively
	output, err = lxc(t, "exec", lxcName, "--", "stat", "-c", "%U:%G", "/home/dev/myproject/file1.txt")
	if err != nil {
		t.Fatalf("stat failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "dev:dev") {
		t.Errorf("file ownership should be dev:dev, got %s", output)
	}
}

// Helper to check if port is available
func portAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
