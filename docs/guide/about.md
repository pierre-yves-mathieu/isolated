# About

lxc-dev-manager is a CLI tool for managing LXC containers as local development environments.

## What It Does

Run multiple AI coding assistants in parallel, each in its own isolated container.

For example:

- 5 containers, 5 Claude instances, 5 tasks running simultaneously
- Each AI has full system access—install packages, run tests, use Docker
- Complete isolation between tasks and from your host
- If something breaks, reset to clean state in seconds
- When done: proxy container to host for manual QA, then branch and push

[See the full workflow →](/guide/workflow)

### Run Docker Inside

Containers have nesting enabled by default, so you can run Docker inside your dev container. This gives you a full development machine where you can use Docker Compose, build images, and run containers—all isolated from your host.

## Why Use It

**Problem**: Development environment setup is repetitive and fragile.

- Installing dependencies directly on your host causes conflicts
- Different projects need different versions of the same tools
- AI coding tools need broad system access to be effective
- VMs are slow and resource-heavy

**Solution**: LXC containers give you isolated Linux environments with near-native performance.

- Full Linux system where you can install anything
- Run Docker inside just like on your host machine
- Give AI tools the freedom they need without risking your host
- Scale horizontally—more containers, more parallel work

lxc-dev-manager wraps LXC to make it practical for development:

| Feature | Benefit |
|---------|---------|
| Project namespacing | Multiple projects can each have a `dev` container without conflicts |
| Auto-configured user | Containers come with `dev` user, passwordless sudo, SSH ready |
| Port forwarding | Access `localhost:3000` instead of `10.x.x.x:3000` |
| Instant snapshots | Save state before risky changes, reset in seconds |
| Reusable images | Configure once, create new containers instantly |

## The Workflow

A typical parallel development setup:

```
┌─────────┬─────────┬─────────┬─────────┬─────────┐
│  dev1   │  dev2   │  dev3   │  dev4   │  dev5   │
│ Claude  │ Claude  │ Claude  │ Claude  │ Claude  │
│ task A  │ task B  │ task C  │ task D  │ task E  │
└─────────┴─────────┴─────────┴─────────┴─────────┘
              ↓ when done ↓
         proxy → QA → branch → push
              ↓
        merge + test (driver)
```

1. **Parallel work** - Each container runs an AI assistant on a separate task
2. **Manual QA** - Use `proxy` to inspect the running app before merging
3. **Push when ready** - AI creates branch and pushes
4. **Central merge** - A driver (human or AI) merges and runs integration tests

[Detailed setup guide →](/guide/workflow)

## How It Works

```
~/projects/webapp/
├── containers.yaml          # Project config
└── (your code)

LXC containers:
├── webapp-dev               # Your dev container
└── webapp-test              # Another container
```

1. **Create a project** - Initializes `containers.yaml` in your project directory
2. **Create containers** - Spins up LXC containers prefixed with project name
3. **Work inside containers** - SSH in, install dependencies, run services
4. **Forward ports** - Access container services on localhost
5. **Snapshot/reset** - Save state and restore when needed

## Requirements

- Linux (Ubuntu, Debian, Fedora, Arch)
- LXD installed and initialized
- User in the `lxd` group

## Compared To

### vs Docker

Docker excels at packaging and deploying applications. It's the standard for production containers and CI/CD pipelines.

| | lxc-dev-manager | Docker |
|-|-----------------|--------|
| **Model** | Full OS container | Single process per container |
| **Persistence** | Stateful by default | Ephemeral by design (uses volumes) |
| **Init system** | Full systemd support | Minimal (PID 1 is your app) |
| **Primary use** | Interactive environments | Application packaging |
| **Run Docker inside** | Yes (nesting) | Requires privileged mode |

**When to use Docker**: Production deployments, CI/CD pipelines, microservices, reproducible builds.

**When to use lxc-dev-manager**: Local development where you want a full Linux environment to work in interactively.

### vs Vagrant

Vagrant manages virtual machines, which provide full isolation but at significant resource cost.

| | lxc-dev-manager | Vagrant |
|-|-----------------|---------|
| **Technology** | LXC containers | VMs (VirtualBox, VMware, etc.) |
| **Startup time** | ~1 second | ~30-60 seconds |
| **Memory overhead** | Minimal (shared kernel) | High (full OS per VM) |
| **Disk usage** | ~500MB per container | ~2-10GB per VM |
| **Performance** | Near-native | 10-30% overhead |
| **Snapshots** | Instant (ZFS) | Slow (full disk copy) |
| **Nested virtualization** | Yes | Requires hardware support |

**When to use Vagrant**: When you need a different kernel, Windows/macOS guests, or hardware-level isolation.

**When to use lxc-dev-manager**: When you want fast, lightweight Linux environments without VM overhead.

### Summary

| | lxc-dev-manager | Docker | Vagrant |
|-|-----------------|--------|---------|
| Startup | ~1s | ~1s | ~30-60s |
| Memory | Minimal | Minimal | High |
| Full OS | Yes | No | Yes |
| Stateful | Yes | No | Yes |
| Best for | Dev environments | App packaging | Multi-OS testing |

## Next Steps

- [Getting Started](/guide/getting-started) - Create your first project
- [Workflow](/guide/workflow) - Set up parallel AI development
- [LXC Setup](/guide/setup) - Install and configure LXC/LXD
