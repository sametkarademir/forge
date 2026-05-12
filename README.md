# forge

`forge` is a developer productivity CLI for macOS, written in Go. Its first module — `docker` — manages per-project database containers on your local Docker daemon. Every project gets an isolated container, volume, and network membership. No database configuration files, no docker-compose clutter, no manually remembered ports.

---

## How it works

### Docker labels as the only source of truth

`forge` does **not** maintain a local state file or database. Every container it creates carries a set of Docker labels that describe it completely:

| Label | Example value | Purpose |
|---|---|---|
| `forge.managed` | `true` | Marks the container as forge-managed |
| `forge.project` | `todeb` | The project this container belongs to |
| `forge.engine` | `postgres` | The database engine |
| `forge.host_port` | `15000` | The host port the container is bound to |
| `forge.created_at` | `2026-05-08T10:00:00Z` | Creation timestamp (RFC 3339) |
| `forge.user` | `forge` | Database username |
| `forge.db` | `forge` | Database name |

When you run `forge docker list` or `forge docker status todeb`, the CLI queries the Docker daemon for containers with `forge.managed=true` and reconstructs all the information it needs from these labels. There is no hidden state.

### Naming conventions

Every managed resource follows a deterministic naming scheme so you can always find them in `docker ps` or `docker volume ls`:

| Resource | Pattern | Example |
|---|---|---|
| Container | `forge-<project>-<engine>` | `forge-todeb-postgres` |
| Volume | `forge-<project>-<engine>-data` | `forge-todeb-postgres-data` |
| Network | `forge-net` | shared across all projects |

### Port allocation

Ports are auto-allocated starting from `port_range_start` (default: 15000). The CLI walks the range with `net.Listen` to probe for a free port, then binds the container to it. The chosen port is stored in the `forge.host_port` label so it can always be recovered from Docker — even if you lose your shell history.

There is no `--port` flag. Port-range exhaustion is an explicit error that instructs you to widen `docker.port_range_end` in config.

### Engine system

Supported database engines implement a single `Engine` interface and self-register via `init()`. Adding a new engine only requires one new file — no changes to command handlers or the Docker client wrapper.

Each engine declares:
- Its Docker image and default internal port
- The environment variables it needs (user, password, database)
- The connection string format (engine-native DSN)
- Password validation rules (e.g., mssql SA_PASSWORD complexity)
- Where its data lives inside the container (for volume mounting)

Three engines ship at launch: **postgres**, **mssql**, **mysql**.

### Safety boundaries

The CLI enforces a strict rule: **it will never touch a container, volume, or network that lacks the `forge.managed=true` label**. Every list, modify, and delete operation filters by this label first. There is no flag, no environment variable, and no config option that bypasses this check.

---

## Installation

**Prerequisites:** Go 1.23+, Docker Desktop (or any Docker Engine) running on the machine.

```bash
# Install from source
make install

# Or via go install
go install github.com/sametkarademir/forge/cmd/forge@latest
```

Verify:

```bash
forge --help
```

---

## Configuration

On first run `forge` creates `~/.forge/config.yaml` with default values:

```yaml
docker:
  default_user: forge
  default_password: forge_dev
  default_db: forge
  port_range_start: 15000
  port_range_end: 15999
  readiness_timeout_seconds: 30
```

Edit this file to change defaults for all future `create` commands. Per-invocation overrides are also available via flags (see below).

> **Note:** The default password `forge_dev` does not meet the mssql SA_PASSWORD complexity requirements. You must pass `--password` explicitly when creating an mssql container.

---

## Docker module

### `engines` — list supported database engines

```bash
forge docker engines
```

```
ENGINE    DEFAULT IMAGE
mssql     mcr.microsoft.com/mssql/server:2022-latest
mysql     mysql:8.4
postgres  postgres:16-alpine
```

Use this to see which engines are available and what Docker image each one uses by default.

---

### `create` — create a project database

```bash
forge docker create <project> --engine <engine> [flags]
```

**Flags:**

