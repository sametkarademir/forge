# Feature Specification: Docker Module — Per-Project Database Container Management

**Feature Branch**: `001-docker-module`
**Created**: 2026-05-08
**Status**: Draft
**Input**: User description: "Build the first module of forge: docker"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Create a project database container (Priority: P1)

A developer working on the `todeb` project runs `forge docker create todeb --engine postgres`
and within seconds has a running Postgres container, isolated volume, and auto-allocated host port.
The connection string is printed at the end so it can be copied immediately.

**Why this priority**: This is the foundational operation. Every other command (list, status, reset,
remove) depends on containers having been created. Without `create`, the module has no value.

**Independent Test**: Run `forge docker create testproj --engine postgres` against a live Docker
daemon, verify a container named `forge-testproj-postgres` is running, a volume named
`forge-testproj-postgres-data` exists, and a postgres connection string is printed.

**Acceptance Scenarios**:

1. **Given** Docker is running and no `todeb` project exists, **When** the developer runs
   `forge docker create todeb --engine postgres`, **Then** a container
   `forge-todeb-postgres` is running, a volume `forge-todeb-postgres-data` is attached,
   a host port ≥ 15000 is allocated, labels `forge.managed=true`, `forge.project=todeb`,
   `forge.engine=postgres`, `forge.created_at=<rfc3339>`, and `forge.host_port=<n>`
   are set, the connection string is printed immediately, and a "waiting for DB…" spinner
   is shown until the port accepts a TCP connection (or a 30-second timeout elapses).

2. **Given** a `todeb` container already exists, **When** the developer runs
   `forge docker create todeb --engine postgres`, **Then** the CLI exits with an error message
   that includes a hint to use `reset` or `remove`.

3. **Given** the target port is in use, **When** the CLI allocates a port, **Then** it increments
   through the configured range until a free port is found and uses that port.

4. **Given** `--user`, `--password`, and `--db` flags are supplied, **When** the container is
   created, **Then** those values override the config defaults for this container.

---

### User Story 2 — List all managed containers (Priority: P2)

A developer runs `forge docker list` and sees a clean table of every forge-managed container
on the machine: project name, engine, status (running / stopped / exited), host port, and uptime.
Unrelated Docker containers are never shown.

**Why this priority**: Discoverability is essential — a developer needs to know which projects have
databases before interacting with them.

**Independent Test**: With two forge containers and several unrelated Docker containers running,
`forge docker list` shows exactly the two forge containers in table format and nothing else.

**Acceptance Scenarios**:

1. **Given** multiple forge-managed containers exist, **When** the developer runs
   `forge docker list`, **Then** a table is printed with columns: Project, Engine, Status,
   Port, Uptime — one row per managed container.

2. **Given** no forge-managed containers exist, **When** the developer runs
   `forge docker list`, **Then** the CLI prints a friendly "no managed containers found" message.

3. **Given** unrelated Docker containers are running, **When** `forge docker list` runs,
   **Then** only containers with `forge.managed=true` label appear in the output.

---

### User Story 3 — Inspect a single project's container (Priority: P2)

A developer runs `forge docker status todeb` and sees detailed information: image, port mapping,
volume name, environment summary with passwords masked, and the connection string. Works even when
the container is stopped, clearly indicating the stopped state.

**Why this priority**: Developers need to retrieve credentials and connection info during active
development without remembering what was generated at creation time.

**Independent Test**: After creating a `todeb` container and then stopping it, `forge docker
status todeb` prints detailed info with "stopped" state indicated and passwords masked.

**Acceptance Scenarios**:

1. **Given** a running `todeb` container, **When** `forge docker status todeb` is run, **Then**
   image, port, volume, masked env vars, and connection string are printed.

2. **Given** a stopped `todeb` container, **When** `forge docker status todeb` is run, **Then**
   the same info is shown with state clearly marked as stopped/exited.

3. **Given** no `todeb` project exists, **When** `forge docker status todeb` is run, **Then**
   an error is shown with a list of known project names.

---

### User Story 4 — Get connection string for piping (Priority: P2)

A developer runs `forge docker conn todeb` and receives only the connection string on stdout,
suitable for piping into `pbcopy` or shell substitution. No table, no extra decoration.

**Why this priority**: Scripting and shell integration are first-class use cases; a clean single-
line output is essential for reliable piping.

**Independent Test**: `forge docker conn todeb | pbcopy` puts a valid DSN in the clipboard with
no trailing whitespace or decorative output.

**Acceptance Scenarios**:

1. **Given** a `todeb` container (running or stopped), **When** `forge docker conn todeb` is
   run, **Then** exactly the connection string is printed to stdout with no additional text.

2. **Given** no `todeb` project exists, **When** `forge docker conn todeb` is run, **Then**
   an error is printed to stderr and the process exits non-zero.

