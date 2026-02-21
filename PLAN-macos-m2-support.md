# macOS M2 Support Plan

## Overview

This document outlines the plan to port lxc-dev-manager to macOS with Apple Silicon (M2) support.

## Current Tool Summary

**lxc-dev-manager** is a CLI that simplifies LXC container management for development:
- Container lifecycle (create, start, stop, delete)
- Automatic dev user setup with SSH and sudo
- TCP port forwarding from localhost to containers
- Snapshot/image management
- Project-based organization with YAML config

## Problem Statement

LXC is Linux-kernel specific and unavailable on macOS. We need an equivalent backend for macOS Apple Silicon.

---

## Option Comparison for macOS Apple Silicon

| Technology | Type | Performance | Ease of Use | LXC-like Experience |
|------------|------|-------------|-------------|---------------------|
| **Lima** | Lightweight Linux VMs | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **OrbStack** | Commercial VMs/Docker | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Colima** | Docker runtime | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |
| **Docker Desktop** | Docker containers | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |
| **UTM/QEMU** | Full VMs | ⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ |

## Recommended Approach: Lima-based Backend

**Lima** is the best fit because:
1. Uses Apple's **Virtualization.framework** (native ARM64 performance)
2. Provides lightweight Linux VMs similar to containers
3. Built-in port forwarding and file sharing
4. Open source, actively maintained
5. Supports multiple instances (like multiple LXC containers)
6. Can use various Linux distros (Ubuntu, Alpine, etc.)

---

## Implementation Plan

### Phase 1: Architecture Refactoring

1. **Create backend abstraction layer**
   ```
   internal/backend/
   ├── backend.go          # Interface definition
   ├── lxc/                 # Linux LXC backend (existing code)
   └── lima/                # macOS Lima backend (new)
   ```

2. **Define unified Backend interface**
   ```go
   type Backend interface {
       Launch(name, image string) error
       Start(name string) error
       Stop(name string) error
       Delete(name string) error
       Exec(name string, cmd []string) error
       GetIP(name string) (string, error)
       GetStatus(name string) (string, error)
       Exists(name string) (bool, error)
       ListAll() ([]string, error)
       Snapshot(name, snapshotName string) error
       PublishSnapshot(name, snapshotName, imageName string) error
       ListImages() ([]Image, error)
       DeleteImage(name string) error
       RenameImage(oldName, newName string) error
       SetupDevUser(name string) error
       EnableSSH(name string) error
       EnableNesting(name string) error
       WaitForReady(name string) error
   }
   ```

3. **Platform detection**
   - Detect OS at runtime (`runtime.GOOS`)
   - Select appropriate backend (LXC on Linux, Lima on macOS)

### Phase 2: Lima Backend Implementation

1. **Instance Management** (maps to container lifecycle)
   - `limactl create` → Launch VM from template
   - `limactl start` → Start VM
   - `limactl stop` → Stop VM
   - `limactl delete` → Remove VM
   - `limactl shell` → SSH into VM

2. **Lima YAML Templates** (equivalent to LXC images)
   ```yaml
   # ~/.lima/templates/dev-ubuntu.yaml
   images:
     - location: "https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-arm64.img"
       arch: "aarch64"
   cpus: 4
   memory: "4GiB"
   disk: "50GiB"
   ssh:
     localPort: 0  # Auto-assign
   portForwards:
     - guestPort: 5173
       hostPort: 5173
     - guestPort: 8000
       hostPort: 8000
   ```

3. **Feature Mapping**

   | LXC Feature | Lima Equivalent |
   |-------------|-----------------|
   | `lxc launch` | `limactl create --name=X template.yaml && limactl start X` |
   | `lxc start/stop` | `limactl start/stop` |
   | `lxc delete` | `limactl delete` |
   | `lxc exec` | `limactl shell` or `lima X cmd` |
   | `lxc list` | `limactl list --json` |
   | Container IP | Lima handles via socket forwarding |
   | Port proxy | Lima's built-in `portForwards` |
   | Snapshots | `limactl copy` or disk snapshots |
   | Nesting (Docker) | Provision script installs Docker |

### Phase 3: Configuration Updates

1. **Extend `containers.yaml` for cross-platform**
   ```yaml
   project: myapp
   backend: auto  # or "lxc", "lima"
   defaults:
     ports: [5173, 8000]
     # Lima-specific
     cpus: 4
     memory: "4GiB"
   containers:
     dev1:
       image: ubuntu:24.04  # Maps to Lima template
   ```

2. **Template management**
   - Store Lima YAML templates in `~/.lxc-dev-manager/templates/`
   - Auto-generate from image names (e.g., `ubuntu:24.04` → Ubuntu ARM64 cloud image)

### Phase 4: Port Forwarding Adaptation

- **Linux (LXC)**: Keep existing TCP proxy (containers have IPs)
- **macOS (Lima)**: Use Lima's native `portForwards` in YAML config
  - Update config on `proxy` command to add ports
  - Restart VM to apply (or use `limactl edit`)

