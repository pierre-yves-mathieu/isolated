package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMv_InvalidDestinationFormat(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")

	// Create a test file
	testFile := filepath.Join(env.dir, "testfile.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	// Missing colon in destination
	err := runMv(nil, []string{testFile, "dev1/home/dev/"})
	if err == nil {
		t.Fatal("expected error for invalid destination format")
	}
	if !strings.Contains(err.Error(), "invalid destination format") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMv_EmptyDestinationPath(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")

	testFile := filepath.Join(env.dir, "testfile.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	err := runMv(nil, []string{testFile, "dev1:"})
	if err == nil {
		t.Fatal("expected error for empty destination path")
	}
	if !strings.Contains(err.Error(), "destination path cannot be empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMv_SourceNotExists(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")

	err := runMv(nil, []string{"/nonexistent/file.txt", "dev1:/home/dev/"})
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMv_ContainerNotInConfig(t *testing.T) {
	env := setupTestEnv(t)
	env.writeMinimalConfig()

	testFile := filepath.Join(env.dir, "testfile.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	err := runMv(nil, []string{testFile, "nonexistent:/home/dev/"})
	if err == nil {
		t.Fatal("expected error for container not in config")
	}
	if !strings.Contains(err.Error(), "not found in project config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMv_ContainerNotInLXC(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerNotExists("dev1")

	testFile := filepath.Join(env.dir, "testfile.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	err := runMv(nil, []string{testFile, "dev1:/home/dev/"})
	if err == nil {
		t.Fatal("expected error for container not in LXC")
	}
	if !strings.Contains(err.Error(), "does not exist in LXC") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMv_TildeExpansion(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: ""
containers:
  dev1:
    image: ubuntu:24.04
defaults:
  user:
    name: myuser
`)
	env.setContainerExists("dev1", true)
	// Mock directory exists check
	env.mock.SetOutput("exec dev1 -- test -d /home/myuser/.ssh", "")
	// Mock file push
	env.mock.SetOutput("file push", "")
	// Mock chown
	env.mock.SetOutput("exec dev1 -- chown myuser:myuser /home/myuser/.ssh/key", "")

	testFile := filepath.Join(env.dir, "key")
	os.WriteFile(testFile, []byte("ssh key content"), 0644)

	err := runMv(nil, []string{testFile, "dev1:~/.ssh/key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file push was called with expanded path
	foundPush := false
	for _, call := range env.mock.Calls {
		callStr := strings.Join(call.Args, " ")
		if strings.Contains(callStr, "file push") && strings.Contains(callStr, "/home/myuser/.ssh/key") {
			foundPush = true
			break
		}
	}
	if !foundPush {
		t.Errorf("expected file push with expanded path, got calls: %v", env.mock.Calls)
	}
}

func TestMv_SuccessfulFileCopy(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", true)
	// Mock directory exists
	env.mock.SetOutput("exec dev1 -- test -d /home/dev", "")
	// Mock file push
	env.mock.SetOutput("file push", "")
	// Mock chown (default user is "dev")
	env.mock.SetOutput("exec dev1 -- chown dev:dev /home/dev/testfile.txt", "")

	testFile := filepath.Join(env.dir, "testfile.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	err := runMv(nil, []string{testFile, "dev1:/home/dev/testfile.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify chown was called
	foundChown := false
	for _, call := range env.mock.Calls {
		callStr := strings.Join(call.Args, " ")
		if strings.Contains(callStr, "chown") && strings.Contains(callStr, "dev:dev") {
			foundChown = true
			break
		}
	}
	if !foundChown {
		t.Errorf("expected chown call, got calls: %v", env.mock.Calls)
	}
}

func TestMv_DirectoryCopyWithRecursiveChown(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfigWithContainer("dev1", "ubuntu:24.04")
	env.setContainerExists("dev1", true)
	// Mock directory exists
	env.mock.SetOutput("exec dev1 -- test -d", "")
	// Mock file push with -r
	env.mock.SetOutput("file push -r", "")
	// Mock recursive chown
	env.mock.SetOutput("exec dev1 -- chown -R dev:dev", "")

	// Create a test directory
	testDir := filepath.Join(env.dir, "myproject")
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content"), 0644)

	err := runMv(nil, []string{testDir, "dev1:/home/dev/myproject"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify recursive chown was called
	foundRecursiveChown := false
	for _, call := range env.mock.Calls {
		callStr := strings.Join(call.Args, " ")
		if strings.Contains(callStr, "chown -R") {
			foundRecursiveChown = true
			break
		}
	}
	if !foundRecursiveChown {
		t.Errorf("expected recursive chown call, got calls: %v", env.mock.Calls)
	}
}

func TestMv_TildeExpansionContainerToHost(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: ""
containers:
  dev1:
    image: ubuntu:24.04
defaults:
  user:
    name: myuser
`)
	env.setContainerExists("dev1", true)
	// Mock file exists check (test -e)
	env.mock.SetOutput("exec dev1 -- test -e /home/myuser/.bashrc", "")
	// Mock is directory check (test -d) - return error to indicate it's a file
	env.mock.SetError("exec dev1 -- test -d /home/myuser/.bashrc", "not a directory")
	// Mock file pull
	env.mock.SetOutput("file pull", "")

	destFile := filepath.Join(env.dir, "bashrc_backup")

	err := runMv(nil, []string{"dev1:~/.bashrc", destFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file pull was called with expanded path
	foundPull := false
	for _, call := range env.mock.Calls {
		callStr := strings.Join(call.Args, " ")
		if strings.Contains(callStr, "file pull") && strings.Contains(callStr, "/home/myuser/.bashrc") {
			foundPull = true
			break
		}
	}
	if !foundPull {
		t.Errorf("expected file pull with expanded path, got calls: %v", env.mock.Calls)
	}
}

func TestMv_TildeExpansionContainerToContainer(t *testing.T) {
	env := setupTestEnv(t)
	env.writeConfig(`project: ""
containers:
  dev1:
    image: ubuntu:24.04
    user:
      name: alice
  dev2:
    image: ubuntu:24.04
    user:
      name: bob
`)
	env.setContainerExists("dev1", true)
	env.setContainerExists("dev2", true)
	// Mock file exists check for source
	env.mock.SetOutput("exec dev1 -- test -e /home/alice/config", "")
	// Mock is directory check - return error to indicate it's a file
	env.mock.SetError("exec dev1 -- test -d /home/alice/config", "not a directory")
	// Mock file pull from dev1 - use callback to create the temp file
	env.mock.SetOutput("file pull", "")
	env.mock.SetCallback("file pull", func(args []string) {
		// args: ["file", "pull", "dev1//home/alice/config", "/tmp/lxc-mv-.../config"]
		// The last arg is the destination path
		if len(args) >= 4 {
			destPath := args[len(args)-1]
			_ = os.WriteFile(destPath, []byte("mock config content"), 0644)
		}
	})
	// Mock directory exists check for dest
	env.mock.SetOutput("exec dev2 -- test -d /home/bob", "")
	// Mock file push to dev2
	env.mock.SetOutput("file push", "")
	// Mock chown on dev2
	env.mock.SetOutput("exec dev2 -- chown bob:bob /home/bob/config", "")

	err := runMv(nil, []string{"dev1:~/config", "dev2:~/config"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file pull was called with alice's home
	foundPull := false
	for _, call := range env.mock.Calls {
		callStr := strings.Join(call.Args, " ")
		if strings.Contains(callStr, "file pull") && strings.Contains(callStr, "/home/alice/config") {
			foundPull = true
			break
		}
	}
	if !foundPull {
		t.Errorf("expected file pull with alice's home path, got calls: %v", env.mock.Calls)
	}

	// Verify the file push was called with bob's home
	foundPush := false
	for _, call := range env.mock.Calls {
		callStr := strings.Join(call.Args, " ")
		if strings.Contains(callStr, "file push") && strings.Contains(callStr, "/home/bob/config") {
			foundPush = true
			break
		}
	}
	if !foundPush {
		t.Errorf("expected file push with bob's home path, got calls: %v", env.mock.Calls)
	}
}
