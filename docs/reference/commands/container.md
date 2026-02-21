# Container Commands

Commands for creating, starting, stopping, and managing containers.

## container create

Create a new container in the current project.

```bash
lxc-dev-manager container create <name> <image>
```

**Aliases**: `c create`

**Arguments**:
| Argument | Description |
|----------|-------------|
| `name` | Container name (local to project) |
| `image` | LXC image or local image alias |

**Examples**:

```bash
# Create from official Ubuntu image
lxc-dev-manager container create dev ubuntu:24.04

# Create from Debian
lxc-dev-manager container create dev debian/12

# Create from Alpine
lxc-dev-manager container create dev images:alpine/3.19

# Create from a saved snapshot
lxc-dev-manager container create dev2 my-base-image

# Using short alias
lxc-dev-manager c create dev ubuntu:24.04
```

**What gets configured**:
- Nesting enabled (Docker support)
- `dev` user created with password `dev`
- Passwordless sudo for `dev` user
- SSH server enabled

**Output**:
```
Creating container 'dev' (LXC: webapp-dev) from image 'ubuntu:24.04'...
Enabling nesting (Docker support)...
Waiting for container to be ready...
Setting up 'dev' user...
Enabling SSH...

Container 'dev' created successfully!
  LXC name: webapp-dev
  IP: 10.87.167.42
  User: dev / Password: dev
  SSH: ssh dev@10.87.167.42

Proxy ports with: lxc-dev-manager proxy dev
```

---

## container clone

Clone an existing container to create a new one.

```bash
lxc-dev-manager container clone <source> <new-name>
lxc-dev-manager container clone <source> <new-name> --snapshot <snapshot-name>
```

**Aliases**: `c clone`

**Arguments**:
| Argument | Description |
|----------|-------------|
| `source` | Source container name |
| `new-name` | Name for the cloned container |

**Flags**:
| Flag | Short | Description |
|------|-------|-------------|
| `--snapshot` | `-s` | Clone from a specific snapshot instead of current state |

**Examples**:

```bash
# Clone the current state of a container
lxc-dev-manager container clone dev dev2

# Clone from a specific snapshot
lxc-dev-manager container clone dev dev2 --snapshot checkpoint

# Using short alias
lxc-dev-manager c clone dev dev2 -s before-refactor
```

**Output**:
```
Cloning container 'dev' to 'dev2'...
Creating initial state snapshot...
Starting cloned container...

Container 'dev2' cloned successfully!
  LXC name: webapp-dev2
  Source: dev
  IP: 10.87.167.43
  User: dev
  SSH: ssh dev@10.87.167.43
```

::: tip
Cloning from a snapshot is useful when you want to create a new container from a known good state, rather than the current (possibly modified) state.
:::

---

## list

List all containers in the current project.

```bash
lxc-dev-manager list
```

**Example output**:
```
Project: webapp

NAME            IMAGE                STATUS     IP              PORTS
---------------------------------------------------------------------------
dev             ubuntu:24.04         RUNNING    10.87.167.42    5173,8000,5432
test            nodejs-ready         STOPPED    -               5173,8000,5432
```

---

## up

Start a stopped container.

```bash
lxc-dev-manager up <name>
```

**Arguments**:
| Argument | Description |
|----------|-------------|
| `name` | Container name |

**Examples**:

```bash
lxc-dev-manager up dev
```

**Output**:
```
Starting container 'dev'...
Container 'dev' started
  IP: 10.87.167.42
```

If the container is already running:
```
Container 'dev' is already running
  IP: 10.87.167.42
```

---

## down

Stop a running container.

```bash
lxc-dev-manager down <name>
```

**Arguments**:
| Argument | Description |
|----------|-------------|
| `name` | Container name |

**Examples**:

```bash
lxc-dev-manager down dev
```

**Output**:
```
Stopping container 'dev'...
Container 'dev' stopped
```

---

## ssh

Open a shell in a container.

```bash
lxc-dev-manager ssh <name> [--user <username>]
```

**Arguments**:
| Argument | Description |
|----------|-------------|
| `name` | Container name |

