#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test4"
ARTIFACTS_DIR="$TEST_DIR/.artifacts"
RUN_DIR="$ARTIFACTS_DIR/run/go"
ORIGIN_DIR="$RUN_DIR/origin"
RELAY_B_DIR="$RUN_DIR/relay-b"
RELAY_C_DIR="$RUN_DIR/relay-c"
BIN_DIR="$ROOT/bin"
RNID_BIN="$BIN_DIR/rnid"
WORK_DIR="$RUN_DIR/announce-loop"

ORIGIN_LOG="$ORIGIN_DIR/logfile"
RELAY_B_LOG="$RELAY_B_DIR/logfile"
RELAY_C_LOG="$RELAY_C_DIR/logfile"
IDENTITY_FILE="$ORIGIN_DIR/loop.id"
ASPECT="${ASPECT:-looptest.echo}"
OBSERVER_TIMEOUT="${OBSERVER_TIMEOUT:-12s}"
OBSERVER_SETTLE="${OBSERVER_SETTLE:-3s}"

mkdir -p "$BIN_DIR" "$WORK_DIR"

require_runtime() {
  local dir="$1"
  local label="$2"
  if [[ ! -f "$dir/config" ]]; then
    echo "Missing runtime config for $label: $dir/config"
    echo "Run ./tests/hand/rnsd/test4/step1_rnsd_go.sh first"
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

echo "[1/7] Building Go rnid"
mkdir -p /tmp/go-cache /tmp/go-tmp
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNID_BIN" ./cmd/rnid

echo "[2/7] Ensuring announce identity exists"
if [[ ! -f "$IDENTITY_FILE" ]]; then
  "$RNID_BIN" -config "$ORIGIN_DIR" -g "$IDENTITY_FILE" -f >"$WORK_DIR/generate.out" 2>&1
fi

echo "[3/7] Resolving destination hash for $ASPECT"
"$RNID_BIN" -config "$ORIGIN_DIR" -i "$IDENTITY_FILE" -H "$ASPECT" >"$WORK_DIR/hash.out" 2>&1
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

echo "[4/7] Starting origin observer"
go run "$TEST_DIR/observe_announces_go.go" \
  -config "$ORIGIN_DIR" \
  -aspect "$ASPECT" \
  -timeout "$OBSERVER_TIMEOUT" \
  -settle "$OBSERVER_SETTLE" >"$WORK_DIR/observer.out" 2>&1 &
OBSERVER_PID=$!
if ! wait_for_line "$WORK_DIR/observer.out" '^READY$'; then
  echo "Observer did not become ready"
  cat "$WORK_DIR/observer.out" 2>/dev/null || true
  exit 1
fi

echo "[5/7] Sending a single announce with rnid"
"$RNID_BIN" -config "$ORIGIN_DIR" -i "$IDENTITY_FILE" -a "$ASPECT" >"$WORK_DIR/send.out" 2>&1

wait "$OBSERVER_PID"
OBSERVER_PID=""

echo "[6/7] Validating observer count and relay traversal"
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

echo "[7/7] Done"
echo "observer count: $OBSERVED_COUNT"
echo "origin path updates: $origin_hits"
echo "artifacts: $WORK_DIR"
