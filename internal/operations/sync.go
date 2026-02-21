package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lxc-dev-manager/internal/config"
	"lxc-dev-manager/internal/lxc"
)

// SyncFiles copies all configured sync entries from host to container.
// Source paths are resolved relative to baseDir (typically the containers.yaml directory).
// Errors are collected per-file; all entries are attempted even if some fail.
func SyncFiles(cfg *config.Config, containerName, baseDir string) error {
	if !cfg.HasContainer(containerName) {
		return fmt.Errorf("container '%s' not found in config", containerName)
	}

	entries := cfg.GetSyncEntries(containerName)
	if len(entries) == 0 {
		return nil
	}

	lxcName := cfg.GetLXCName(containerName)
	if !lxc.Exists(lxcName) {
		return fmt.Errorf("container '%s' does not exist in LXC", lxcName)
	}

	status, err := lxc.GetStatus(lxcName)
	if err != nil {
		return fmt.Errorf("failed to get container status: %w", err)
	}
	if status != "RUNNING" {
		return fmt.Errorf("container '%s' is not running (status: %s)", containerName, status)
	}

	var errors []string
	for _, entry := range entries {
		if err := syncEntry(cfg, containerName, baseDir, entry); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", entry.Source, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("sync errors:\n  %s", strings.Join(errors, "\n  "))
	}
	return nil
}

// syncEntry copies a single file/directory from host to container.
func syncEntry(cfg *config.Config, containerName, baseDir string, entry config.SyncEntry) error {
	// Resolve source path
	source := entry.Source
	if !filepath.IsAbs(source) {
		source = filepath.Join(baseDir, source)
	}

	// Check source exists
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source does not exist")
		}
		return fmt.Errorf("cannot access source: %w", err)
	}

	// Use existing CopyToContainer which handles dir creation and ownership
	return CopyToContainer(cfg, containerName, source, entry.Dest, CopyOpts{AutoCreateDir: true})
}
