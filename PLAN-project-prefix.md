# Project Prefix Feature - Implementation Plan

## Overview

Add project-based namespacing to lxc-dev-manager so containers are prefixed with project names, enabling safe multi-project usage without name collisions.

## Feature Requirements

1. **Project initialization**: `lxc-dev-manager create` initializes a project
   - Saves project name from folder name (default) or optional `--name` argument
   - Creates `containers.yaml` with `project` field
   - `create` becomes alias for `project create`

2. **Container prefixing**: All container names are prefixed with `<project>-`
   - Example: project `webapp` + container `dev1` → LXC name `webapp-dev1`
   - Display shows short names, LXC uses full prefixed names

3. **Project delete**: `project delete` command removes all containers and config
   - Lists all containers to be deleted
   - Prompts for confirmation (with `--force` to skip)
   - Deletes all LXC containers and removes `containers.yaml`

---

## Config Changes

### Current containers.yaml
```yaml
defaults:
    ports:
        - 5173
        - 8000
        - 5432
containers:
    dev1:
        image: ubuntu:24.04
```

### New containers.yaml
```yaml
project: webapp          # NEW: Required field set on project create
defaults:
    ports:
        - 5173
        - 8000
        - 5432
containers:
    dev1:                # Short name (display)
        image: ubuntu:24.04
        # LXC name: webapp-dev1 (computed)
```

---

## Implementation Tasks

### Phase 1: Config Layer Changes

#### 1.1 Update Config Struct (`internal/config/config.go`)

```go
type Config struct {
    Project    string               `yaml:"project"`           // NEW
    Defaults   Defaults             `yaml:"defaults"`
    Containers map[string]Container `yaml:"containers"`
}
```

#### 1.2 Add Helper Methods (`internal/config/config.go`)

```go
// GetLXCName returns the full LXC container name with project prefix
func (c *Config) GetLXCName(shortName string) string {
    if c.Project == "" {
        return shortName
    }
    return c.Project + "-" + shortName
}

// GetShortName extracts short name from LXC name by stripping project prefix
func (c *Config) GetShortName(lxcName string) string {
    if c.Project == "" {
        return lxcName
    }
    prefix := c.Project + "-"
    if strings.HasPrefix(lxcName, prefix) {
        return strings.TrimPrefix(lxcName, prefix)
    }
    return lxcName
}

// HasProject returns true if project is initialized
func (c *Config) HasProject() bool {
    return c.Project != ""
}

// GetProjectFromFolder returns the current directory name
func GetProjectFromFolder() (string, error) {
    cwd, err := os.Getwd()
    if err != nil {
        return "", err
    }
    return filepath.Base(cwd), nil
}
```

#### 1.3 Update Default Config

Change `Load()` to NOT create defaults when file doesn't exist. Project must be explicitly created.

```go
func Load() (*Config, error) {
    data, err := os.ReadFile(ConfigFile)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil  // No config = no project
        }
        return nil, err
    }
    // ... parse yaml
}
```

---

### Phase 2: New Project Commands

#### 2.1 Create Project Command (`cmd/project.go`)

**Command Structure:**
```
lxc-dev-manager project create [--name <project-name>]
lxc-dev-manager project delete [--force]
lxc-dev-manager create  # Alias for project create
```

**project create Implementation:**

```go
var projectCmd = &cobra.Command{
    Use:   "project",
    Short: "Manage lxc-dev-manager projects",
}

var projectCreateCmd = &cobra.Command{
    Use:   "create",
    Short: "Initialize a new project in the current directory",
    Long: `Creates a containers.yaml file with the project name.

