package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		{"a1b2c3", false, ""},

		// Invalid: empty
		{"", true, "cannot be empty"},

		// Invalid: starts with number
		{"1dev", true, "must start with a letter"},
		{"123", true, "must start with a letter"},

		// Invalid: spaces
		{"my container", true, "cannot contain spaces"},

		// Invalid: special characters
		{"my_container", true, "cannot contain underscores"},
		{"my.container", true, "invalid characters"},
		{"my@container", true, "invalid characters"},

		// Invalid: hyphens at edges
		{"-dev", true, "invalid characters"},
		{"dev-", true, "cannot start or end with a hyphen"},

		// Invalid: consecutive hyphens
		{"dev--test", true, "consecutive hyphens"},

		// Invalid: reserved
		{"list", true, "reserved name"},
		{"delete", true, "reserved name"},
		{"create", true, "reserved name"},
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

func TestValidateContainerName_TooLong(t *testing.T) {
	// Create a name that's exactly at the limit
	maxName := "a" + strings.Repeat("b", MaxContainerNameLength-1)
	if err := ValidateContainerName(maxName); err != nil {
		t.Errorf("name at max length should be valid: %v", err)
	}

	// Create a name that's one over the limit
	tooLong := maxName + "c"
	err := ValidateContainerName(tooLong)
	if err == nil {
		t.Error("expected error for name over max length")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("expected 'too long' error, got: %v", err)
	}
}

func TestValidateFullContainerName(t *testing.T) {
	tests := []struct {
		project   string
		container string
		wantErr   bool
		errMsg    string
	}{
		// Valid combinations
		{"", "dev", false, ""},
		{"myproject", "dev", false, ""},
		{"project", "container", false, ""},

		// Invalid container name
		{"project", "1dev", true, "must start with a letter"},
		{"project", "", true, "cannot be empty"},

		// Combined length too long
		{strings.Repeat("a", 30), strings.Repeat("b", 35), true, "too long"},
	}

	for _, tt := range tests {
		name := tt.project + "/" + tt.container
		t.Run(name, func(t *testing.T) {
			err := ValidateFullContainerName(tt.project, tt.container)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFullContainerName(%q, %q) error = %v, wantErr %v",
					tt.project, tt.container, err, tt.wantErr)
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
		t.Run(string(rune(tt.port)), func(t *testing.T) {
			err := ValidatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%d) error = %v, wantErr %v",
					tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePorts(t *testing.T) {
	tests := []struct {
		name    string
		ports   []int
		wantErr bool
		errMsg  string
	}{
		{"empty", []int{}, false, ""},
		{"single valid", []int{8080}, false, ""},
		{"multiple valid", []int{3000, 8000, 5432}, false, ""},
		{"invalid port", []int{8080, 99999}, true, "invalid port"},
		{"duplicate", []int{8080, 3000, 8080}, true, "duplicate"},
		{"zero", []int{0}, true, "invalid port"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePorts(tt.ports)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePorts(%v) error = %v, wantErr %v",
					tt.ports, err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
				}
			}
		})
	}
}

// ValidateSourcePath tests

func TestValidateSourcePath_Valid(t *testing.T) {
	tmpDir := t.TempDir()

	resolvedPath, warning, err := ValidateSourcePath(tmpDir)
	if err != nil {
		t.Errorf("ValidateSourcePath(%q) unexpected error: %v", tmpDir, err)
	}
	if warning != "" {
		t.Errorf("ValidateSourcePath(%q) unexpected warning: %s", tmpDir, warning)
	}
	if resolvedPath == "" {
		t.Error("ValidateSourcePath should return resolved path")
	}
	// The resolved path should be a valid absolute path
	if !filepath.IsAbs(resolvedPath) {
		t.Errorf("resolved path should be absolute, got: %s", resolvedPath)
	}
}

func TestValidateSourcePath_NotExists(t *testing.T) {
	nonExistent := "/path/that/does/not/exist/anywhere"

	_, _, err := ValidateSourcePath(nonExistent)
	if err == nil {
		t.Error("ValidateSourcePath should return error for non-existent path")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %v", err)
	}
}

func TestValidateSourcePath_IsFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testfile.txt")

	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, _, err := ValidateSourcePath(tmpFile)
	if err == nil {
		t.Error("ValidateSourcePath should return error for file path")
	}
	if !strings.Contains(err.Error(), "must be a directory") {
		t.Errorf("expected 'must be a directory' error, got: %v", err)
	}
}

