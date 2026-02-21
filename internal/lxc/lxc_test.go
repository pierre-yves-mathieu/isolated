package lxc

import (
	"errors"
	"strings"
	"testing"
)

func setupMock(t *testing.T) *MockExecutor {
	t.Helper()
	mock := NewMockExecutor()
	SetExecutor(mock)
	t.Cleanup(func() {
		ResetExecutor()
	})
	return mock
}

func TestGetIP_ParsesOutput(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("list dev1 -c4 -f csv", "10.10.10.45 (eth0)")

	ip, err := GetIP("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "10.10.10.45" {
		t.Errorf("expected 10.10.10.45, got %s", ip)
	}
}

func TestGetIP_NoIP(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("list dev1 -c4 -f csv", "")

	_, err := GetIP("dev1")
	if err == nil {
		t.Fatal("expected error for no IP")
	}
}

func TestGetIP_MultipleIPs(t *testing.T) {
	mock := setupMock(t)
	// Sometimes LXC returns multiple IPs separated by newlines
	mock.SetOutput("list dev1 -c4 -f csv", "10.10.10.45 (eth0)\n192.168.1.100 (eth1)")

	ip, err := GetIP("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return first IP
	if ip != "10.10.10.45" {
		t.Errorf("expected 10.10.10.45, got %s", ip)
	}
}

func TestGetIP_CommandError(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("list dev1 -c4 -f csv", "container not found")

	_, err := GetIP("dev1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetStatus_Running(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("list dev1 -cs -f csv", "RUNNING")

	status, err := GetStatus("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "RUNNING" {
		t.Errorf("expected RUNNING, got %s", status)
	}
}

func TestGetStatus_Stopped(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("list dev1 -cs -f csv", "STOPPED")

	status, err := GetStatus("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "STOPPED" {
		t.Errorf("expected STOPPED, got %s", status)
	}
}

func TestGetStatus_CommandError(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("list dev1 -cs -f csv", "failed")

	_, err := GetStatus("dev1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListAll_ParsesCSV(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("list -c ns4 -f csv", `dev1,RUNNING,10.10.10.45 (eth0)
dev2,STOPPED,
dev3,RUNNING,10.10.10.46 (eth0)`)

	containers, err := ListAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(containers) != 3 {
		t.Fatalf("expected 3 containers, got %d", len(containers))
	}

	if containers[0].Name != "dev1" || containers[0].Status != "RUNNING" || containers[0].IP != "10.10.10.45" {
		t.Errorf("unexpected container 0: %+v", containers[0])
	}
	if containers[1].Name != "dev2" || containers[1].Status != "STOPPED" {
		t.Errorf("unexpected container 1: %+v", containers[1])
	}
}

func TestListAll_Empty(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("list -c ns4 -f csv", "")

	containers, err := ListAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(containers) != 0 {
		t.Errorf("expected 0 containers, got %d", len(containers))
	}
}

func TestListAll_CommandError(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("list -c ns4 -f csv", "permission denied")

	_, err := ListAll()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExists_True(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("info dev1", "Name: dev1\n...")

	if !Exists("dev1") {
		t.Error("expected Exists to return true")
	}
}

func TestExists_False(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("info dev1", "not found")

	if Exists("dev1") {
		t.Error("expected Exists to return false")
	}
}

func TestLaunch_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("launch ubuntu:24.04 dev1", "Creating dev1...")

	err := Launch("dev1", "ubuntu:24.04")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("launch", "ubuntu:24.04", "dev1") {
		t.Error("expected launch command to be called")
	}
}

func TestLaunch_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("launch ubuntu:24.04 dev1", "image not found")

	err := Launch("dev1", "ubuntu:24.04")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStart_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("start dev1", "")

	err := Start("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("start", "dev1") {
		t.Error("expected start command to be called")
	}
}

func TestStart_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("start dev1", "container not found")

	err := Start("dev1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStop_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("stop dev1 --timeout=5", "")

	err := Stop("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("stop", "dev1", "--timeout=5") {
		t.Error("expected stop command to be called")
	}
}

func TestStop_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("stop dev1 --timeout=5", "container not found")

	err := Stop("dev1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("delete dev1 --force", "")

	err := Delete("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("delete", "dev1", "--force") {
		t.Error("expected delete command to be called")
	}
}

func TestDelete_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("delete dev1 --force", "container not found")

	err := Delete("dev1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPublish_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("publish dev1 --alias my-image", "")

	err := Publish("dev1", "my-image")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("publish", "dev1", "--alias", "my-image") {
		t.Error("expected publish command to be called")
	}
}

func TestPublish_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("publish dev1 --alias my-image", "container not found")

	err := Publish("dev1", "my-image")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConfigSet_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("config set dev1 security.nesting true", "")

	err := ConfigSet("dev1", "security.nesting", "true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("config", "set", "dev1", "security.nesting", "true") {
		t.Error("expected config set command to be called")
	}
}

func TestEnableNesting_Success(t *testing.T) {
	mock := setupMock(t)
	// All config commands succeed
	mock.DefaultResponse = MockResponse{Output: []byte("")}

	err := EnableNesting("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have called config set for nesting options
	if !mock.HasCallPrefix("config", "set", "dev1", "security.nesting") {
		t.Error("expected nesting config to be set")
	}
}

func TestEnableNesting_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("config set dev1 security.nesting true", "permission denied")

	err := EnableNesting("dev1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExec_Success(t *testing.T) {
	mock := setupMock(t)
	mock.DefaultResponse = MockResponse{Output: []byte("output")}

	err := Exec("dev1", "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("exec", "dev1", "--", "echo", "hello") {
		t.Error("expected exec command to be called")
	}
}

func TestExec_Error(t *testing.T) {
	mock := setupMock(t)
	mock.DefaultResponse = MockResponse{Err: errors.New("command failed")}

	err := Exec("dev1", "false")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMockExecutor_CallTracking(t *testing.T) {
	mock := NewMockExecutor()

	mock.Run("info", "dev1")
	mock.Run("list", "-c", "ns4")
	mock.RunCombined("start", "dev1")

	if mock.CallCount() != 3 {
		t.Errorf("expected 3 calls, got %d", mock.CallCount())
	}

	last := mock.LastCall()
	if len(last.Args) != 2 || last.Args[0] != "start" {
		t.Errorf("unexpected last call: %v", last)
	}

	if !mock.HasCall("info", "dev1") {
		t.Error("expected HasCall to find 'info dev1'")
	}

	if mock.HasCall("nonexistent", "command") {
		t.Error("HasCall should return false for non-existent call")
	}
}

func TestMockExecutor_Reset(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetOutput("test", "output")
	mock.Run("test")

	mock.Reset()

	if mock.CallCount() != 0 {
		t.Error("calls should be cleared after reset")
	}
	if len(mock.Responses) != 0 {
		t.Error("responses should be cleared after reset")
	}
}

// Tests for Snapshot function
func TestSnapshot_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("snapshot dev1 snap1", "")

	err := Snapshot("dev1", "snap1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("snapshot", "dev1", "snap1") {
		t.Error("expected snapshot command to be called")
	}
}

func TestSnapshot_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("snapshot dev1 snap1", "container not found")

	err := Snapshot("dev1", "snap1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to create snapshot") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Tests for DeleteSnapshot function
func TestDeleteSnapshot_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("delete dev1/snap1", "")

	err := DeleteSnapshot("dev1", "snap1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("delete", "dev1/snap1") {
		t.Error("expected delete snapshot command to be called")
	}
}

func TestDeleteSnapshot_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("delete dev1/snap1", "snapshot not found")

	err := DeleteSnapshot("dev1", "snap1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to delete snapshot") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Tests for ListImages function
func TestListImages_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image list --format=csv -c lfsd", `my-base,abc123def456,500MiB,Ubuntu 24.04
dev-image,def789ghi012,1.2GiB,Custom dev image`)

	images, err := ListImages(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}

	if images[0].Alias != "my-base" {
		t.Errorf("expected alias 'my-base', got '%s'", images[0].Alias)
	}
	if images[0].Fingerprint != "abc123def456" {
		t.Errorf("expected fingerprint 'abc123def456', got '%s'", images[0].Fingerprint)
	}
	if images[0].Size != "500MiB" {
		t.Errorf("expected size '500MiB', got '%s'", images[0].Size)
	}
	if images[0].Description != "Ubuntu 24.04" {
		t.Errorf("expected description 'Ubuntu 24.04', got '%s'", images[0].Description)
	}
}

func TestListImages_Empty(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image list --format=csv -c lfsd", "")

	images, err := ListImages(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(images) != 0 {
		t.Errorf("expected 0 images, got %d", len(images))
	}
}

func TestListImages_FiltersCachedImages(t *testing.T) {
	mock := setupMock(t)
	// One aliased, one cached (no alias)
	mock.SetOutput("image list --format=csv -c lfsd", `my-base,abc123,500MiB,Ubuntu
,def456,300MiB,cached image`)

	images, err := ListImages(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only return aliased image when all=false
	if len(images) != 1 {
		t.Errorf("expected 1 image (filtered), got %d", len(images))
	}
	if images[0].Alias != "my-base" {
		t.Errorf("expected 'my-base', got '%s'", images[0].Alias)
	}
}

func TestListImages_ShowsAllWithFlag(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image list --format=csv -c lfsd", `my-base,abc123,500MiB,Ubuntu
,def456,300MiB,cached image`)

	images, err := ListImages(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return all images when all=true
	if len(images) != 2 {
		t.Errorf("expected 2 images, got %d", len(images))
	}
}

func TestListImages_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("image list --format=csv -c lfsd", "permission denied")

	_, err := ListImages(false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListImages_PartialCSV(t *testing.T) {
	mock := setupMock(t)
	// Only 3 columns (no description)
	mock.SetOutput("image list --format=csv -c lfsd", "my-base,abc123,500MiB")

	images, err := ListImages(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	if images[0].Description != "" {
		t.Errorf("expected empty description, got '%s'", images[0].Description)
	}
}

// Tests for DeleteImage function
func TestDeleteImage_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image delete my-base", "")

	err := DeleteImage("my-base")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("image", "delete", "my-base") {
		t.Error("expected image delete command to be called")
	}
}

func TestDeleteImage_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("image delete my-base", "image in use")

	err := DeleteImage("my-base")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to delete image") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Tests for GetImageFingerprint function
func TestGetImageFingerprint_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image list my-base --format=csv -c f", "abc123def456")

	fp, err := GetImageFingerprint("my-base")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fp != "abc123def456" {
		t.Errorf("expected 'abc123def456', got '%s'", fp)
	}
}

func TestGetImageFingerprint_NotFound(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image list my-base --format=csv -c f", "")

	_, err := GetImageFingerprint("my-base")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGetImageFingerprint_MultipleLines(t *testing.T) {
	mock := setupMock(t)
	// Sometimes multiple fingerprints may be returned
	mock.SetOutput("image list my-base --format=csv -c f", "abc123\ndef456")

	fp, err := GetImageFingerprint("my-base")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return first fingerprint
	if fp != "abc123" {
		t.Errorf("expected 'abc123', got '%s'", fp)
	}
}

func TestGetImageFingerprint_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("image list my-base --format=csv -c f", "permission denied")

	_, err := GetImageFingerprint("my-base")
	if err == nil {
		t.Fatal("expected error")
	}
}

// Tests for RenameImage function
func TestRenameImage_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image list old-name --format=csv -c f", "abc123def456")
	mock.SetOutput("image alias create new-name abc123def456", "")
	mock.SetOutput("image alias delete old-name", "")

	err := RenameImage("old-name", "new-name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCallPrefix("image", "alias", "create", "new-name") {
		t.Error("expected alias create command")
	}
	if !mock.HasCallPrefix("image", "alias", "delete", "old-name") {
		t.Error("expected alias delete command")
	}
}

func TestRenameImage_OldNotFound(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image list old-name --format=csv -c f", "")

	err := RenameImage("old-name", "new-name")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRenameImage_CreateAliasFails(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image list old-name --format=csv -c f", "abc123")
	mock.SetError("image alias create new-name abc123", "alias exists")

	err := RenameImage("old-name", "new-name")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to create new alias") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRenameImage_DeleteAliasFails(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image list old-name --format=csv -c f", "abc123")
	mock.SetOutput("image alias create new-name abc123", "")
	mock.SetError("image alias delete old-name", "failed")

	err := RenameImage("old-name", "new-name")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to delete old alias") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Tests for ImageExists function
func TestImageExists_True(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image list my-image --format=csv -c f", "abc123")

	if !ImageExists("my-image") {
		t.Error("expected ImageExists to return true")
	}
}

func TestImageExists_False(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("image list my-image --format=csv -c f", "")

	if ImageExists("my-image") {
		t.Error("expected ImageExists to return false")
	}
}

func TestImageExists_OnError(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("image list my-image --format=csv -c f", "error")

	if ImageExists("my-image") {
		t.Error("expected ImageExists to return false on error")
	}
}

// Tests for Restore function
func TestRestore_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("restore dev1 snap1", "")

	err := Restore("dev1", "snap1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("restore", "dev1", "snap1") {
		t.Error("expected restore command to be called")
	}
}

func TestRestore_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("restore dev1 snap1", "snapshot not found")

	err := Restore("dev1", "snap1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to restore snapshot") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Tests for SnapshotExists function
func TestSnapshotExists_True(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("info dev1/snap1", "Name: snap1")

	if !SnapshotExists("dev1", "snap1") {
		t.Error("expected SnapshotExists to return true")
	}
}

func TestSnapshotExists_False(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("info dev1/snap1", "not found")

	if SnapshotExists("dev1", "snap1") {
		t.Error("expected SnapshotExists to return false")
	}
}

// Tests for ListSnapshots function
func TestListSnapshots_Success(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("query /1.0/instances/dev1/snapshots",
		`["/1.0/instances/dev1/snapshots/initial-state","/1.0/instances/dev1/snapshots/checkpoint"]`)

	snapshots, err := ListSnapshots("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
	if snapshots[0] != "initial-state" {
		t.Errorf("expected 'initial-state', got '%s'", snapshots[0])
	}
	if snapshots[1] != "checkpoint" {
		t.Errorf("expected 'checkpoint', got '%s'", snapshots[1])
	}
}

func TestListSnapshots_Empty(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("query /1.0/instances/dev1/snapshots", "[]")

	snapshots, err := ListSnapshots("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(snapshots))
	}
}

func TestListSnapshots_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("query /1.0/instances/dev1/snapshots", "container not found")

	_, err := ListSnapshots("dev1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListSnapshots_InvalidJSON(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("query /1.0/instances/dev1/snapshots", "not json")

	_, err := ListSnapshots("dev1")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// Tests for DeviceAdd function
func TestDeviceAdd(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("config device add dev1 repo disk source=/host/path path=/container/path", "")

	config := map[string]string{
		"source": "/host/path",
		"path":   "/container/path",
	}
	err := DeviceAdd("dev1", "repo", "disk", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCallPrefix("config", "device", "add", "dev1", "repo", "disk") {
		t.Error("expected device add command to be called")
	}
}

func TestDeviceAdd_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("config device add dev1 repo disk source=/host/path path=/container/path", "container not found")

	config := map[string]string{
		"source": "/host/path",
		"path":   "/container/path",
	}
	err := DeviceAdd("dev1", "repo", "disk", config)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to add device") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Tests for DeviceRemove function
func TestDeviceRemove(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("config device remove dev1 repo", "")

	err := DeviceRemove("dev1", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.HasCall("config", "device", "remove", "dev1", "repo") {
		t.Error("expected device remove command to be called")
	}
}

func TestDeviceRemove_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("config device remove dev1 repo", "device not found")

	err := DeviceRemove("dev1", "repo")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to remove device") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Tests for DeviceList function
func TestDeviceList(t *testing.T) {
	mock := setupMock(t)
	yamlOutput := `repo:
  type: disk
  source: /host/path
  path: /container/path`
	mock.SetOutput("config device show dev1", yamlOutput)

	devices, err := DeviceList("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}

	if devices[0].Name != "repo" {
		t.Errorf("expected device name 'repo', got '%s'", devices[0].Name)
	}
	if devices[0].Type != "disk" {
		t.Errorf("expected device type 'disk', got '%s'", devices[0].Type)
	}
	if devices[0].Config["source"] != "/host/path" {
		t.Errorf("expected source '/host/path', got '%s'", devices[0].Config["source"])
	}
	if devices[0].Config["path"] != "/container/path" {
		t.Errorf("expected path '/container/path', got '%s'", devices[0].Config["path"])
	}
}

func TestDeviceList_Empty(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("config device show dev1", "{}")

	devices, err := DeviceList("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestDeviceList_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("config device show dev1", "container not found")

	_, err := DeviceList("dev1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to list devices") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestDeviceList_MultipleDiskDevices(t *testing.T) {
	mock := setupMock(t)
	yamlOutput := `repo:
  type: disk
  source: /host/repo
  path: /container/repo
data:
  type: disk
  source: /host/data
  path: /container/data`
	mock.SetOutput("config device show dev1", yamlOutput)

	devices, err := DeviceList("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}

	// Find devices by name (order is not guaranteed with maps)
	deviceMap := make(map[string]DeviceInfo)
	for _, d := range devices {
		deviceMap[d.Name] = d
	}

	repo, ok := deviceMap["repo"]
	if !ok {
		t.Fatal("expected device 'repo' to exist")
	}
	if repo.Type != "disk" {
		t.Errorf("expected repo type 'disk', got '%s'", repo.Type)
	}
	if repo.Config["source"] != "/host/repo" {
		t.Errorf("expected repo source '/host/repo', got '%s'", repo.Config["source"])
	}

	data, ok := deviceMap["data"]
	if !ok {
		t.Fatal("expected device 'data' to exist")
	}
	if data.Type != "disk" {
		t.Errorf("expected data type 'disk', got '%s'", data.Type)
	}
	if data.Config["source"] != "/host/data" {
		t.Errorf("expected data source '/host/data', got '%s'", data.Config["source"])
	}
}

// Tests for DeviceExists function
func TestDeviceExists_True(t *testing.T) {
	mock := setupMock(t)
	yamlOutput := `repo:
  type: disk
  source: /host/path
  path: /container/path`
	mock.SetOutput("config device show dev1", yamlOutput)

	exists, err := DeviceExists("dev1", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected DeviceExists to return true")
	}
}

func TestDeviceExists_False(t *testing.T) {
	mock := setupMock(t)
	yamlOutput := `other:
  type: disk
  source: /host/path
  path: /container/path`
	mock.SetOutput("config device show dev1", yamlOutput)

	exists, err := DeviceExists("dev1", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected DeviceExists to return false")
	}
}

// Tests for IsPrivileged function
func TestIsPrivileged_True(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("config get dev1 security.privileged", "true")

	privileged, err := IsPrivileged("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !privileged {
		t.Error("expected IsPrivileged to return true")
	}
}

func TestIsPrivileged_False(t *testing.T) {
	mock := setupMock(t)
	mock.SetOutput("config get dev1 security.privileged", "false")

	privileged, err := IsPrivileged("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if privileged {
		t.Error("expected IsPrivileged to return false for 'false'")
	}

	// Also test empty value
	mock.SetOutput("config get dev1 security.privileged", "")
	privileged, err = IsPrivileged("dev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if privileged {
		t.Error("expected IsPrivileged to return false for empty string")
	}
}

func TestIsPrivileged_Error(t *testing.T) {
	mock := setupMock(t)
	mock.SetError("config get dev1 security.privileged", "container not found")

	_, err := IsPrivileged("dev1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to get privileged status") {
		t.Errorf("unexpected error message: %v", err)
	}
}