The project name defaults to the current folder name, or can be
specified with --name. All containers will be prefixed with
the project name in LXC.`,
    RunE: runProjectCreate,
}

var projectNameFlag string

func init() {
    rootCmd.AddCommand(projectCmd)
    projectCmd.AddCommand(projectCreateCmd)
    projectCreateCmd.Flags().StringVarP(&projectNameFlag, "name", "n", "", "Project name (defaults to folder name)")

    // Make 'create' an alias for 'project create' at root level
    rootCmd.AddCommand(&cobra.Command{
        Use:   "create",
        Short: "Initialize a new project (alias for 'project create')",
        RunE:  runProjectCreate,
    })
}

func runProjectCreate(cmd *cobra.Command, args []string) error {
    // Check if project already exists
    cfg, err := config.Load()
    if err != nil {
        return err
    }
    if cfg != nil {
        return fmt.Errorf("project already exists: %s\nUse 'project delete' first to remove it", cfg.Project)
    }

    // Determine project name
    projectName := projectNameFlag
    if projectName == "" {
        projectName, err = config.GetProjectFromFolder()
        if err != nil {
            return fmt.Errorf("failed to get folder name: %w", err)
        }
    }

    // Validate project name (alphanumeric, hyphens, underscores)
    if !isValidProjectName(projectName) {
        return fmt.Errorf("invalid project name %q: must contain only letters, numbers, hyphens, and underscores", projectName)
    }

    // Create config
    cfg = &config.Config{
        Project: projectName,
        Defaults: config.Defaults{
            Ports: []int{5173, 8000, 5432},
        },
        Containers: make(map[string]config.Container),
    }

    if err := cfg.Save(); err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }

    fmt.Printf("✓ Project '%s' created\n", projectName)
    fmt.Printf("  Config: %s\n", config.ConfigFile)
    fmt.Printf("\nNext steps:\n")
    fmt.Printf("  lxc-dev-manager container create dev1 ubuntu:24.04\n")

    return nil
}
```

#### 2.2 Project Delete Command

```go
var projectDeleteCmd = &cobra.Command{
    Use:   "delete",
    Short: "Delete the project and all its containers",
    Long: `Deletes all containers belonging to this project and removes
the containers.yaml file. This action is destructive and irreversible.`,
    RunE: runProjectDelete,
}

var projectDeleteForce bool

func init() {
    projectCmd.AddCommand(projectDeleteCmd)
    projectDeleteCmd.Flags().BoolVarP(&projectDeleteForce, "force", "f", false, "Skip confirmation prompt")
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }
    if cfg == nil {
        return fmt.Errorf("no project found in current directory")
    }

    // List containers to be deleted
    fmt.Printf("Project: %s\n", cfg.Project)
    fmt.Printf("Config:  %s\n\n", config.ConfigFile)

    if len(cfg.Containers) > 0 {
        fmt.Println("Containers to be deleted:")
        for name := range cfg.Containers {
            lxcName := cfg.GetLXCName(name)
            status, _ := lxc.GetStatus(lxcName)
            fmt.Printf("  - %s (%s) [%s]\n", name, lxcName, status)
        }
        fmt.Println()
    } else {
        fmt.Println("No containers defined.\n")
    }

    // Confirm deletion
    if !projectDeleteForce {
        fmt.Print("Are you sure you want to delete this project? [y/N]: ")
        var response string
        fmt.Scanln(&response)
        if response != "y" && response != "Y" {
            fmt.Println("Cancelled.")
            return nil
        }
    }

    // Delete all containers
    var deleteErrors []string
    for name := range cfg.Containers {
        lxcName := cfg.GetLXCName(name)
        fmt.Printf("Deleting container '%s'... ", name)

        if exists, _ := lxc.Exists(lxcName); exists {
            if err := lxc.Delete(lxcName); err != nil {
                fmt.Printf("FAILED: %v\n", err)
                deleteErrors = append(deleteErrors, fmt.Sprintf("%s: %v", name, err))
                continue
            }
        }
        fmt.Println("✓")
    }

    // Remove config file
    fmt.Printf("Removing %s... ", config.ConfigFile)
    if err := os.Remove(config.ConfigFile); err != nil {
        return fmt.Errorf("failed to remove config: %w", err)
    }
    fmt.Println("✓")

    if len(deleteErrors) > 0 {
        fmt.Printf("\nWarning: Some containers failed to delete:\n")
        for _, e := range deleteErrors {
            fmt.Printf("  - %s\n", e)
        }
    }

    fmt.Printf("\n✓ Project '%s' deleted\n", cfg.Project)
    return nil
}
```

---

### Phase 3: Rename Container Commands

#### 3.1 Move Current create.go → container.go

Restructure commands under `container` subcommand:

```
lxc-dev-manager container create <name> <image>
lxc-dev-manager container remove <name> [--force]
lxc-dev-manager container list
lxc-dev-manager container up <name>
lxc-dev-manager container down <name>
lxc-dev-manager container ssh <name>
lxc-dev-manager container proxy <name>
```

