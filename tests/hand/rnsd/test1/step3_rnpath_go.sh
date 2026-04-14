#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test1"
RUN_DIR="$TEST_DIR/.artifacts/run/go"
BIN_DIR="$ROOT/bin"
RNPATH_BIN="$BIN_DIR/rnpath"
GO_PIDFILE="$RUN_DIR/rnsd.pid"
PY_RUN_DIR="$TEST_DIR/.artifacts/run/py"
PY_PIDFILE="$PY_RUN_DIR/python-rnsd.pid"

resolve_active_run_dir() {
  local pid

  if [[ -f "$GO_PIDFILE" ]]; then
    pid="$(cat "$GO_PIDFILE" 2>/dev/null || true)"
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
      ACTIVE_LABEL="Go"
      ACTIVE_RUN_DIR="$RUN_DIR"
      return 0
    fi
  fi

  if [[ -f "$PY_PIDFILE" ]]; then
    pid="$(cat "$PY_PIDFILE" 2>/dev/null || true)"
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
      ACTIVE_LABEL="Python"
      ACTIVE_RUN_DIR="$PY_RUN_DIR"
      return 0
    fi
  fi

  echo "No running rnsd found, run step1_rnsd_go.sh or step1_rnsd_py.sh first"
  exit 1
}

mkdir -p "$BIN_DIR"
resolve_active_run_dir
CFG="$ACTIVE_RUN_DIR"

DEST_HASH="${1:-}"

echo "[1/4] Building rnpath"
mkdir -p /tmp/go-cache /tmp/go-tmp
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNPATH_BIN" ./cmd/rnpath

echo "[2/4] Path table (before)"
echo "Using $ACTIVE_LABEL rnsd"
echo "Command: $RNPATH_BIN -config $CFG -t"
"$RNPATH_BIN" -config "$CFG" -t || true

if [[ -z "${DEST_HASH:-}" ]]; then
  echo
  echo "[2.5/4] Selecting destination from table"
  echo "Command: $RNPATH_BIN -config $CFG -t -j"
  TABLE_JSON="$("$RNPATH_BIN" -config "$CFG" -t -j 2>/dev/null || true)"
  # JSON includes multiple 32-hex values (destination hash, via hash, etc).
  # Only pick the first destination `"hash":"..."`.
  DEST_HASH="$(
    printf '%s\n' "$TABLE_JSON" |
      rg -o '"hash"\s*:\s*"[0-9a-f]{32}"' |
      head -n 1 |
      rg -o '[0-9a-f]{32}' || true
  )"
  DEST_HASH="$(printf '%s' "${DEST_HASH:-}" | tr -d '\r\n')"
  if [[ -z "${DEST_HASH:-}" ]]; then
    echo "Could not auto-select destination from path table."
    echo "Usage: $0 <destination_hash_hex_32chars>"
    exit 2
  fi
  echo "Selected: $DEST_HASH"
fi

echo
echo "[3/4] Requesting path"
echo "Command: $RNPATH_BIN -config $CFG -w 15 $DEST_HASH"
"$RNPATH_BIN" -config "$CFG" -w 15 "$DEST_HASH"

echo
echo "[4/4] Path table (after)"
echo "Command: $RNPATH_BIN -config $CFG -t"
"$RNPATH_BIN" -config "$CFG" -t
