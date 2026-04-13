#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test1"
RUN_DIR="$TEST_DIR/.run/py"
CFG="$RUN_DIR"
PIDFILE="$RUN_DIR/python-rnsd.pid"
PYTHON="${PYTHON:-python3}"

RNPATH_PY="$ROOT/python/RNS/Utilities/rnpath.py"

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

echo "[1/4] Path table (before)"
echo "Command: PYTHONPATH=$ROOT/python $PYTHON $RNPATH_PY --config $CFG -t"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNPATH_PY" --config "$CFG" -t || true

if [[ -z "${DEST_HASH:-}" ]]; then
  echo
  echo "[1.5/4] Selecting destination from table"
  # Python rnpath text output contains two hashes per line: "<dest> ... via <via>".
  # Only pick the destination hash at start-of-line.
  TABLE_TXT="$(PYTHONPATH="$ROOT/python" "$PYTHON" "$RNPATH_PY" --config "$CFG" -t 2>/dev/null || true)"
  DEST_HASH="$(printf '%s\n' "$TABLE_TXT" | rg -o '^<[0-9a-f]{32}>' | head -n 1 | tr -d '<>' || true)"
  if [[ -z "${DEST_HASH:-}" ]]; then
    echo "Could not auto-select destination from path table."
    echo "Usage: $0 <destination_hash_hex_32chars>"
    exit 2
  fi
  echo "Selected: $DEST_HASH"
fi

echo
echo "[2/4] Requesting path"
echo "Command: PYTHONPATH=$ROOT/python $PYTHON $RNPATH_PY --config $CFG -w 15 $DEST_HASH"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNPATH_PY" --config "$CFG" -w 15 "$DEST_HASH"

echo
echo "[3/4] Path table (after)"
echo "Command: PYTHONPATH=$ROOT/python $PYTHON $RNPATH_PY --config $CFG -t"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNPATH_PY" --config "$CFG" -t

echo
echo "[4/4] Done"
