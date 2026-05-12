#!/usr/bin/env bash
set -euo pipefail

BINARY="${BINARY:-forge}"
PROJECT="smoke-create-$$"

cleanup() {
  "$BINARY" docker remove "$PROJECT" --yes 2>/dev/null || true
}
trap cleanup EXIT

echo "=== smoke: docker create ==="

# Create a postgres container
output=$("$BINARY" docker create "$PROJECT" --engine postgres 2>&1)
echo "$output"

# Verify container is running
state=$(docker inspect "forge-${PROJECT}-postgres" --format '{{.State.Running}}' 2>/dev/null)
if [[ "$state" != "true" ]]; then
  echo "FAIL: container not running (state=$state)"
  exit 1
fi

# Verify all five required labels
labels=$(docker inspect "forge-${PROJECT}-postgres" --format '{{json .Config.Labels}}' 2>/dev/null)
for key in forge.managed forge.project forge.engine forge.created_at forge.host_port; do
  if ! echo "$labels" | grep -q "\"$key\""; then
    echo "FAIL: missing label $key"
    exit 1
  fi
done

# Verify connection string printed
if ! echo "$output" | grep -q "postgres://"; then
  echo "FAIL: connection string not printed"
  exit 1
fi

# Verify duplicate create is rejected
set +e
dup_output=$("$BINARY" docker create "$PROJECT" --engine postgres 2>&1)
dup_exit=$?
set -e
if [[ $dup_exit -eq 0 ]]; then
  echo "FAIL: duplicate create should have failed"
  exit 1
fi

echo "PASS: docker create"
