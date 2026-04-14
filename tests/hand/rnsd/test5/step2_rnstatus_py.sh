#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test5"
RUN_DIR="$TEST_DIR/.artifacts/run/py"
CFG="$RUN_DIR"
PYTHON="${PYTHON:-python3}"
RNSTATUS_CMD=("$PYTHON" "$ROOT/python/RNS/Utilities/rnstatus.py")
PIDFILE="$RUN_DIR/python-rnsd.pid"

if [[ ! -f "$PIDFILE" ]]; then
  echo "Python rnsd.pid not found, run step1_rnsd_py.sh first"
  exit 1
fi

pid="$(cat "$PIDFILE" 2>/dev/null || true)"
if [[ -z "${pid:-}" ]] || ! kill -0 "$pid" 2>/dev/null; then
  echo "Python rnsd is not running, run step1_rnsd_py.sh first"
  exit 1
fi

echo "[1/2] Querying shared instance"
echo "Command: PYTHONPATH=$ROOT/python ${RNSTATUS_CMD[*]} --config $CFG -a"

status_ok=0
for _ in {1..100}; do
  set +e
    STATUS_OUT="$(PYTHONPATH="$ROOT/python" "${RNSTATUS_CMD[@]}" --config "$CFG" -a 2>&1)"
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
