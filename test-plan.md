# Test Implementation Plan

## Overview

Comprehensive test suite covering unit tests, integration tests, and end-to-end tests.

## Test Strategy

### 1. Unit Tests (no LXC required)
- Test internal packages with mocked dependencies
- Fast, run on every commit

### 2. Integration Tests (mocked LXC)
- Test command logic with mocked LXC calls
- Verify correct LXC commands are invoked

### 3. End-to-End Tests (real LXC)
- Test full workflow with actual containers
- Run manually or in CI with LXC available
- Tagged with `//go:build e2e`

---

## Unit Tests

### internal/config - Config Package

| Test | Description |
|------|-------------|
| `TestLoad_FileNotExists` | Returns default config when file doesn't exist |
| `TestLoad_ValidYAML` | Parses valid YAML correctly |
| `TestLoad_InvalidYAML` | Returns error on malformed YAML |
| `TestLoad_EmptyFile` | Handles empty file gracefully |
| `TestSave_CreatesFile` | Creates file with correct YAML |
| `TestSave_OverwritesFile` | Overwrites existing file |
| `TestAddContainer` | Adds container to map |
| `TestAddContainer_Duplicate` | Overwrites existing entry |
| `TestRemoveContainer` | Removes container from map |
| `TestRemoveContainer_NotExists` | No error when removing non-existent |
| `TestGetPorts_ContainerSpecific` | Returns container-specific ports |
| `TestGetPorts_DefaultFallback` | Returns default ports when not specified |
| `TestGetPorts_EmptyDefaults` | Handles empty defaults |
| `TestHasContainer_Exists` | Returns true for existing container |
| `TestHasContainer_NotExists` | Returns false for non-existent |

### internal/proxy - TCP Proxy Package

| Test | Description |
|------|-------------|
| `TestProxy_Start` | Starts listening on port |
| `TestProxy_StartPortInUse` | Returns error if port in use |
| `TestProxy_Stop` | Stops cleanly |
| `TestProxy_ForwardsData` | Data flows local → remote |
| `TestProxy_BidirectionalData` | Data flows both directions |
| `TestProxy_MultipleConnections` | Handles concurrent connections |
| `TestProxy_RemoteUnavailable` | Handles unreachable remote |
| `TestProxy_LargeData` | Handles large data transfers |
| `TestManager_Add` | Adds proxy to manager |
| `TestManager_AddMultiple` | Manages multiple proxies |
| `TestManager_StopAll` | Stops all proxies cleanly |

### internal/lxc - LXC Wrapper Package

| Test | Description |
|------|-------------|
| `TestGetIP_ParsesOutput` | Correctly parses `lxc list` output |
| `TestGetIP_NoIP` | Returns error when no IP |
| `TestGetIP_MultipleIPs` | Returns first IP |
| `TestGetStatus_Running` | Parses RUNNING status |
| `TestGetStatus_Stopped` | Parses STOPPED status |
| `TestListAll_ParsesCSV` | Correctly parses CSV output |
| `TestListAll_Empty` | Handles no containers |

---

## Integration Tests (Mocked LXC)

### cmd/create

| Test | Description |
|------|-------------|
| `TestCreate_Success` | Full create flow with mocked LXC |
| `TestCreate_AlreadyExistsInConfig` | Error when name in config |
| `TestCreate_AlreadyExistsInLXC` | Error when container exists in LXC |
| `TestCreate_LaunchFails` | Handles launch failure |
| `TestCreate_NestingFails` | Continues with warning |
| `TestCreate_UserSetupFails` | Returns error |
| `TestCreate_SSHSetupFails` | Returns error |
| `TestCreate_SavesConfig` | Config file updated |
| `TestCreate_MissingArgs` | Error with usage hint |

### cmd/up

| Test | Description |
|------|-------------|
| `TestUp_Success` | Starts stopped container |
| `TestUp_AlreadyRunning` | Shows message, no error |
| `TestUp_NotExists` | Error when container doesn't exist |
| `TestUp_StartFails` | Handles start failure |
| `TestUp_MissingArgs` | Error with usage hint |

### cmd/down

| Test | Description |
|------|-------------|
| `TestDown_Success` | Stops running container |
| `TestDown_AlreadyStopped` | Shows message, no error |
| `TestDown_NotExists` | Error when container doesn't exist |
| `TestDown_StopFails` | Handles stop failure |
| `TestDown_MissingArgs` | Error with usage hint |

### cmd/list

| Test | Description |
|------|-------------|
| `TestList_Empty` | Shows "no containers" message |
| `TestList_WithContainers` | Shows formatted table |
| `TestList_MixedStatus` | Shows running and stopped |
| `TestList_ContainerNotInLXC` | Shows NOT FOUND status |
| `TestList_CustomPorts` | Shows custom ports |

### cmd/snapshot

