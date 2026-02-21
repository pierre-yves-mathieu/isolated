# Issue #9: Race Condition on Config File Access

## Severity: Medium
## Category: Concurrency / Data Integrity

---

## Implementation Status: DONE

### Implementation Complete
Changes made to fix the race condition:

1. **Atomic Write** (`internal/config/config.go`):
   - Added `atomicWriteFile()` function using temp file + sync + rename pattern
   - Updated `Save()` to use atomic writes

2. **File Locking** (`internal/config/config.go`):
   - Added `ConfigLock` type with exclusive file locking
   - Added `AcquireLock()` with timeout (5 seconds)
   - Added `Release()` method
   - Added `LoadWithLock()` for atomic load + lock

3. **Helper Functions** (`cmd/helpers.go`):
   - Added `requireProjectWithLock()` - loads config with lock held
   - Added `requireContainerWithLock()` - loads config and validates container with lock held

4. **Updated Commands** to use locking:
   - `cmd/container.go`: `runContainerCreate`, `runContainerClone`
   - `cmd/container_snapshot.go`: `runSnapshotCreate`, `runSnapshotDelete`
   - `cmd/remove.go`: `runRemove`

### Original Implementation Plan

1. **Atomic Write** - Update `Save()` to use temp file + rename pattern:
   ```go
   func (c *Config) Save() error {
       data, err := yaml.Marshal(c)
       if err != nil {
           return err
       }
       return atomicWriteFile(ConfigFile, data, 0644)
   }
   ```

2. **Add atomicWriteFile function**:
   - Write to temp file in same directory
   - Sync to disk
   - Rename (atomic on POSIX)
   - Cleanup temp on error

3. **Add file locking** - Create `ConfigLock` type:
   ```go
   type ConfigLock struct {
       file *os.File
   }

   func AcquireLock() (*ConfigLock, error)
   func (l *ConfigLock) Release() error
   func LoadWithLock() (*Config, *ConfigLock, error)
   ```

4. **Update commands** that do Load→Modify→Save:
   - `cmd/container.go` - runContainerCreate, runContainerClone
   - `cmd/container_snapshot.go` - runSnapshotCreate, runSnapshotDelete
   - `cmd/remove.go` - runRemove

### Files to Modify
- `internal/config/config.go` - Add atomic write + locking
- `cmd/container.go` - Use LoadWithLock for create/clone
- `cmd/container_snapshot.go` - Use LoadWithLock
- `cmd/remove.go` - Use LoadWithLock

---

## Problem Summary

The `containers.yaml` configuration file can be corrupted when multiple instances of the tool run simultaneously, as there is no file locking or atomic write mechanism in place.

---

## Affected Code

**File:** `internal/config/config.go`

```go
func (c *Config) Save() error {
    data, err := yaml.Marshal(c)
    if err != nil {
        return err
    }
    return os.WriteFile(ConfigFile, data, 0644)  // <-- NOT ATOMIC
}
```

**Multiple places that Load → Modify → Save:**

```go
// cmd/container.go:157-160
cfg.AddContainer(name, image)
if err := cfg.Save(); err != nil {
    return fmt.Errorf("failed to save config: %w", err)
}

// cmd/container.go:167-170
cfg.AddSnapshot(name, "initial-state", "Initial state after setup")
cfg.Save()  // Second save shortly after first
```

---

## Why This Is a Problem

### Scenario: Two Terminals Creating Containers

```
Terminal 1                          Terminal 2
----------                          ----------
t=0:  Load config
      containers: {dev1}
                                    t=1:  Load config
                                          containers: {dev1}
t=2:  Add dev2
      containers: {dev1, dev2}
                                    t=3:  Add dev3
                                          containers: {dev1, dev3}
t=4:  Save config
      File: {dev1, dev2}
                                    t=5:  Save config
                                          File: {dev1, dev3}  <-- dev2 LOST!
```

**Result:** Container `dev2` was created in LXC but lost from the config file.

### Scenario: Partial Write

If the process is killed during `os.WriteFile`:

```
t=0:  WriteFile starts
      File content: "project: test\ncontain"  <-- TRUNCATED
t=1:  Process killed (Ctrl+C, power loss, etc.)
t=2:  File is corrupted, tool won't start
```

### Scenario: Concurrent Snapshot Operations

```go
// User runs two commands quickly:
// Terminal 1: lxc-dev-manager container snapshot create dev1 snap1
// Terminal 2: lxc-dev-manager container snapshot create dev1 snap2

// Both load the same config, both add a snapshot, one overwrites the other
```

---

## Demonstration

### Reproducing the Race Condition

```bash
#!/bin/bash
# race_test.sh

cd /tmp
mkdir race-test && cd race-test

# Initialize project
lxc-dev-manager create --name racetest

# Run two container creates simultaneously
lxc-dev-manager container create dev1 ubuntu:24.04 &
lxc-dev-manager container create dev2 ubuntu:24.04 &
wait

# Check config - one container may be missing
cat containers.yaml
```

### Reproducing Corruption

