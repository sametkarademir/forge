# Tasks: Docker Module — Per-Project Database Container Management

**Input**: Design documents from `specs/001-docker-module/`
**Prerequisites**: plan.md ✅ spec.md ✅ research.md ✅ data-model.md ✅ contracts/cli-schema.md ✅ quickstart.md ✅

**Tests**: No test tasks — not requested in the feature specification.
**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1–US7)
- Exact file paths are included in every description

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Go module initialization and project directory structure

- [x] T001 Initialize Go module with `go mod init github.com/sametkarademir/forge` and create all directories from the plan: `cmd/forge/`, `internal/core/{config,logger,ui,registry}/`, `internal/modules/docker/{commands,service,client,engines}/`, `test/smoke/`
- [x] T002 Add all required dependencies to `go.mod` via `go get`: `github.com/spf13/cobra@v1.8`, `github.com/spf13/viper@v1.19`, `github.com/docker/docker/client@v27`, `github.com/AlecAivazis/survey/v2@v2.3`, `github.com/fatih/color@v1.18`, `github.com/olekukonko/tablewriter@v0.0.5`, `github.com/stretchr/testify@latest`; then run `go mod tidy`
- [x] T003 [P] Create `Makefile` with targets: `build` (`go build -o forge ./cmd/forge`), `install` (`go install ./cmd/forge`), `vet` (`go vet ./...`), `fmt` (`gofmt -l ./...`), `smoke` (`bash test/smoke/docker_create.sh && bash test/smoke/docker_reset.sh && bash test/smoke/docker_remove.sh`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before any user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Create `internal/core/registry/registry.go` defining the `Module` interface (`Name() string`, `Command() *cobra.Command`) and a global registry with `Register(m Module)` and `Commands() []*cobra.Command` functions protected by `sync.Mutex`
- [x] T005 [P] Create `internal/core/config/config.go` that on `Init()` calls `viper.SetDefault` for all six docker config keys (`docker.default_user=forge`, `docker.default_password=forge_dev`, `docker.default_db=forge`, `docker.port_range_start=15000`, `docker.port_range_end=15999`, `docker.readiness_timeout_seconds=30`), sets config file to `~/.forge/config.yaml`, reads it if present, and writes defaults to that path on first run; expose typed getters (`DefaultUser() string`, `DefaultPassword() string`, `DefaultDB() string`, `PortRangeStart() int`, `PortRangeEnd() int`, `ReadinessTimeoutSeconds() int`)
- [x] T006 [P] Create `internal/core/logger/logger.go` with `Success(msg string)` (green `✓` prefix to stdout), `Error(msg string)` (red `✗` prefix to stderr), `Warn(msg string)` (yellow `⚠` prefix to stdout), and `Info(msg string)` (plain stdout) using `github.com/fatih/color`
- [x] T007 [P] Create `internal/core/ui/ui.go` with `Confirm(question string) (bool, error)` using `github.com/AlecAivazis/survey/v2` and `RenderTable(headers []string, rows [][]string)` using `github.com/olekukonko/tablewriter` writing to stdout
- [x] T008 Create `internal/modules/docker/engines/engine.go` defining the `Engine` interface (`Name() string`, `DefaultImage() string`, `DefaultPort() int`, `EnvVars(user, password, db string) map[string]string`, `ConnectionString(host string, hostPort int, user, password, db string) string`, `ValidatePassword(password string) error`) and a `sync.Mutex`-protected global registry with `Register(e Engine)`, `Get(name string) (Engine, bool)`, and `All() []Engine` functions
- [x] T009 [P] Create `internal/modules/docker/client/client.go` with `NewClient() (*DockerClient, error)` (calls `client.NewClientWithOpts(client.FromEnv)` and pings the daemon — on failure prints "Docker is not running — run: open -a Docker" and returns error), `ListManaged(ctx) ([]types.Container, error)` (filters by `forge.managed=true`), `InspectByProject(ctx, project string) (types.ContainerJSON, error)` (filters by both `forge.managed=true` and `forge.project=<project>`), `RunContainer(ctx, config *RunConfig) (string, error)`, `StopContainer(ctx, id string) error`, `RemoveContainer(ctx, id string) error`, `EnsureNetwork(ctx, name string) error`, `VolumeCreate(ctx, name string, labels map[string]string) error`, `VolumeRemove(ctx, name string) error`
- [x] T010 [P] Create `internal/modules/docker/service/naming.go` with pure functions: `ContainerName(project, engine string) string` (returns `forge-<project>-<engine>`), `VolumeName(project, engine string) string` (returns `forge-<project>-<engine>-data`), `NetworkName() string` (returns `forge-net`)
- [x] T011 [P] Create `internal/modules/docker/service/ports.go` with `IsPortFree(port int) bool` (attempts `net.Listen("tcp", ":<port>")` and immediately closes on success) and `NextFreePort(start, end int) (int, error)` (walks the range calling `IsPortFree`, returns first free port or error if range exhausted)
- [x] T012 Create `cmd/forge/main.go` with a root `cobra.Command` (`Use: "forge"`) that calls `config.Init()` in `PersistentPreRun`, iterates `registry.Commands()` and adds each to the root command, then executes; blank-imports all module packages: `_ "github.com/sametkarademir/forge/internal/modules/docker"`
- [x] T013 Create `internal/modules/docker/module.go` that in `init()` registers a `dockerModule` struct (implementing `registry.Module`) which returns a `*cobra.Command` with `Use: "docker"` and `Short: "Manage per-project database containers"`; the command's subcommands are added via `AddCommand` calls for each command file (create, list, status, conn, reset, remove, engines — wire up as each phase completes)

**Checkpoint**: Run `go vet ./...` and `gofmt -l ./...` — both must pass. Run `forge --help` — must show `docker` subcommand.

---

## Phase 3: User Story 1 — Create a Project Database Container (Priority: P1) 🎯 MVP

**Goal**: `forge docker create <project> --engine postgres` creates a container, prints the connection string, and shows a readiness spinner.

**Independent Test**: `forge docker create testproj --engine postgres` against a live Docker daemon produces a running container `forge-testproj-postgres`, volume `forge-testproj-postgres-data`, all five required labels, and prints the connection string.

- [x] T014 [US1] Create `internal/modules/docker/engines/postgres.go` implementing the `Engine` interface: `Name()="postgres"`, `DefaultImage()="postgres:16-alpine"`, `DefaultPort()=5432`, `EnvVars` returns `POSTGRES_USER/POSTGRES_PASSWORD/POSTGRES_DB`, `ConnectionString` returns `postgres://<user>:<password>@<host>:<hostPort>/<db>`, `ValidatePassword` always returns nil; call `engines.Register(&Postgres{})` from `init()`
- [x] T015 [US1] Create `internal/modules/docker/service/service.go` with `CreateProject(ctx context.Context, opts CreateOptions) (*ProjectInfo, error)`: (1) validate project name against `^[a-z0-9][a-z0-9-]{0,62}$`, (2) look up engine via `engines.Get`, (3) call `engine.ValidatePassword`, (4) call `client.InspectByProject` and return error with reset/remove hint if container already exists, (5) call `NextFreePort` or return port-range-exhausted error, (6) call `client.EnsureNetwork("forge-net")`, (7) call `client.VolumeCreate` with all five forge labels, (8) call `client.RunContainer` with container name, image, env vars, port binding, volume mount, and labels (`forge.managed=true`, `forge.project`, `forge.engine`, `forge.created_at` RFC3339, `forge.host_port`), (9) print connection string via `logger.Success`, (10) start readiness goroutine; return `*ProjectInfo`
- [x] T016 [US1] Add `WaitForReady(ctx context.Context, port int, timeoutSecs int)` to `internal/modules/docker/service/service.go`: print connection string immediately (already done in T015), then in a loop poll `net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)` every 500ms up to `timeoutSecs` seconds, printing a spinner (`\r⠿ waiting for DB…`) on each attempt; on success print elapsed time; on timeout call `logger.Warn` with "DB did not become ready within Ns — the connection string is still valid; try again in a moment." and return nil (not an error)
- [x] T017 [US1] Create `internal/modules/docker/commands/create.go` with `NewCreateCommand() *cobra.Command` (`Use: "create <project>"`, `Args: cobra.ExactArgs(1)`), flags: `--engine/-e` (required string), `--image` (string), `--user` (string, default from config), `--password` (string, default from config), `--db` (string, default from config); calls `service.CreateProject` with a `CreateOptions` built from args+flags+config defaults; add `NewCreateCommand()` call inside `module.go`'s `init` wiring

**Checkpoint**: `forge docker create testproj --engine postgres` runs end-to-end on a live Docker daemon; `docker inspect forge-testproj-postgres` shows all five labels.

---

## Phase 4: User Story 2 — List All Managed Containers (Priority: P2)

**Goal**: `forge docker list` shows a table of forge-managed containers only; unrelated containers are invisible.

**Independent Test**: With two forge containers and several unrelated containers running, `forge docker list` shows exactly the two forge containers in PROJECT/ENGINE/STATUS/PORT/UPTIME table format.

- [x] T018 [US2] Add `ListProjects(ctx context.Context) ([]*ProjectInfo, error)` to `internal/modules/docker/service/service.go`: call `client.ListManaged` (filter `forge.managed=true`), map each container to `ProjectInfo` reading all five labels, compute `Uptime` from `StartedAt` (zero when stopped), and return the slice
- [x] T019 [US2] Create `internal/modules/docker/commands/list.go` with `NewListCommand() *cobra.Command` (`Use: "list"`): calls `service.ListProjects`, on empty slice prints "No managed containers found.", otherwise calls `ui.RenderTable` with headers `[PROJECT ENGINE STATUS PORT UPTIME]` and one row per `ProjectInfo` (uptime as "—" when zero); add `NewListCommand()` to `module.go` wiring

**Checkpoint**: `forge docker list` prints correct table with two managed containers visible and unrelated containers absent.

---

## Phase 5: User Story 3 — Inspect a Single Project's Container (Priority: P2)

**Goal**: `forge docker status <project>` shows detailed info with masked passwords; works for running and stopped containers.

**Independent Test**: After creating and stopping a `todeb` container, `forge docker status todeb` prints all fields including state "exited" and passwords shown as `****`.

- [x] T020 [US3] Add `GetProjectStatus(ctx context.Context, project string) (*ProjectInfo, error)` to `internal/modules/docker/service/service.go`: call `client.InspectByProject`, build full `ProjectInfo` including `EnvSummary` map (replace any env var value containing the password with `"****"`), masked `ConnectionString` (using `****` in place of the password), and `Uptime`; return error listing known projects when not found (call `ListProjects` for the list)
- [x] T021 [US3] Create `internal/modules/docker/commands/status.go` with `NewStatusCommand() *cobra.Command` (`Use: "status <project>"`, `Args: cobra.ExactArgs(1)`): calls `service.GetProjectStatus`, prints labeled fields (Project, Engine, Status, Image, Port, Volume, Created, Uptime, blank line, "Environment:" block with masked values, blank line, "Connection:" with masked DSN) to stdout using `fmt.Fprintf`; on error calls `logger.Error` with known-projects list; add `NewStatusCommand()` to `module.go` wiring

**Checkpoint**: `forge docker status todeb` shows all fields; passwords appear as `****` in Environment section.

---

## Phase 6: User Story 4 — Get Connection String for Piping (Priority: P2)

**Goal**: `forge docker conn <project>` prints only the real DSN to stdout with no decoration; errors go to stderr.

**Independent Test**: `forge docker conn todeb | wc -l` prints `1`; the line is a valid DSN with the real password.

- [x] T022 [US4] Add `GetConnectionString(ctx context.Context, project string) (string, error)` to `internal/modules/docker/service/service.go`: call `client.InspectByProject`, look up engine by `forge.engine` label, return the unmasked `engine.ConnectionString(...)` using env vars and label values
- [x] T023 [US4] Create `internal/modules/docker/commands/conn.go` with `NewConnCommand() *cobra.Command` (`Use: "conn <project>"`, `Args: cobra.ExactArgs(1)`): calls `service.GetConnectionString`, on success calls `fmt.Println(connStr)` to stdout (no decoration), on error calls `fmt.Fprintln(os.Stderr, "✗ "+err.Error())` and `os.Exit(1)`; add `NewConnCommand()` to `module.go` wiring

**Checkpoint**: `forge docker conn todeb | pbcopy` puts a valid DSN in clipboard; `echo $?` prints `0`.

---

## Phase 7: User Story 5 — Reset a Project's Database (Priority: P3)

**Goal**: `forge docker reset <project>` wipes and recreates the database on the same port with the same credentials; `--yes` skips the prompt.

**Independent Test**: After seeding data in `todeb`, `forge docker reset todeb --yes` leaves an empty DB on the same port; running twice succeeds.

- [x] T024 [US5] Add `ResetProject(ctx context.Context, project string) error` to `internal/modules/docker/service/service.go`: (1) call `client.InspectByProject` (error if not found), (2) read port from `forge.host_port` label, (3) call `IsPortFree(port)` — if occupied return error with instructions to free port or use remove+create, (4) stop and force-remove container, (5) call `client.VolumeRemove`, (6) reconstruct `CreateOptions` from the original container labels (project, engine, image, user, password, db from env vars), (7) call `CreateProject` with those options to re-create on the same port
- [x] T025 [US5] Create `internal/modules/docker/commands/reset.go` with `NewResetCommand() *cobra.Command` (`Use: "reset <project>"`, `Args: cobra.ExactArgs(1)`), flag `--yes/-y bool`: if `--yes` is not set, call `ui.Confirm("This will DELETE all data for project \"<project>\". Continue?")` and exit 0 on denial; calls `service.ResetProject`; add `NewResetCommand()` to `module.go` wiring

**Checkpoint**: `forge docker reset todeb --yes` reports success; `forge docker status todeb` shows same port; DB is empty.

---

## Phase 8: User Story 6 — Remove a Project Entirely (Priority: P3)

**Goal**: `forge docker remove <project>` removes container, volume, and network membership idempotently; refuses unmanaged containers.

**Independent Test**: `forge docker remove todeb --yes` twice exits 0 both times; unmanaged container with matching name is refused with safety message.

- [x] T026 [US6] Add `RemoveProject(ctx context.Context, project string) error` to `internal/modules/docker/service/service.go`: (1) look up container by name `forge-<project>-<engine>` via `client.InspectByProject`; if not found return nil (idempotent), (2) verify `forge.managed=true` label is present — if missing return error "container exists but is not managed by forge (missing label forge.managed=true) — refusing to remove", (3) call `client.StopContainer` (ignore "not running" error), (4) call `client.RemoveContainer`, (5) call `client.VolumeRemove` (ignore "not found" error), (6) disconnect container from `forge-net` (ignore "not found" error); network itself is never removed
- [x] T027 [US6] Create `internal/modules/docker/commands/remove.go` with `NewRemoveCommand() *cobra.Command` (`Use: "remove <project>"`, `Args: cobra.ExactArgs(1)`), flag `--yes/-y bool`: if `--yes` is not set call `ui.Confirm("Remove project \"<project>\" and all its data?")` and exit 0 on denial; calls `service.RemoveProject`; add `NewRemoveCommand()` to `module.go` wiring

**Checkpoint**: `forge docker remove todeb --yes` exits 0; re-running exits 0; unmanaged container triggers safety refusal.

---

## Phase 9: User Story 7 — List Supported Engines (Priority: P3)

**Goal**: `forge docker engines` prints a table of all registered engines with their default images; all three launch engines are present.

**Independent Test**: `forge docker engines` prints a table with exactly three rows: postgres, mssql, mysql — each with the correct default image tag.

- [x] T028 [US7] Create `internal/modules/docker/engines/mssql.go` implementing the `Engine` interface: `Name()="mssql"`, `DefaultImage()="mcr.microsoft.com/mssql/server:2022-latest"`, `DefaultPort()=1433`, `EnvVars` returns `ACCEPT_EULA=Y`, `SA_PASSWORD=<password>`, `MSSQL_PID=Developer`, `ConnectionString` returns `Server=localhost,<hostPort>;Database=<db>;User Id=sa;Password=<password>;TrustServerCertificate=true`, `ValidatePassword` checks: len≥8, at least one uppercase (A–Z), one lowercase (a–z), one digit (0–9), one special from `!@#$%^&*()-_+=[]{}|;:,.<>?` — returns combined error message listing each failed rule; call `engines.Register(&MSSQL{})` from `init()`
- [x] T029 [P] [US7] Create `internal/modules/docker/engines/mysql.go` implementing the `Engine` interface: `Name()="mysql"`, `DefaultImage()="mysql:8.4"`, `DefaultPort()=3306`, `EnvVars` returns `MYSQL_USER=<user>`, `MYSQL_PASSWORD=<password>`, `MYSQL_DATABASE=<db>`, `MYSQL_ROOT_PASSWORD=<password>`, `ConnectionString` returns `mysql://<user>:<password>@localhost:<hostPort>/<db>`, `ValidatePassword` always returns nil; call `engines.Register(&MySQL{})` from `init()`
- [x] T030 [US7] Create `internal/modules/docker/commands/engines.go` with `NewEnginesCommand() *cobra.Command` (`Use: "engines"`): calls `engines.All()`, sorts by name, calls `ui.RenderTable` with headers `[ENGINE DEFAULT IMAGE]` and one row per engine; add `NewEnginesCommand()` to `module.go` wiring

**Checkpoint**: `forge docker engines` prints a three-row table; `forge docker create testmssql --engine mssql` with default password fails with complexity error; with `--password 'Str0ng!Pass'` succeeds.

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Smoke tests, final validation, and documentation

- [x] T031 Create `test/smoke/docker_create.sh`: builds and installs the binary via `make install`, runs `forge docker create testproj --engine postgres`, asserts container `forge-testproj-postgres` is running (`docker inspect --format '{{.State.Running}}'`), asserts all five labels are present (`docker inspect --format '{{.Config.Labels}}'`), asserts stdout contains a `postgres://` connection string, then cleans up via `forge docker remove testproj --yes`; exits non-zero on any assertion failure
- [x] T032 [P] Create `test/smoke/docker_reset.sh`: creates `testproj` with Postgres, seeds a table via `psql`, resets with `--yes`, verifies table is gone (`\dt` returns empty), verifies port is unchanged, cleans up; exits non-zero on any assertion failure
- [x] T033 [P] Create `test/smoke/docker_remove.sh`: creates `testproj`, removes with `--yes`, asserts no container or volume remains; removes again and asserts exit 0 (idempotence); manually creates an unmanaged container `forge-ghost-postgres` without labels, runs `forge docker remove ghost --yes`, asserts exit 1 and safety message; cleans up with `docker rm -f forge-ghost-postgres`; exits non-zero on any assertion failure
- [x] T034 [P] Update `README.md` with a Usage section covering all seven subcommands with example invocations and expected output matching `quickstart.md` Steps 1–13; include installation section (`go install ./cmd/forge` and `make install`)
- [x] T035 Run `go vet ./...` and `gofmt -l ./...` across all Go files in `cmd/` and `internal/`; fix every reported issue; confirm both commands produce zero output

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 completion — **BLOCKS all user stories**
- **User Story Phases (3–9)**: All depend on Phase 2 completion; phases 3–9 must run in priority order (P1 → P2 → P3) since later commands may call service functions defined in earlier phases
- **Polish (Phase 10)**: Depends on all user story phases being complete

### User Story Dependencies

- **US1 — Create (P1)**: Requires Foundational complete. No other story dependency.
- **US2 — List (P2)**: Requires US1 complete (Docker client and naming patterns proven).
- **US3 — Status (P2)**: Requires US1 complete (InspectByProject must work).
- **US4 — Conn (P2)**: Requires US1 complete (engine ConnectionString must work).
- **US5 — Reset (P3)**: Requires US1 complete (reuses CreateProject internally).
- **US6 — Remove (P3)**: Requires US1 complete (reuses Docker client).
- **US7 — Engines (P3)**: Requires US1 complete (Postgres engine already registered); MSSQL/MySQL engines added here.

### Within Each Phase

- Foundational [P] tasks (T005–T011) can all run in parallel after T004
- T012 and T013 depend on T004 (registry)
- Within each user story phase: service method before command file

### Parallel Opportunities

- T003 (Makefile) can run in parallel with T004 (registry)
- T005, T006, T007, T009, T010, T011 can all run in parallel (different packages/files)
- T028 (mssql) and T029 (mysql) can run in parallel (different files)
- T031, T032, T033 (smoke tests) can be written in parallel; run sequentially (need live Docker)
- T034 (README) can run in parallel with T035 (vet/fmt)

---

## Parallel Example: Phase 2 Foundational

```bash
# Launch all parallel foundational tasks together (T004 must complete first):
Task T005: internal/core/config/config.go
Task T006: internal/core/logger/logger.go
Task T007: internal/core/ui/ui.go
Task T009: internal/modules/docker/client/client.go
Task T010: internal/modules/docker/service/naming.go
Task T011: internal/modules/docker/service/ports.go

# Then sequentially:
Task T008: internal/modules/docker/engines/engine.go (after T004)
Task T012: cmd/forge/main.go (after T004, T005)
Task T013: internal/modules/docker/module.go (after T008, T012)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001–T003)
2. Complete Phase 2: Foundational (T004–T013) — **CRITICAL**
3. Complete Phase 3: User Story 1 — Create (T014–T017)
4. **STOP and VALIDATE**: `forge docker create testproj --engine postgres` runs end-to-end
5. Proceed to Phase 4 (list) once MVP is confirmed

### Incremental Delivery

1. Setup + Foundational → scaffolding ready
2. Add US1 (create) → test independently → MVP!
3. Add US2 (list) + US3 (status) + US4 (conn) → full read-path working → test independently
4. Add US5 (reset) + US6 (remove) → destructive commands → test independently
5. Add US7 (engines) + MSSQL + MySQL → all engines supported → test independently
6. Polish → smoke tests + README + `go vet` clean

### Phase Rollout Alignment

| Plan Phase | Tasks | Exit Criteria |
|---|---|---|
| Plan Phase 1 | T001–T013 | `forge --help` shows `docker`; `go vet` clean |
| Plan Phase 2 | T014–T019 | `create` + `list` work end-to-end with Postgres |
| Plan Phase 3 | T020–T027 | All 7 subcommands pass manual smoke test |
| Plan Phase 4 | T028–T033 | Smoke tests green for all three engines |
| Plan Phase 5 | T034–T035 | README done; `go vet`/`gofmt` clean |

---

## Notes

- `[P]` tasks touch different files — no file-level conflicts when run in parallel
- Each user story phase is independently testable before the next begins
- `service.go` grows across phases — each task adds one or two methods; name each method clearly
- `module.go` must be updated (`AddCommand`) each time a new command file is added in T017, T019, T021, T023, T025, T027, T030 — include this in each command task
- Docker label `forge.managed=true` is the mandatory safety gate — every read and write operation must check it first
- mssql default password `forge_dev` intentionally fails `ValidatePassword` — this is by design (FR-012 + research decision)
- The `forge-net` network is never removed by any command — only created on first use
