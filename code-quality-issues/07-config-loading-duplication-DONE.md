# Issue #7: Repeated Config Loading Pattern (Code Duplication)

## Severity: Medium
## Category: Code Quality / DRY Principle

---

## Problem Summary

Nearly every command in the `cmd/` package contains an identical config loading and validation block. This violates the DRY (Don't Repeat Yourself) principle and increases maintenance burden.

---

## Affected Code

The following pattern appears **11+ times** across the codebase:

```go
// Load config (project must exist)
cfg, err := config.Load()
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}
if cfg == nil {
    return fmt.Errorf("no project found. Run 'lxc-dev-manager create' first to initialize a project")
}
```

### Files Containing This Pattern

| File | Function | Lines |
|------|----------|-------|
| `cmd/container.go` | `runContainerCreate` | 95-101 |
| `cmd/container.go` | `runContainerReset` | 189-196 |
| `cmd/container.go` | `runContainerClone` | 265-272 |
| `cmd/proxy.go` | `runProxy` | 44-50 |
| `cmd/ssh.go` | `runSSH` | 59-65 |
| `cmd/up.go` | `runUp` | ~40-46 |
| `cmd/down.go` | `runDown` | ~40-46 |
| `cmd/list.go` | `runList` | ~35-41 |
| `cmd/remove.go` | `runRemove` | ~45-51 |
| `cmd/container_snapshot.go` | multiple functions | various |
| `cmd/image_create.go` | `runImageCreate` | ~45-51 |

---

## Why This Is a Problem

### 1. Maintenance Burden

If the error message needs to change, or the loading logic needs modification, you must update **11+ locations**.

Example: If you want to add a suggestion for `lxc-dev-manager project create` instead of just `create`:
```go
// Must change in 11+ places:
return fmt.Errorf("no project found. Run 'lxc-dev-manager project create' first")
```

### 2. Inconsistency Risk

Different commands might drift and have slightly different error messages:
```go
// In one file:
"no project found. Run 'lxc-dev-manager create' first to initialize a project"

// In another file (hypothetical drift):
"no project found in current directory"

// In yet another:
"project not initialized"
```

### 3. Testing Complexity

Each command needs separate tests for the "no project" case, when they could share a single well-tested helper.

### 4. Violation of Single Responsibility

Each command function is responsible for both:
1. Config validation (repeated)
2. Command-specific logic (unique)

---

## Extended Pattern Analysis

Beyond just loading, there's also repeated validation:

```go
// Pattern 2: Container existence check (repeated ~8 times)
if !cfg.HasContainer(name) {
    return fmt.Errorf("container '%s' not found in config", name)
}
lxcName := cfg.GetLXCName(name)
if !lxc.Exists(lxcName) {
    return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
}

// Pattern 3: Running check (repeated ~5 times)
status, err := lxc.GetStatus(lxcName)
if err != nil {
    return err
}
if status != "RUNNING" {
    return fmt.Errorf("container '%s' is not running (status: %s)", name, status)
}
```

---

## Recommended Fix

### Option A: Helper Functions (Minimal Change)

Create a new file `cmd/helpers.go`:

```go
package cmd

import (
    "fmt"

    "lxc-dev-manager/internal/config"
    "lxc-dev-manager/internal/lxc"
)

// requireProject loads config and ensures a project exists
func requireProject() (*config.Config, error) {
    cfg, err := config.Load()
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }
    if cfg == nil {
        return nil, fmt.Errorf("no project found. Run 'lxc-dev-manager project create' first to initialize a project")
    }
    return cfg, nil
}

// requireContainer ensures a container exists in both config and LXC
// Returns the config, LXC name, and any error
func requireContainer(name string) (*config.Config, string, error) {
    cfg, err := requireProject()
    if err != nil {
        return nil, "", err
    }

    if !cfg.HasContainer(name) {
        return nil, "", fmt.Errorf("container '%s' not found in project config", name)
    }

    lxcName := cfg.GetLXCName(name)
    if !lxc.Exists(lxcName) {
        return nil, "", fmt.Errorf("container '%s' does not exist in LXC (expected: %s)", name, lxcName)
    }

    return cfg, lxcName, nil
}

// requireRunningContainer ensures a container exists and is running
func requireRunningContainer(name string) (*config.Config, string, error) {
    cfg, lxcName, err := requireContainer(name)
    if err != nil {
        return nil, "", err
    }

    status, err := lxc.GetStatus(lxcName)
    if err != nil {
        return nil, "", fmt.Errorf("failed to get container status: %w", err)
    }
    if status != "RUNNING" {
        return nil, "", fmt.Errorf("container '%s' is not running (status: %s). Start it with: lxc-dev-manager up %s", name, status, name)
    }

    return cfg, lxcName, nil
}

// requireStoppedContainer ensures a container exists and is stopped
func requireStoppedContainer(name string) (*config.Config, string, error) {
    cfg, lxcName, err := requireContainer(name)
    if err != nil {
        return nil, "", err
    }

    status, err := lxc.GetStatus(lxcName)
    if err != nil {
        return nil, "", fmt.Errorf("failed to get container status: %w", err)
    }
    if status == "RUNNING" {
        return nil, "", fmt.Errorf("container '%s' is running. Stop it first with: lxc-dev-manager down %s", name, name)
    }

    return cfg, lxcName, nil
}
```

### Refactored Command Example

**Before:**
```go
func runProxy(cmd *cobra.Command, args []string) error {
    name := args[0]

    // Load config (project must exist)
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }
    if cfg == nil {
        return fmt.Errorf("no project found. Run 'lxc-dev-manager create' first to initialize a project")
    }

    // Get full LXC name with prefix
    lxcName := cfg.GetLXCName(name)

    // Check if container exists
    if !lxc.Exists(lxcName) {
        return fmt.Errorf("container '%s' does not exist", name)
    }

    // Check if running
    status, err := lxc.GetStatus(lxcName)
    if err != nil {
        return err
    }
    if status != "RUNNING" {
        return fmt.Errorf("container '%s' is not running (status: %s)", name, status)
    }

    // ... rest of function
}
```

**After:**
```go
func runProxy(cmd *cobra.Command, args []string) error {
    name := args[0]

    cfg, lxcName, err := requireRunningContainer(name)
    if err != nil {
        return err
    }

    // ... rest of function (now just the proxy-specific logic)
}
```

**Lines of code:** 25 → 8 (68% reduction in boilerplate)

---

### Option B: Cobra Middleware Pattern (More Advanced)

Use Cobra's `PersistentPreRunE` for common setup:

```go
// cmd/root.go
var projectConfig *config.Config

var rootCmd = &cobra.Command{
    Use:   "lxc-dev-manager",
    Short: "Manage LXC containers for local development",
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // Skip for commands that don't need a project
        if cmd.Name() == "create" || cmd.Name() == "help" || cmd.Name() == "version" {
            return nil
        }

        cfg, err := config.Load()
        if err != nil {
            return fmt.Errorf("failed to load config: %w", err)
        }
        if cfg == nil {
            return fmt.Errorf("no project found. Run 'lxc-dev-manager project create' first")
        }
        projectConfig = cfg
        return nil
    },
}

// In commands, just use the global:
func runProxy(cmd *cobra.Command, args []string) error {
    name := args[0]
    cfg := projectConfig // Already loaded and validated

    // ... rest of function
}
```

**Pros:** Single place for project requirement logic
**Cons:** Global state, harder to test, implicit dependencies

---

### Option C: Context-Based Injection (Best Practice)

```go
// cmd/context.go
type cmdContext struct {
    config  *config.Config
    lxcName string
}

func withProject(fn func(*cobra.Command, []string, *config.Config) error) func(*cobra.Command, []string) error {
    return func(cmd *cobra.Command, args []string) error {
        cfg, err := requireProject()
        if err != nil {
            return err
        }
        return fn(cmd, args, cfg)
    }
}

func withContainer(fn func(*cobra.Command, []string, *config.Config, string) error) func(*cobra.Command, []string) error {
    return func(cmd *cobra.Command, args []string) error {
        if len(args) < 1 {
            return fmt.Errorf("container name required")
        }
        cfg, lxcName, err := requireContainer(args[0])
        if err != nil {
            return err
        }
        return fn(cmd, args, cfg, lxcName)
    }
}

// Usage:
var proxyCmd = &cobra.Command{
    Use:   "proxy <name>",
    Args:  cobra.ExactArgs(1),
    RunE:  withRunningContainer(runProxyImpl),
}

func runProxyImpl(cmd *cobra.Command, args []string, cfg *config.Config, lxcName string) error {
    // Direct access to validated config and lxcName
    // No boilerplate needed
}
```

---

## Testing the Helpers

```go
// cmd/helpers_test.go
package cmd

import (
    "testing"
    "lxc-dev-manager/internal/lxc"
)

func TestRequireProject_NoConfig(t *testing.T) {
    env := setupTestEnv(t)
    // Don't create config file

    _, err := requireProject()
    if err == nil {
        t.Error("expected error for missing project")
    }
    if !strings.Contains(err.Error(), "no project found") {
        t.Errorf("unexpected error: %v", err)
    }
}

func TestRequireContainer_NotInConfig(t *testing.T) {
    env := setupTestEnv(t)
    env.writeConfig(`project: test
containers: {}
`)

    _, _, err := requireContainer("nonexistent")
    if err == nil {
        t.Error("expected error")
    }
    if !strings.Contains(err.Error(), "not found in project config") {
        t.Errorf("unexpected error: %v", err)
    }
}

func TestRequireRunningContainer_Stopped(t *testing.T) {
    env := setupTestEnv(t)
    env.writeConfig(`project: test
containers:
  dev1:
    image: ubuntu:24.04
`)
    env.setContainerExists("test-dev1", false) // exists but stopped

    _, _, err := requireRunningContainer("dev1")
    if err == nil {
        t.Error("expected error")
    }
    if !strings.Contains(err.Error(), "not running") {
        t.Errorf("unexpected error: %v", err)
    }
}
```

---

## Migration Strategy

1. **Add helpers.go** with the new functions
2. **Add tests** for the helpers
3. **Refactor one command** (e.g., `proxy.go`) as proof of concept
4. **Refactor remaining commands** incrementally
5. **Remove duplicated code** from each command

This can be done incrementally without breaking anything.

---

## Metrics

| Metric | Before | After |
|--------|--------|-------|
| Duplicated blocks | 11+ | 0 |
| Lines per command (avg) | 50-80 | 30-50 |
| Places to update for error message change | 11+ | 1 |
| Test cases for "no project" scenario | 11+ | 1 |

---

## References

- [DRY Principle](https://en.wikipedia.org/wiki/Don%27t_repeat_yourself)
- [Cobra Middleware](https://github.com/spf13/cobra/blob/main/user_guide.md#prerun-and-postrun-hooks)
- [Go Error Handling Best Practices](https://go.dev/blog/error-handling-and-go)
