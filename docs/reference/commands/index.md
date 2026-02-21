# Command Reference

Complete reference for all lxc-dev-manager commands.

## Quick Reference

| Command | Description |
|---------|-------------|
| [`create`](./project#create) | Initialize a new project |
| [`project delete`](./project#project-delete) | Delete project and all containers |
| [`container create`](./container#container-create) | Create a container |
| [`container clone`](./container#container-clone) | Clone an existing container |
| [`list`](./container#list) | List project containers |
| [`up`](./container#up) | Start a container |
| [`down`](./container#down) | Stop a container |
| [`ssh`](./container#ssh) | Open shell in container |
| [`exec`](./container#exec) | Execute a command in container |
| [`proxy`](./container#proxy) | Forward ports to localhost |
| [`mv`](./container#mv) | Copy file/folder to container |
| [`remove`](./container#remove) | Delete a container |
| [`container reset`](./snapshot#container-reset) | Reset container to snapshot |
| [`container snapshot create`](./snapshot#container-snapshot-create) | Create named snapshot |
| [`container snapshot list`](./snapshot#container-snapshot-list) | List container snapshots |
| [`container snapshot delete`](./snapshot#container-snapshot-delete) | Delete a snapshot |
| [`image create`](./image#image-create) | Create image from container |
| [`image list`](./image#image-list) | List local images |
| [`image delete`](./image#image-delete) | Delete an image |
| [`image rename`](./image#image-rename) | Rename image alias |

## Command Categories

### [Project Commands](./project)
Initialize and manage projects.

### [Container Commands](./container)
Create, start, stop, and manage containers.

### [Snapshot Commands](./snapshot)
Save and restore container state.

### [Image Commands](./image)
Create and manage reusable images.

## Global Options

These options are available for all commands:

| Flag | Description |
|------|-------------|
| `--help` | Display help for the command |

**Examples**:

```bash
# Get help for any command
lxc-dev-manager --help
lxc-dev-manager container --help
lxc-dev-manager container create --help
```
