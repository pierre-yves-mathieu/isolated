# lxc-dev-manager

Manage LXC containers for local development. Create isolated, reproducible development environments with a simple CLI.

## Features

- **Project-based namespacing** - Containers are prefixed with project names to avoid conflicts
- **Ready-to-use containers** - Comes with `dev` user, passwordless sudo, and SSH enabled
- **Port forwarding** - Access container services on localhost
- **Instant snapshots** - Create checkpoints and reset containers instantly
- **Image creation** - Save your configured environment as a reusable image

## Quick Start

```bash
# Prerequisites: LXC/LXD installed and initialized
# See docs for setup: https://pierre-yves-mathieu.github.io/lxc-dev-manager/guide/setup

# Create a project
cd ~/projects/webapp
lxc-dev-manager create

# Create a container
lxc-dev-manager container create dev ubuntu:24.04

# Open a shell
lxc-dev-manager ssh dev

# Forward ports to localhost
lxc-dev-manager proxy dev
```

## Installation

### Download Binary

```bash
curl -LO https://github.com/pierre-yves-mathieu/lxc-dev-manager/releases/latest/download/lxc-dev-manager-linux-amd64
chmod +x lxc-dev-manager-linux-amd64
sudo mv lxc-dev-manager-linux-amd64 /usr/local/bin/lxc-dev-manager
```

### Build from Source

```bash
git clone https://github.com/pierre-yves-mathieu/lxc-dev-manager.git
cd lxc-dev-manager
go build -o lxc-dev-manager .
sudo mv lxc-dev-manager /usr/local/bin/
```

## Documentation

Full documentation is available at: **https://pierre-yves-mathieu.github.io/lxc-dev-manager/**

- [LXC Setup Guide](https://pierre-yves-mathieu.github.io/lxc-dev-manager/guide/setup) - Install and configure LXC/LXD
- [Getting Started Tutorial](https://pierre-yves-mathieu.github.io/lxc-dev-manager/guide/getting-started) - Create your first project
- [Command Reference](https://pierre-yves-mathieu.github.io/lxc-dev-manager/reference/commands) - All commands and options
- [Configuration Reference](https://pierre-yves-mathieu.github.io/lxc-dev-manager/reference/configuration) - containers.yaml format

## Commands Overview

| Command | Description |
|---------|-------------|
| `create` | Initialize a new project |
| `container create <name> <image>` | Create a container |
| `container clone <source> <name>` | Clone an existing container |
| `container reset <name> [snapshot]` | Reset container to snapshot |
| `container snapshot create` | Create named snapshot |
| `container snapshot list` | List container snapshots |
| `container snapshot delete` | Delete a snapshot |
| `list` | List project containers |
| `up <name>` | Start a container |
| `down <name>` | Stop a container |
| `ssh <name>` | Open shell in container |
| `proxy <name>` | Forward ports to localhost |
| `image create <container> <image>` | Create image from container |
| `image list` | List local images |
| `image delete <name>` | Delete an image |
| `image rename <old> <new>` | Rename image alias |
| `remove <name>` | Delete a container |
| `project delete` | Delete project and all containers |

## License

MIT
