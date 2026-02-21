# Configuration Reference

lxc-dev-manager uses a `containers.yaml` file to store project configuration. This file is created automatically when you run `lxc-dev-manager create`.

## File Location

The configuration file is always located in the current working directory:

```
~/projects/webapp/
└── containers.yaml
```

## File Format

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
  test:
    image: nodejs-ready
    ports:
      - 3000
      - 4000
```

## Fields

### project

**Type**: `string`
**Required**: Yes

The project name used to prefix all container names in LXC.

```yaml
project: webapp
```

When you create a container named `dev`, the actual LXC container will be named `webapp-dev`.

::: warning
The project name must contain only:
- Letters (a-z, A-Z)
- Numbers (0-9)
- Hyphens (-)
- Underscores (_)
:::

---

### defaults

**Type**: `object`
**Required**: No

Default settings applied to all containers unless overridden.

```yaml
defaults:
  ports:
    - 5173
    - 8000
    - 5432
```

#### defaults.ports

**Type**: `array of integers`
**Required**: No
**Default**: `[5173, 8000, 5432]`

Default ports to forward when running `lxc-dev-manager proxy <container>`.

Common port conventions:
| Port | Common Use |
|------|------------|
| 3000 | React, Next.js, Express |
| 4000 | GraphQL |
| 5000 | Flask |
| 5173 | Vite |
| 5432 | PostgreSQL |
| 6379 | Redis |
| 8000 | Django, FastAPI |
| 8080 | General HTTP |
| 27017 | MongoDB |

#### defaults.user

**Type**: `object`
**Required**: No

Default user credentials for all containers in the project.

```yaml
defaults:
  user:
    name: developer
    password: secret123
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | `dev` | Username to create in containers |
| `password` | string | `dev` | Password for the user |

::: tip
If not specified, containers default to username `dev` with password `dev`.
:::

::: info
The `ssh` command uses this user configuration by default. Running `lxc-dev-manager ssh dev` will log in as the configured user. Use `-u root` to get a root shell instead.
:::

---

### containers

**Type**: `object`
**Required**: No

Map of container names to their configurations.

```yaml
containers:
  dev:
    image: ubuntu:24.04
  staging:
    image: my-base-image
    ports:
      - 3000
```

Each container has the following fields:

#### containers.\<name\>.image

**Type**: `string`
**Required**: Yes (when container exists)

The image used to create the container. This is recorded when you run `container create`.

Can be:
- Official LXC image: `ubuntu:24.04`, `debian/12`, `images:alpine/3.19`
- Local snapshot image: `my-base-image`

```yaml
containers:
  dev:
    image: ubuntu:24.04
```

#### containers.\<name\>.ports

**Type**: `array of integers`
**Required**: No

Override the default ports for this specific container.

```yaml
containers:
  dev:
    image: ubuntu:24.04
    ports:
      - 3000    # React
      - 3001    # React admin
      - 5432    # PostgreSQL
```

When you run `lxc-dev-manager proxy dev`, only these ports will be forwarded, not the defaults.

#### containers.\<name\>.user

**Type**: `object`
**Required**: No

Override the default user credentials for this specific container.

```yaml
containers:
  dev:
    image: ubuntu:24.04
    user:
      name: admin
      password: adminpass
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Username for this container |
| `password` | string | Password for this container |

::: tip
Per-container user settings override project defaults. Useful when different containers need different credentials. The `ssh` command will automatically use this user when connecting to the container.
:::

#### containers.\<name\>.snapshots

**Type**: `array`
**Required**: No (auto-managed)

Metadata for named snapshots. This field is automatically populated when you create snapshots using `container snapshot create`.

```yaml
containers:
  dev:
    image: ubuntu:24.04
    snapshots:
      - name: before-refactor
        description: "Before major refactor"
        created: "2024-01-15T14:22:00Z"
      - name: checkpoint
        created: "2024-01-15T16:45:00Z"
