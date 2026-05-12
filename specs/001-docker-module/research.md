# Research: Docker Module

**Branch**: `001-docker-module` | **Date**: 2026-05-08
**Status**: Complete — all unknowns resolved

---

## 1. Docker SDK v27 — Label Filtering and Container Lifecycle

**Decision**: Use `github.com/docker/docker/client` v27 with `filters.NewArgs()` for all label
queries. Every list/inspect call passes `filters.KeyValuePair{Key: "label", Value: "forge.managed=true"}`.

**Rationale**: The Docker Go SDK v27 is the current stable release aligned with Docker Engine
27.x (the default on Docker Desktop). `filters.Args` is the canonical, API-stable way to filter
by label on `ContainerList`. Using SDK v27 avoids the deprecated `docker/docker` v24 module path.

**Key API surface used**:
- `client.ContainerList(ctx, container.ListOptions{All: true, Filters: filters})` — list managed
- `client.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, name)` — create
- `client.ContainerStart/Stop/Remove(ctx, id, options)` — lifecycle
- `client.VolumeCreate/Remove(ctx, options)` — volume management
- `client.NetworkCreate/Connect/Disconnect(ctx, ...)` — network management
- `client.ContainerInspect(ctx, id)` — read labels + env for status/conn

**Alternatives considered**: `docker/compose` SDK — too heavy; plain HTTP to Docker socket —
reinvents the SDK unnecessarily.

---

## 2. Free-Port Allocation on macOS

**Decision**: Walk the range `[port_range_start, port_range_end]` and attempt `net.Listen("tcp",
":N")`. The first port for which the listen succeeds (and is immediately closed) is claimed. Store
the chosen port as Docker label `forge.host_port=N`.

**Rationale**: Asking the kernel via `net.Listen` is the only reliable TOCTOU-safe method on
macOS without parsing `/proc` (which doesn't exist on macOS anyway). The listen-and-close
approach is standard Go practice for port probing.

**TOCTOU note**: There is a narrow race between the probe and Docker binding the port. In
practice this is negligible for a single-developer tool; a retry loop on container start failure
with `bind: address already in use` can be added as a future hardening step.

**Alternatives considered**: Parsing `netstat -an` output — brittle, shell-out overhead.
Using `:0` (kernel-assigned) — gives a random port outside the configured range.

---

## 3. DB Readiness Check (TCP Ping)

**Decision**: After `ContainerStart`, print the connection string immediately, then poll
`net.DialTimeout("tcp", "localhost:<port>", 1*time.Second)` in a loop with 500 ms sleep between
attempts, up to `docker.readiness_timeout_seconds` (default 30). Show a spinner using a simple
goroutine + `fmt.Print("\r⠿ waiting for DB…")` pattern (no extra library).

**Rationale**: The user chose Option C in clarification — print immediately, then show a spinner.
A lightweight spinner avoids pulling in a TUI library just for this one indicator. Using `net.Dial`
(not a DB-protocol handshake) is engine-agnostic and sufficient to confirm the port is accepting
connections.

**Timeout behavior**: On timeout, print a yellow warning `⚠ DB did not become ready within Ns —
the connection string is still valid; try again in a moment.` Exit code 0 (not a failure).

**Alternatives considered**: Engine-specific readiness (e.g., `pg_isready`) — breaks engine-
agnostic design; requires shelling out. HTTP health endpoint — databases don't expose one.

---

## 4. mssql SA_PASSWORD Complexity Rules

**Decision**: Each `Engine` implementation exposes a `ValidatePassword(password string) error`
method. The mssql engine validates: length ≥ 8, contains uppercase, lowercase, digit, and one
of `!@#$%^&*`. Validation runs in `service.CreateProject` before any Docker call. On failure,
exit with a red `✗` message naming each failed rule.

**SA_PASSWORD requirements (SQL Server 2019+)**:
- Minimum 8 characters
- At least one uppercase letter (A–Z)
- At least one lowercase letter (a–z)
- At least one digit (0–9)
- At least one non-alphanumeric character from the set: `!@#$%^&*()-_+=[]{}|;:,.<>?`

**Rationale**: Pre-flight validation (Option B from clarification) gives the developer an
actionable error rather than waiting for the SQL Server container to silently fail its init
script. The `Engine` interface owns the rule, keeping command handlers ignorant of engine
specifics.

**Alternatives considered**: Auto-generate compliant password — convenient but surprises users
who expect their configured password to be used.

---

## 5. Reset Port Conflict Handling

**Decision**: During `reset`, re-read the port from the `forge.host_port` label, then call
`IsPortFree(port)`. If the port is occupied, print a red `✗` error: `port <N> is in use — free
it or use 'forge docker remove <project> && forge docker create <project> --engine <engine>'
to start fresh.` Exit non-zero without touching the container.

**Rationale**: Hard error (Option A from clarification). The developer's tooling is likely bound
to the original port; silently reallocating would create a confusing silent breakage.

---

## 6. Cobra Module Registry Pattern

**Decision**: Define a `Module` interface in `internal/core/registry/`:
```go
type Module interface {
    Name() string
    Command() *cobra.Command
}
```
Each module calls `registry.Register(m)` from its `init()` function. `main.go` calls
`registry.Commands()` to get all registered `*cobra.Command` instances and adds them to the root
command. `main.go` imports each module package for its `init()` side effect via blank import:
`_ "github.com/sametkarademir/forge/internal/modules/docker"`.

**Rationale**: This is the standard Go plugin pattern using `init()`. It satisfies Constitution
Principle I exactly: `main.go` does not know about any module by name — it only knows the
`registry` package. Adding a new module requires zero changes to `main.go` or any other module.

**Alternatives considered**: Config-file-driven plugin loading — overkill for a personal CLI;
no dynamic loading needed. Interface with reflection — unnecessary complexity.

---

## 7. viper Config Initialisation

**Decision**: On startup, `internal/core/config` calls `viper.SetConfigFile("~/.forge/config.yaml")`.
If the file does not exist, it writes the default config using `viper.WriteConfigAs`. Defaults
are set via `viper.SetDefault(...)` before the file is loaded so they are always available.

**Default config values**:
```yaml
docker:
  default_user: forge
  default_password: forge_dev
  default_db: forge
  port_range_start: 15000
  port_range_end: 15999
  readiness_timeout_seconds: 30
```

**Note**: The default password `forge_dev` does NOT satisfy mssql complexity rules. The CLI
will detect this and error on `create --engine mssql`, instructing the user to set
`docker.default_password` to a compliant value or pass `--password`.

---

## 8. Engine Interface Summary

```go
type Engine interface {
    Name()             string          // "postgres", "mssql", "mysql"
    DefaultImage()     string          // e.g. "postgres:16-alpine"
    DefaultPort()      int             // internal container port
    EnvVars(user, password, db string) map[string]string
    ConnectionString(host string, hostPort int, user, password, db string) string
    ValidatePassword(password string) error  // nil = OK
}
```

Each engine file calls `Register(e Engine)` from its `init()`. The global registry is a
`sync.Mutex`-protected `map[string]Engine`. Command handlers access engines via
`engines.Get("postgres")` — never by direct import.
