#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test2"
RUN_DIR="$TEST_DIR/.run/go"
CFG="$RUN_DIR"
BIN_DIR="$ROOT/bin"
RNPROBE_BIN="$BIN_DIR/rnprobe"
PIDFILE="$RUN_DIR/rnsd.pid"
LOGFILE="$RUN_DIR/logfile"

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

echo "[1/4] Building rnprobe"
mkdir -p /tmp/go-cache /tmp/go-tmp
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNPROBE_BIN" ./cmd/rnprobe

echo "[2/4] Finding probe responder in logfile"
if [[ -z "${DEST_HASH:-}" ]]; then
  echo "Log: $LOGFILE"
  DEST_HASH="$(
    rg "respond to probe requests on <" "$LOGFILE" |
      tail -n 1 |
      sed -E 's/.*:<?([0-9a-fA-F]{32})>?>.*/\1/' || true
  )"
  if [[ -z "${DEST_HASH:-}" ]]; then
    echo "Could not auto-select probe responder hash from $LOGFILE"
    echo "Usage: $0 <probe_destination_hash_hex_32chars>"
    exit 2
  fi
fi
echo "Selected: $DEST_HASH"

echo
echo "[3/4] Probing rnstransport.probe"
echo "Command: $RNPROBE_BIN -config $CFG -n 3 -w 0.2 -t 15 rnstransport.probe $DEST_HASH"
"$RNPROBE_BIN" -config "$CFG" -n 3 -w 0.2 -t 15 rnstransport.probe "$DEST_HASH"

echo
echo "[4/4] Done"
