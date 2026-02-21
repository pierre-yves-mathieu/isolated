# Project Commands

Commands for initializing and managing projects.

## create

Initialize a new project in the current directory.

```bash
lxc-dev-manager create [--name <project-name>]
```

**Aliases**: `project create`

**Flags**:
| Flag | Short | Description |
|------|-------|-------------|
| `--name` | `-n` | Project name (defaults to folder name) |

**Examples**:

```bash
# Use folder name as project name
cd ~/projects/webapp
lxc-dev-manager create

# Specify custom project name
lxc-dev-manager create --name my-webapp
```

**Output**:
```
Project 'webapp' created
  Config: containers.yaml

Next steps:
  lxc-dev-manager container create dev1 ubuntu:24.04
```

---

## project delete

Delete the project and all its containers.

```bash
lxc-dev-manager project delete [--force]
```

**Flags**:
| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Skip confirmation prompt |

**Examples**:

```bash
# Interactive deletion (asks for confirmation)
lxc-dev-manager project delete

# Skip confirmation
lxc-dev-manager project delete --force
```

::: danger
This command is destructive. It will delete all containers in the project and remove the `containers.yaml` file.
:::
