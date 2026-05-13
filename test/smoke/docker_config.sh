#!/usr/bin/env bash
set -euo pipefail

BINARY="${BINARY:-forge}"
CONFIG_FILE="${HOME}/.forge/config.yaml"

# Preserve existing config so this test is non-destructive.
backup=""
if [[ -f "$CONFIG_FILE" ]]; then
  backup=$(mktemp)
  cp "$CONFIG_FILE" "$backup"
fi

cleanup() {
  if [[ -n "$backup" ]]; then
    cp "$backup" "$CONFIG_FILE"
    rm -f "$backup"
  else
    # Config didn't exist before; remove whatever the test wrote.
    rm -f "$CONFIG_FILE"
  fi
}
trap cleanup EXIT

echo "=== smoke: docker config ==="

# 1. config show must exit 0 and print known keys.
output=$("$BINARY" docker config show 2>&1)
echo "$output"

for key in docker.default_user docker.default_password docker.default_db \
            docker.port_range_start docker.port_range_end \
            docker.readiness_timeout_seconds \
            docker.engines.postgres.default_image \
            docker.engines.mysql.default_image \
            docker.engines.mssql.default_image; do
  if ! echo "$output" | grep -q "$key"; then
    echo "FAIL: config show missing key $key"
    exit 1
  fi
done

# All rows should show 'default' source on a clean config.
if ! echo "$output" | grep -q "default"; then
  echo "FAIL: expected at least one 'default' source row"
  exit 1
fi

# 2. config set with a valid key must exit 0.
"$BINARY" docker config set engines.postgres.default_image postgres:17-alpine
output=$("$BINARY" docker config show 2>&1)

# The overridden key should now show 'user' source.
if ! echo "$output" | grep -q "user"; then
  echo "FAIL: expected 'user' source after config set"
  exit 1
fi

# The value should be updated.
if ! echo "$output" | grep -q "postgres:17-alpine"; then
  echo "FAIL: updated value not visible in config show"
  exit 1
fi

# 3. config set with the full prefix form must also work.
"$BINARY" docker config set docker.default_user smoketest-user

# 4. config set with an unknown key must exit non-zero.
set +e
bad_output=$("$BINARY" docker config set bogus.unknown.key somevalue 2>&1)
bad_exit=$?
set -e
if [[ $bad_exit -eq 0 ]]; then
  echo "FAIL: config set with unknown key should have failed"
  exit 1
fi

# 5. forge docker create without --engine on a non-TTY must fail with a clear message.
set +e
notty_output=$("$BINARY" docker create smoke-notty-$$ < /dev/null 2>&1)
notty_exit=$?
set -e
if [[ $notty_exit -eq 0 ]]; then
  echo "FAIL: create without --engine on non-TTY should have failed"
  exit 1
fi
if ! echo "$notty_output" | grep -qi "engine"; then
  echo "FAIL: error message should mention 'engine'"
  exit 1
fi

echo "PASS: docker config"
