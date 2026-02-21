package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"

	"github.com/spf13/cobra"
)

// pathSpec represents a source or destination path
type pathSpec struct {
	isContainer bool
	container   string // container name or glob pattern (empty if host path)
	path        string
}

// parsePath parses a path argument into a pathSpec.
// Container paths have format "container:/path", host paths are just "/path" or "./path"
func parsePath(p string) pathSpec {
	// Check for container:path format
	// Be careful with Windows-style paths and paths starting with /
	if idx := strings.Index(p, ":"); idx > 0 {
		// Make sure this isn't just a path like "/foo" where idx would be -1
		// or "C:\path" on Windows (single letter before colon)
		container := p[:idx]
		// Container names should be alphanumeric with possible * for glob
		// Skip if it looks like an absolute path or single letter (Windows drive)
		if len(container) > 1 || container == "*" || strings.Contains(container, "*") {
			return pathSpec{
				isContainer: true,
				container:   container,
				path:        p[idx+1:],
			}
		}
	}
	return pathSpec{
		isContainer: false,
		path:        p,
	}
}

// matchContainers returns container names matching the glob pattern.
// Supports: "*" (all containers), "prefix*" (containers starting with prefix)
func matchContainers(cfg *config.Config, pattern string) []string {
	var matches []string

	if pattern == "*" {
		// All containers
		for name := range cfg.Containers {
			matches = append(matches, name)
		}
	} else if strings.HasSuffix(pattern, "*") {
		// Prefix match (e.g., "dev*")
		prefix := strings.TrimSuffix(pattern, "*")
		for name := range cfg.Containers {
			if strings.HasPrefix(name, prefix) {
				matches = append(matches, name)
			}
		}
	} else {
		// Exact match - return single container if it exists
		if _, ok := cfg.Containers[pattern]; ok {
			matches = append(matches, pattern)
		}
	}

	sort.Strings(matches) // Consistent ordering
	return matches
}

// validateContainer checks that a container exists in config and LXC
func validateContainer(cfg *config.Config, name string) error {
	if !cfg.HasContainer(name) {
		return fmt.Errorf("container '%s' not found in project config", name)
	}
	lxcName := cfg.GetLXCName(name)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC (expected: %s)", name, lxcName)
	}
	return nil
}

// copyToContainer copies a file or directory from host to a single container
func copyToContainer(cfg *config.Config, containerName, source, remotePath string, sourceInfo os.FileInfo, autoCreate bool) error {
	lxcName := cfg.GetLXCName(containerName)

	// Expand ~ to user's home directory
	if strings.HasPrefix(remotePath, "~/") {
		user := cfg.GetUser(containerName)
		remotePath = "/home/" + user.Name + remotePath[1:]
	} else if remotePath == "~" {
		user := cfg.GetUser(containerName)
		remotePath = "/home/" + user.Name
	}

	// Determine if recursive (directory)
	recursive := sourceInfo.IsDir()

	// Get the destination directory to check/create
	destDir := path.Dir(remotePath)

	// Check if destination directory exists
	user := cfg.GetUser(containerName)
	if !lxc.DirExists(lxcName, destDir) {
		if !autoCreate && !confirmPrompt(fmt.Sprintf("Directory '%s' does not exist in %s. Create it?", destDir, containerName)) {
			return fmt.Errorf("destination directory does not exist")
		}
		if err := lxc.Exec(lxcName, "mkdir", "-p", destDir); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		lxc.Exec(lxcName, "chown", user.Name+":"+user.Name, destDir)
	}

	// Push the file
	pushPath := remotePath
	if recursive {
		pushPath = path.Dir(remotePath)
	}

	if err := lxc.FilePush(lxcName, source, pushPath, recursive); err != nil {
		return err
	}

	// Fix ownership
	if recursive {
		if err := lxc.Exec(lxcName, "chown", "-R", user.Name+":"+user.Name, remotePath); err != nil {
			return fmt.Errorf("could not set ownership: %w", err)
		}
	} else {
		if err := lxc.Exec(lxcName, "chown", user.Name+":"+user.Name, remotePath); err != nil {
			return fmt.Errorf("could not set ownership: %w", err)
		}
	}

	return nil
}