### Phase 5: User Setup Adaptation

Lima handles SSH automatically, but we need to:

1. Create provisioning script for `dev` user setup
2. Include in Lima template:
   ```yaml
   provision:
     - mode: system
       script: |
         useradd -m -s /bin/bash dev
         echo 'dev:dev' | chpasswd
         usermod -aG sudo dev
         echo 'dev ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers.d/dev
   ```

### Phase 6: Command Adaptations

| Command | Linux Behavior | macOS Behavior |
|---------|----------------|----------------|
| `create` | Initialize project | Same |
| `container create` | `lxc launch` + setup | `limactl create` + start |
| `up` | `lxc start` | `limactl start` |
| `down` | `lxc stop` | `limactl stop` |
| `ssh` | `lxc exec bash` | `limactl shell` |
| `proxy` | TCP proxy manager | Configure Lima portForwards |
| `list` | `lxc list` | `limactl list` |
| `snapshot` | `lxc snapshot` + publish | Disk image copy |
| `remove` | `lxc delete` | `limactl delete` |

---

## Implementation Tasks

### Phase 1: Refactoring (Foundation)
- [ ] Create `internal/backend/backend.go` interface
- [ ] Move LXC code to `internal/backend/lxc/`
- [ ] Add platform detection in `cmd/root.go`
- [ ] Update all commands to use backend interface

### Phase 2: Lima Backend (Core)
- [ ] Implement `internal/backend/lima/lima.go`
- [ ] Create Lima template generator
- [ ] Implement all Backend interface methods
- [ ] Handle Lima's JSON output parsing

### Phase 3: Templates & Config (UX)
- [ ] Create default Lima templates for common distros
- [ ] Extend config schema for cross-platform options
- [ ] Add template management commands

### Phase 4: Testing (Quality)
- [ ] Add Lima mock executor for tests
- [ ] Cross-platform CI (Linux + macOS runners)
- [ ] Integration tests on macOS

### Phase 5: Documentation (Adoption)
- [ ] Update README for macOS installation
- [ ] Document Lima prerequisites (`brew install lima`)
- [ ] Platform-specific usage notes

---

## Prerequisites for macOS Users

```bash
# Install Lima
brew install lima

# Initialize Lima (first time)
limactl start default  # Optional: creates default instance

# Install lxc-dev-manager
# (same binary works on both platforms)
```

---

## Trade-offs & Considerations

| Aspect | LXC (Linux) | Lima (macOS) |
|--------|-------------|--------------|
| **Startup time** | ~5 seconds | ~30-60 seconds |
| **Resource overhead** | Minimal (containers) | Higher (VMs) |
| **Isolation** | Namespace-based | Full VM |
| **Networking** | Direct IP access | Socket forwarding |
| **Docker nesting** | Config flags | Full Docker in VM |
| **Snapshots** | Native, fast | Disk copy, slower |

---

## File Structure After Implementation

```
lxc-dev-manager/
├── main.go
├── cmd/
│   ├── root.go              # Platform detection added
│   ├── project.go
│   ├── container.go         # Uses backend interface
│   ├── up.go
│   ├── down.go
│   ├── list.go
│   ├── ssh.go
│   ├── proxy.go
│   ├── snapshot.go
│   ├── remove.go
│   └── image.go
├── internal/
│   ├── backend/
│   │   ├── backend.go       # Interface definition
│   │   ├── lxc/
│   │   │   ├── lxc.go       # Existing LXC code
│   │   │   ├── executor.go
│   │   │   └── mock.go
│   │   └── lima/
│   │       ├── lima.go      # Lima implementation
│   │       ├── templates.go # Template management
│   │       └── mock.go
│   ├── config/
│   │   └── config.go        # Extended for cross-platform
│   └── proxy/
│       └── proxy.go
├── templates/
│   ├── ubuntu-24.04.yaml    # Lima template
│   ├── ubuntu-22.04.yaml
│   └── alpine.yaml
└── containers.yaml
```

---

## Timeline Phases

1. **Phase 1**: Architecture refactoring
2. **Phase 2**: Lima backend implementation
3. **Phase 3**: Configuration and templates
4. **Phase 4**: Testing infrastructure
5. **Phase 5**: Documentation and release

---

## Open Questions

1. Should we rename the tool since "lxc" is Linux-specific?
   - Option A: Keep name (LXC-like experience)
   - Option B: Rename to `dev-manager` or `devbox`

2. How to handle image/template mapping between platforms?
   - Need registry of Lima templates for common images

3. Should Lima port forwarding replace the TCP proxy entirely on macOS?
   - Lima's native forwarding is simpler but less flexible

---

## References

- [Lima GitHub](https://github.com/lima-vm/lima)
- [Lima Documentation](https://lima-vm.io/)
- [Apple Virtualization.framework](https://developer.apple.com/documentation/virtualization)
- [LXC/LXD Documentation](https://linuxcontainers.org/)
