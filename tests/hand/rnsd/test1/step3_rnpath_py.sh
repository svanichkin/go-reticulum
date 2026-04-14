#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test1"
RUN_DIR="$TEST_DIR/.artifacts/run/py"
PYTHON="${PYTHON:-python3}"

RNPATH_PY="$ROOT/python/RNS/Utilities/rnpath.py"
GO_RUN_DIR="$TEST_DIR/.artifacts/run/go"
GO_PIDFILE="$GO_RUN_DIR/rnsd.pid"
PY_PIDFILE="$RUN_DIR/python-rnsd.pid"

resolve_active_run_dir() {
  local pid

  if [[ -f "$GO_PIDFILE" ]]; then
    pid="$(cat "$GO_PIDFILE" 2>/dev/null || true)"
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
      ACTIVE_LABEL="Go"
      ACTIVE_PIDFILE="$GO_PIDFILE"
      ACTIVE_RUN_DIR="$GO_RUN_DIR"
      return 0
    fi
  fi

  if [[ -f "$PY_PIDFILE" ]]; then
    pid="$(cat "$PY_PIDFILE" 2>/dev/null || true)"
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
      ACTIVE_LABEL="Python"
      ACTIVE_PIDFILE="$PY_PIDFILE"
      ACTIVE_RUN_DIR="$RUN_DIR"
      return 0
    fi
  fi

  echo "No running rnsd found, run step1_rnsd_go.sh or step1_rnsd_py.sh first"
  exit 1
}

resolve_active_run_dir
CFG="$ACTIVE_RUN_DIR"

DEST_HASH="${1:-}"

echo "[1/4] Path table (before)"
echo "Using $ACTIVE_LABEL rnsd"
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
