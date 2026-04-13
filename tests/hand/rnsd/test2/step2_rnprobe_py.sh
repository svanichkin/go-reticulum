#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test2"
RUN_DIR="$TEST_DIR/.run/py"
CFG="$RUN_DIR"
PIDFILE="$RUN_DIR/python-rnsd.pid"
LOGFILE="$RUN_DIR/logfile.py"
PYTHON="${PYTHON:-python3}"
RNPROBE_CMD=("$PYTHON" "$ROOT/python/RNS/Utilities/rnprobe.py")

if [[ ! -f "$PIDFILE" ]]; then
  echo "Python rnsd.pid not found, run step1_rnsd_py.sh first"
  exit 1
fi

pid="$(cat "$PIDFILE" 2>/dev/null || true)"
if [[ -z "${pid:-}" ]] || ! kill -0 "$pid" 2>/dev/null; then
  echo "Python rnsd is not running, run step1_rnsd_py.sh first"
  exit 1
fi

DEST_HASH="${1:-}"

echo "[1/3] Finding probe responder in logfile"
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
echo "[2/3] Probing rnstransport.probe"
echo "Command: PYTHONPATH=$ROOT/python ${RNPROBE_CMD[*]} --config $CFG -n 3 -w 0.2 -t 15 rnstransport.probe $DEST_HASH"
PYTHONPATH="$ROOT/python" "${RNPROBE_CMD[@]}" --config "$CFG" -n 3 -w 0.2 -t 15 rnstransport.probe "$DEST_HASH"

echo
echo "[3/3] Done"
