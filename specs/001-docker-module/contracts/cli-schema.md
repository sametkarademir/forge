# CLI Contract: `forge docker`

**Branch**: `001-docker-module` | **Date**: 2026-05-08

This document is the authoritative contract for the `forge docker` command tree. Each
subcommand entry defines flags, arguments, exit codes, stdout/stderr behaviour, and example
invocations.

---

## Global conventions

- **Argument validation** happens before any Docker call. Invalid input exits 1 with a message
  to stderr; no Docker operation is started.
- **Error output** goes to stderr with a red `✗` prefix. Success output goes to stdout.
- **`--yes` / `-y`** skips confirmation prompts on destructive commands (`reset`, `remove`).
- All commands accept `--help` / `-h` and return exit 0.

---

## `forge docker create <project> --engine <engine>`

**Purpose**: Create a new managed database container for `<project>`.

### Flags

| Flag | Short | Type | Default | Required | Description |
|---|---|---|---|---|---|
| `--engine` | `-e` | string | — | ✅ | Engine name (`postgres`, `mssql`, `mysql`) |
| `--image` | | string | engine default | | Override Docker image tag |
| `--user` | | string | config default | | Database username |
| `--password` | | string | config default | | Database password |
| `--db` | | string | config default | | Database name |

### Arguments

| Position | Name | Description |
|---|---|---|
| 1 | `project` | Project slug. Must match `^[a-z0-9][a-z0-9-]{0,62}$`. |

### Behaviour

1. Validate project name regex.
2. Validate engine is registered.
3. Validate password against engine rules.
4. Check no container `forge-<project>-<engine>` already exists.
5. Find next free port in `[port_range_start, port_range_end]`.
6. Ensure `forge-net` network exists (create if not).
7. Create volume `forge-<project>-<engine>-data`.
8. Create and start container with labels and env vars.
9. Print connection string to stdout.
10. Show "waiting for DB…" spinner until port accepts TCP or timeout.

### Exit codes

| Code | Condition |
|---|---|
| 0 | Container created and running |
| 1 | Validation error, project already exists, port range exhausted, Docker error |

### stdout (success)

```
✓ Created forge-todeb-postgres on port 15000
  Connection: postgres://forge:forge_dev@localhost:15000/forge
⠿ waiting for DB… (ready in 2s)
```

### stderr (error examples)

```
✗ project "todeb" already exists — use 'forge docker reset todeb' or 'forge docker remove todeb'
✗ password does not meet mssql requirements: missing uppercase letter, missing special character
✗ port range 15000–15999 is exhausted — increase docker.port_range_end in ~/.forge/config.yaml
```

---

## `forge docker list`

**Purpose**: List all forge-managed containers on the local Docker daemon.

### Flags

None.

### Behaviour

1. Query Docker for all containers with label `forge.managed=true`.
2. Render table to stdout.

### Exit codes

| Code | Condition |
|---|---|
| 0 | Always (empty list is not an error) |

### stdout

```
PROJECT    ENGINE    STATUS    PORT    UPTIME
todeb      postgres  running   15000   2h 14m
mediazone  mysql     exited    15001   —
```

Empty state: `No managed containers found.`

---

## `forge docker status <project>`

**Purpose**: Print detailed info for one project's container.

### Arguments

| Position | Name | Description |
|---|---|---|
| 1 | `project` | Project slug |

### Behaviour

1. Find container by label `forge.project=<project>` AND `forge.managed=true`.
2. Inspect container; mask passwords in env summary.
3. Print structured output to stdout.

### Exit codes

| Code | Condition |
|---|---|
| 0 | Container found (running or stopped) |
| 1 | No managed container for project |

### stdout (success)

```
Project:     todeb
Engine:      postgres
Status:      running
Image:       postgres:16-alpine
Port:        15000
Volume:      forge-todeb-postgres-data
Created:     2026-05-08T10:00:00Z
Uptime:      2h 14m

Environment:
  POSTGRES_USER      forge
  POSTGRES_PASSWORD  ****
  POSTGRES_DB        forge

Connection: postgres://forge:****@localhost:15000/forge
```

### stderr (error)

```
✗ no managed container found for project "todeb"
  Known projects: mediazone, internal-tools
```

---

## `forge docker conn <project>`

**Purpose**: Print only the connection string (with real password) to stdout for piping.

### Arguments

| Position | Name | Description |
|---|---|---|
| 1 | `project` | Project slug |

### Behaviour

1. Find container by labels.
2. Print connection string to stdout. No decoration, no trailing newline beyond `\n`.

### Exit codes

| Code | Condition |
|---|---|
| 0 | Connection string printed |
| 1 | No managed container for project |

### stdout (success)

```
postgres://forge:forge_dev@localhost:15000/forge
```

### stderr (error)

```
✗ no managed container found for project "todeb"
```

---

## `forge docker reset <project>`

**Purpose**: Wipe and recreate the database for a project (same port, same config).

### Flags

| Flag | Short | Type | Default | Description |
|---|---|---|---|---|
| `--yes` | `-y` | bool | false | Skip confirmation |

### Arguments

| Position | Name | Description |
|---|---|---|
| 1 | `project` | Project slug |

### Behaviour

1. Find managed container; error if not found.
2. Read port from `forge.host_port` label.
3. Check port is still free; hard error if occupied.
4. Prompt for confirmation (skip if `--yes`).
5. Stop and remove container (force-remove).
6. Remove volume.
7. Re-run create flow with same config (from labels).
8. Print connection string + readiness spinner.

### Exit codes

| Code | Condition |
|---|---|
| 0 | Reset complete |
| 1 | Project not found, port conflict, user cancelled, Docker error |

### stdout (success)

```
⚠ This will DELETE all data for project "todeb". Continue? (y/N) y
✓ Reset forge-todeb-postgres on port 15000
  Connection: postgres://forge:forge_dev@localhost:15000/forge
⠿ waiting for DB… (ready in 3s)
```

### stderr (error — port conflict)

```
✗ port 15000 is occupied by another process
  Free the port or use:
    forge docker remove todeb && forge docker create todeb --engine postgres
```

---

## `forge docker remove <project>`

**Purpose**: Fully remove a project's container, volume, and network membership.

### Flags

| Flag | Short | Type | Default | Description |
|---|---|---|---|---|
| `--yes` | `-y` | bool | false | Skip confirmation |

### Arguments

| Position | Name | Description |
|---|---|---|
| 1 | `project` | Project slug |

### Behaviour

1. Find container by labels.
2. Prompt for confirmation (skip if `--yes`).
3. Stop container (if running).
4. Remove container.
5. Remove volume.
6. Disconnect from `forge-net` (network itself is NOT removed).
7. If any resource is already absent, continue silently (idempotent).

### Exit codes

| Code | Condition |
|---|---|
| 0 | Remove complete (including already-absent case) |
| 1 | Container found but lacks `forge.managed=true` (safety boundary violation), Docker error |

### stderr (safety refusal)

```
✗ container "forge-todeb-postgres" exists but is not managed by forge
  (missing label forge.managed=true) — refusing to remove
```

---

## `forge docker engines`

**Purpose**: List registered engines and their default images.

### Flags

None.

### stdout

```
ENGINE    DEFAULT IMAGE
postgres  postgres:16-alpine
mssql     mcr.microsoft.com/mssql/server:2022-latest
mysql     mysql:8.4
```

### Exit codes

| Code | Condition |
|---|---|
| 0 | Always |
