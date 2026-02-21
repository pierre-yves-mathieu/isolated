# LXC Setup Guide

This guide walks you through installing LXC/LXD on Linux and configuring it for use with lxc-dev-manager.

## Prerequisites

- A Linux distribution: Ubuntu 22.04+, Debian 12+, Fedora 38+, or Arch Linux
- sudo access on your machine
- At least 10GB of free disk space (for container images)

## Step 1: Install LXC/LXD

::: code-group

```bash [Ubuntu/Debian (snap)]
# Install LXD via snap (recommended)
sudo snap install lxd

# Or install from apt (older version)
sudo apt update
sudo apt install lxd lxd-client
```

```bash [Fedora]
# Enable COPR repository
sudo dnf copr enable ganto/lxc4
sudo dnf install lxd

# Enable and start the service
sudo systemctl enable --now lxd
```

```bash [Arch Linux]
# Install from official repositories
sudo pacman -S lxd

# Enable and start the service
sudo systemctl enable --now lxd.socket
```

:::

## Step 2: Initialize LXD

Run `lxd init` to configure LXD. For most development use cases, the minimal setup works well:

```bash
sudo lxd init --minimal
```

This creates a default storage pool and network with sensible defaults.

### Interactive Setup (Advanced)

If you want more control, run `lxd init` without `--minimal`:

```bash
sudo lxd init
```

Here are recommended answers for each prompt:

| Prompt | Recommended Answer | Notes |
|--------|-------------------|-------|
| Would you like to use LXD clustering? | no | Single-machine setup |
| Do you want to configure a new storage pool? | yes | Required for containers |
| Name of the new storage pool | default | Standard name |
| Name of the storage backend to use | dir | Simple, works everywhere. Use `zfs` or `btrfs` for faster snapshots |
| Would you like to connect to a MAAS server? | no | Not needed for development |
| Would you like to create a new local network bridge? | yes | Required for container networking |
| What should the new bridge be called? | lxdbr0 | Standard name |
| What IPv4 address should be used? | auto | Let LXD choose |
| What IPv6 address should be used? | auto | Or `none` if you don't need IPv6 |
| Would you like LXD to be available over the network? | no | Local development only |
| Would you like stale cached images to be updated automatically? | yes | Keeps images current |
| Would you like a YAML "lxd init" preseed to be printed? | no | Optional |

### Storage Backend Comparison

| Backend | Speed | Snapshots | Disk Usage | Best For |
|---------|-------|-----------|------------|----------|
| `dir` | Good | Slow (copies data) | Higher | Simplicity, any filesystem |
| `zfs` | Fast | Instant (copy-on-write) | Efficient | Performance, if ZFS is available |
| `btrfs` | Fast | Instant (copy-on-write) | Efficient | Performance, native on some distros |

::: tip
If your system uses ZFS or btrfs, choose that backend for significantly faster snapshot operations.
:::

## Step 3: Configure User Permissions

Add your user to the `lxd` group to run LXC commands without sudo:

```bash
sudo usermod -aG lxd $USER
```

Apply the group change:

```bash
newgrp lxd
```

::: warning
If `newgrp` doesn't work, log out and log back in for the group change to take effect.
:::

Verify LXD is working:

```bash
lxc list
```

Expected output (empty list is fine):
```
+------+-------+------+------+------+-----------+
| NAME | STATE | IPV4 | IPV6 | TYPE | SNAPSHOTS |
+------+-------+------+------+------+-----------+
```

## Step 4: Install lxc-dev-manager

### Option A: Download Binary (Recommended)

Download the latest release for your architecture:

```bash
# Download (replace VERSION and ARCH as needed)
curl -LO https://github.com/yourusername/lxc-dev-manager/releases/latest/download/lxc-dev-manager-linux-amd64

# Make executable
chmod +x lxc-dev-manager-linux-amd64

# Move to PATH
sudo mv lxc-dev-manager-linux-amd64 /usr/local/bin/lxc-dev-manager
```

### Option B: Build from Source

Requires Go 1.21 or later:

```bash
# Clone the repository
git clone https://github.com/yourusername/lxc-dev-manager.git
cd lxc-dev-manager

# Build
go build -o lxc-dev-manager .

# Install
sudo mv lxc-dev-manager /usr/local/bin/
```

Or use `go install`:

```bash
go install github.com/yourusername/lxc-dev-manager@latest
```

## Step 5: Verify Installation

Check that lxc-dev-manager is installed correctly:

```bash
lxc-dev-manager --help
```

Expected output:
```
lxc-dev-manager is a CLI tool to manage LXC containers for local development.

It provides easy container lifecycle management and port proxying to make
containers feel like local services.

Usage:
  lxc-dev-manager [command]

Available Commands:
  container   Manage containers within the project
  create      Initialize a new project (alias for 'project create')
  down        Stop a container
  help        Help about any command
  image       Manage images
  list        List all containers
  project     Manage lxc-dev-manager projects
  proxy       Proxy ports from localhost to container
  remove      Remove a container
  image       Manage images (create, list, delete, rename)
  ssh         Open a shell in a container
  up          Start a container
...
```

## Troubleshooting

### "permission denied" when running lxc commands

**Cause**: Your user is not in the `lxd` group, or the group change hasn't taken effect.

**Solution**:
```bash
# Check if you're in the lxd group
groups

# If lxd is not listed, add yourself
sudo usermod -aG lxd $USER

# Then either log out/in, or run:
newgrp lxd
```

### "lxc: command not found"

**Cause**: LXD is not installed or not in your PATH.

**Solution**:
```bash
# Check if lxc is installed
which lxc

# If using snap, it should be at /snap/bin/lxc
# Make sure /snap/bin is in your PATH
echo $PATH | grep -q /snap/bin || echo 'export PATH=$PATH:/snap/bin' >> ~/.bashrc
source ~/.bashrc
```

### Storage pool errors

**Cause**: LXD was not initialized or storage pool is misconfigured.

**Solution**:
```bash
# Check storage pools
lxc storage list

# If empty, reinitialize
sudo lxd init
```

### Container fails to start with network errors

**Cause**: Network bridge is not configured.

**Solution**:
```bash
# Check network configuration
lxc network list

# If lxdbr0 doesn't exist, create it
lxc network create lxdbr0

# Or reinitialize LXD
sudo lxd init
```

### "Error: not found" when launching containers

**Cause**: The image name is incorrect or the image server is unreachable.

**Solution**:
```bash
# List available images
lxc image list images: | head -50

# Use the correct image name format
# Examples:
#   ubuntu:24.04
#   debian/12
#   images:alpine/3.19
```

## Next Steps

Now that LXC is set up, continue to the [Getting Started Tutorial](/guide/getting-started) to create your first development container.
