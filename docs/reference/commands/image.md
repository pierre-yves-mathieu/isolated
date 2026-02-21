# Image Commands

Commands for creating and managing reusable images.

## image create

Create a reusable image from a container.

```bash
lxc-dev-manager image create <container> <image-name>
```

**Arguments**:
| Argument | Description |
|----------|-------------|
| `container` | Source container name |
| `image-name` | Name for the new image |

**Examples**:

```bash
# Create an image from the dev container
lxc-dev-manager image create dev nodejs-ready

# Create an image with a descriptive name
lxc-dev-manager image create dev python-ml-base
```

**Output**:
```
[1/4] Stopping container 'dev'...
      ✓ Stopped
[2/4] Creating snapshot...
      ✓ Snapshot created (instant with ZFS)
[3/4] Publishing image 'nodejs-ready'...

      Transferring image: 100% (312.45MB/s)

      ✓ Image published
[4/4] Restarting container 'dev'...
      ✓ Started (10.87.167.42)

Image 'nodejs-ready' created successfully!

Create new containers from it with:
  lxc-dev-manager container create <name> nodejs-ready
```

::: tip
The container is automatically stopped before creating the image and restarted afterward (if it was running).
:::

---

## image list

List local images.

```bash
lxc-dev-manager image list [--all]
```

**Aliases**: `images`

**Flags**:
| Flag | Short | Description |
|------|-------|-------------|
| `--all` | `-a` | Show all images including cached remote images |

**Examples**:

```bash
# List custom/snapshot images only
lxc-dev-manager image list
lxc-dev-manager images

# List all images including cached
lxc-dev-manager images --all
```

**Output**:
```
ALIAS                     FINGERPRINT    SIZE       DESCRIPTION
---------------------------------------------------------------------------
nodejs-ready              a1b2c3d4e5f6   1.2GB      Ubuntu 24.04 LTS
python-ml-base            f6e5d4c3b2a1   2.8GB      Ubuntu 24.04 LTS
```

---

## image delete

Delete a local image.

```bash
lxc-dev-manager image delete <name> [--force]
```

**Arguments**:
| Argument | Description |
|----------|-------------|
| `name` | Image alias or fingerprint |

**Flags**:
| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Skip confirmation prompt |

**Examples**:

```bash
# Interactive deletion
lxc-dev-manager image delete nodejs-ready

# Skip confirmation
lxc-dev-manager image delete nodejs-ready --force
```

**Output**:
```
Image: nodejs-ready
  Size: 1.2GB
  Description: Ubuntu 24.04 LTS

Are you sure you want to delete image 'nodejs-ready'? [y/N]: y
Deleting image 'nodejs-ready'...
Image 'nodejs-ready' deleted
```

---

## image rename

Rename an image alias.

```bash
lxc-dev-manager image rename <old-name> <new-name>
```

**Arguments**:
| Argument | Description |
|----------|-------------|
| `old-name` | Current image alias |
| `new-name` | New image alias |

**Examples**:

```bash
lxc-dev-manager image rename my-base-image production-base
```

**Output**:
```
Renaming image 'my-base-image' → 'production-base'...
Image renamed: my-base-image → production-base
```
