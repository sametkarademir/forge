#!/usr/bin/env bash
set -euo pipefail

BINARY="${BINARY:-forge}"
PROJECT="smoke-create-$$"

cleanup() {
  "$BINARY" docker remove "$PROJECT" --purge --yes 2>/dev/null || true
}
trap cleanup EXIT

echo "=== smoke: docker create ==="

# Non-interactive flag-driven create + immediate run.
"$BINARY" docker create "$PROJECT" \
  --engine postgres \
  --user smokeuser \
  --password SmokePass1! \
  --db smokedb \
  --run

# Verify container is running
state=$(docker inspect "forge-${PROJECT}" --format '{{.State.Running}}' 2>/dev/null)
if [[ "$state" != "true" ]]; then
  echo "FAIL: container not running (state=$state)"
  exit 1
fi

# Verify required labels (v2 schema uses forge.preset not forge.project)
labels=$(docker inspect "forge-${PROJECT}" --format '{{json .Config.Labels}}' 2>/dev/null)
for key in forge.managed forge.preset forge.engine forge.created_at forge.host_port; do
  if ! echo "$labels" | grep -q "\"$key\""; then
    echo "FAIL: missing label $key"
    exit 1
  fi
done

# Verify connection string printed
conn=$("$BINARY" docker conn "$PROJECT" --raw)
if [[ -z "$conn" ]]; then
  echo "FAIL: connection string empty"
  exit 1
fi

# Verify duplicate create is rejected.
set +e
"$BINARY" docker create "$PROJECT" \
  --engine postgres \
  --user smokeuser \
  --password SmokePass1! \
  --db smokedb 2>&1
dup_exit=$?
set -e
if [[ $dup_exit -eq 0 ]]; then
  echo "FAIL: duplicate create should have failed"
  exit 1
fi

echo "PASS: docker create"