func TestValidateSourcePath_BlockedRoot(t *testing.T) {
	_, _, err := ValidateSourcePath("/")
	if err == nil {
		t.Error("ValidateSourcePath should return error for root path")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %v", err)
	}
}

func TestValidateSourcePath_BlockedEtc(t *testing.T) {
	_, _, err := ValidateSourcePath("/etc")
	if err == nil {
		t.Error("ValidateSourcePath should return error for /etc")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %v", err)
	}
}

func TestValidateSourcePath_BlockedPattern(t *testing.T) {
	// Create a temp directory with .ssh suffix to test pattern blocking
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")

	if err := os.Mkdir(sshDir, 0755); err != nil {
		t.Fatalf("failed to create .ssh directory: %v", err)
	}

	_, _, err := ValidateSourcePath(sshDir)
	if err == nil {
		t.Error("ValidateSourcePath should return error for .ssh pattern")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %v", err)
	}
}

func TestValidateSourcePath_RiskyPath(t *testing.T) {
	// Test /tmp which is in RiskyHostPaths
	// Note: /tmp must exist on the system for this test to work
	if _, err := os.Stat("/tmp"); os.IsNotExist(err) {
		t.Skip("/tmp does not exist on this system")
	}

	resolvedPath, warning, err := ValidateSourcePath("/tmp")
	if err != nil {
		t.Errorf("ValidateSourcePath(/tmp) should not return error, got: %v", err)
	}
	if warning == "" {
		t.Error("ValidateSourcePath(/tmp) should return warning for risky path")
	}
	if !strings.Contains(warning, "risky") {
		t.Errorf("expected warning containing 'risky', got: %s", warning)
	}
	if resolvedPath == "" {
		t.Error("ValidateSourcePath should return resolved path even for risky paths")
	}
}

func TestValidateSourcePath_ResolvesSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real directory
	realDir := filepath.Join(tmpDir, "realdir")
	if err := os.Mkdir(realDir, 0755); err != nil {
		t.Fatalf("failed to create real directory: %v", err)
	}

	// Create a symlink pointing to the real directory
	symlinkPath := filepath.Join(tmpDir, "symlink")
	if err := os.Symlink(realDir, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	resolvedPath, _, err := ValidateSourcePath(symlinkPath)
	if err != nil {
		t.Errorf("ValidateSourcePath(%q) unexpected error: %v", symlinkPath, err)
	}

	// The resolved path should be the real directory, not the symlink
	if resolvedPath != realDir {
		t.Errorf("expected resolved path %q, got %q", realDir, resolvedPath)
	}
}

// ValidateContainerPath tests

func TestValidateContainerPath_Valid(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"simple path", "/home/user"},
		{"root path", "/data"},
		{"nested path", "/opt/myapp/data"},
		{"with numbers", "/home/user123/data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContainerPath(tt.path)
			if err != nil {
				t.Errorf("ValidateContainerPath(%q) unexpected error: %v", tt.path, err)
			}
		})
	}
}

func TestValidateContainerPath_RelativePath(t *testing.T) {
	relativePaths := []string{
		"data",
		"./data",
		"home/user",
	}

	for _, path := range relativePaths {
		t.Run(path, func(t *testing.T) {
			err := ValidateContainerPath(path)
			if err == nil {
				t.Errorf("ValidateContainerPath(%q) should return error for relative path", path)
			}
			if !strings.Contains(err.Error(), "must be absolute") {
				t.Errorf("expected 'must be absolute' error, got: %v", err)
			}
		})
	}
}

func TestValidateContainerPath_Traversal(t *testing.T) {
	// Note: filepath.Clean resolves .. in absolute paths, so /home/../etc becomes /etc
	// The implementation allows paths with .. as long as they resolve to allowed paths after cleaning.
	// The .. check only triggers for paths where .. remains after cleaning (relative paths).
	// Since relative paths are already rejected by the absolute path check, this test verifies
	// that the cleaned path is validated against blocked paths.

	// This path resolves to /proc after cleaning, which is blocked
	err := ValidateContainerPath("/home/../proc")
	if err == nil {
		t.Error("ValidateContainerPath(/home/../proc) should return error (resolves to blocked /proc)")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %v", err)
	}

	// This path with .. resolves to /etc which is not blocked in container paths
	err = ValidateContainerPath("/home/../etc/passwd")
	if err != nil {
		t.Errorf("ValidateContainerPath(/home/../etc/passwd) should succeed (resolves to /etc/passwd), got: %v", err)
	}
}