**Keep aliases at root for backwards compatibility:**
```
lxc-dev-manager up <name>      # alias for container up
lxc-dev-manager down <name>    # alias for container down
lxc-dev-manager list           # alias for container list
lxc-dev-manager ssh <name>     # alias for container ssh
lxc-dev-manager proxy <name>   # alias for container proxy
lxc-dev-manager remove <name>  # alias for container remove
```

#### 3.2 Update Container Create (`cmd/container.go`)

```go
var containerCmd = &cobra.Command{
    Use:   "container",
    Short: "Manage containers within the project",
    Aliases: []string{"c"},
}

var containerCreateCmd = &cobra.Command{
    Use:   "create <name> <image>",
    Short: "Create a new container in the current project",
    Args:  cobra.ExactArgs(2),
    RunE:  runContainerCreate,
}

func runContainerCreate(cmd *cobra.Command, args []string) error {
    name := args[0]
    image := args[1]

    // Load config (project must exist)
    cfg, err := config.Load()
    if err != nil {
        return err
    }
    if cfg == nil {
        return fmt.Errorf("no project found. Run 'lxc-dev-manager create' first to initialize a project")
    }

    // Check short name doesn't exist in config
    if cfg.HasContainer(name) {
        return fmt.Errorf("container %q already exists in config", name)
    }

    // Get full LXC name with prefix
    lxcName := cfg.GetLXCName(name)

    // Check LXC name doesn't exist
    if exists, _ := lxc.Exists(lxcName); exists {
        return fmt.Errorf("container %q already exists in LXC", lxcName)
    }

    fmt.Printf("Creating container '%s' (LXC: %s)...\n", name, lxcName)

    // Launch with full LXC name
    if err := lxc.Launch(lxcName, image); err != nil {
        return fmt.Errorf("failed to launch container: %w", err)
    }

    // ... rest of setup (nesting, user, ssh) using lxcName ...

    // Save to config with short name
    cfg.AddContainer(name, image)
    if err := cfg.Save(); err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }

    // Display with short name
    fmt.Printf("\n✓ Container '%s' ready\n", name)
    fmt.Printf("  LXC name: %s\n", lxcName)
    fmt.Printf("  IP:       %s\n", ip)

    return nil
}
```

---

### Phase 4: Update All Commands to Use Prefix

Each command needs to:
1. Load config and get project prefix
2. Convert short name → LXC name for operations
3. Display short names to user

#### 4.1 Commands to Update

| Command | File | Changes |
|---------|------|---------|
| up | `cmd/up.go` | Use `cfg.GetLXCName(name)` for lxc.Start() |
| down | `cmd/down.go` | Use `cfg.GetLXCName(name)` for lxc.Stop() |
| remove | `cmd/remove.go` | Use `cfg.GetLXCName(name)` for lxc.Delete() |
| ssh | `cmd/ssh.go` | Use `cfg.GetLXCName(name)` for lxc.Exec() |
| proxy | `cmd/proxy.go` | Use `cfg.GetLXCName(name)` for IP lookup |
| list | `cmd/list.go` | Show short names, query by LXC names |
| snapshot | `cmd/snapshot.go` | Use `cfg.GetLXCName(name)` for snapshot |

#### 4.2 List Command Changes

```go
func runList(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }
    if cfg == nil {
        return fmt.Errorf("no project found")
    }

    fmt.Printf("Project: %s\n\n", cfg.Project)

    // Print header
    fmt.Printf("%-15s %-20s %-10s %-15s %s\n",
        "NAME", "IMAGE", "STATUS", "IP", "PORTS")

    for name, container := range cfg.Containers {
        lxcName := cfg.GetLXCName(name)
        status, _ := lxc.GetStatus(lxcName)
        ip, _ := lxc.GetIP(lxcName)
        ports := cfg.GetPorts(name)

        // Display SHORT name, not LXC name
        fmt.Printf("%-15s %-20s %-10s %-15s %v\n",
            name, container.Image, status, ip, ports)
    }

    return nil
}
```

---

### Phase 5: Backwards Compatibility

#### 5.1 Handle Legacy Config (no project field)

When loading a config without a `project` field:
- Prompt user to migrate or set project name
- Or auto-migrate using folder name

```go
func Load() (*Config, error) {
    // ... load yaml ...

    // Migration check
    if cfg.Project == "" && len(cfg.Containers) > 0 {
        // Legacy config detected
        fmt.Println("Warning: Legacy config detected (no project field)")
        fmt.Println("Run 'lxc-dev-manager migrate' to add project prefix")
        // Continue working with empty prefix for now
    }

    return cfg, nil
}
```