---

### User Story 5 — Reset a project's database (Priority: P3)

A developer runs `forge docker reset todeb`, confirms the prompt, and gets a fresh empty
database. The container keeps the same name, the same port, and the same credentials. The old
volume (and all its data) is deleted.

**Why this priority**: A fast wipe-and-restart cycle is central to the disposable-database
workflow but less frequently needed than create/list/status.

**Independent Test**: After seeding data in `todeb`, run `forge docker reset todeb --yes`, then
verify the volume was recreated (new creation timestamp), the container is running on the same
port, and the database is empty.

**Acceptance Scenarios**:

1. **Given** a running `todeb` container with data, **When** the developer runs
   `forge docker reset todeb` and confirms, **Then** the container is stopped, the volume
   deleted, and a new container is started on the same port with the same config. Data is gone.

2. **Given** the original port is occupied by another process at reset time, **When** the
   developer runs `forge docker reset todeb`, **Then** the CLI exits with an error naming
   the port conflict and instructing the developer to free the port or use `remove` + `create`.

2. **Given** `--yes` / `-y` is passed, **When** `forge docker reset todeb --yes` is run,
   **Then** the confirmation prompt is skipped entirely.

4. **Given** no `todeb` project exists, **When** `forge docker reset todeb` is run, **Then**
   an error is shown with a list of known project names.

---

### User Story 6 — Remove a project entirely (Priority: P3)

A developer runs `forge docker remove todeb`, confirms, and the container, volume, and network
membership for `todeb` are all gone. The command is idempotent — running it twice does not error.

**Why this priority**: Cleanup is important but less frequent than daily development operations.

**Independent Test**: Run `forge docker remove todeb --yes`. Verify no container, volume, or
network membership remains for `todeb`. Run the same command again and verify no error is returned.

**Acceptance Scenarios**:

1. **Given** a `todeb` container exists, **When** `forge docker remove todeb --yes` is run,
   **Then** the container, volume, and network membership are removed with no error.

2. **Given** `todeb` is already removed, **When** `forge docker remove todeb --yes` is run
   again, **Then** the CLI exits successfully (idempotent).

3. **Given** a container named `forge-todeb-postgres` exists but lacks the
   `forge.managed=true` label, **When** `forge docker remove todeb` is run, **Then** the
   CLI refuses with an explanation of the safety boundary.

---

### User Story 7 — List supported engines (Priority: P3)

A developer runs `forge docker engines` and sees a table of supported database engines with
their default image tags.

**Why this priority**: Discovery of available engines matters during onboarding and when adding
new projects with unfamiliar engines.

**Independent Test**: `forge docker engines` prints a table with at least postgres, mssql, and
mysql rows, each showing the default image tag.

**Acceptance Scenarios**:

1. **Given** the CLI is installed, **When** `forge docker engines` is run, **Then** a table
   listing engine name and default image for each registered engine is printed.

---

### Edge Cases

- Docker daemon not running → print friendly error mentioning `open -a Docker` and exit non-zero.
- Port range exhausted (all ports from `port_range_start` to `port_range_end` in use) →
  error with message explaining the configured range and instructing the developer to widen it
  via `docker.port_range_end` in config. No `--port` flag exists on `create`.
- Project name contains characters invalid for container naming (spaces, slashes, etc.) →
  validate and reject early with a clear message.
- Container exists but its engine label is missing → treat as unmanaged and refuse to touch it.
- Password fails engine-specific complexity rules (e.g., mssql SA_PASSWORD) → validate before
  any Docker call and exit with a message naming the specific rule(s) violated.
- Port occupied by another process during `reset` → hard error naming the port and the conflict;
  instruct developer to free it or use `remove` + `create` on a fresh port.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The module MUST support engines `postgres`, `mssql`, and `mysql` at launch. The
  engine registry MUST be extensible without touching command handlers or the Docker client.
- **FR-002**: All resources created by the module MUST carry labels: `forge.managed=true`,
  `forge.project=<name>`, `forge.engine=<name>`, `forge.created_at=<rfc3339>`.
- **FR-003**: Container names MUST follow `forge-<project>-<engine>`. Volume names MUST follow
  `forge-<project>-<engine>-data`. The shared network MUST be named `forge-net`.
- **FR-004**: Host ports MUST be auto-allocated starting at `port_range_start` (default 15000),
  scanning forward to find a free port. The chosen port MUST be stored as label
  `forge.host_port=<n>` on the container so it is recoverable from Docker alone.
- **FR-005**: Default credentials MUST come from config keys `docker.default_user` and
  `docker.default_password`. These MUST be overridable per-invocation via `--user`, `--password`,
  and `--db` flags on `create`.
