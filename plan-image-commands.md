# Plan: Image Management Commands

## New Commands

### 1. `lxc-dev-manager images`

List all local images (custom and cached).

```bash
$ lxc-dev-manager images

NAME              ALIAS              SIZE      CREATED
my-base-image     my-base-image      423MB     2024-01-14 10:30
dev-with-docker   dev-with-docker    1.2GB     2024-01-13 15:00
ubuntu:24.04      -                  512MB     2024-01-10 (cached)
```

**LXC command**: `lxc image list --format=csv`

**Flags**:
- `--all` / `-a`: Show cached images too (default: only aliased/custom)

### 2. `lxc-dev-manager image delete <name>`

Delete a local image.

```bash
$ lxc-dev-manager image delete my-base-image
Image 'my-base-image' deleted
```

**LXC command**: `lxc image delete <alias>`

**Behavior**:
- Confirm before delete (unless `--force`)
- Error if image is in use by containers

### 3. `lxc-dev-manager image rename <old> <new>`

Rename an image alias.

```bash
$ lxc-dev-manager image rename my-base-image production-base
Image renamed: my-base-image → production-base
```

**LXC commands**:
```bash
lxc image alias create <new> <fingerprint>
lxc image alias delete <old>
```

**Note**: LXC doesn't have native rename, so we create new alias then delete old.

### 4. `lxc-dev-manager image info <name>` (optional)

Show detailed image info.

```bash
$ lxc-dev-manager image info my-base-image

Name:         my-base-image
Size:         423MB
Created:      2024-01-14 10:30:45
Architecture: x86_64
OS:           ubuntu
Release:      24.04
Fingerprint:  abc123...
```

## Implementation

### New file: `cmd/image.go`

```go
// Parent command for image subcommands
var imageCmd = &cobra.Command{
    Use:   "image",
    Short: "Manage images",
}

// Subcommands
var imageListCmd    // lxc-dev-manager image list (or just 'images')
var imageDeleteCmd  // lxc-dev-manager image delete <name>
var imageRenameCmd  // lxc-dev-manager image rename <old> <new>
var imageInfoCmd    // lxc-dev-manager image info <name>
```

### New LXC wrapper functions

```go
// internal/lxc/lxc.go

type ImageInfo struct {
    Alias        string
    Fingerprint  string
    Size         int64
    CreatedAt    time.Time
    Architecture string
    Description  string
}

func ListImages(all bool) ([]ImageInfo, error)
func DeleteImage(alias string) error
func RenameImage(oldAlias, newAlias string) error
func GetImageInfo(alias string) (*ImageInfo, error)
func GetImageFingerprint(alias string) (string, error)
```

## Command Structure Options

**Option A**: Subcommands under `image`
```
lxc-dev-manager image list
lxc-dev-manager image delete <name>
lxc-dev-manager image rename <old> <new>
```

**Option B**: Top-level `images` + subcommands
```
lxc-dev-manager images              # list
lxc-dev-manager image delete <name>
lxc-dev-manager image rename <old> <new>
```

**Option C**: All top-level (like Docker)
```
lxc-dev-manager images              # list
lxc-dev-manager rmi <name>          # delete (docker style)
lxc-dev-manager tag <old> <new>     # rename (docker style)
```

## Recommendation

**Option B** - familiar pattern:
- `images` for quick listing (common operation)
- `image <verb>` for other operations

## Implementation Order

1. [ ] Add LXC wrapper functions for images
2. [ ] Implement `images` (list) command
3. [ ] Implement `image delete` command
4. [ ] Implement `image rename` command
5. [ ] Add tests
6. [ ] Optional: `image info` command

## Example Session

```bash
# List images
$ lxc-dev-manager images
NAME              SIZE      CREATED
my-base-image     423MB     2024-01-14

# Create new container from image
$ lxc-dev-manager create dev2 my-base-image

# Rename image
$ lxc-dev-manager image rename my-base-image prod-base

# Delete unused image
$ lxc-dev-manager image delete old-image
```