**Flags**:
| Flag | Short | Description |
|------|-------|-------------|
| `--user` | `-u` | Override user (e.g., `-u root` for root shell) |

By default, logs in as the user configured in `containers.yaml`. The user is determined by:
1. Container-specific `user.name` if set
2. Project `defaults.user.name` if set
3. Falls back to `dev`

**Examples**:

```bash
# Login as configured user (default: dev)
lxc-dev-manager ssh dev

# Login as root
lxc-dev-manager ssh dev -u root
lxc-dev-manager ssh dev --user root

# Login as a specific user
lxc-dev-manager ssh dev -u ubuntu
```

::: tip
This uses `lxc exec` under the hood, which is faster than SSH and doesn't require network access. For true SSH access, use:
```bash
ssh dev@<container-ip>
```
:::

---

## exec

Execute a command in a container.

```bash
lxc-dev-manager exec <name> -- <command> [args...]
```

**Arguments**:
| Argument | Description |
|----------|-------------|
| `name` | Container name |
| `command` | Command to execute (required) |
| `args` | Arguments to pass to the command |

**Flags**:
| Flag | Short | Description |
|------|-------|-------------|
| `--user` | `-u` | Run as user (default: configured user) |

**Examples**:

```bash
# Run a command
lxc-dev-manager exec dev -- htop

# Run a terminal multiplexer
lxc-dev-manager exec dev -- zellij

# Run as root
lxc-dev-manager exec dev -u root -- apt update

# Run npm command
lxc-dev-manager exec dev -- npm run dev

# Nested -- works (for commands that use -- themselves)
lxc-dev-manager exec dev -- zellij run -- ls -la

# Explicit shell (if you want a shell via exec)
lxc-dev-manager exec dev -- bash
```

::: tip
Use `ssh` for interactive shells, use `exec` for running specific commands. The `exec` command requires a command after `--`.
:::

---

## proxy

Forward ports from localhost to a container.

```bash
lxc-dev-manager proxy <name>
```

**Arguments**:
| Argument | Description |
|----------|-------------|
| `name` | Container name |

**Examples**:

```bash
lxc-dev-manager proxy dev
```

**Output**:
```
Proxying dev (10.87.167.42):
  localhost:5173 -> 10.87.167.42:5173
  localhost:8000 -> 10.87.167.42:8000
  localhost:5432 -> 10.87.167.42:5432

Press Ctrl+C to stop
```

The proxy runs in the foreground. Press `Ctrl+C` to stop it.

::: tip
The ports forwarded are determined by the container's configuration in `containers.yaml`, or the project defaults if not specified.
:::

---

## mv

Copy a file or directory from the host to a container.

```bash
lxc-dev-manager mv <source> <container>:<dest>
```

**Arguments**:
| Argument | Description |
|----------|-------------|
| `source` | Local file or directory path |
| `container:dest` | Container name and destination path |

**Examples**:

```bash
# Copy a single file
lxc-dev-manager mv ./config.json dev:/home/dev/

# Copy a directory
lxc-dev-manager mv ./myproject dev:/home/dev/myproject

# Copy to a specific path
lxc-dev-manager mv ./app.py dev:/opt/app/
```

**Output**:
```
Copying file './config.json' to dev:/home/dev/...
Done.
```

For directories:
```
Copying directory './myproject' to dev:/home/dev/myproject...
Done.
```

::: tip
Directories are automatically detected and copied recursively. The destination path must exist in the container.
:::

---

## remove

Remove a container and delete it from the config.

```bash
lxc-dev-manager remove <name> [--force]
```

**Arguments**:
| Argument | Description |
|----------|-------------|
| `name` | Container name |

**Flags**:
| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Skip confirmation prompt |

**Examples**:

```bash
# Interactive deletion
lxc-dev-manager remove dev

# Skip confirmation
lxc-dev-manager remove dev --force
```

**Output**:
```
Container: dev (LXC: webapp-dev)
  Status: RUNNING
  IP: 10.87.167.42
  In config: yes

Are you sure you want to delete container 'dev'? [y/N]: y
Deleting container 'dev'...
Container 'dev' removed
```

::: warning
This will forcefully delete the container even if it's running. All data in the container will be lost unless you've created a snapshot.
:::
