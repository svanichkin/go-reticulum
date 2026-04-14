#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test1"
RUN_DIR="$TEST_DIR/.artifacts/run/py"
PYTHON="${PYTHON:-python3}"
RNSTATUS_CMD=("$PYTHON" "$ROOT/python/RNS/Utilities/rnstatus.py")
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

echo "[1/2] Querying shared instance"
echo "Using $ACTIVE_LABEL rnsd"
echo "Command: ${RNSTATUS_CMD[*]} --config $CFG -a"

status_ok=0
for _ in {1..100}; do
  set +e
    STATUS_OUT="$("${RNSTATUS_CMD[@]}" --config "$CFG" -a 2>&1)"
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