```bash
#!/bin/bash
# corruption_test.sh

# Start a long-running save operation
(
    for i in $(seq 1 1000); do
        lxc-dev-manager container snapshot create dev1 "snap$i" -d "test" 2>/dev/null
    done
) &
PID=$!

# Kill it mid-operation
sleep 0.5
kill -9 $PID

# Check for corruption
cat containers.yaml
```

---

## Recommended Fixes

### Option A: File Locking (Cross-Process Safety)

```go
// internal/config/config.go
package config

import (
    "os"
    "syscall"
)

// lockFile represents a locked config file
type lockFile struct {
    file *os.File
}

// acquireLock gets an exclusive lock on the config file
func acquireLock() (*lockFile, error) {
    // Open or create lock file
    f, err := os.OpenFile(ConfigFile+".lock", os.O_CREATE|os.O_RDWR, 0644)
    if err != nil {
        return nil, fmt.Errorf("failed to open lock file: %w", err)
    }

    // Acquire exclusive lock (blocks until available)
    if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
        f.Close()
        return nil, fmt.Errorf("failed to acquire lock: %w", err)
    }

    return &lockFile{file: f}, nil
}

// release releases the lock
func (l *lockFile) release() error {
    if l.file == nil {
        return nil
    }
    syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
    return l.file.Close()
}

// LoadWithLock loads config with exclusive lock held
func LoadWithLock() (*Config, *lockFile, error) {
    lock, err := acquireLock()
    if err != nil {
        return nil, nil, err
    }

    cfg, err := Load()
    if err != nil {
        lock.release()
        return nil, nil, err
    }

    return cfg, lock, nil
}

// Example usage in commands:
func runContainerCreate(cmd *cobra.Command, args []string) error {
    cfg, lock, err := config.LoadWithLock()
    if err != nil {
        return err
    }
    defer lock.release()  // Always release lock

    // ... modify config ...

    return cfg.Save()
}
```

### Option B: Atomic Write (Corruption Safety)

```go
// internal/config/config.go
package config

import (
    "io"
    "os"
    "path/filepath"
)

func (c *Config) Save() error {
    data, err := yaml.Marshal(c)
    if err != nil {
        return err
    }

    return atomicWriteFile(ConfigFile, data, 0644)
}

// atomicWriteFile writes data to a file atomically
// It writes to a temp file first, then renames (atomic on POSIX)
func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
    // Get directory for temp file (same filesystem required for atomic rename)
    dir := filepath.Dir(filename)

    // Create temp file in same directory
    tmp, err := os.CreateTemp(dir, ".containers.yaml.tmp.*")
    if err != nil {
        return fmt.Errorf("failed to create temp file: %w", err)
    }
    tmpName := tmp.Name()

    // Clean up temp file on any error
    success := false
    defer func() {
        if !success {
            os.Remove(tmpName)
        }
    }()

    // Write data
    if _, err := tmp.Write(data); err != nil {
        tmp.Close()
        return fmt.Errorf("failed to write temp file: %w", err)
    }

    // Sync to disk (ensure data is persisted before rename)
    if err := tmp.Sync(); err != nil {
        tmp.Close()
        return fmt.Errorf("failed to sync temp file: %w", err)
    }

    // Close before rename
    if err := tmp.Close(); err != nil {
        return fmt.Errorf("failed to close temp file: %w", err)
    }

    // Set permissions
    if err := os.Chmod(tmpName, perm); err != nil {
        return fmt.Errorf("failed to set permissions: %w", err)
    }

    // Atomic rename
    if err := os.Rename(tmpName, filename); err != nil {
        return fmt.Errorf("failed to rename temp file: %w", err)
    }

    success = true
    return nil
}
```

### Option C: Combined Locking + Atomic Write (Full Solution)

