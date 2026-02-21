# Issue #8: Missing Input Validation for Container Names and Ports

## Severity: Medium
## Category: Input Validation / Robustness

---

## Problem Summary

User-provided input for container names and port numbers is not validated against system constraints. This can lead to confusing error messages from LXC, invalid configurations, or unexpected behavior.

---

## Affected Areas

### 1. Container Names

**File:** `cmd/container.go`

```go
func runContainerCreate(cmd *cobra.Command, args []string) error {
    name := args[0]  // <-- NO VALIDATION
    image := args[1]

    // ... proceeds to create container with potentially invalid name
}
```

### 2. Port Numbers

**File:** `internal/config/config.go`

```go
type Defaults struct {
    Ports []int `yaml:"ports"`  // <-- NO VALIDATION ON LOAD
}
```

When loading from YAML:
```yaml
defaults:
  ports:
    - 99999      # Invalid: above 65535
    - -1         # Invalid: negative
    - 0          # Invalid: reserved
    - 80         # Potentially problematic: requires root
```

---

## Why This Is a Problem

### Container Name Issues

LXC has specific naming requirements. Invalid names cause cryptic errors:

```bash
$ lxc launch ubuntu:24.04 "my container"
Error: Invalid instance name: my container

$ lxc launch ubuntu:24.04 "123-start-with-number"
Error: Invalid instance name

$ lxc launch ubuntu:24.04 "name_with_very_long_identifier_that_exceeds_limits..."
Error: Instance name too long
```

**LXC naming rules:**
- Must start with a letter
- Can contain letters, numbers, and hyphens
- Maximum 63 characters
- No spaces or special characters

### Port Number Issues

Invalid ports cause runtime failures or security issues:

```go
// Port 99999 - fails at runtime
net.Listen("tcp", ":99999")
// Error: invalid port

// Port 0 - random port assignment (probably not intended)
net.Listen("tcp", ":0")
// Assigns random available port

// Port 80 - requires root privileges
net.Listen("tcp", ":80")
// Error: permission denied (if not root)
```

### Combined with Project Prefix

The full LXC name is `project-containername`. If both are max length:
```
project name (63) + "-" (1) + container name (63) = 127 characters
```
This exceeds LXC's limit even if individual parts are valid.

---

## Current Behavior Examples

### Invalid Container Name
```bash
$ lxc-dev-manager container create "my container" ubuntu:24.04
Creating container 'my container' (LXC: myproject-my container) from image 'ubuntu:24.04'...
Error: failed to launch container: Invalid instance name: myproject-my container
```

The error comes from LXC, not from validation. Users don't know what's wrong.

### Invalid Port in Config
```yaml
# containers.yaml
defaults:
  ports:
    - 70000
```

```bash
$ lxc-dev-manager proxy dev1
Proxying dev1 (10.10.10.5):
Error: failed to start proxy for port 70000: listen tcp :70000: invalid port
```

Again, runtime error instead of config validation.

---

## Recommended Fix

### 1. Container Name Validation

Create `internal/validation/validation.go`:

