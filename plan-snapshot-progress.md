# Plan: Snapshot Progress Reporting

## Current Behavior

```
Stopping container 'dev1'...
Creating image 'my-image' from container 'dev1'...
[hangs with no feedback until complete]
Restarting container 'dev1'...
```

The `lxc publish` command can take minutes for large containers with no visible progress.

## Options

### Option 1: Stream LXC Output (Recommended)

`lxc publish` outputs progress by default (e.g., `Transferring image: 45%`).

**Approach**: Instead of capturing output, stream it directly to stdout.

```go
cmd := exec.Command("lxc", "publish", name, "--alias", imageName)
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
err := cmd.Run()
```

**Pros**:
- Simple, uses LXC's built-in progress
- Accurate percentage from LXC

**Cons**:
- Less control over formatting

### Option 2: Background Polling

Start `lxc publish` in background, poll `lxc operation list` for progress.

```go
// Start publish
cmd := exec.Command("lxc", "publish", name, "--alias", imageName)
cmd.Start()

// Poll operation status
for {
    ops := exec.Command("lxc", "operation", "list", "--format=json")
    // Parse JSON, find our operation, show progress
    time.Sleep(500ms)
}
```

**Pros**:
- Full control over display format
- Can show spinner, progress bar, ETA

**Cons**:
- More complex
- Requires parsing operation JSON

### Option 3: Step-Based Progress

Show meaningful steps even without percentage:

```
[1/4] Stopping container...           ✓
[2/4] Creating filesystem snapshot... ✓
[3/4] Compressing image...            ⠋ (this may take a few minutes)
[4/4] Restarting container...
```

**Approach**: Use intermediate LXC commands:
1. `lxc stop`
2. `lxc snapshot` (instant with ZFS)
3. `lxc publish container/snapshot` (slow part)
4. `lxc start`

With ZFS, step 2 is instant. Step 3 is where time is spent.

**Pros**:
- Clear progress indication
- Works even if LXC doesn't output progress

**Cons**:
- Still no percentage for compress step

### Option 4: Hybrid (Best UX)

Combine step-based with streamed output:

```
[1/4] Stopping container...           ✓
[2/4] Creating snapshot...            ✓
[3/4] Publishing image...
      Transferring image: 67% (142MB/212MB)
[4/4] Restarting container...         ✓
```

## Recommended Implementation

**Option 4 (Hybrid)** provides the best user experience:

### Changes Required

1. **Update `internal/lxc/lxc.go`**:
   - Add `PublishWithProgress(name, alias string, stdout, stderr io.Writer) error`
   - Add `Snapshot(container, snapshotName string) error`
   - Add `PublishSnapshot(container, snapshotName, alias string, stdout, stderr io.Writer) error`

2. **Update `cmd/snapshot.go`**:
   - Show numbered steps
   - Stream LXC output during publish step
   - Add spinner for non-streaming steps

3. **Optional: Add progress bar package**:
   - `github.com/schollz/progressbar/v3` or
   - `github.com/cheggaaa/pb/v3`
   - Or simple custom spinner

### Implementation Steps

1. [ ] Add `Snapshot` function to create named snapshot
2. [ ] Add `PublishWithProgress` that streams to stdout
3. [ ] Refactor `runSnapshot` to use step-based approach
4. [ ] Add simple spinner for waiting steps
5. [ ] Test with real container
6. [ ] Add tests for new functions

### Example Output After Implementation

```
$ lxc-dev-manager snapshot dev1 my-base

[1/4] Stopping container 'dev1'...
      ✓ Stopped

[2/4] Creating snapshot...
      ✓ Snapshot created (instant with ZFS)

[3/4] Publishing image 'my-base'...
      Packing filesystem: 100% (423.12MB/s)
      Creating image: 100%
      ✓ Image published

[4/4] Restarting container 'dev1'...
      ✓ Started (10.186.84.216)

Image 'my-base' created successfully!
Create new containers: lxc-dev-manager create <name> my-base
```

## Decision Needed

Which option do you prefer?
1. **Simple**: Just stream LXC output (Option 1)
2. **Polished**: Hybrid with steps + streaming (Option 4)