```go
// internal/config/config.go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "syscall"
    "time"
)

const lockTimeout = 5 * time.Second

type ConfigManager struct {
    lockFile *os.File
    config   *Config
}

// Open loads config with exclusive lock
func Open() (*ConfigManager, error) {
    // Acquire lock with timeout
    lockPath := ConfigFile + ".lock"
    f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
    if err != nil {
        return nil, fmt.Errorf("failed to open lock file: %w", err)
    }

    // Try to acquire lock with timeout
    deadline := time.Now().Add(lockTimeout)
    for {
        err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
        if err == nil {
            break // Lock acquired
        }
        if time.Now().After(deadline) {
            f.Close()
            return nil, fmt.Errorf("timeout waiting for config lock (another instance may be running)")
        }
        time.Sleep(100 * time.Millisecond)
    }

    // Load config while holding lock
    cfg, err := Load()
    if err != nil {
        syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
        f.Close()
        return nil, err
    }

    // Handle nil config (no project)
    if cfg == nil {
        cfg = &Config{Containers: make(map[string]Container)}
    }

    return &ConfigManager{
        lockFile: f,
        config:   cfg,
    }, nil
}

// Config returns the loaded config
func (m *ConfigManager) Config() *Config {
    return m.config
}

// Save saves config atomically while holding lock
func (m *ConfigManager) Save() error {
    return m.config.saveAtomic()
}

// Close releases the lock
func (m *ConfigManager) Close() error {
    if m.lockFile == nil {
        return nil
    }
    syscall.Flock(int(m.lockFile.Fd()), syscall.LOCK_UN)
    err := m.lockFile.Close()
    m.lockFile = nil
    return err
}

func (c *Config) saveAtomic() error {
    data, err := yaml.Marshal(c)
    if err != nil {
        return err
    }

    dir := filepath.Dir(ConfigFile)
    tmp, err := os.CreateTemp(dir, ".containers.yaml.tmp.*")
    if err != nil {
        return fmt.Errorf("failed to create temp file: %w", err)
    }
    tmpName := tmp.Name()

    defer func() {
        if tmp != nil {
            tmp.Close()
            os.Remove(tmpName)
        }
    }()

    if _, err := tmp.Write(data); err != nil {
        return fmt.Errorf("failed to write: %w", err)
    }
    if err := tmp.Sync(); err != nil {
        return fmt.Errorf("failed to sync: %w", err)
    }
    if err := tmp.Close(); err != nil {
        return fmt.Errorf("failed to close: %w", err)
    }
    tmp = nil // Prevent deferred cleanup

    if err := os.Chmod(tmpName, 0600); err != nil {
        os.Remove(tmpName)
        return fmt.Errorf("failed to chmod: %w", err)
    }
    if err := os.Rename(tmpName, ConfigFile); err != nil {
        os.Remove(tmpName)
        return fmt.Errorf("failed to rename: %w", err)
    }

    return nil
}
```

**Usage in commands:**

```go
func runContainerCreate(cmd *cobra.Command, args []string) error {
    name := args[0]
    image := args[1]

    // Open with lock
    mgr, err := config.Open()
    if err != nil {
        return err
    }
    defer mgr.Close()

    cfg := mgr.Config()

    // ... do work ...

    cfg.AddContainer(name, image)
    return mgr.Save()
}
```

---

## Testing Race Conditions

```go
// internal/config/config_race_test.go
package config

import (
    "sync"
    "testing"
)

func TestConcurrentSave(t *testing.T) {
    dir := t.TempDir()
    // ... setup ...

    var wg sync.WaitGroup
    errors := make(chan error, 10)

    // Simulate 10 concurrent saves
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()

            mgr, err := Open()
            if err != nil {
                errors <- err
                return
            }
            defer mgr.Close()

            cfg := mgr.Config()
            cfg.AddContainer(fmt.Sprintf("container%d", n), "ubuntu:24.04")

            if err := mgr.Save(); err != nil {
                errors <- err
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    for err := range errors {
        t.Errorf("concurrent save error: %v", err)
    }

    // Verify all containers were saved
    cfg, _ := Load()
    if len(cfg.Containers) != 10 {
        t.Errorf("expected 10 containers, got %d", len(cfg.Containers))
    }
}
```

---

## Platform Considerations

### Linux/macOS
`flock()` and atomic rename work as expected.

### Windows
- `flock()` is not available; use `LockFileEx` instead
- Rename may fail if file is open by another process
- Consider using a cross-platform library like `github.com/gofrs/flock`

```go
// Cross-platform locking
import "github.com/gofrs/flock"

func acquireLock() (*flock.Flock, error) {
    lock := flock.New(ConfigFile + ".lock")

    locked, err := lock.TryLock()
    if err != nil {
        return nil, err
    }
    if !locked {
        return nil, fmt.Errorf("config is locked by another process")
    }

    return lock, nil
}
```

---

## Alternative: Single-Instance Enforcement

Instead of file locking, prevent multiple instances entirely:

```go
// cmd/root.go
func init() {
    // Create PID file
    pidFile := filepath.Join(os.TempDir(), "lxc-dev-manager.pid")

    // Check if already running
    if data, err := os.ReadFile(pidFile); err == nil {
        pid, _ := strconv.Atoi(string(data))
        if process, err := os.FindProcess(pid); err == nil {
            if err := process.Signal(syscall.Signal(0)); err == nil {
                fmt.Fprintln(os.Stderr, "Another instance is already running")
                os.Exit(1)
            }
        }
    }

    // Write our PID
    os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
}
```

---

## Summary of Solutions

| Solution | Corruption Safe | Race Safe | Complexity |
|----------|-----------------|-----------|------------|
| Current (none) | No | No | None |
| Atomic write only | Yes | No | Low |
| File locking only | No | Yes | Medium |
| Atomic + Locking | Yes | Yes | Medium |
| Single instance | Yes | Yes | Low |

**Recommendation:** Implement **Atomic + Locking** for the most robust solution.

---

## References

- [Atomic File Operations](https://lwn.net/Articles/457667/)
- [File Locking in Go](https://pkg.go.dev/syscall#Flock)
- [gofrs/flock - Cross-platform locking](https://github.com/gofrs/flock)
- [Write-Ahead Logging](https://en.wikipedia.org/wiki/Write-ahead_logging) (for more complex scenarios)