// copyFromContainer copies a file or directory from container to host
func copyFromContainer(cfg *config.Config, containerName, remotePath, localPath string) error {
	lxcName := cfg.GetLXCName(containerName)

	// Expand ~ to user's home directory
	if strings.HasPrefix(remotePath, "~/") {
		user := cfg.GetUser(containerName)
		remotePath = "/home/" + user.Name + remotePath[1:]
	} else if remotePath == "~" {
		user := cfg.GetUser(containerName)
		remotePath = "/home/" + user.Name
	}

	// Check if source exists in container
	if !lxc.FileExists(lxcName, remotePath) {
		return fmt.Errorf("source '%s' does not exist in container %s", remotePath, containerName)
	}

	// Determine if recursive (directory)
	recursive := lxc.IsDir(lxcName, remotePath)

	// Ensure local destination directory exists
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Pull the file
	if err := lxc.FilePull(lxcName, remotePath, localPath, recursive); err != nil {
		return err
	}

	return nil
}

var mvCmd = &cobra.Command{
	Use:   "mv <source> <dest>",
	Short: "Copy files between host and container(s)",
	Long: `Copy files or directories between host and containers.

Supports multiple directions:
  Host → Container:      mv ./file container:/path
  Container → Host:      mv container:/path ./local
  Container → Container: mv container1:/path container2:/path

Use * to target all containers, or prefix* to match by name prefix.

Examples:
  lxc-dev-manager mv ./app dev1:/home/dev/app       # host → container
  lxc-dev-manager mv ./config.json *:/etc/app/      # host → all containers
  lxc-dev-manager mv dev1:/etc/config ./backup/     # container → host
  lxc-dev-manager mv dev1:/app/config *:/app/       # container → all containers
  lxc-dev-manager mv dev1:/data dev2:/data          # container → container
  lxc-dev-manager mv ./data dev1:/opt/data -y       # auto-create directory`,
	Args: cobra.ExactArgs(2),
	RunE: runMv,
}

var mvYes bool

func init() {
	rootCmd.AddCommand(mvCmd)
	mvCmd.Flags().BoolVarP(&mvYes, "yes", "y", false, "Auto-create destination directory if it doesn't exist")
}

func runMv(cmd *cobra.Command, args []string) error {
	src := parsePath(args[0])
	dst := parsePath(args[1])

	// Check for common mistake: container/path instead of container:/path
	if !src.isContainer && !dst.isContainer {
		// If destination looks like it might be a container path (starts with alphanumeric, contains /)
		// and doesn't start with . or / (common host path prefixes), warn the user
		if len(dst.path) > 0 && dst.path[0] != '.' && dst.path[0] != '/' && strings.Contains(dst.path, "/") {
			return fmt.Errorf("invalid destination format. Use container:path (e.g., dev1:/home/dev/)")
		}
		return fmt.Errorf("both source and destination are local paths; use 'cp' instead")
	}

	switch {
	case !src.isContainer && dst.isContainer:
		// Host → Container(s)
		return hostToContainer(src, dst)

	case src.isContainer && !dst.isContainer:
		// Container → Host
		return containerToHost(src, dst)

	case src.isContainer && dst.isContainer:
		// Container → Container(s)
		return containerToContainer(src, dst)

	default:
		return fmt.Errorf("unexpected path combination")
	}
}

// hostToContainer handles copying from host to one or more containers
func hostToContainer(src, dst pathSpec) error {
	// Validate source exists on host
	info, err := os.Stat(src.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source '%s' does not exist", src.path)
		}
		return fmt.Errorf("cannot access source '%s': %w", src.path, err)
	}

	if dst.path == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	cfg, err := config.Load(projectDir)
	if err != nil {
		return err
	}

	// Check for glob pattern
	if strings.Contains(dst.container, "*") {
		matches := matchContainers(cfg, dst.container)
		if len(matches) == 0 {
			return fmt.Errorf("no containers match pattern %q", dst.container)
		}

		fmt.Printf("Targeting %d container(s): %s\n", len(matches), strings.Join(matches, ", "))

		var errors []string
		for _, name := range matches {
			if err := validateContainer(cfg, name); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", name, err))
				fmt.Printf("✗ %s failed: %v\n", name, err)
				continue
			}

			printCopyMessage(src.path, name, dst.path, info.IsDir())

			if err := copyToContainer(cfg, name, src.path, dst.path, info, mvYes); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", name, err))
				fmt.Printf("✗ %s failed: %v\n", name, err)
				continue
			}
			fmt.Printf("✓ %s done\n", name)
		}

		if len(errors) > 0 {
			return fmt.Errorf("failed for %d container(s):\n  %s", len(errors), strings.Join(errors, "\n  "))
		}
		fmt.Println("All done.")
		return nil
	}

	// Single container
	if err := validateContainer(cfg, dst.container); err != nil {
		return err
	}

	printCopyMessage(src.path, dst.container, dst.path, info.IsDir())

	if err := copyToContainer(cfg, dst.container, src.path, dst.path, info, mvYes); err != nil {
		return err
	}

	fmt.Println("Done.")
	return nil
}

