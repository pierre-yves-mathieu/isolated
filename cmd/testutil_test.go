package cmd

import (
	"bytes"
	"os"
	"testing"

	"lxc-dev-manager/internal/lxc"
)

// testEnv holds test environment state
type testEnv struct {
	t      *testing.T
	dir    string
	oldDir string
	mock   *lxc.MockExecutor
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

// setupTestEnv creates an isolated test environment
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	dir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	mock := lxc.NewMockExecutor()
	lxc.SetExecutor(mock)

	env := &testEnv{
		t:      t,
		dir:    dir,
		oldDir: oldDir,
		mock:   mock,
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	}

	t.Cleanup(func() {
		os.Chdir(oldDir)
		lxc.ResetExecutor()
	})

	return env
}

// writeConfig writes a containers.yaml file
func (e *testEnv) writeConfig(yaml string) {
	e.t.Helper()
	if err := os.WriteFile("containers.yaml", []byte(yaml), 0644); err != nil {
		e.t.Fatal(err)
	}
}

// readConfig reads the containers.yaml file
func (e *testEnv) readConfig() string {
	e.t.Helper()
	data, err := os.ReadFile("containers.yaml")
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		e.t.Fatal(err)
	}
	return string(data)
}

// configExists checks if config file exists
func (e *testEnv) configExists() bool {
	_, err := os.Stat("containers.yaml")
	return err == nil
}

// setContainerExists mocks a container as existing
func (e *testEnv) setContainerExists(name string, running bool) {
	e.mock.SetOutput("info "+name, "Name: "+name)
	status := "STOPPED"
	if running {
		status = "RUNNING"
	}
	e.mock.SetOutput("list "+name+" -cs -f csv", status)
	if running {
		e.mock.SetOutput("list "+name+" -c4 -f csv", "10.10.10.100 (eth0)")
	} else {
		e.mock.SetOutput("list "+name+" -c4 -f csv", "")
	}
}

// setContainerNotExists mocks a container as not existing
func (e *testEnv) setContainerNotExists(name string) {
	e.mock.SetError("info "+name, "not found")
}

// setLaunchSuccess mocks successful container launch
func (e *testEnv) setLaunchSuccess() {
	e.mock.DefaultResponse = lxc.MockResponse{Output: []byte("")}
	// Mock cloud-init status to return done quickly
	e.mock.SetOutput("exec", "status: done")
}

// setListAllContainers sets the output for ListAll
func (e *testEnv) setListAllContainers(csv string) {
	e.mock.SetOutput("list -c ns4 -f csv", csv)
}

// writeMinimalConfig writes a minimal config with empty project
// Use this when the test doesn't need specific config values
// Empty project means no prefix is added to container names
func (e *testEnv) writeMinimalConfig() {
	e.writeConfig(`project: ""
containers: {}
`)
}

// writeConfigWithContainer writes a config with a single container defined
// Use this when testing commands that require a container to exist in config
func (e *testEnv) writeConfigWithContainer(name, image string) {
	e.writeConfig(`project: ""
containers:
  ` + name + `:
    image: ` + image + `
`)
}
