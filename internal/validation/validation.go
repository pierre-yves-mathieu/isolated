package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// MaxContainerNameLength is the max length for a container name
	MaxContainerNameLength = 63
	// MaxCombinedLength is LXC's limit for full container name (project-container)
	MaxCombinedLength = 63
	// MinPort is the minimum valid port number
	MinPort = 1
	// MaxPort is the maximum valid port number
	MaxPort = 65535
	// PrivilegedPortMax is the highest port requiring root privileges
	PrivilegedPortMax = 1023
	// MaxContainerPathLength is the maximum length for a container path
	MaxContainerPathLength = 4096
	// MaxMountNameLength is the maximum length for a mount name
	MaxMountNameLength = 50
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

	// BlockedHostPaths are paths that cannot be mounted from the host
	BlockedHostPaths = []string{
		"/",
		"/root",
		"/etc",
		"/boot",
		"/proc",
		"/sys",
		"/dev",
		"/var/lib/lxd",
		"/var/lib/lxc",
	}

	// BlockedHostPatterns are path suffixes that cannot be mounted from the host
	BlockedHostPatterns = []string{
		"/.ssh",
		"/.aws",
		"/.gnupg",
		"/.config/gcloud",
	}

	// RiskyHostPaths are paths that trigger a warning (but not an error)
	RiskyHostPaths = []string{
		"/home",
		"/var",
		"/tmp",
		"/opt",
	}

	// BlockedContainerPaths are paths that cannot be mounted inside containers
	BlockedContainerPaths = []string{
		"/",
		"/proc",
		"/sys",
		"/dev",
	}
)

// ValidateContainerName checks if a container name is valid for LXC
func ValidateContainerName(name string) error {
	name = strings.TrimSpace(name)

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

// ValidateSourcePath validates a host source path for mounting.
// Returns the resolved absolute path, a warning message (empty if none), and an error.
func ValidateSourcePath(source string) (resolvedPath string, warning string, err error) {
	if source == "" {
		return "", "", fmt.Errorf("source path cannot be empty")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(source)
	if err != nil {
		return "", "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Resolve symlinks (CRITICAL for security)
	resolvedPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("source path does not exist: %s", absPath)
		}
		return "", "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Clean path
	resolvedPath = filepath.Clean(resolvedPath)

	// Check path exists
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("source path does not exist: %s", resolvedPath)
		}
		return "", "", fmt.Errorf("failed to stat source path: %w", err)
	}

	// Check is directory (not file)
	if !info.IsDir() {
		return "", "", fmt.Errorf("source path must be a directory, not a file: %s", resolvedPath)
	}

	// Check against BlockedHostPaths
	for _, blocked := range BlockedHostPaths {
		if resolvedPath == blocked {
			return "", "", fmt.Errorf("mounting '%s' is not allowed for security reasons", resolvedPath)
		}
	}

	// Check against BlockedHostPatterns (suffix match)
	for _, pattern := range BlockedHostPatterns {
		if strings.HasSuffix(resolvedPath, pattern) {
			return "", "", fmt.Errorf("mounting paths matching '%s' is not allowed for security reasons", pattern)
		}
	}

	// Check against RiskyHostPaths (return warning, not error)
	for _, risky := range RiskyHostPaths {
		if resolvedPath == risky {
			warning = fmt.Sprintf("mounting '%s' is risky and may expose sensitive data", resolvedPath)
			break
		}
	}

	return resolvedPath, warning, nil
}

// ValidateContainerPath validates a path inside a container
func ValidateContainerPath(path string) error {
	if path == "" {
		return fmt.Errorf("container path cannot be empty")
	}

	// Must be absolute (starts with /)
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("container path must be absolute (start with /): %s", path)
	}

	// Max length check
	if len(path) > MaxContainerPathLength {
		return fmt.Errorf("container path too long: %d characters (max %d)", len(path), MaxContainerPathLength)
	}

	// Check for control characters
	for _, c := range path {
		if c == '\x00' || c == '\n' || c == '\r' || c == '\t' {
			return fmt.Errorf("container path cannot contain control characters")
		}
	}

	// Clean path
	cleanPath := filepath.Clean(path)

	// No .. traversal after cleaning
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("container path cannot contain path traversal (..)")
	}

	// Check against BlockedContainerPaths
	for _, blocked := range BlockedContainerPaths {
		if cleanPath == blocked {
			return fmt.Errorf("mounting to '%s' inside container is not allowed", blocked)
		}
	}

	return nil
}

// ValidateMountName validates a mount/device name using the same rules as container names
func ValidateMountName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return fmt.Errorf("mount name cannot be empty")
	}

	if len(name) > MaxMountNameLength {
		return fmt.Errorf("mount name too long: %d characters (max %d)",
			len(name), MaxMountNameLength)
	}

	if !containerNameRegex.MatchString(name) {
		if name[0] >= '0' && name[0] <= '9' {
			return fmt.Errorf("mount name must start with a letter, not '%c'", name[0])
		}
		if strings.Contains(name, " ") {
			return fmt.Errorf("mount name cannot contain spaces")
		}
		if strings.Contains(name, "_") {
			return fmt.Errorf("mount name cannot contain underscores (use hyphens instead)")
		}
		return fmt.Errorf("mount name contains invalid characters (allowed: letters, numbers, hyphens)")
	}

	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("mount name cannot start or end with a hyphen")
	}

	if strings.Contains(name, "--") {
		return fmt.Errorf("mount name cannot contain consecutive hyphens")
	}

	return nil
}

// GenerateMountName generates a safe mount name from a source path
func GenerateMountName(sourcePath string) string {
	// Get base name from path
	name := filepath.Base(sourcePath)

	// Replace invalid characters with hyphens
	var result strings.Builder
	prevWasHyphen := false

	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result.WriteRune(c)
			prevWasHyphen = false
		} else if !prevWasHyphen && result.Len() > 0 {
			// Replace invalid chars with hyphen (avoid consecutive hyphens)
			result.WriteRune('-')
			prevWasHyphen = true
		}
	}

	name = result.String()

	// Remove trailing hyphen
	name = strings.TrimSuffix(name, "-")

	// Ensure starts with letter (prefix with "mount-" if starts with number or empty)
	if name == "" || (name[0] >= '0' && name[0] <= '9') {
		name = "mount-" + name
	}

	// Trim to max length
	if len(name) > MaxMountNameLength {
		name = name[:MaxMountNameLength]
		// Remove trailing hyphen after truncation
		name = strings.TrimSuffix(name, "-")
	}

	return name
}