func TestValidateContainerPath_ControlChars(t *testing.T) {
	controlCharPaths := []struct {
		name string
		path string
	}{
		{"newline", "/home/user\n/data"},
		{"null byte", "/home/user\x00/data"},
		{"carriage return", "/home/user\r/data"},
		{"tab", "/home/user\t/data"},
	}

	for _, tt := range controlCharPaths {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContainerPath(tt.path)
			if err == nil {
				t.Errorf("ValidateContainerPath should return error for path with control chars")
			}
			if !strings.Contains(err.Error(), "control characters") {
				t.Errorf("expected 'control characters' error, got: %v", err)
			}
		})
	}
}

func TestValidateContainerPath_TooLong(t *testing.T) {
	longPath := "/" + strings.Repeat("a", MaxContainerPathLength+1)

	err := ValidateContainerPath(longPath)
	if err == nil {
		t.Error("ValidateContainerPath should return error for path over 4096 chars")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("expected 'too long' error, got: %v", err)
	}
}

func TestValidateContainerPath_BlockedRoot(t *testing.T) {
	err := ValidateContainerPath("/")
	if err == nil {
		t.Error("ValidateContainerPath should return error for root path")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %v", err)
	}
}

func TestValidateContainerPath_BlockedProc(t *testing.T) {
	procPaths := []string{
		"/proc",
	}

	for _, path := range procPaths {
		t.Run(path, func(t *testing.T) {
			err := ValidateContainerPath(path)
			if err == nil {
				t.Errorf("ValidateContainerPath(%q) should return error for /proc", path)
			}
			if !strings.Contains(err.Error(), "not allowed") {
				t.Errorf("expected 'not allowed' error, got: %v", err)
			}
		})
	}
}

// ValidateMountName tests

func TestValidateMountName_Valid(t *testing.T) {
	validNames := []string{
		"data",
		"my-mount",
		"MyMount",
		"mount1",
		"a",
		"a1b2c3",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateMountName(name)
			if err != nil {
				t.Errorf("ValidateMountName(%q) unexpected error: %v", name, err)
			}
		})
	}
}

func TestValidateMountName_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		errMsg string
	}{
		{"starts with number", "1mount", "must start with a letter"},
		{"special chars", "my@mount", "invalid characters"},
		{"underscore", "my_mount", "underscores"},
		{"space", "my mount", "spaces"},
		{"leading hyphen", "-mount", "invalid characters"},
		{"trailing hyphen", "mount-", "cannot start or end with a hyphen"},
		{"consecutive hyphens", "my--mount", "consecutive hyphens"},
		{"empty", "", "cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMountName(tt.input)
			if err == nil {
				t.Errorf("ValidateMountName(%q) should return error", tt.input)
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got: %v", tt.errMsg, err)
			}
		})
	}
}

// GenerateMountName tests

func TestGenerateMountName_Simple(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/home/user/projects", "projects"},
		{"/data", "data"},
		{"/opt/myapp", "myapp"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := GenerateMountName(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateMountName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateMountName_SpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"underscore", "/home/my_project", "my-project"},
		{"dot", "/home/my.project", "my-project"},
		{"space", "/home/my project", "my-project"},
		{"multiple special", "/home/my@project#1", "my-project-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateMountName(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateMountName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateMountName_StartsWithNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/home/123project", "mount-123project"},
		{"/data/1test", "mount-1test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := GenerateMountName(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateMountName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateMountName_TooLong(t *testing.T) {
	// Create a path with a very long base name
	longName := strings.Repeat("a", 100)
	input := "/home/" + longName

	result := GenerateMountName(input)
	if len(result) > MaxMountNameLength {
		t.Errorf("GenerateMountName should truncate to %d chars, got %d chars: %s",
			MaxMountNameLength, len(result), result)
	}
	if len(result) != MaxMountNameLength {
		t.Errorf("GenerateMountName should truncate to exactly %d chars, got %d chars: %s",
			MaxMountNameLength, len(result), result)
	}
}