| Flag | Short | Default | Description |
|---|---|---|---|
| `--engine` | `-e` | *(required)* | Engine name: `postgres`, `mssql`, `mysql` |
| `--image` | | engine default | Override the Docker image tag |
| `--user` | | config default | Database username |
| `--password` | | config default | Database password |
| `--db` | | config default | Database name |

**Example — Postgres with defaults:**

```bash
forge docker create todeb --engine postgres
```

```
✓ Created forge-todeb-postgres on port 15000
  Connection: postgres://forge:forge_dev@localhost:15000/forge
⠿ waiting for DB… ✓ DB ready (in 2s)
```

**Example — MySQL with custom credentials:**

```bash
forge docker create myapp --engine mysql \
  --user myuser --password 'MyPass1!' --db myapp_db
```

**Example — mssql with a compliant password:**

```bash
forge docker create erp --engine mssql --password 'Str0ng!Pass'
```

**Example — pin a specific image version:**

```bash
forge docker create legacy --engine postgres --image postgres:14-alpine
```

**What `create` does, step by step:**

1. Validates the project name (must match `^[a-z0-9][a-z0-9-]{0,62}$`)
2. Checks the engine is registered
3. Validates the password against the engine's complexity rules
4. Checks no container for this project already exists
5. Finds the next free port in the configured range
6. Creates the `forge-net` network if it does not yet exist
7. Creates the named volume with forge labels
8. Creates and starts the container with all labels, port binding, and volume mount
9. Prints the connection string immediately
10. Polls `localhost:<port>` via TCP until the database is ready (or the readiness timeout elapses)

If the project already exists, `create` exits with an error and suggests `reset` or `remove`.

---

### `list` — list all managed containers

```bash
forge docker list
```

```
PROJECT    ENGINE    STATUS    PORT    UPTIME
todeb      postgres  running   15000   2h 14m
mediazone  mysql     exited    15001   —
erp        mssql     running   15002   45m
```

Only containers with `forge.managed=true` are shown. All other Docker containers on the machine are invisible to this command. If there are no managed containers, a friendly message is printed.

---

### `status` — inspect a project's container

```bash
forge docker status <project>
```

```
Project:   todeb
Engine:    postgres
Status:    running
Image:     postgres:16-alpine
Port:      15000
Volume:    forge-todeb-postgres-data
Created:   2026-05-08T10:00:00Z
Uptime:    2h 14m

Environment:
  POSTGRES_USER              forge
  POSTGRES_PASSWORD          ****
  POSTGRES_DB                forge

Connection: postgres://forge:****@localhost:15000/forge
```

Passwords are always masked in `status` output. Use `conn` to get the real password for piping into tools.

Works even when the container is stopped — state will show as `exited`.

If the project does not exist, the error message lists all known project names.

---

### `conn` — get connection string for piping

```bash
forge docker conn <project>
```

```
postgres://forge:forge_dev@localhost:15000/forge
```

`conn` prints **only** the connection string to stdout — no labels, no decoration, no trailing spaces. Designed for piping and shell substitution.

**Common patterns:**

```bash
# Copy to clipboard (macOS)
forge docker conn todeb | pbcopy

# Connect directly with psql
psql "$(forge docker conn todeb)"

# Use in a script
export DATABASE_URL="$(forge docker conn todeb)"

# Pass to a migration tool
migrate -database "$(forge docker conn todeb)" -path ./migrations up
```

Errors are printed to stderr and exit code is non-zero, so piping fails safely.

---

### `reset` — wipe and recreate the database

```bash
forge docker reset <project> [--yes]
```

Stops the container, deletes the volume (and all data), and creates a fresh container on the **same port** with the **same credentials**. The project name, engine, image, and connection string stay identical.

```bash
forge docker reset todeb          # prompts: "This will DELETE all data…"
forge docker reset todeb --yes    # skips prompt
```

**Use case:** You want a clean database without having to remember the port and credentials you used at creation time.

**What reset does NOT do:** It does not pick a new port. If the original port is now occupied by another process, `reset` exits with a hard error and tells you exactly what to do:

```
✗ port 15000 is occupied by another process
  Free the port or use:
    forge docker remove todeb && forge docker create todeb --engine postgres
```

---

### `remove` — remove a project entirely

