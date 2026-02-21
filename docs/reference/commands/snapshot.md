# Snapshot Commands

Commands for saving and restoring container state.

Snapshots let you save and restore container state instantly. Every container automatically gets an `initial-state` snapshot when created.

## container reset

Reset a container to a previous snapshot state.

```bash
lxc-dev-manager container reset <container> [snapshot]
```

**Aliases**: `c reset`

**Arguments**:
| Argument | Description |
|----------|-------------|
| `container` | Container name |
| `snapshot` | Snapshot name (defaults to `initial-state`) |

**Examples**:

```bash
# Reset to initial state (created when container was first made)
lxc-dev-manager container reset dev

# Reset to a named snapshot
lxc-dev-manager container reset dev before-refactor

# Using short alias
lxc-dev-manager c reset dev checkpoint
```

**Output**:
```
Resetting container 'dev' to snapshot 'initial-state'...
Container 'dev' reset successfully
  IP: 10.87.167.42
```

::: tip
Reset preserves the container's running/stopped state. If the container was running before reset, it will be running after.
:::

---

## container snapshot create

Create a named snapshot of a container.

```bash
lxc-dev-manager container snapshot create <container> <name> [--description <text>]
```

**Aliases**: `c snapshot create`

**Arguments**:
| Argument | Description |
|----------|-------------|
| `container` | Container name |
| `name` | Snapshot name |

**Flags**:
| Flag | Short | Description |
|------|-------|-------------|
| `--description` | `-d` | Add a description for the snapshot |

**Examples**:

```bash
# Create a simple snapshot
lxc-dev-manager container snapshot create dev checkpoint

# Create with description
lxc-dev-manager container snapshot create dev before-refactor -d "Before major refactor"

# Using short alias
lxc-dev-manager c snapshot create dev working-state
```

**Output**:
```
Creating snapshot 'before-refactor' for container 'dev'...
Snapshot 'before-refactor' created
```

::: tip
With ZFS storage, snapshots are instant and space-efficient. They only store the differences from the current state.
:::

---

## container snapshot list

List all snapshots for a container.

```bash
lxc-dev-manager container snapshot list <container>
```

**Aliases**: `c snapshot list`

**Arguments**:
| Argument | Description |
|----------|-------------|
| `container` | Container name |

**Examples**:

```bash
lxc-dev-manager container snapshot list dev

# Using short alias
lxc-dev-manager c snapshot list dev
```

**Output**:
```
Snapshots for container 'dev':

NAME              CREATED              DESCRIPTION
---------------------------------------------------------------------------
initial-state     2024-01-15 10:30    Initial container state
before-refactor   2024-01-15 14:22    Before major refactor
checkpoint        2024-01-15 16:45    -
```

---

## container snapshot delete

Delete a snapshot from a container.

```bash
lxc-dev-manager container snapshot delete <container> <name>
```

**Aliases**: `c snapshot delete`

**Arguments**:
| Argument | Description |
|----------|-------------|
| `container` | Container name |
| `name` | Snapshot name to delete |

**Examples**:

```bash
lxc-dev-manager container snapshot delete dev checkpoint

# Using short alias
lxc-dev-manager c snapshot delete dev old-snapshot
```

**Output**:
```
Deleting snapshot 'checkpoint' from container 'dev'...
Snapshot 'checkpoint' deleted
```

::: warning
The `initial-state` snapshot cannot be deleted. It's protected to ensure you can always reset to the original container state.
:::
