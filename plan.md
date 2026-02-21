# lxc-dev-manager - Plan

## Overview

A CLI tool to manage LXC containers for local development, with config-driven setup and easy port proxying.

## Commands

```
lxc-dev-manager create <name> <image>      Create container from image, add to config
lxc-dev-manager up <name>                  Start container
lxc-dev-manager down <name>                Stop container
lxc-dev-manager snapshot <name> <snap>     Publish container as reusable image
lxc-dev-manager remove <name>              Delete container and remove from config
lxc-dev-manager proxy <name>               Forward ports from localhost to container
lxc-dev-manager list                       Show all containers and status
```

## Config File

`containers.yaml`:

```yaml
defaults:
  ports:
    - 5173
    - 8000
    - 5432

containers:
  dev1:
    image: ubuntu:24.04
    # ports: override if needed

  dev2:
    image: my-custom-base
    ports:
      - 3000
      - 8080
```

## Command Details

### `create <name> <image>`

```bash
$ lxc-dev-manager create dev1 ubuntu:24.04
```

1. Run `lxc launch <image> <name>`
2. Enable nesting for Docker support:
   - `lxc config set <name> security.nesting true`
   - `lxc config set <name> security.syscalls.intercept.mknod true`
   - `lxc config set <name> security.syscalls.intercept.setxattr true`
3. Wait for container to be ready (cloud-init)
4. Create `dev` user with password `dev`, add to sudoers
5. Enable SSH
6. Add entry to `containers.yaml`
7. Print success + IP

### `up <name>`

```bash
$ lxc-dev-manager up dev1
```

1. Check container exists
2. Run `lxc start <name>`
3. Wait for network ready
4. Print IP

### `down <name>`

```bash
$ lxc-dev-manager down dev1
```

1. Run `lxc stop <name>`

### `snapshot <name> <snapshot-name>`

```bash
$ lxc-dev-manager snapshot dev1 my-base-image
```

1. Stop container if running
2. Run `lxc publish <name> --alias <snapshot-name>`
3. Optionally restart container
4. Print: "Image 'my-base-image' created. Use with: lxc-dev-manager create <name> my-base-image"

### `remove <name>`

```bash
$ lxc-dev-manager remove dev1
```

1. Run `lxc delete <name> --force`
2. Remove from `containers.yaml`

### `proxy <name>`

```bash
$ lxc-dev-manager proxy dev1
```

1. Get container IP via `lxc list <name> -c4 -f csv`
2. Get ports from config (container-specific or defaults)
3. Start TCP proxy for each port: `localhost:<port>` → `<ip>:<port>`
4. Print active proxies
5. Wait for Ctrl+C, clean shutdown

### `list`

```bash
$ lxc-dev-manager list

NAME   IMAGE            STATUS   IP            PORTS
dev1   ubuntu:24.04     RUNNING  10.10.10.45   5173,8000,5432
dev2   my-base-image    STOPPED  -             3000,8080
```

## Project Structure

```
lxc-dev-manager/
├── main.go
├── go.mod
├── cmd/
│   ├── create.go
│   ├── up.go
│   ├── down.go
│   ├── snapshot.go
│   ├── remove.go
│   ├── proxy.go
│   └── list.go
├── internal/
│   ├── config/
│   │   └── config.go      # YAML load/save
│   ├── lxc/
│   │   └── lxc.go         # LXC command wrappers
│   └── proxy/
│       └── proxy.go       # TCP proxy implementation
└── containers.yaml        # Created on first use
```

## Dependencies

- Go 1.21+
- `gopkg.in/yaml.v3` for YAML parsing
- `github.com/spf13/cobra` for CLI (optional, could use stdlib)

## Decisions

1. **CLI framework**: Cobra
2. **Config location**: Current directory (`containers.yaml`)
3. **Proxy implementation**: Pure Go TCP proxy (handles websocket, SSE, any TCP; UDP future)
4. **Auto-setup on create**:
   - Enable nesting (for Docker-in-LXC support)
   - Create `dev` user with password `dev`
   - Add to sudoers (NOPASSWD)
   - Enable SSH
   - No extra packages (user installs what they need, then snapshots)

## Nesting (Docker Support)

To run Docker inside LXC containers:

```bash
# Enable nesting on container
lxc config set <name> security.nesting true

# May also need (for some Docker features):
lxc config set <name> security.syscalls.intercept.mknod true
lxc config set <name> security.syscalls.intercept.setxattr true
```

The `create` command will set these automatically.

## Implementation Order

1. [ ] Project scaffolding (go.mod, main.go, cobra setup)
2. [ ] Config loading/saving (internal/config)
3. [ ] LXC wrapper functions (internal/lxc)
4. [ ] Commands: create, up, down, list
5. [ ] Commands: snapshot, remove
6. [ ] TCP proxy (internal/proxy)
7. [ ] Command: proxy