```bash
forge docker remove <project> [--yes]
```

Removes the container, its volume, and its membership from the shared network. Running `remove` on a project that no longer exists exits successfully — the operation is idempotent.

```bash
forge docker remove todeb          # prompts for confirmation
forge docker remove todeb --yes    # skips prompt
```

**Safety check:** If a container named `forge-<project>-<engine>` exists but lacks the `forge.managed=true` label, `remove` refuses with an explicit explanation:

```
✗ container "forge-ghost-postgres" exists but is not managed by forge
  (missing label forge.managed=true) — refusing to remove
```

The `forge-net` network is **never** removed by `remove` — it is shared across all projects.

---

## Edge cases

| Situation | Behaviour |
|---|---|
| Docker daemon not running | Friendly error: "Docker is not running — run: open -a Docker" |
| Project name with spaces or slashes | Rejected early: must match `^[a-z0-9][a-z0-9-]{0,62}$` |
| Port range exhausted | Error with instructions to widen `docker.port_range_end` in config |
| mssql with weak password | Password validated before any Docker call; clear message naming each failed rule |
| Port conflict during `reset` | Hard error naming the port; instructions to remove + recreate |
| Container with missing engine label | Treated as unmanaged; CLI refuses to touch it |
| Running a destructive command twice | Exits 0 both times (idempotent) |
| DB readiness timeout | Warns but exits 0; connection string is still printed and valid |

---

## Project structure

```
forge/
├── cmd/forge/main.go                    # entry point; loads modules via registry
├── internal/
│   ├── core/
│   │   ├── config/config.go               # viper-backed config + auto-write on first run
│   │   ├── logger/logger.go               # colored stdout/stderr (fatih/color)
│   │   ├── registry/registry.go           # Module interface + global registry
│   │   └── ui/ui.go                       # confirm prompt + table renderer
│   └── modules/
│       └── docker/
│           ├── module.go                  # self-registers; returns root cobra command
│           ├── commands/                  # one file per subcommand (thin cobra wrappers)
│           │   ├── create.go
│           │   ├── list.go
│           │   ├── status.go
│           │   ├── conn.go
│           │   ├── reset.go
│           │   ├── remove.go
│           │   └── engines.go
│           ├── service/                   # business logic; no cobra imports
│           │   ├── service.go             # CreateProject, ListProjects, Reset…
│           │   ├── naming.go              # ContainerName, VolumeName, NetworkName
│           │   └── ports.go              # NextFreePort, IsPortFree
│           ├── client/client.go           # Docker SDK wrapper
│           └── engines/                   # one file per engine
│               ├── engine.go              # Engine interface + registry
│               ├── postgres.go
│               ├── mssql.go
│               └── mysql.go
├── test/smoke/
│   ├── docker_create.sh
│   ├── docker_reset.sh
│   └── docker_remove.sh
├── Makefile
└── go.mod
```

---

## Makefile targets

```bash
make build    # compile binary to ./forge
make install  # go install (places binary on PATH)
make vet      # go vet ./...
make fmt      # gofmt -l ./...  (lists unformatted files)
make smoke    # run all smoke tests (requires Docker)
```

---

## Adding a new engine

Create a single file under `internal/modules/docker/engines/`:

```go
package engines

import "fmt"

type redis struct{}

func init() { Register(&redis{}) }

func (r *redis) Name() string           { return "redis" }
func (r *redis) DefaultImage() string   { return "redis:7-alpine" }
func (r *redis) DefaultPort() int       { return 6379 }
func (r *redis) DataDir() string        { return "/data" }
func (r *redis) PasswordEnvKey() string { return "REDIS_PASSWORD" }

func (r *redis) EnvVars(user, password, db string) map[string]string {
    return map[string]string{"REDIS_PASSWORD": password}
}

func (r *redis) ConnectionString(host string, hostPort int, user, password, db string) string {
    return fmt.Sprintf("redis://:%s@%s:%d", password, host, hostPort)
}

func (r *redis) ValidatePassword(password string) error { return nil }
```

No other files need to change. `forge docker engines` will list it immediately, and `forge docker create myproject --engine redis` will work.
