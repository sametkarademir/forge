#!/usr/bin/env bash
set -euo pipefail

BINARY="${BINARY:-forge}"
PROJECT="smoke-reset-$$"

cleanup() {
  "$BINARY" docker remove "$PROJECT" --yes 2>/dev/null || true
}
trap cleanup EXIT

echo "=== smoke: docker reset ==="

"$BINARY" docker create "$PROJECT" --engine postgres

# Record original port
port_before=$("$BINARY" docker conn "$PROJECT" | grep -oE ':[0-9]+/' | tr -d ':/')

# Reset
"$BINARY" docker reset "$PROJECT" --yes

# Verify container still running
state=$(docker inspect "forge-${PROJECT}-postgres" --format '{{.State.Running}}' 2>/dev/null)
if [[ "$state" != "true" ]]; then
  echo "FAIL: container not running after reset"
  exit 1
fi

# Verify same port
port_after=$("$BINARY" docker conn "$PROJECT" | grep -oE ':[0-9]+/' | tr -d ':/')
if [[ "$port_before" != "$port_after" ]]; then
  echo "FAIL: port changed after reset (before=$port_before after=$port_after)"
  exit 1
fi

# Run reset a second time to verify it succeeds again
"$BINARY" docker reset "$PROJECT" --yes

echo "PASS: docker reset"
