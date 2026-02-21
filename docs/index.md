---
layout: home

hero:
  name: lxc-dev-manager
  text: Isolated Dev Environments
  tagline: Create, manage, and share development containers with LXC. Reproducible setups in seconds.
  actions:
    - theme: brand
      text: Get Started
      link: /guide/about
    - theme: alt
      text: View on GitHub
      link: https://github.com/yourusername/lxc-dev-manager

features:
  - icon: 📦
    title: Isolated Environments
    details: Each project gets its own container with full root access. Install anything without affecting your host system.
  - icon: 📸
    title: Images & Snapshots
    details: Create reusable images from containers. Use snapshots for quick checkpoints and reset.
  - icon: 🗂️
    title: Multi-Project Support
    details: Work on multiple projects simultaneously. Each project is namespaced to avoid container name conflicts.
  - icon: 🔌
    title: Port Forwarding
    details: Access container services on localhost. No complex networking setup required.
---

## Quick Start

```bash
# Install LXC/LXD (Ubuntu)
sudo snap install lxd
sudo lxd init --minimal
sudo usermod -aG lxd $USER && newgrp lxd

# Download lxc-dev-manager
# (see installation guide for download link)

# Create your first project
cd ~/projects/my-webapp
lxc-dev-manager create
lxc-dev-manager container create dev ubuntu:24.04
lxc-dev-manager ssh dev
```

## Why lxc-dev-manager?

**Problem**: Setting up development environments is repetitive and error-prone. Installing dependencies on your host can lead to conflicts. Virtual machines are heavy and slow.

**Solution**: LXC containers provide near-native performance with full isolation. lxc-dev-manager makes them easy to use for development workflows:

- **Project-based**: Containers are namespaced per project (e.g., `webapp-dev`, `api-dev`)
- **Ready to use**: Containers come with a `dev` user, passwordless sudo, and SSH enabled
- **Shareable**: Snapshot your configured environment and share it with your team
- **Lightweight**: Start containers in seconds, not minutes
