# Getting Started

This tutorial walks you through creating your first project with lxc-dev-manager, from setup to a working development environment.

::: tip Prerequisites
LXC/LXD must be installed and initialized. See [LXC Setup](/guide/setup) if you haven't done this yet.
:::

## Understanding Projects

lxc-dev-manager organizes containers by **project**. A project is simply a directory containing a `containers.yaml` configuration file.

When you create containers, they're automatically prefixed with the project name. For example, if your project is called `webapp` and you create a container named `dev`, the actual LXC container will be named `webapp-dev`.

This namespacing prevents conflicts when working on multiple projects:

```
~/projects/
├── webapp/           # Project: webapp
│   └── containers.yaml
│       └── dev → LXC: webapp-dev
│       └── test → LXC: webapp-test
│
└── api-service/      # Project: api-service
    └── containers.yaml
        └── dev → LXC: api-service-dev
```

## Your First Project

### Step 1: Create a Project Directory

```bash
mkdir -p ~/projects/webapp
cd ~/projects/webapp
```

### Step 2: Initialize the Project

```bash
lxc-dev-manager create
```

Output:
```
Project 'webapp' created
  Config: containers.yaml

Next steps:
  lxc-dev-manager container create dev1 ubuntu:24.04
```

This creates a `containers.yaml` file in the current directory:

```yaml
project: webapp
defaults:
  ports:
    - 5173
    - 8000
    - 5432
containers: {}
```

::: tip
The project name defaults to the folder name. Use `--name` to override:
```bash
lxc-dev-manager create --name my-custom-name
```
:::

### Step 3: Create a Development Container

```bash
lxc-dev-manager container create dev ubuntu:24.04
```

Output:
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

The container is automatically configured with:
- A `dev` user with password `dev`
- Passwordless sudo for the dev user
- SSH server enabled
- Nesting enabled (for running Docker inside the container)

### Step 4: Connect to Your Container

Open a shell inside the container:

```bash
lxc-dev-manager ssh dev
```

You're now inside the container as the `dev` user. To get a root shell instead:

```bash
lxc-dev-manager ssh dev -u root
```

Or use SSH directly:

```bash
ssh dev@10.87.167.42
# Password: dev
```

### Step 5: Set Up Your Development Environment

Inside the container, install your project dependencies:

```bash
# Update packages
sudo apt update

# Example: Node.js development
sudo apt install -y nodejs npm
node --version

# Example: Python development
sudo apt install -y python3 python3-pip python3-venv
python3 --version

# Example: Go development
sudo snap install go --classic
go version
```

::: tip
The container has internet access and can install any packages. Your host system remains clean.
:::

## Port Forwarding

Containers have their own IP addresses on a private network. To access services running inside the container from your host browser, use port forwarding.

### Start Port Forwarding

```bash
lxc-dev-manager proxy dev
```

Output:
```
Proxying dev (10.87.167.42):
  localhost:5173 -> 10.87.167.42:5173
  localhost:8000 -> 10.87.167.42:8000
  localhost:5432 -> 10.87.167.42:5432

Press Ctrl+C to stop
```

Now you can access:
- `http://localhost:5173` - Vite dev server
- `http://localhost:8000` - Django/FastAPI
- `localhost:5432` - PostgreSQL

### Custom Ports

Edit `containers.yaml` to configure custom ports for a container:

```yaml
project: webapp
defaults:
  ports:
    - 5173
    - 8000
    - 5432
containers:
  dev:
    image: ubuntu:24.04
    ports:      # Override defaults
      - 3000    # React/Next.js
      - 4000    # GraphQL
```

## Creating Reusable Images

Once you've set up your development environment, save it as a reusable image:

### Create an Image

```bash
lxc-dev-manager image create dev nodejs-ready
```

Output:
```
[1/4] Stopping container 'dev'...
      ✓ Stopped
[2/4] Creating snapshot...
      ✓ Snapshot created (instant with ZFS)
[3/4] Publishing image 'nodejs-ready'...

      Transferring image: 100% (312.45MB/s)

      ✓ Image published
[4/4] Restarting container 'dev'...
      ✓ Started (10.87.167.42)

Image 'nodejs-ready' created successfully!

Create new containers from it with:
  lxc-dev-manager container create <name> nodejs-ready
```

### Create Containers from Images

Now you can create new containers instantly with your pre-configured environment:

```bash
lxc-dev-manager container create dev2 nodejs-ready
```

This creates a container with all your installed packages and configurations.

### Clone Existing Containers

Alternatively, you can clone a container directly without creating an image first:

```bash
# Clone the current state
lxc-dev-manager container clone dev dev2

# Or clone from a specific snapshot
lxc-dev-manager container clone dev dev2 --snapshot checkpoint
```

Cloning is faster for one-off copies, while images are better for sharing or reusing across projects.

### List Your Images

```bash
lxc-dev-manager images
```

Output:
```
ALIAS                     FINGERPRINT    SIZE       DESCRIPTION
nodejs-ready              a1b2c3d4e5f6   1.2GB      Ubuntu 24.04 LTS
python-ml-base            f6e5d4c3b2a1   2.8GB      Ubuntu 24.04 LTS
```