```go
package validation

import (
    "fmt"
    "regexp"
    "strings"
)

const (
    MaxContainerNameLength = 63
    MaxProjectNameLength   = 63
    MaxCombinedLength      = 63  // LXC limit for full name
)

var (
    // LXC naming rules: start with letter, alphanumeric + hyphens
    containerNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)

    // Reserved names that conflict with LXC commands/concepts
    reservedNames = map[string]bool{
        "list":     true,
        "create":   true,
        "delete":   true,
        "start":    true,
        "stop":     true,
        "snapshot": true,
        "image":    true,
        "config":   true,
    }
)

// ValidateContainerName checks if a container name is valid for LXC
func ValidateContainerName(name string) error {
    if name == "" {
        return fmt.Errorf("container name cannot be empty")
    }

    if len(name) > MaxContainerNameLength {
        return fmt.Errorf("container name too long: %d characters (max %d)",
            len(name), MaxContainerNameLength)
    }

    if !containerNameRegex.MatchString(name) {
        if name[0] >= '0' && name[0] <= '9' {
            return fmt.Errorf("container name must start with a letter, not '%c'", name[0])
        }
        if strings.Contains(name, " ") {
            return fmt.Errorf("container name cannot contain spaces")
        }
        if strings.Contains(name, "_") {
            return fmt.Errorf("container name cannot contain underscores (use hyphens instead)")
        }
        return fmt.Errorf("container name contains invalid characters (allowed: letters, numbers, hyphens)")
    }

    if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
        return fmt.Errorf("container name cannot start or end with a hyphen")
    }

    if strings.Contains(name, "--") {
        return fmt.Errorf("container name cannot contain consecutive hyphens")
    }

    nameLower := strings.ToLower(name)
    if reservedNames[nameLower] {
        return fmt.Errorf("'%s' is a reserved name", name)
    }

    return nil
}

// ValidateFullContainerName checks if project + container name combination is valid
func ValidateFullContainerName(project, container string) error {
    if err := ValidateContainerName(container); err != nil {
        return err
    }

    fullName := container
    if project != "" {
        fullName = project + "-" + container
    }

    if len(fullName) > MaxCombinedLength {
        return fmt.Errorf("full container name '%s' too long: %d characters (max %d). "+
            "Use a shorter project or container name",
            fullName, len(fullName), MaxCombinedLength)
    }

    return nil
}
```

### 2. Port Validation

```go
package validation

import "fmt"

const (
    MinPort           = 1
    MaxPort           = 65535
    PrivilegedPortMax = 1023
)

// Common ports that might cause conflicts
var wellKnownPorts = map[int]string{
    22:   "SSH",
    80:   "HTTP",
    443:  "HTTPS",
    3306: "MySQL",
    5432: "PostgreSQL",
    6379: "Redis",
    27017: "MongoDB",
}

// ValidatePort checks if a port number is valid
func ValidatePort(port int) error {
    if port < MinPort || port > MaxPort {
        return fmt.Errorf("invalid port %d: must be between %d and %d",
            port, MinPort, MaxPort)
    }
    return nil
}

// ValidatePorts checks a list of ports
func ValidatePorts(ports []int) error {
    seen := make(map[int]bool)

    for _, port := range ports {
        if err := ValidatePort(port); err != nil {
            return err
        }

        if seen[port] {
            return fmt.Errorf("duplicate port %d in configuration", port)
        }
        seen[port] = true
    }

    return nil
}

// PortWarnings returns non-fatal warnings about port configuration
func PortWarnings(ports []int) []string {
    var warnings []string

    for _, port := range ports {
        if port <= PrivilegedPortMax {
            warnings = append(warnings,
                fmt.Sprintf("port %d requires root privileges", port))
        }

        if name, ok := wellKnownPorts[port]; ok {
            warnings = append(warnings,
                fmt.Sprintf("port %d is commonly used by %s", port, name))
        }
    }

    return warnings
}
```

### 3. Integration with Config Loading

**File:** `internal/config/config.go`

```go
import "lxc-dev-manager/internal/validation"

func Load() (*Config, error) {
    data, err := os.ReadFile(ConfigFile)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil
        }
        return nil, err
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("invalid YAML in %s: %w", ConfigFile, err)
    }

    // Validate after loading
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    if cfg.Containers == nil {
        cfg.Containers = make(map[string]Container)
    }

    return &cfg, nil
}

func (c *Config) Validate() error {
    // Validate project name
    if c.Project != "" && !IsValidProjectName(c.Project) {
        return fmt.Errorf("invalid project name %q", c.Project)
    }

    // Validate default ports
    if err := validation.ValidatePorts(c.Defaults.Ports); err != nil {
        return fmt.Errorf("invalid default ports: %w", err)
    }

    // Validate each container
    for name, container := range c.Containers {
        if err := validation.ValidateFullContainerName(c.Project, name); err != nil {
            return fmt.Errorf("container '%s': %w", name, err)
        }

        if len(container.Ports) > 0 {
            if err := validation.ValidatePorts(container.Ports); err != nil {
                return fmt.Errorf("container '%s': %w", name, err)
            }
        }
    }

    return nil
}
```

