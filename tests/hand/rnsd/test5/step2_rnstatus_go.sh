#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test5"
RUN_DIR="$TEST_DIR/.artifacts/run/go"
CFG="$RUN_DIR"
BIN_DIR="$ROOT/bin"
RNSTATUS_BIN="$BIN_DIR/rnstatus"
PIDFILE="$RUN_DIR/rnsd.pid"

mkdir -p "$BIN_DIR"

if [[ ! -f "$PIDFILE" ]]; then
  echo "Go rnsd.pid not found, run step1_rnsd_go.sh first"
  exit 1
fi

pid="$(cat "$PIDFILE" 2>/dev/null || true)"
if [[ -z "${pid:-}" ]] || ! kill -0 "$pid" 2>/dev/null; then
  echo "Go rnsd is not running, run step1_rnsd_go.sh first"
  exit 1
fi

echo "[1/2] Building rnstatus"
mkdir -p /tmp/go-cache /tmp/go-tmp
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNSTATUS_BIN" ./cmd/rnstatus

echo "[2/2] Querying shared instance"
echo "Command: $RNSTATUS_BIN -config $CFG -a"
status_ok=0
for _ in {1..100}; do
  set +e
    STATUS_OUT="$("$RNSTATUS_BIN" -config "$CFG" -a 2>&1)"
  STATUS_CODE=$?
  set -e
  if [[ "$STATUS_CODE" -eq 0 ]]; then
    status_ok=1
    break
  fi
  sleep 0.1
done

if [[ "$status_ok" -ne 1 ]]; then
  echo "rnstatus could not reach shared instance"
  echo "$STATUS_OUT"
  exit 1
fi

echo "$STATUS_OUT"