## Working with Snapshots

Snapshots let you save and restore container state instantly. This is useful for creating checkpoints before risky changes or experiments.

### Automatic Initial Snapshot

Every container automatically gets an `initial-state` snapshot when created. This lets you reset to a clean state at any time.

### Create a Checkpoint

Before making significant changes, create a named snapshot:

```bash
lxc-dev-manager container snapshot create dev before-refactor -d "Before major refactor"
```

Output:
```
Creating snapshot 'before-refactor' for container 'dev'...
Snapshot 'before-refactor' created
```

### List Your Snapshots

See all snapshots for a container:

```bash
lxc-dev-manager container snapshot list dev
```

Output:
```
Snapshots for container 'dev':

NAME              CREATED              DESCRIPTION
---------------------------------------------------------------------------
initial-state     2024-01-15 10:30    Initial container state
before-refactor   2024-01-15 14:22    Before major refactor
```

### Reset to a Snapshot

If something goes wrong, reset to a previous state:

```bash
# Reset to your named snapshot
lxc-dev-manager container reset dev before-refactor

# Or reset to the initial clean state
lxc-dev-manager container reset dev
```

Output:
```
Resetting container 'dev' to snapshot 'before-refactor'...
Container 'dev' reset successfully
  IP: 10.87.167.42
```

::: tip
With ZFS storage, snapshots and resets are instant operations. You can create as many snapshots as you need without significant storage overhead.
:::

### Delete Old Snapshots

Remove snapshots you no longer need:

```bash
lxc-dev-manager container snapshot delete dev before-refactor
```

::: warning
The `initial-state` snapshot cannot be deleted. It's protected to ensure you can always reset to the original container state.
:::

## Multi-Project Workflow

Work on multiple projects simultaneously without conflicts:

```bash
# Project 1: Frontend
cd ~/projects/frontend
lxc-dev-manager create
lxc-dev-manager container create dev ubuntu:24.04

# Project 2: Backend API
cd ~/projects/backend
lxc-dev-manager create
lxc-dev-manager container create dev ubuntu:24.04

# Project 3: ML Pipeline
cd ~/projects/ml-pipeline
lxc-dev-manager create
lxc-dev-manager container create dev ubuntu:24.04
```

Each project has its own `dev` container with no naming conflicts:

```bash
lxc list
```

```
+------------------+---------+----------------------+------+-----------+
| NAME             | STATE   | IPV4                 | TYPE | SNAPSHOTS |
+------------------+---------+----------------------+------+-----------+
| frontend-dev     | RUNNING | 10.87.167.42 (eth0)  | ...  | 0         |
| backend-dev      | RUNNING | 10.87.167.43 (eth0)  | ...  | 0         |
| ml-pipeline-dev  | RUNNING | 10.87.167.44 (eth0)  | ...  | 0         |
+------------------+---------+----------------------+------+-----------+
```

## Container Lifecycle

### List Project Containers

```bash
cd ~/projects/webapp
lxc-dev-manager list
```

```
Project: webapp

NAME            IMAGE                STATUS     IP              PORTS
---------------------------------------------------------------------------
dev             ubuntu:24.04         RUNNING    10.87.167.42    5173,8000,5432
dev2            nodejs-ready         RUNNING    10.87.167.45    5173,8000,5432
```

### Stop a Container

```bash
lxc-dev-manager down dev
```

```
Stopping container 'dev'...
Container 'dev' stopped
```

### Start a Container

```bash
lxc-dev-manager up dev
```

```
Starting container 'dev'...
Container 'dev' started
  IP: 10.87.167.42
```

### Remove a Container

```bash
lxc-dev-manager remove dev2
```

```
Container: dev2 (LXC: webapp-dev2)
  Status: RUNNING
  IP: 10.87.167.45
  In config: yes

Are you sure you want to delete container 'dev2'? [y/N]: y
Deleting container 'dev2'...
Container 'dev2' removed
```

Use `--force` to skip confirmation:

```bash
lxc-dev-manager remove dev2 --force
```

## Cleanup

### Delete a Single Container

```bash
lxc-dev-manager remove dev
```

### Delete Entire Project

Remove all containers and the config file:

```bash
lxc-dev-manager project delete
```

```
Project: webapp
Config:  containers.yaml

Containers to be deleted:
  - dev (webapp-dev) [RUNNING]
  - dev2 (webapp-dev2) [STOPPED]

Are you sure you want to delete this project? [y/N]: y
Deleting container 'dev'... done
Deleting container 'dev2'... done
Removing containers.yaml... done

Project 'webapp' deleted
```

### Delete an Image

```bash
lxc-dev-manager image delete nodejs-ready
```

```
Image: nodejs-ready
  Size: 1.2GB
  Description: Ubuntu 24.04 LTS

Are you sure you want to delete image 'nodejs-ready'? [y/N]: y
Deleting image 'nodejs-ready'...
Image 'nodejs-ready' deleted
```

## Next Steps

- See the [Command Reference](/reference/commands/) for all available commands
- See the [Configuration Reference](/reference/configuration) for `containers.yaml` options
