#!/usr/bin/env bash
# Smoke test for the rabbitmq engine.
# Creates a preset directly (bypassing the interactive wizard), runs it, verifies
# both the AMQP port and the Management UI port are bound on the host, exercises
# `forge docker conn` in pretty and raw modes, checks `show` output, resets and
# verifies port preservation, then tears everything down.
#
# Prerequisites: Docker running, forge binary in PATH (or BINARY env var set).
set -euo pipefail

BINARY="${BINARY:-forge}"
PRESET="smoke-rabbitmq-$$"
PRESETS_DIR="${HOME}/.forge/presets"

cleanup() {
  "$BINARY" docker remove "$PRESET" --yes 2>/dev/null || true
}
trap cleanup EXIT

echo "=== smoke: rabbitmq engine ==="

# --- 1. Create preset YAML directly (create command is interactive-only) ---
mkdir -p "$PRESETS_DIR"
cat >"$PRESETS_DIR/${PRESET}.yaml" <<YAML
schema_version: 2
name: ${PRESET}
engine: rabbitmq
image: rabbitmq:3-management-alpine
database: /
username: admin
password: smokepass
internal_port: 5672
host_port: 0
created_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
YAML

# --- 2. Run: both AMQP and Management UI ports should be bound ---
run_output=$("$BINARY" docker run "$PRESET" 2>&1)
echo "$run_output"

# Verify AMQP primary port bound
CONTAINER_NAME="forge-${PRESET}"
amqp_port=$(docker inspect "$CONTAINER_NAME" \
  --format '{{(index (index .NetworkSettings.Ports "5672/tcp") 0).HostPort}}' 2>/dev/null)
if [[ -z "$amqp_port" ]]; then
  echo "FAIL: AMQP port (5672) not bound on host"
  exit 1
fi
echo "  AMQP host port: $amqp_port"

# Verify Management UI port bound
mgmt_port=$(docker inspect "$CONTAINER_NAME" \
  --format '{{(index (index .NetworkSettings.Ports "15672/tcp") 0).HostPort}}' 2>/dev/null)
if [[ -z "$mgmt_port" ]]; then
  echo "FAIL: Management UI port (15672) not bound on host"
  exit 1
fi
echo "  Management UI host port: $mgmt_port"

# Verify forge.extra_port.mgmt_host_port label is set
extra_label=$(docker inspect "$CONTAINER_NAME" \
  --format '{{index .Config.Labels "forge.extra_port.mgmt_host_port"}}' 2>/dev/null)
if [[ "$extra_label" != "$mgmt_port" ]]; then
  echo "FAIL: forge.extra_port.mgmt_host_port label=$extra_label, expected=$mgmt_port"
  exit 1
fi

# --- 3. conn --raw: single-line unmasked AMQP DSN ---
raw_dsn=$("$BINARY" docker conn "$PRESET" --raw 2>&1)
if [[ "$raw_dsn" != amqp://* ]]; then
  echo "FAIL: --raw did not return an AMQP DSN, got: $raw_dsn"
  exit 1
fi
if [[ "$raw_dsn" != *"smokepass"* ]]; then
  echo "FAIL: --raw DSN should contain the unmasked password"
  exit 1
fi
echo "  conn --raw: $raw_dsn"

# --- 4. show: must include Management UI row and masked DSN ---
show_output=$("$BINARY" docker show "$PRESET" 2>&1)
echo "$show_output"
if ! echo "$show_output" | grep -q "Management UI"; then
  echo "FAIL: 'show' output missing Management UI row"
  exit 1
fi
if ! echo "$show_output" | grep -q "amqp://"; then
  echo "FAIL: 'show' output missing AMQP connection row"
  exit 1
fi
if echo "$show_output" | grep -q "smokepass"; then
  echo "FAIL: 'show' output must not contain unmasked password"
  exit 1
fi

# --- 5. reset: verify mgmt port is preserved ---
"$BINARY" docker reset "$PRESET" --yes 2>&1
mgmt_port_after=$(docker inspect "$CONTAINER_NAME" \
  --format '{{(index (index .NetworkSettings.Ports "15672/tcp") 0).HostPort}}' 2>/dev/null)
if [[ "$mgmt_port_after" != "$mgmt_port" ]]; then
  echo "FAIL: mgmt port changed after reset (before=$mgmt_port after=$mgmt_port_after)"
  exit 1
fi
echo "  mgmt port preserved across reset: $mgmt_port_after"

echo "PASS: rabbitmq"