#### 5.2 Migration Command (Optional)

```
lxc-dev-manager migrate [--name <project>]
```

- Adds project field to existing config
- Renames existing LXC containers to add prefix
- Updates config with new names

---

## File Changes Summary

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/config/config.go` | MODIFY | Add Project field, GetLXCName(), GetShortName() |
| `cmd/project.go` | NEW | project create, project delete commands |
| `cmd/container.go` | NEW | container subcommand with create |
| `cmd/create.go` | DELETE | Replaced by project.go and container.go |
| `cmd/up.go` | MODIFY | Use GetLXCName() |
| `cmd/down.go` | MODIFY | Use GetLXCName() |
| `cmd/remove.go` | MODIFY | Use GetLXCName() |
| `cmd/ssh.go` | MODIFY | Use GetLXCName() |
| `cmd/proxy.go` | MODIFY | Use GetLXCName() |
| `cmd/list.go` | MODIFY | Show project header, use short names |
| `cmd/snapshot.go` | MODIFY | Use GetLXCName() |
| `cmd/root.go` | MODIFY | Add root-level aliases |

---

## New Command Structure

```
lxc-dev-manager
├── create [--name <project>]        # Alias for 'project create'
├── project
│   ├── create [--name <project>]    # Initialize project
│   └── delete [--force]             # Delete project and all containers
├── container (alias: c)
│   ├── create <name> <image>        # Create container
│   ├── remove <name> [--force]      # Remove container
│   ├── list                         # List containers
│   ├── up <name>                    # Start container
│   ├── down <name>                  # Stop container
│   ├── ssh <name> [--user]          # Shell into container
│   └── proxy <name>                 # Forward ports
├── up <name>                        # Alias for 'container up'
├── down <name>                      # Alias for 'container down'
├── list                             # Alias for 'container list'
├── ssh <name>                       # Alias for 'container ssh'
├── proxy <name>                     # Alias for 'container proxy'
├── remove <name>                    # Alias for 'container remove'
├── image
│   ├── list [--all]
│   ├── delete <name> [--force]
│   └── rename <old> <new>
└── snapshot <name> <image-name>
```

---

## Example Workflow

```bash
# In ~/projects/webapp/
$ lxc-dev-manager create
✓ Project 'webapp' created
  Config: containers.yaml

# Or with custom name
$ lxc-dev-manager create --name my-app
✓ Project 'my-app' created

# Create containers (automatically prefixed)
$ lxc-dev-manager container create dev1 ubuntu:24.04
Creating container 'dev1' (LXC: webapp-dev1)...
✓ Container 'dev1' ready
  LXC name: webapp-dev1
  IP:       10.x.x.x

$ lxc-dev-manager list
Project: webapp

NAME            IMAGE                STATUS     IP              PORTS
dev1            ubuntu:24.04         RUNNING    10.x.x.x        5173,8000,5432

# In LXC directly
$ lxc list
+-------------+---------+...
| webapp-dev1 | RUNNING |...
+-------------+---------+...

# Delete entire project
$ lxc-dev-manager project delete
Project: webapp
Config:  containers.yaml

Containers to be deleted:
  - dev1 (webapp-dev1) [RUNNING]

Are you sure you want to delete this project? [y/N]: y
Deleting container 'dev1'... ✓
Removing containers.yaml... ✓

✓ Project 'webapp' deleted
```

---

## Testing Plan

1. **Unit tests for config helpers**
   - `GetLXCName()` with/without project
   - `GetShortName()` strips prefix correctly
   - `GetProjectFromFolder()` returns correct name

2. **Integration tests for commands**
   - `project create` creates config with project field
   - `project create --name` uses custom name
   - `project create` fails if project exists
   - `project delete` removes all containers
   - `project delete --force` skips confirmation
   - `container create` uses prefixed name in LXC
   - All commands work with prefixed names

3. **E2E tests**
   - Full workflow: create project → create containers → delete project
   - Multi-project: two projects in different folders don't conflict

---

## Estimated Changes

- **Config**: ~50 lines
- **project.go**: ~150 lines
- **container.go**: ~100 lines (restructure)
- **Command updates**: ~5-10 lines each × 7 commands = ~50 lines
- **Tests**: ~200 lines
- **Total**: ~550 lines new/modified code
