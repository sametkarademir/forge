# Implementation Plan: Docker Module — Per-Project Database Container Management

**Branch**: `001-docker-module` | **Date**: 2026-05-08 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/001-docker-module/spec.md`

## Summary

Build the `docker` module of forge — a self-contained Go package under
`internal/modules/docker` that manages per-project database containers on the developer's local
Docker daemon. The module exposes seven subcommands (`create`, `list`, `status`, `conn`, `reset`,
`remove`, `engines`) under the `forge docker` top-level command. Docker labels are the sole
source of truth; no local state files are written. Three database engines (postgres, mssql, mysql)
ship at launch, each as a single file implementing a shared `Engine` interface registered via
`init()`.

## Technical Context

**Language/Version**: Go 1.23+
**Primary Dependencies**: spf13/cobra v1.8, spf13/viper v1.19, github.com/docker/docker/client v27,
  AlecAivazis/survey/v2 v2.3, fatih/color v1.18, olekukonko/tablewriter v0.0.5,
  testify/require (test only)
**Storage**: Docker daemon labels (no local DB or file state)
**Testing**: `go test ./...` (unit); shell-based smoke tests under `test/smoke/`
**Target Platform**: macOS (Apple Silicon primary, Intel supported); Linux nice-to-have
**Project Type**: CLI tool (single static binary, Homebrew-installable)
**Performance Goals**: `--help` and `list` under 100 ms cold; `create` connection string printed
  in under 5 s (image pre-cached); DB readiness spinner resolves within 30 s
**Constraints**: Static binary, no runtime dependencies beyond Docker Engine; zero unmanaged
  container mutations guaranteed by label filter
**Scale/Scope**: Single developer machine; tens of managed containers at most

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Gate | Status |
|---|---|---|
| I. Modular Architecture | `main.go` MUST NOT import `modules/docker` directly; module self-registers | ✅ PASS |
| II. Single Static Binary | No CGO, no runtime deps beyond Docker Engine; cold start ≤ 100 ms | ✅ PASS |
| III. Docker as Source of Truth | Labels are authoritative; no state files written | ✅ PASS |
| IV. Engine Pluggability | `Engine` interface + `init()` registration; `create` handler has zero engine-specific code | ✅ PASS |
| V. Predictable Naming & Idempotence | Names follow `forge-<project>-<engine>`; reset/remove are idempotent | ✅ PASS |
| VI. Safety Boundaries | Every Docker operation filters by `forge.managed=true` first; no override path | ✅ PASS |
| VII. Spec-First, Atomic Tasks | Spec + clarifications complete; plan → tasks → implement workflow | ✅ PASS |

**Post-design re-check**: All gates still pass after Phase 1 design. No complexity violations.

## Project Structure

### Documentation (this feature)

```text
specs/001-docker-module/
├── plan.md          # This file
├── research.md      # Phase 0 output
├── data-model.md    # Phase 1 output
├── quickstart.md    # Phase 1 output
├── contracts/       # Phase 1 output
│   └── cli-schema.md
└── tasks.md         # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
forge/
├── cmd/forge/
│   └── main.go                        # root cobra cmd; loads modules via registry
├── internal/
│   ├── core/
│   │   ├── config/                    # ~/.forge/config.yaml (viper)
│   │   ├── logger/                    # colored stdout/stderr (fatih/color)
│   │   ├── ui/                        # confirm prompt (survey/v2), table writer
│   │   └── registry/                  # Module interface + global registry
│   └── modules/
│       └── docker/
│           ├── module.go              # self-registers; returns root cobra cmd
│           ├── commands/              # one file per subcommand (cobra thin wrappers)
│           │   ├── create.go
│           │   ├── list.go
│           │   ├── status.go
│           │   ├── conn.go
│           │   ├── reset.go
│           │   ├── remove.go
│           │   └── engines.go
│           ├── service/               # business logic; zero cobra imports
│           │   ├── service.go         # CreateProject, ListProjects, ResetProject, …
│           │   ├── naming.go          # ContainerName, VolumeName, NetworkName
│           │   └── ports.go           # NextFreePort, IsPortFree
│           ├── client/                # Docker SDK wrapper
│           │   └── client.go          # NewClient, RunContainer, RemoveContainer,
│           │                          # ListManaged, InspectByProject, EnsureNetwork
│           └── engines/               # one file per engine
│               ├── engine.go          # Engine interface + global registry
│               ├── postgres.go
│               ├── mssql.go
│               └── mysql.go
├── test/
│   └── smoke/
│       ├── docker_create.sh
│       ├── docker_reset.sh
│       └── docker_remove.sh
├── .specify/
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

**Structure Decision**: Single-project layout. `internal/core/` is shared infrastructure; each
module under `internal/modules/` is fully self-contained. The `docker` module owns all files
under `internal/modules/docker/`.

## Complexity Tracking

> No constitution violations detected. Table left intentionally empty.

## Phased Rollout

| Phase | Scope | Exit Criteria |
|---|---|---|
| 1 | Core scaffolding: `core/` packages, registry, `cmd/forge/main.go`, docker module stub | `go vet ./...`, `gofmt -l` pass; `forge --help` shows `docker` subcommand |
| 2 | Engine interface + Postgres + Docker client wrapper + `create` + `list` | `forge docker create testproj --engine postgres` runs end-to-end; `list` shows it |
| 3 | `status`, `conn`, `reset`, `remove`, `engines`; `--yes` flag on destructive cmds | All 7 subcommands pass manual smoke test with Postgres |
| 4 | MSSQL and MySQL engines; smoke tests for all three engines | `docker_create.sh`, `docker_reset.sh`, `docker_remove.sh` green for all engines |
| 5 | Makefile, Homebrew formula stub, README usage examples, `forge completion zsh` | `make install` works; README quickstart matches quickstart.md |