// containerToHost handles copying from container to host
func containerToHost(src, dst pathSpec) error {
	if src.path == "" {
		return fmt.Errorf("source path cannot be empty")
	}

	cfg, err := config.Load(projectDir)
	if err != nil {
		return err
	}

	// Glob not supported on source
	if strings.Contains(src.container, "*") {
		return fmt.Errorf("glob patterns not supported for source container")
	}

	if err := validateContainer(cfg, src.container); err != nil {
		return err
	}

	fmt.Printf("Copying from %s:%s to %s...\n", src.container, src.path, dst.path)

	if err := copyFromContainer(cfg, src.container, src.path, dst.path); err != nil {
		return err
	}

	fmt.Println("Done.")
	return nil
}

// containerToContainer handles copying from one container to one or more containers
func containerToContainer(src, dst pathSpec) error {
	if src.path == "" {
		return fmt.Errorf("source path cannot be empty")
	}
	if dst.path == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	cfg, err := config.Load(projectDir)
	if err != nil {
		return err
	}

	// Glob not supported on source
	if strings.Contains(src.container, "*") {
		return fmt.Errorf("glob patterns not supported for source container")
	}

	if err := validateContainer(cfg, src.container); err != nil {
		return err
	}

	// Create temp directory for intermediate storage
	tempDir, err := os.MkdirTemp("", "lxc-mv-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Pull from source container to temp
	tempPath := filepath.Join(tempDir, filepath.Base(src.path))
	fmt.Printf("Pulling from %s:%s...\n", src.container, src.path)
	if err := copyFromContainer(cfg, src.container, src.path, tempPath); err != nil {
		return fmt.Errorf("failed to pull from source: %w", err)
	}

	// Get info about the pulled file/directory
	info, err := os.Stat(tempPath)
	if err != nil {
		return fmt.Errorf("failed to stat temp file: %w", err)
	}

	// Push to destination container(s)
	if strings.Contains(dst.container, "*") {
		matches := matchContainers(cfg, dst.container)
		if len(matches) == 0 {
			return fmt.Errorf("no containers match pattern %q", dst.container)
		}

		fmt.Printf("Targeting %d container(s): %s\n", len(matches), strings.Join(matches, ", "))

		var errors []string
		for _, name := range matches {
			// Skip source container if it matches
			if name == src.container {
				fmt.Printf("⊘ %s skipped (source container)\n", name)
				continue
			}

			if err := validateContainer(cfg, name); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", name, err))
				fmt.Printf("✗ %s failed: %v\n", name, err)
				continue
			}

			printCopyMessage(src.path, name, dst.path, info.IsDir())

			if err := copyToContainer(cfg, name, tempPath, dst.path, info, mvYes); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", name, err))
				fmt.Printf("✗ %s failed: %v\n", name, err)
				continue
			}
			fmt.Printf("✓ %s done\n", name)
		}

		if len(errors) > 0 {
			return fmt.Errorf("failed for %d container(s):\n  %s", len(errors), strings.Join(errors, "\n  "))
		}
		fmt.Println("All done.")
		return nil
	}

	// Single destination container
	if err := validateContainer(cfg, dst.container); err != nil {
		return err
	}

	printCopyMessage(src.path, dst.container, dst.path, info.IsDir())

	if err := copyToContainer(cfg, dst.container, tempPath, dst.path, info, mvYes); err != nil {
		return err
	}

	fmt.Println("Done.")
	return nil
}

func printCopyMessage(source, container, dest string, isDir bool) {
	if isDir {
		fmt.Printf("Copying directory '%s' to %s:%s...\n", source, container, dest)
	} else {
		fmt.Printf("Copying file '%s' to %s:%s...\n", source, container, dest)
	}
}