```

::: warning
Do not manually edit this field. Use the `container snapshot create` and `container snapshot delete` commands to manage snapshots.
:::

---

## Examples

### Minimal Configuration

```yaml
project: webapp
defaults:
  ports:
    - 5173
    - 8000
    - 5432
containers: {}
```

### Single Container

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
```

### Multiple Containers with Custom Ports

```yaml
project: fullstack-app
defaults:
  ports:
    - 8000
containers:
  frontend:
    image: nodejs-ready
    ports:
      - 3000    # React dev server
      - 6006    # Storybook
  backend:
    image: python-base
    ports:
      - 8000    # Django
      - 5432    # PostgreSQL
  worker:
    image: python-base
    ports:
      - 6379    # Redis
```

### Microservices Setup

```yaml
project: microservices
defaults:
  ports: []
containers:
  gateway:
    image: nodejs-ready
    ports:
      - 4000
  users-service:
    image: go-base
    ports:
      - 4001
  orders-service:
    image: go-base
    ports:
      - 4002
  notifications:
    image: nodejs-ready
    ports:
      - 4003
```

### Data Science Environment

```yaml
project: ml-project
defaults:
  ports:
    - 8888    # Jupyter
    - 6006    # TensorBoard
    - 5000    # MLflow
containers:
  dev:
    image: ubuntu:24.04
  gpu:
    image: nvidia-cuda-base
    ports:
      - 8888
      - 6006
```

### Custom User Configuration

```yaml
project: secure-app
defaults:
  user:
    name: developer
    password: devpass123
  ports:
    - 5173
    - 8000
containers:
  dev:
    image: ubuntu:24.04
    # Uses default user: developer/devpass123
  admin:
    image: ubuntu:24.04
    user:
      name: admin
      password: adminpass456
    # Uses override: admin/adminpass456
  test:
    image: ubuntu:24.04
    user:
      name: tester
    # Uses: tester/devpass123 (password falls back to default)
```

## Editing the Configuration

You can edit `containers.yaml` directly with any text editor. Changes to ports take effect immediately when you run `lxc-dev-manager proxy`.

::: warning
Do not manually add or remove containers from the `containers` section. Use the `container create` and `remove` commands instead, as they also manage the actual LXC containers.
:::

### Safe to Edit

- `defaults.ports` - Change default ports anytime
- `defaults.user` - Change default user for new containers (doesn't affect existing)
- `containers.<name>.ports` - Change per-container ports anytime

### Avoid Editing

- `project` - Changing this will break the link to existing LXC containers
- `containers.<name>.image` - This is just metadata; changing it doesn't affect the container
- `containers.<name>.user` - Changing this doesn't update the user inside an existing container
- `containers.<name>.snapshots` - Auto-managed by snapshot commands

## Configuration Precedence

### Ports

When determining which ports to forward for a container:

1. If `containers.<name>.ports` is specified, use those ports
2. Otherwise, use `defaults.ports`
3. If neither is specified, use the built-in defaults: `[5173, 8000, 5432]`

```yaml
project: webapp
defaults:
  ports:
    - 5173
    - 8000
containers:
  dev:
    image: ubuntu:24.04
    # Uses defaults: 5173, 8000
  api:
    image: ubuntu:24.04
    ports:
      - 3000
    # Uses override: 3000 only
```

### User Credentials

When determining user credentials for container creation:

1. If `containers.<name>.user.name` is specified, use it
2. Otherwise, use `defaults.user.name`
3. If neither is specified, use `dev`

The same precedence applies to passwords:

1. If `containers.<name>.user.password` is specified, use it
2. Otherwise, use `defaults.user.password`
3. If neither is specified, use `dev`

```yaml
project: webapp
defaults:
  user:
    name: developer
    password: secret
containers:
  dev:
    image: ubuntu:24.04
    # Uses defaults: developer/secret
  admin:
    image: ubuntu:24.04
    user:
      name: admin
      password: adminpass
    # Uses override: admin/adminpass
  hybrid:
    image: ubuntu:24.04
    user:
      name: custom
    # Uses: custom/secret (name overridden, password from defaults)
```
