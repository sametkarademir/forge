# Quickstart: Docker Module

**Branch**: `001-docker-module` | **Date**: 2026-05-08

This guide validates the docker module end-to-end after implementation. Run these steps in order
on a macOS machine with Docker Desktop running.

---

## Prerequisites

- Docker Desktop is running (`docker ps` succeeds)
- `forge` binary is on `PATH` (built via `make install` or `go install ./cmd/forge`)
- No pre-existing containers with names starting `forge-` (clean state)

---

## Step 1 — Check available engines

```bash
forge docker engines
```

Expected output (columns: ENGINE, DEFAULT IMAGE):
```
ENGINE    DEFAULT IMAGE
postgres  postgres:16-alpine
mssql     mcr.microsoft.com/mssql/server:2022-latest
mysql     mysql:8.4
```

---

## Step 2 — Create a Postgres container

```bash
forge docker create todeb --engine postgres
```

Expected output:
```
✓ Created forge-todeb-postgres on port 15000
  Connection: postgres://forge:forge_dev@localhost:15000/forge
⠿ waiting for DB… (ready in ~2s)
```

Verify with Docker:
```bash
docker inspect forge-todeb-postgres --format '{{.Config.Labels}}'
# Should contain: forge.managed=true, forge.project=todeb, forge.engine=postgres, forge.host_port=15000
```

---

## Step 3 — List managed containers

```bash
forge docker list
```

Expected: one row for `todeb` with status `running`.

---

## Step 4 — Inspect status

```bash
forge docker status todeb
```

Expected: detailed output with image, port, volume, masked password, and connection string.
Verify password is shown as `****` in environment summary but real value appears in connection
string field.

---

## Step 5 — Get connection string for piping

```bash
forge docker conn todeb | pbcopy
```

Paste into a DB client (TablePlus, psql, etc.) and confirm a connection is established.

```bash
# Verify it's exactly one line, no trailing decoration
forge docker conn todeb | wc -l   # should print 1
```

---

## Step 6 — Create a second project

```bash
forge docker create mediazone --engine postgres
```

Expected: port 15001 allocated (15000 already in use). `forge docker list` now shows two rows.

---

## Step 7 — Duplicate create is rejected

```bash
forge docker create todeb --engine postgres
```

Expected: error exit 1 with hint to use `reset` or `remove`.

---

## Step 8 — Reset (wipes data)

First seed something:
```bash
psql "$(forge docker conn todeb)" -c "CREATE TABLE demo (id serial);"
```

Then reset:
```bash
forge docker reset todeb --yes
```

Verify the table is gone:
```bash
psql "$(forge docker conn todeb)" -c "\dt"   # should print "Did not find any relations."
```

Verify port is the same:
```bash
forge docker status todeb | grep Port   # should still show 15000
```

---

## Step 9 — Remove

```bash
forge docker remove todeb --yes
```

Verify:
```bash
docker ps -a | grep forge-todeb   # should print nothing
docker volume ls | grep forge-todeb   # should print nothing
```

---

## Step 10 — Idempotent remove

```bash
forge docker remove todeb --yes   # run again
echo $?   # should print 0
```

---

## Step 11 — mssql password validation

```bash
forge docker create testmssql --engine mssql
# With default password "forge_dev" — should fail with complexity error
```

Expected:
```
✗ password does not meet mssql requirements: missing uppercase letter, missing special character
```

Retry with a compliant password:
```bash
forge docker create testmssql --engine mssql --password 'Str0ng!Pass'
```

Expected: container created and running.

---

## Step 12 — Safety boundary

Manually create an unmanaged container with the right name:
```bash
docker run -d --name forge-ghost-postgres postgres:16-alpine
forge docker remove ghost --yes
```

Expected:
```
✗ container "forge-ghost-postgres" exists but is not managed by forge
  (missing label forge.managed=true) — refusing to remove
```

Cleanup:
```bash
docker rm -f forge-ghost-postgres
```

---

## Step 13 — Cold-start performance

```bash
time forge docker --help
```

Expected: real time < 100 ms.

---

## Teardown

Remove all test containers created during this guide:
```bash
forge docker remove mediazone --yes
forge docker remove testmssql --yes
```

Verify `forge docker list` prints "No managed containers found."
