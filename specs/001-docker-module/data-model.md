# Data Model: Docker Module

**Branch**: `001-docker-module` | **Date**: 2026-05-08

The docker module has no database. All persistent state is encoded in Docker labels on
containers and volumes. The types below describe the Go structs the service layer works with.

---

## ProjectInfo

Derived by inspecting a container's labels and metadata. Read-only from the CLI's perspective
(Docker is the authoritative source).

| Field | Type | Source | Notes |
|---|---|---|---|
| `Name` | `string` | `forge.project` label | Validated `[a-z0-9-]+`, max 64 chars |
| `Engine` | `string` | `forge.engine` label | Must match a registered engine name |
| `ContainerID` | `string` | Docker container ID | Short ID for display |
| `ContainerName` | `string` | Docker container name | `forge-<project>-<engine>` |
| `VolumeName` | `string` | Derived from labels | `forge-<project>-<engine>-data` |
| `HostPort` | `int` | `forge.host_port` label | Host-side port; >0 |
| `Status` | `string` | Docker container state | `running`, `exited`, `paused`, … |
| `Image` | `string` | Docker image field | e.g. `postgres:16-alpine` |
| `CreatedAt` | `time.Time` | `forge.created_at` label | RFC 3339 |
| `Uptime` | `time.Duration` | Computed from `StartedAt` | Zero when stopped |
| `EnvSummary` | `map[string]string` | Container env vars | Passwords replaced with `****` |
| `ConnectionString` | `string` | Engine.ConnectionString() | Engine-native format |

**Validation rules**:
- `Name` MUST match `^[a-z0-9][a-z0-9-]{0,62}$`
- `Engine` MUST be a key in the engine registry
- `HostPort` MUST be in `[port_range_start, port_range_end]`

**State transitions**:

```
(none) ──create──► running ──stop──► exited
                      ▲                │
                      └──start─────────┘
running/exited ──reset──► (volume deleted) ──► running (fresh)
running/exited ──remove──► (none)
```

---

## CreateOptions

Input to `service.CreateProject`. Populated by the `create` command from flags + config defaults.

| Field | Type | Default | Notes |
|---|---|---|---|
| `ProjectName` | `string` | (required) | Validated before use |
| `Engine` | `string` | (required via `--engine`) | Must be registered |
| `Image` | `string` | `engine.DefaultImage()` | Overridable via `--image` |
| `User` | `string` | `config.docker.default_user` | Overridable via `--user` |
| `Password` | `string` | `config.docker.default_password` | Overridable via `--password` |
| `Database` | `string` | `config.docker.default_db` | Overridable via `--db` |

---

## Engine (interface)

Implemented by each engine file. Stateless — no fields.

| Method | Returns | Notes |
|---|---|---|
| `Name()` | `string` | Canonical lowercase name (`"postgres"`, etc.) |
| `DefaultImage()` | `string` | Docker image tag |
| `DefaultPort()` | `int` | Container-internal port |
| `EnvVars(user, password, db string)` | `map[string]string` | Env vars for `ContainerConfig.Env` |
| `ConnectionString(host string, hostPort int, user, password, db string)` | `string` | Engine-native DSN |
| `ValidatePassword(password string)` | `error` | `nil` = valid |

---

## Config (viper-backed)

Read from `~/.forge/config.yaml`. Accessed via typed getters in `internal/core/config`.

| Key | Go type | Default | Notes |
|---|---|---|---|
| `docker.default_user` | `string` | `"forge"` | |
| `docker.default_password` | `string` | `"forge_dev"` | Fails mssql validation — intentional |
| `docker.default_db` | `string` | `"forge"` | |
| `docker.port_range_start` | `int` | `15000` | |
| `docker.port_range_end` | `int` | `15999` | |
| `docker.readiness_timeout_seconds` | `int` | `30` | |

---

## Docker Labels (contract with daemon)

Labels applied to every managed container by `create` and validated by every read operation.

| Label key | Value | Required |
|---|---|---|
| `forge.managed` | `"true"` | ✅ |
| `forge.project` | project slug | ✅ |
| `forge.engine` | engine name | ✅ |
| `forge.host_port` | decimal port number | ✅ |
| `forge.created_at` | RFC 3339 timestamp | ✅ |

Any container missing `forge.managed=true` is treated as unmanaged and MUST NOT be touched.

---

## Naming Derivation (from `service/naming.go`)

| Resource | Pattern | Example |
|---|---|---|
| Container | `forge-<project>-<engine>` | `forge-todeb-postgres` |
| Volume | `forge-<project>-<engine>-data` | `forge-todeb-postgres-data` |
| Network | `forge-net` | `forge-net` (shared; never removed) |