### 4. Command-Level Validation

**File:** `cmd/container.go`

```go
import "lxc-dev-manager/internal/validation"

func runContainerCreate(cmd *cobra.Command, args []string) error {
    name := args[0]
    image := args[1]

    // Validate container name FIRST
    if err := validation.ValidateContainerName(name); err != nil {
        return fmt.Errorf("invalid container name: %w", err)
    }

    cfg, err := requireProject()
    if err != nil {
        return err
    }

    // Validate combined name
    if err := validation.ValidateFullContainerName(cfg.Project, name); err != nil {
        return err
    }

    // ... rest of function
}
```

---

## Improved Error Messages

### Before
```
Error: failed to launch container: Invalid instance name: my-project-my container
```

### After
```
Error: invalid container name: container name cannot contain spaces
```

```
Error: invalid container name: container name must start with a letter, not '1'
```

```
Error: full container name 'very-long-project-name-here-very-long-container-name-here'
too long: 72 characters (max 63). Use a shorter project or container name
```

```
Error: invalid configuration: invalid default ports: invalid port 99999:
must be between 1 and 65535
```

---

## Testing

```go
// internal/validation/validation_test.go
package validation

import "testing"

func TestValidateContainerName(t *testing.T) {
    tests := []struct {
        name    string
        wantErr bool
        errMsg  string
    }{
        // Valid names
        {"dev", false, ""},
        {"dev1", false, ""},
        {"my-container", false, ""},
        {"a", false, ""},
        {"MyContainer", false, ""},

        // Invalid: empty
        {"", true, "cannot be empty"},

        // Invalid: starts with number
        {"1dev", true, "must start with a letter"},
        {"123", true, "must start with a letter"},

        // Invalid: spaces
        {"my container", true, "cannot contain spaces"},
        {"my  container", true, "cannot contain spaces"},

        // Invalid: special characters
        {"my_container", true, "cannot contain underscores"},
        {"my.container", true, "invalid characters"},
        {"my@container", true, "invalid characters"},

        // Invalid: hyphens
        {"-dev", true, "cannot start or end with a hyphen"},
        {"dev-", true, "cannot start or end with a hyphen"},
        {"dev--test", true, "consecutive hyphens"},

        // Invalid: too long
        {string(make([]byte, 64)), true, "too long"},

        // Invalid: reserved
        {"list", true, "reserved name"},
        {"delete", true, "reserved name"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateContainerName(tt.name)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateContainerName(%q) error = %v, wantErr %v",
                    tt.name, err, tt.wantErr)
            }
            if tt.wantErr && tt.errMsg != "" {
                if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
                    t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
                }
            }
        })
    }
}

func TestValidatePort(t *testing.T) {
    tests := []struct {
        port    int
        wantErr bool
    }{
        {80, false},
        {8080, false},
        {65535, false},
        {1, false},
        {0, true},
        {-1, true},
        {65536, true},
        {99999, true},
    }

    for _, tt := range tests {
        t.Run(fmt.Sprintf("port_%d", tt.port), func(t *testing.T) {
            err := ValidatePort(tt.port)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidatePort(%d) error = %v, wantErr %v",
                    tt.port, err, tt.wantErr)
            }
        })
    }
}
```

---

## Edge Cases to Consider

1. **Unicode characters**: Should `container-日本語` be allowed?
2. **Case sensitivity**: LXC is case-sensitive, but `Dev` and `dev` might confuse users
3. **Emoji**: `container-🚀` - definitely reject
4. **Leading/trailing whitespace**: Trim before validation
5. **Reserved ports**: Warn about 22, 80, 443, etc.

---

## References

- [LXC Container Naming](https://linuxcontainers.org/lxd/docs/latest/instances/)
- [IANA Port Assignments](https://www.iana.org/assignments/service-names-port-numbers/)
- [Input Validation Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html)
