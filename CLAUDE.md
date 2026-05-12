<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
at `specs/001-docker-module/plan.md`.
<!-- SPECKIT END -->

# forge — Claude Code Operating Guide

You are working on **forge**, a personal developer-productivity CLI for macOS
written in Go. The full architectural rules live in `.specify/memory/constitution.md`.
This file tells you **how to operate day-to-day** in the repo.

## Quick facts

- **Language**: Go 1.23+
- **Entry point**: `cmd/forge/main.go`
- **CLI framework**: `spf13/cobra` + `spf13/viper`
- **Docker SDK**: `github.com/docker/docker/client` (wrapped — see "Boundaries" below)
- **Target OS**: macOS (Apple Silicon primary)
- **Config file**: `~/.forge/config.yaml`
- **Spec Kit workflow**: `/specify` → `/clarify` → `/plan` → `/tasks` → `/analyze` → `/implement`

## Repository map

```
cmd/forge/                # main.go — root cobra command, loads modules from registry
internal/
  core/
    config/                 # ~/.forge/config.yaml (viper)
    logger/                 # colored stdout/stderr — USE THIS, never fmt.Println
    ui/                     # confirm prompt, table writer — USE THIS, never raw survey/tablewriter
    registry/               # Module interface + central registry
  modules/
    docker/
      module.go             # self-registration entry point
      commands/             # one file per subcommand (thin cobra wrappers)
      service/              # business logic — NO cobra imports allowed here
      client/               # Docker SDK wrapper — the ONLY package that imports docker/client
      engines/              # one file per engine, self-register via init()
test/smoke/                 # shell-based end-to-end tests
.specify/                   # Spec Kit artifacts (constitution, specs, plans, tasks)
.claude/                    # this dir — commands, skills, settings
```

## Operating rules

### Architecture boundaries (HARD RULES)
1. `service/` packages MUST NOT import `cobra`, `viper`, or any UI package.
   Commands are thin adapters; logic lives in services.
2. The Docker SDK (`github.com/docker/docker/client`) is imported **only** by
   `internal/modules/docker/client/`. Everything else uses the wrapper.
3. `main.go` does NOT import individual modules. Modules self-register via
   `init()` calling `registry.Register(...)`. To add a module, add a blank
   import in `cmd/forge/main.go` and nothing else.
4. Each engine is one file. Adding Redis = create `engines/redis.go` with a
   type implementing `Engine`, plus `init() { Register(...) }`. No other file
   changes allowed.

### Docker resource discipline
5. **Every** managed resource carries these labels:
   `forge.managed=true`, `forge.project=<name>`, `forge.engine=<name>`,
   `forge.created_at=<rfc3339>`, `forge.host_port=<n>`.
6. Resource names are deterministic:
   - Container: `forge-<project>-<engine>`
   - Volume:    `forge-<project>-<engine>-data`
   - Network:   `forge-net` (shared)
7. Any list/bulk operation MUST filter by `forge.managed=true`. There is no
   bypass flag — never add one.

### Output and UX
8. All user-facing output goes through `internal/core/logger`. Never use
   `fmt.Println` directly in commands or services. The four levels:
   `logger.Info`, `logger.Success`, `logger.Warn`, `logger.Error`.
   Use `logger.Plain` only when the output is meant to be machine-piped
   (e.g. `forge docker conn <project>` for `pbcopy`).
9. Tables use `ui.NewTable(...)`. Confirmation prompts use `ui.Confirm(...)`.
10. Destructive commands (`reset`, `remove`) prompt by default and accept
    `--yes` / `-y` to skip. The flag bypasses the prompt — never the safety
    label check from rule 7.

### Error handling
11. Errors returned from `service/` are wrapped with `fmt.Errorf("...: %w", err)`
    and include enough context to debug without a stack trace.
12. Commands convert errors to `logger.Error(...)` + non-zero exit. They never
    print Go error formatting directly.
13. Docker daemon unreachable is a recognised condition: detect the specific
    error and print the macOS-friendly hint `Docker daemon is not running — run \`open -a Docker\` to start it`.

### Testing and quality gates
14. Before any commit, these MUST pass:
    - `go vet ./...`
    - `gofmt -l .` (empty output)
    - The smoke test for the touched module if one exists
15. A new public command MUST come with a smoke test under `test/smoke/`.
16. Service-layer logic that does not touch Docker SHOULD have a unit test
    next to it (`service_test.go`).

### Commit hygiene
17. One task per commit. Conventional commit format:
    `feat(docker): add reset command`, `fix(core): handle missing config dir`,
    `refactor(engines): extract password validator`, `docs: …`, `test: …`,
    `chore: …`.
18. Keep diffs small. If you find yourself touching more than three packages
    in one task, stop and ask whether the task should be split.

## What to do when stuck

- If a request requires changes outside the current task's scope, surface it
  rather than expanding scope silently.
- If a design decision contradicts the constitution, stop and ask. Don't
  rationalise around it.
- If you would need to import `docker/client` outside `client/`, you have the
  abstraction wrong — extend the wrapper instead.
- If you would need to import `cobra` inside `service/`, the responsibility is
  in the wrong layer — move it to the command file.

## Things to never do

- Never call `fmt.Println` / `fmt.Printf` in commands or services.
- Never create a Docker resource without the full label set from rule 5.
- Never touch a Docker resource that lacks `forge.managed=true`.
- Never add a flag whose only purpose is to bypass a safety check.
- Never write `panic(...)` in non-test code. Return errors.
- Never store secrets in plaintext anywhere other than the Docker container
  environment (which is the engine's required interface).
- Never edit `.specify/memory/constitution.md` without an explicit user request
  and a version bump.

## Useful one-liners

```bash
# Build locally
go build -o ./bin/forge ./cmd/forge

# Install to $GOBIN
go install ./cmd/forge

# Lint & vet
gofmt -l . && go vet ./...

# List forge-managed Docker resources directly
docker ps -a --filter label=forge.managed=true
docker volume ls --filter label=forge.managed=true

# Nuke everything forge-managed (use with care, scripts only)
docker ps -aq --filter label=forge.managed=true | xargs -r docker rm -f
docker volume ls -q --filter label=forge.managed=true | xargs -r docker volume rm
```
