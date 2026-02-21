# Workflow

Run multiple AI coding assistants in parallel, each in its own isolated container. This guide covers the full setup.

## Parallel Development

Work on multiple tasks simultaneously, each in its own container. Each Claude instance can run tests, install packages, use Docker, and experiment freely—completely isolated from the others. When a task is done, QA it via proxy, then push.

### The Setup

```
project/
├── containers.yaml
└── src/

Containers:
├── project-dev1  # Working on feature A
├── project-dev2  # Working on feature B
├── project-dev3  # Bug fix
├── project-dev4  # Experiment
└── project-dev5  # Another task
```

Each container has:
- Its own copy of the repo
- Full isolation from other containers
- Ability to run tests, Docker, and any tools
- Claude or other AI tools with unrestricted access

Containers are reusable—when you finish a task, just switch branches and start the next one. No need to recreate containers for each branch.

### Step-by-Step Setup

**1. Create the project**

```bash
cd ~/projects/myapp
lxc-dev-manager create
```

**2. Create a base container and configure it**

```bash
lxc-dev-manager container create dev1 ubuntu:24.04
lxc-dev-manager ssh dev1

# Inside container: install dependencies
sudo apt update
sudo apt install -y git nodejs npm docker.io
git clone https://github.com/you/myapp.git ~/myapp
cd ~/myapp && npm install

# Install Claude Code or other AI tools
# ...

exit
```

**3. Save as a reusable image**

```bash
lxc-dev-manager image create dev1 myapp-ready
```

**4. Create more containers**

```bash
lxc-dev-manager container create dev2 myapp-ready
lxc-dev-manager container create dev3 myapp-ready
lxc-dev-manager container create dev4 myapp-ready
lxc-dev-manager container create dev5 myapp-ready
```

**5. Start working on different branches**

```bash
# dev1 - feature work
lxc-dev-manager ssh dev1
cd ~/myapp && git checkout -b feature/new-auth
exit

# dev2 - another feature
lxc-dev-manager ssh dev2
cd ~/myapp && git checkout -b feature/dashboard
exit

# And so on...
```

When you finish a task in dev3, just switch branches and reuse it:

```bash
lxc-dev-manager ssh dev3
cd ~/myapp
git checkout main && git pull
git checkout -b feature/next-task
```

### Managing with Zellij or Tmux

Use a terminal multiplexer to control all containers from one screen.

**Zellij layout example:**

```
┌─────────────────┬─────────────────┐
│ dev1            │ dev2            │
│ claude working  │ claude working  │
│ on auth         │ on dashboard    │
├─────────────────┼─────────────────┤
│ dev3            │ dev4            │
│ running tests   │ claude trying   │
│                 │ new approach    │
├─────────────────┴─────────────────┤
│ dev5 - available                  │
└───────────────────────────────────┘
```

**Start sessions:**

```bash
# Terminal 1
lxc-dev-manager ssh dev1
cd ~/myapp && claude

# Terminal 2
lxc-dev-manager ssh dev2
cd ~/myapp && claude

# And so on...
```

**Zellij config (`~/.config/zellij/layouts/dev.kdl`):**

```kdl
layout {
    pane split_direction="vertical" {
        pane {
            command "lxc-dev-manager"
            args "ssh" "dev1"
        }
        pane {
            command "lxc-dev-manager"
            args "ssh" "dev2"
        }
    }
    pane split_direction="vertical" {
        pane {
            command "lxc-dev-manager"
            args "ssh" "dev3"
        }
        pane {
            command "lxc-dev-manager"
            args "ssh" "dev4"
        }
    }
}
```

Start with: `zellij --layout dev`

### Running Tests and E2E

Each container can run its own test suite without interference:

```bash
# In dev1 container
cd ~/myapp
npm test
npm run e2e

# In dev2 container (simultaneously)
cd ~/myapp
npm test
npm run e2e
```

Tests in one container don't affect others. Database states, file changes, and running services are all isolated.

### Using AI Tools Freely

AI coding tools like Claude Code can:
- Edit any files
- Run arbitrary commands
- Install packages
- Start services
- Use MCP servers

All within the container's isolation. If something breaks, reset:

```bash
lxc-dev-manager container reset dev1
```

### Inspecting Before Merge

Before merging a branch, proxy the container to inspect the running application:

```bash
# From your host
lxc-dev-manager proxy dev1
```

Now open `http://localhost:3000` in your browser to see the feature running. Check the UI, test interactions, verify everything works.

```bash
# Check another branch
lxc-dev-manager proxy dev2
```

### The Driver

Once tasks are complete and QA'd, someone (or something) needs to merge and run integration tests. This is the "driver" role.

The driver can be:
- **You** - Manually review PRs and merge
- **A Claude instance** - In another container or on your host, reviewing and merging branches
- **CI/CD** - Let your pipeline handle merges after approval

A typical flow:

1. dev1 finishes task → you QA via proxy → ask Claude to create branch and push
2. dev2 finishes task → you QA via proxy → ask Claude to create branch and push
3. Driver merges branches, runs full test suite, resolves conflicts
4. Repeat

The driver doesn't need to be in a container—it can run on your host machine. But if you want full isolation for the merge/test process, create a dedicated container for it.

### Quick Reset

If an experiment goes wrong:

```bash
# Reset to initial state
lxc-dev-manager container reset dev4

# Or reset to a checkpoint you created
lxc-dev-manager container reset dev4 before-risky-change
```

Container is back to a clean state in seconds.

### Tips

- **Create snapshots before risky changes**: `lxc-dev-manager container snapshot create dev1 checkpoint`
- **Reuse containers**: When done with a task, just switch branches instead of creating new containers
- **Clone for quick setup**: `lxc-dev-manager container clone dev1 dev6` copies current state instantly
- **Reset when stuck**: If a container gets messy, reset it and start fresh
