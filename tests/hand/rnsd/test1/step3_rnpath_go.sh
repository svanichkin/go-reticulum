#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test1"
RUN_DIR="$TEST_DIR/.artifacts/run/go"
CFG="$RUN_DIR"
BIN_DIR="$ROOT/bin"
RNPATH_BIN="$BIN_DIR/rnpath"
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

DEST_HASH="${1:-}"

echo "[1/4] Building rnpath"
mkdir -p /tmp/go-cache /tmp/go-tmp
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNPATH_BIN" ./cmd/rnpath

echo "[2/4] Path table (before)"
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
