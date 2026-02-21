# Documentation Plan

## Goal
Create comprehensive documentation for lxc-dev-manager that helps newcomers:
1. Understand what the tool does and why they'd use it
2. Set up LXC on their Linux machine from scratch
3. Get started quickly with their first project

---

## Documentation Structure

### 1. README.md (Main Entry Point)

```
# lxc-dev-manager

## What is this?
- One-paragraph explanation
- Key benefits (isolated dev environments, snapshot/restore, multi-project support)

## Quick Start (5 commands to success)
- Install LXC
- Install lxc-dev-manager
- Create project
- Create container
- SSH in

## Features
- Project-based namespacing
- Port forwarding
- Snapshot to image
- Dev user with passwordless sudo

## Documentation
- Link to docs/SETUP.md
- Link to docs/USAGE.md
- Link to docs/COMMANDS.md
```

### 2. docs/SETUP.md (Linux Setup Guide)

**Target audience**: Complete beginner on Linux who has never used LXC

```
# Setting Up LXC on Linux

## Prerequisites
- Ubuntu 22.04+ / Debian 12+ / Fedora 38+ / Arch Linux
- sudo access

## Step 1: Install LXC and LXD
### Ubuntu/Debian
- apt install commands
- snap install lxd (if using snap)

### Fedora
- dnf commands

### Arch
- pacman commands

## Step 2: Initialize LXD
- lxd init walkthrough
- Recommended answers for each prompt
- Storage pool explanation (dir vs zfs vs btrfs)

## Step 3: Configure User Permissions
- Add user to lxd group
- newgrp lxd (or logout/login)
- Verify with: lxc list

## Step 4: Install lxc-dev-manager
### Option A: Download binary
- GitHub releases link
- chmod +x, move to PATH

### Option B: Build from source
- go install command
- Or clone + go build

## Step 5: Verify Installation
- lxc-dev-manager --help
- Expected output

## Troubleshooting
- "permission denied" → lxd group
- "lxc: command not found" → PATH
- Storage pool errors → lxd init
- Network issues → lxd network
```

### 3. docs/USAGE.md (Getting Started Guide)

**Target audience**: Someone with LXC set up, learning lxc-dev-manager

```
# Getting Started with lxc-dev-manager

## Concept: Projects
- What is a project?
- Why namespacing matters
- containers.yaml explained

## Your First Project

### Step 1: Create a project directory
mkdir ~/projects/my-app
cd ~/projects/my-app

### Step 2: Initialize the project
lxc-dev-manager create
# Output explanation

### Step 3: Create a development container
lxc-dev-manager container create dev ubuntu:24.04
# What happens behind the scenes
# - LXC container created as "my-app-dev"
# - User 'dev' with password 'dev'
# - Passwordless sudo
# - SSH enabled

### Step 4: Connect to your container
lxc-dev-manager ssh dev
# You're now inside the container!

### Step 5: Set up your dev environment
# Inside container:
sudo apt update
sudo apt install nodejs npm
# etc.

## Port Forwarding

### Why port forwarding?
- Container has its own IP
- Access services on localhost

### Using proxy command
lxc-dev-manager proxy dev
# Forwards configured ports (5173, 8000, 5432 by default)

### Custom ports in containers.yaml
containers:
  dev:
    image: ubuntu:24.04
    ports: [3000, 8080]

## Saving Your Work (Snapshots)

### Create a snapshot image
lxc-dev-manager snapshot dev my-base
# Now "my-base" is an image you can reuse

### Create container from snapshot
lxc-dev-manager container create dev2 my-base
# Instant clone with all your setup!

## Multi-Project Workflow

### Project A: ~/projects/frontend
cd ~/projects/frontend
lxc-dev-manager create
lxc-dev-manager container create dev ubuntu:24.04
# LXC name: frontend-dev

### Project B: ~/projects/backend
cd ~/projects/backend
lxc-dev-manager create
lxc-dev-manager container create dev ubuntu:24.04
# LXC name: backend-dev

### No conflicts!
lxc list
# frontend-dev | RUNNING | ...
# backend-dev  | RUNNING | ...

## Cleanup

### Remove a container
lxc-dev-manager remove dev

### Delete entire project
lxc-dev-manager project delete
# Removes all containers and config
```

### 4. docs/COMMANDS.md (Command Reference)

```
# Command Reference

## Project Commands

### lxc-dev-manager create
Initialize a new project.
Usage: lxc-dev-manager create [--name <project-name>]
Flags:
  -n, --name   Project name (defaults to folder name)

### lxc-dev-manager project delete
Delete project and all containers.
Usage: lxc-dev-manager project delete [--force]
Flags:
  -f, --force  Skip confirmation

## Container Commands

### lxc-dev-manager container create
Create a new container.
Usage: lxc-dev-manager container create <name> <image>
Aliases: c create

### lxc-dev-manager up
Start a stopped container.
Usage: lxc-dev-manager up <name>

### lxc-dev-manager down
Stop a running container.
Usage: lxc-dev-manager down <name>

### lxc-dev-manager ssh
Open shell in container.
Usage: lxc-dev-manager ssh <name> [--user <username>]
Flags:
  -u, --user   Username (default: dev)

### lxc-dev-manager proxy
Forward ports to container.
Usage: lxc-dev-manager proxy <name>

### lxc-dev-manager remove
Remove a container.
Usage: lxc-dev-manager remove <name> [--force]
Flags:
  -f, --force  Skip confirmation

### lxc-dev-manager list
List all containers in project.
Usage: lxc-dev-manager list

## Image Commands

### lxc-dev-manager snapshot
Create image from container.
Usage: lxc-dev-manager snapshot <container> <image-name>

### lxc-dev-manager image list
List local images.
Usage: lxc-dev-manager image list [--all]

### lxc-dev-manager image delete
Delete an image.
Usage: lxc-dev-manager image delete <name> [--force]

### lxc-dev-manager image rename
Rename an image.
Usage: lxc-dev-manager image rename <old-name> <new-name>

## Configuration

### containers.yaml
project: my-app
defaults:
  ports:
    - 5173
    - 8000
    - 5432
containers:
  dev:
    image: ubuntu:24.04
  dev2:
    image: my-base
    ports:
      - 3000
```

---

## Files to Create

| File | Purpose | Priority |
|------|---------|----------|
| `README.md` | Main entry, quick start | High |
| `docs/SETUP.md` | LXC installation guide | High |
| `docs/USAGE.md` | Getting started tutorial | High |
| `docs/COMMANDS.md` | Command reference | Medium |

---

## Key Principles

1. **Start with "why"** - Explain benefits before how-to
2. **Copy-paste friendly** - All commands should work when pasted
3. **Show expected output** - So users know they're on track
4. **Progressive complexity** - Basic → intermediate → advanced
5. **Troubleshooting sections** - Anticipate common problems
6. **Real examples** - Use concrete project names, not "foo/bar"

---

## Estimated Effort

- README.md: ~100 lines
- docs/SETUP.md: ~200 lines
- docs/USAGE.md: ~250 lines
- docs/COMMANDS.md: ~150 lines
- **Total**: ~700 lines of documentation
