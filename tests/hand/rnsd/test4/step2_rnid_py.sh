#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test4"
ARTIFACTS_DIR="$TEST_DIR/.artifacts"
RUN_DIR="$ARTIFACTS_DIR/run/py"
ORIGIN_DIR="$RUN_DIR/origin"
RELAY_B_DIR="$RUN_DIR/relay-b"
RELAY_C_DIR="$RUN_DIR/relay-c"
PYTHON="${PYTHON:-python3}"
RNID_PY="$ROOT/python/RNS/Utilities/rnid.py"
WORK_DIR="$RUN_DIR/announce-loop"

ORIGIN_LOG="$ORIGIN_DIR/logfile.py"
RELAY_B_LOG="$RELAY_B_DIR/logfile.py"
RELAY_C_LOG="$RELAY_C_DIR/logfile.py"
IDENTITY_FILE="$ORIGIN_DIR/loop.id"
ASPECT="${ASPECT:-looptest.echo}"
OBSERVER_TIMEOUT="${OBSERVER_TIMEOUT:-12}"
OBSERVER_SETTLE="${OBSERVER_SETTLE:-3}"

mkdir -p "$WORK_DIR"

require_runtime() {
  local dir="$1"
  local label="$2"
  if [[ ! -f "$dir/config" ]]; then
    echo "Missing runtime config for $label: $dir/config"
    echo "Run ./tests/hand/rnsd/test4/step1_rnsd_py.sh first"
    exit 1
  fi
}

parse_hash() {
  local file="$1"
  sed -nE 's/.*destination for this Identity is <([0-9a-fA-F]+)>.*/\1/p' "$file" | tail -n 1
}

count_updates_after() {
  local file="$1"
  local line_no="$2"
  local hash="$3"
  local start=$((line_no + 1))
  sed -n "${start},\$p" "$file" 2>/dev/null | grep -F -c "Destination <${hash}> is now" || true
}

wait_for_line() {
  local file="$1"
  local pattern="$2"
  for _ in {1..150}; do
    if [[ -f "$file" ]] && grep -q "$pattern" "$file" 2>/dev/null; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

cleanup() {
  if [[ -n "${OBSERVER_PID:-}" ]] && kill -0 "$OBSERVER_PID" 2>/dev/null; then
    kill "$OBSERVER_PID" 2>/dev/null || true
    wait "$OBSERVER_PID" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

require_runtime "$ORIGIN_DIR" "origin"
require_runtime "$RELAY_B_DIR" "relay-b"
require_runtime "$RELAY_C_DIR" "relay-c"

echo "[1/6] Ensuring announce identity exists"
if [[ ! -f "$IDENTITY_FILE" ]]; then
  PYTHONPATH="$ROOT/python" "$PYTHON" "$RNID_PY" --config "$ORIGIN_DIR" -g "$IDENTITY_FILE" >"$WORK_DIR/generate.out" 2>&1
fi

echo "[2/6] Resolving destination hash for $ASPECT"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNID_PY" --config "$ORIGIN_DIR" -i "$IDENTITY_FILE" -H "$ASPECT" >"$WORK_DIR/hash.out" 2>&1
DEST_HASH="$(parse_hash "$WORK_DIR/hash.out")"
if [[ -z "${DEST_HASH:-}" ]]; then
  echo "Could not parse destination hash"
  cat "$WORK_DIR/hash.out"
  exit 1
fi
echo "destination hash: <$DEST_HASH>"

origin_line="$(wc -l <"$ORIGIN_LOG" 2>/dev/null || echo 0)"
relay_b_line="$(wc -l <"$RELAY_B_LOG" 2>/dev/null || echo 0)"
relay_c_line="$(wc -l <"$RELAY_C_LOG" 2>/dev/null || echo 0)"

echo "[3/6] Starting origin observer"
PYTHONPATH="$ROOT/python" PYTHONUNBUFFERED=1 "$PYTHON" "$TEST_DIR/_observe_announces_py.py" \
  --config "$ORIGIN_DIR" \
  --aspect "$ASPECT" \
  --timeout "$OBSERVER_TIMEOUT" \
  --settle "$OBSERVER_SETTLE" >"$WORK_DIR/observer.out" 2>&1 &
OBSERVER_PID=$!
if ! wait_for_line "$WORK_DIR/observer.out" '^READY$'; then
  echo "Observer did not become ready"
  cat "$WORK_DIR/observer.out" 2>/dev/null || true
  exit 1
fi

echo "[4/6] Sending a single announce with rnid.py"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNID_PY" --config "$ORIGIN_DIR" -i "$IDENTITY_FILE" -a "$ASPECT" >"$WORK_DIR/send.out" 2>&1

wait "$OBSERVER_PID"
OBSERVER_PID=""

echo "[5/6] Validating observer count and relay traversal"
OBSERVED_COUNT="$(sed -nE 's/^COUNT=([0-9]+)$/\1/p' "$WORK_DIR/observer.out" | tail -n 1)"
if [[ -z "${OBSERVED_COUNT:-}" ]]; then
  echo "Could not parse observer count"
  cat "$WORK_DIR/observer.out"
  exit 1
fi

relay_b_hits="$(count_updates_after "$RELAY_B_LOG" "$relay_b_line" "$DEST_HASH")"
relay_c_hits="$(count_updates_after "$RELAY_C_LOG" "$relay_c_line" "$DEST_HASH")"
origin_hits="$(sed -n "$((origin_line + 1)),\$p" "$ORIGIN_LOG" 2>/dev/null | grep -F -c "Rebroadcasting announce for <${DEST_HASH}>" || true)"

if [[ "$origin_hits" -gt 1 ]]; then
  echo "Origin saw duplicate rebroadcast log entries for <$DEST_HASH>: $origin_hits"
  exit 1
fi

echo "[6/6] Done"
echo "observer count: $OBSERVED_COUNT"
echo "origin path updates: $origin_hits"
echo "artifacts: $WORK_DIR"
