#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test1"
RUN_DIR="$TEST_DIR/.run/py"
CFG="$RUN_DIR"
CONFIG_FILE="$RUN_DIR/config"
PYTHON="${PYTHON:-python3}"
LOGFILE="$RUN_DIR/logfile.py"
PIDFILE="$RUN_DIR/python-rnsd.pid"
RNSD_CMD=("$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --config "$CFG" -vvvvvvv)
SHARED_PORT=37430
CONTROL_PORT=37431

mkdir -p "$RUN_DIR"
cp "$TEST_DIR/config" "$CONFIG_FILE"
perl -0pi -e "s/share_instance = Yes\\n/share_instance = Yes\\nshared_instance_port = $SHARED_PORT\\ninstance_control_port = $CONTROL_PORT\\n/" "$CONFIG_FILE"

if [[ -f "$PIDFILE" ]]; then
  old_pid="$(cat "$PIDFILE" 2>/dev/null || true)"
  if [[ -n "${old_pid:-}" ]] && kill -0 "$old_pid" 2>/dev/null; then
    echo "python rnsd is already running with pid $old_pid"
    exit 1
  fi
  rm -f "$PIDFILE"
fi

cleanup() {
  if [[ -f "$PIDFILE" ]]; then
    pid="$(cat "$PIDFILE" 2>/dev/null || true)"
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
      echo
      echo "Stopping python rnsd pid=$pid"
      kill "$pid" 2>/dev/null || true
      for _ in {1..50}; do
        if ! kill -0 "$pid" 2>/dev/null; then
          break
        fi
        sleep 0.1
      done
      if kill -0 "$pid" 2>/dev/null; then
        kill -9 "$pid" 2>/dev/null || true
      fi
    fi
    rm -f "$PIDFILE"
  fi
}

trap cleanup EXIT INT TERM

rm -f "$LOGFILE" "$LOGFILE.1"

echo "Command: ${RNSD_CMD[*]}"
"${RNSD_CMD[@]}" >"$LOGFILE" 2>&1 &
RNSD_PID=$!
echo "$RNSD_PID" > "$PIDFILE"

echo "python rnsd started"
echo "pid: $RNSD_PID"
echo "log: $LOGFILE"
echo "next: ./tests/hand/rnsd/test1/step2_rnstatus_py.sh"
echo "Press Ctrl+C to stop python rnsd"
echo
echo "===== logfile stream ====="
tail -n +1 -f "$LOGFILE" &
TAIL_PID=$!

wait "$RNSD_PID"
RNSD_EXIT=$?
kill "$TAIL_PID" 2>/dev/null || true

if [[ "$RNSD_EXIT" -ne 0 ]]; then
  echo "python rnsd exited with code $RNSD_EXIT"
  exit "$RNSD_EXIT"
fi