| Test | Description |
|------|-------------|
| `TestSnapshot_Success` | Creates image from container |
| `TestSnapshot_StopsAndRestarts` | Stops running container, restarts after |
| `TestSnapshot_AlreadyStopped` | Works without restart |
| `TestSnapshot_NotExists` | Error when container doesn't exist |
| `TestSnapshot_PublishFails` | Handles publish failure |
| `TestSnapshot_MissingArgs` | Error with usage hint |

### cmd/remove

| Test | Description |
|------|-------------|
| `TestRemove_Success` | Deletes container and config |
| `TestRemove_OnlyInConfig` | Removes from config when not in LXC |
| `TestRemove_OnlyInLXC` | Deletes from LXC when not in config |
| `TestRemove_NotExists` | No error when doesn't exist |
| `TestRemove_DeleteFails` | Handles delete failure |
| `TestRemove_MissingArgs` | Error with usage hint |

### cmd/proxy

| Test | Description |
|------|-------------|
| `TestProxy_Success` | Starts proxies for all ports |
| `TestProxy_NotExists` | Error when container doesn't exist |
| `TestProxy_NotRunning` | Error when container stopped |
| `TestProxy_NoIP` | Error when no IP available |
| `TestProxy_NoPorts` | Error when no ports configured |
| `TestProxy_PortInUse` | Error when port already bound |
| `TestProxy_CustomPorts` | Uses container-specific ports |
| `TestProxy_MissingArgs` | Error with usage hint |

---

## End-to-End Tests (Real LXC)

Require actual LXC/LXD installation. Tagged with `//go:build e2e`.

| Test | Description |
|------|-------------|
| `TestE2E_FullWorkflow` | create → up → down → remove |
| `TestE2E_CreateWithUbuntu` | Creates Ubuntu 24.04 container |
| `TestE2E_DevUserExists` | Verifies dev user created |
| `TestE2E_SSHWorks` | SSH connection succeeds |
| `TestE2E_NestingEnabled` | Docker can run inside |
| `TestE2E_Snapshot` | Create and use custom image |
| `TestE2E_ProxyForwarding` | Proxy actually forwards traffic |
| `TestE2E_ProxyWebSocket` | WebSocket works through proxy |

---

## Test File Structure

```
lxc-dev-manager/
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── lxc/
│   │   ├── lxc.go
│   │   ├── lxc_test.go
│   │   └── mock.go          # Mock LXC executor
│   └── proxy/
│       ├── proxy.go
│       └── proxy_test.go
├── cmd/
│   ├── create.go
│   ├── create_test.go
│   ├── up.go
│   ├── up_test.go
│   ├── down.go
│   ├── down_test.go
│   ├── list.go
│   ├── list_test.go
│   ├── snapshot.go
│   ├── snapshot_test.go
│   ├── remove.go
│   ├── remove_test.go
│   ├── proxy.go
│   └── proxy_test.go
└── e2e/
    └── e2e_test.go          # //go:build e2e
```

---

## Mocking Strategy

### LXC Command Mocking

Create an interface for command execution:

```go
// internal/lxc/executor.go
type Executor interface {
    Run(name string, args ...string) ([]byte, error)
}

type RealExecutor struct{}
type MockExecutor struct {
    Responses map[string]MockResponse
}
```

Commands use `DefaultExecutor` which can be swapped in tests.

### Config File Mocking

Tests use temp directories:

```go
func TestSomething(t *testing.T) {
    dir := t.TempDir()
    oldDir, _ := os.Getwd()
    os.Chdir(dir)
    defer os.Chdir(oldDir)

    // Test runs in isolated directory
}
```

### Proxy Testing

Use local TCP servers:

```go
func TestProxy_ForwardsData(t *testing.T) {
    // Start mock server on random port
    server := startMockServer(t)

    // Start proxy: localPort → server.Port
    proxy := New(0, "127.0.0.1", server.Port)

    // Connect to proxy, verify data flows
}
```

---

## Implementation Order

1. [ ] Add mock infrastructure (executor interface)
2. [ ] `internal/config/config_test.go` - 14 tests
3. [ ] `internal/proxy/proxy_test.go` - 11 tests
4. [ ] `internal/lxc/lxc_test.go` - 7 tests
5. [ ] `cmd/create_test.go` - 9 tests
6. [ ] `cmd/up_test.go` - 5 tests
7. [ ] `cmd/down_test.go` - 5 tests
8. [ ] `cmd/list_test.go` - 5 tests
9. [ ] `cmd/snapshot_test.go` - 6 tests
10. [ ] `cmd/remove_test.go` - 6 tests
11. [ ] `cmd/proxy_test.go` - 8 tests
12. [ ] `e2e/e2e_test.go` - 8 tests

**Total: 84 tests**

---

## Running Tests

```bash
# Unit + Integration tests
go test ./...

# With coverage
go test -cover ./...

# Verbose
go test -v ./...

# E2E tests (requires LXC)
go test -tags=e2e ./e2e/

# Specific package
go test -v ./internal/config/
```
