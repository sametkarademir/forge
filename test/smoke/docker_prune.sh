#!/usr/bin/env bash
set -euo pipefail

BINARY="${BINARY:-forge}"
PRESET="smoke-prune-$$"

cleanup() {
  # Remove any leftover containers or volumes from this run.
  docker rm -f "forge-${PRESET}" 2>/dev/null || true
  docker volume rm "forge-${PRESET}-data" 2>/dev/null || true
  "$BINARY" docker remove "$PRESET" --purge --yes 2>/dev/null || true
}
trap cleanup EXIT

echo "=== smoke: docker prune ==="

# Create a preset but do NOT run it.
"$BINARY" docker create "$PRESET" --engine postgres \
  --user smokeuser --password SmokePass1! --db smokedb

# prune with nothing to remove — should say "Nothing to prune." and exit 0.
"$BINARY" docker prune --yes

# Delete the preset YAML manually to simulate an orphan.
rm -f ~/.forge/presets/"${PRESET}".yaml

# Run the container directly so there is an orphan.
docker run -d \
  --label forge.managed=true \
  --label "forge.preset=${PRESET}" \
  --label forge.engine=postgres \
  --name "forge-${PRESET}" \
  postgres:16-alpine tail -f /dev/null

# prune should detect the orphan and remove it.
"$BINARY" docker prune --yes

# Container must be gone.
if docker inspect "forge-${PRESET}" &>/dev/null; then
  echo "FAIL: orphan container still exists after prune"
  exit 1
fi

echo "PASS: docker prune"
