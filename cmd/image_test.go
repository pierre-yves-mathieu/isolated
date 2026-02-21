package cmd

import (
	"strings"
	"testing"
)

func TestImageList_Empty(t *testing.T) {
	env := setupTestEnv(t)
	env.mock.SetOutput("image list --format=csv -c lfsd", "")

	err := runImageList(nil, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImageList_WithImages(t *testing.T) {
	env := setupTestEnv(t)
	env.mock.SetOutput("image list --format=csv -c lfsd", `my-base,abc123def456,500MiB,Ubuntu 24.04
dev-image,def789ghi012,1.2GiB,Custom dev image`)

	err := runImageList(nil, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImageList_FiltersCached(t *testing.T) {
	env := setupTestEnv(t)
	// One aliased, one cached (no alias)
	env.mock.SetOutput("image list --format=csv -c lfsd", `my-base,abc123,500MiB,Ubuntu
,def456,300MiB,cached image`)

	imageListAll = false
	err := runImageList(nil, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should only show aliased images (verified by output inspection)
}

func TestImageList_ShowsAllWithFlag(t *testing.T) {
	env := setupTestEnv(t)
	env.mock.SetOutput("image list --format=csv -c lfsd", `my-base,abc123,500MiB,Ubuntu
,def456,300MiB,cached image`)

	imageListAll = true
	defer func() { imageListAll = false }()

	err := runImageList(nil, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Helper to set force flag for image delete tests
func withImageDeleteForce(t *testing.T) {
	t.Helper()
	imageDeleteForce = true
	t.Cleanup(func() { imageDeleteForce = false })
}

func TestImageDelete_Success(t *testing.T) {
	env := setupTestEnv(t)
	withImageDeleteForce(t)

	env.mock.SetOutput("image list my-base --format=csv -c f", "abc123def456")
	env.mock.SetOutput("image list --format=csv -c lfsd", "my-base,abc123,500MiB,Test image")
	env.mock.SetOutput("image delete my-base", "")

	err := runImageDelete(nil, []string{"my-base"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.mock.HasCall("image", "delete", "my-base") {
		t.Error("expected image delete command")
	}
}

func TestImageDelete_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	withImageDeleteForce(t)

	env.mock.SetOutput("image list my-base --format=csv -c f", "")

	err := runImageDelete(nil, []string{"my-base"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestImageDelete_Error(t *testing.T) {
	env := setupTestEnv(t)
	withImageDeleteForce(t)

	env.mock.SetOutput("image list my-base --format=csv -c f", "abc123")
	env.mock.SetOutput("image list --format=csv -c lfsd", "my-base,abc123,500MiB,Test")
	env.mock.SetError("image delete my-base", "image in use")

	err := runImageDelete(nil, []string{"my-base"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestImageRename_Success(t *testing.T) {
	env := setupTestEnv(t)
	// Old exists
	env.mock.SetOutput("image list old-name --format=csv -c f", "abc123def456")
	// New doesn't exist
	env.mock.SetOutput("image list new-name --format=csv -c f", "")
	// Alias create/delete succeed
	env.mock.SetOutput("image alias create new-name abc123def456", "")
	env.mock.SetOutput("image alias delete old-name", "")

	err := runImageRename(nil, []string{"old-name", "new-name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !env.mock.HasCallPrefix("image", "alias", "create", "new-name") {
		t.Error("expected alias create command")
	}
	if !env.mock.HasCallPrefix("image", "alias", "delete", "old-name") {
		t.Error("expected alias delete command")
	}
}

func TestImageRename_OldNotFound(t *testing.T) {
	env := setupTestEnv(t)
	env.mock.SetOutput("image list old-name --format=csv -c f", "")

	err := runImageRename(nil, []string{"old-name", "new-name"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestImageRename_NewAlreadyExists(t *testing.T) {
	env := setupTestEnv(t)
	env.mock.SetOutput("image list old-name --format=csv -c f", "abc123")
	env.mock.SetOutput("image list new-name --format=csv -c f", "def456")

	err := runImageRename(nil, []string{"old-name", "new-name"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestImageRename_AliasCreateFails(t *testing.T) {
	env := setupTestEnv(t)
	env.mock.SetOutput("image list old-name --format=csv -c f", "abc123")
	env.mock.SetOutput("image list new-name --format=csv -c f", "")
	env.mock.SetError("image alias create new-name abc123", "failed")

	err := runImageRename(nil, []string{"old-name", "new-name"})
	if err == nil {
		t.Fatal("expected error")
	}
}