- **FR-006**: The Docker image MUST default to the engine's registered default and be overridable
  via `--image` on `create`.
- **FR-007**: The `reset` and `remove` commands MUST prompt for confirmation before destructive
  action. Passing `--yes` / `-y` MUST skip the prompt.
- **FR-008**: `list` and any bulk operation MUST filter exclusively by `forge.managed=true`
  label. The CLI MUST NEVER inspect, modify, or remove a resource lacking this label.
- **FR-009**: All tabular output MUST use the shared table writer. All log messages MUST use the
  shared logger. Only `conn` outputs raw text to stdout.
- **FR-010**: On first run (no config file present), the CLI MUST create `~/.forge/config.yaml`
  with documented default values.
- **FR-012**: For engines with password complexity requirements (e.g., mssql SA_PASSWORD), the
  CLI MUST validate the password against those rules before contacting Docker. If the password
  fails validation, the CLI MUST exit with a clear message describing the requirements and MUST
  NOT start or modify any container. Each engine's `Engine` implementation declares its own
  validation rule.
- **FR-011**: After `create`, the CLI MUST print the connection string immediately, then display
  a "waiting for DB…" spinner that exits once the database port accepts a TCP connection. The
  readiness timeout MUST be configurable (`docker.readiness_timeout_seconds`, default 30). On
  timeout, the CLI MUST warn that the DB is not yet ready but MUST NOT treat it as a failure.

### Key Entities

- **Project**: Identified by a slug name (e.g., `todeb`). Owns one container, one volume, and
  network membership. Fully described by Docker labels — no local state file required.
- **Engine**: A registered database type (postgres, mssql, mysql). Provides default image,
  default port, native connection string template (engine-specific format), password validation
  rules, and required environment variable mappings.
- **ManagedContainer**: A Docker container carrying all forge labels. The CLI's view of a
  running or stopped project database.
- **Config**: `~/.forge/config.yaml` — stores default credentials, port range, and other
  user-level preferences.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can go from zero to a printed connection string for a new project in
  under 5 seconds on a machine where Docker is already running and the engine image is cached.
  The DB readiness spinner resolves within 30 seconds for all three launch engines.
- **SC-002**: `forge docker list` returns in under 500 ms regardless of the number of
  unmanaged containers on the machine (up to 200 total containers).
- **SC-003**: Running a destructive command (`reset`, `remove`) twice in succession MUST produce
  a success exit code both times (idempotence).
- **SC-004**: Zero unmanaged Docker containers are ever modified or removed during any
  forge operation — verified by running the full test suite against a Docker host with
  pre-existing unrelated containers.
- **SC-005**: The `--help` output for the `docker` subcommand and all its sub-commands renders
  in under 100 ms (cold start budget from the constitution).
- **SC-006**: All three launch engines (postgres, mssql, mysql) produce a valid, connectable
  database when used with `create` on a standard macOS Apple Silicon machine.

## Clarifications

### Session 2026-05-08

- Q: Should a `--port` flag exist on `create` for manual host-port override when the range is exhausted? → A: No `--port` flag. Port-range exhaustion is an error; the developer resolves it by widening `docker.port_range_end` in config.
- Q: After `create`, does the CLI wait for the DB to be reachable before returning? → A: Print the connection string immediately, then show a "waiting for DB…" spinner that exits once the port accepts a TCP connection (with a configurable timeout, default 30 s).
- Q: How should the CLI handle mssql's SA_PASSWORD complexity requirements? → A: Validate the password meets mssql rules before starting the container; fail fast with a clear, human-readable message if it does not.
- Q: What connection string format should `conn` and `status` emit? → A: Engine-native format per engine (e.g., `postgres://…` for Postgres, `Server=…;` for mssql). Each engine's implementation owns its connection string template.
- Q: If the original port is occupied during `reset`, should the CLI pick a new port or error? → A: Hard error — report the conflict clearly and instruct the developer to free the port or use `remove` + `create`.

## Assumptions

- Docker Desktop (or equivalent) is already installed on the developer's machine. The CLI does
  not install or upgrade Docker.
- The developer's machine has internet access to pull engine images on first use; subsequent
  runs use the local image cache.
- One project maps to exactly one database container (multi-container projects are out of scope
  for this version).
- The shared network `forge-net` is created on first use if it does not already exist; it is
  never removed by `remove` (it is shared across projects).
- Project slug names are case-insensitive and validated to contain only `[a-z0-9-]` characters
  before any Docker operation.
- The `port_range_end` defaults to `port_range_start + 999` (i.e., 15000–15999) unless
  configured otherwise.
- Remote Docker hosts and Docker contexts other than the default are out of scope.
- Backup/restore, docker-compose generation, and multi-container projects are out of scope.
