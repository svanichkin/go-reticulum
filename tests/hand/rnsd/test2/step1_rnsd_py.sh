#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test2"
RUN_DIR="$TEST_DIR/.run/py"
CFG="$RUN_DIR"
CONFIG_FILE="$RUN_DIR/config"
PYTHON="${PYTHON:-python3}"
LOGFILE="$RUN_DIR/logfile.py"
PIDFILE="$RUN_DIR/python-rnsd.pid"
SHARED_PORT=37434
CONTROL_PORT=37435
RNSD_CMD=("$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --config "$CFG" -vvvvvvv)

mkdir -p "$RUN_DIR"
cp "$TEST_DIR/config" "$CONFIG_FILE"
perl -0pi -e "s/(\\[reticulum\\]\\n)/\\1shared_instance_port = $SHARED_PORT\\ninstance_control_port = $CONTROL_PORT\\n/" "$CONFIG_FILE"

if [[ -f "$PIDFILE" ]]; then
  old_pid="$(cat "$PIDFILE" 2>/dev/null || true)"
  if [[ -n "${old_pid:-}" ]] && kill -0 "$old_pid" 2>/dev/null; then
    echo "python rnsd is already running with pid $old_pid"
    exit 1
  fi
  rm -f "$PIDFILE"
fi

cleanup() {
  if [[ -n "${TAIL_PID:-}" ]] && kill -0 "${TAIL_PID}" 2>/dev/null; then
    kill "$TAIL_PID" 2>/dev/null || true
  fi
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

echo "[1/2] Starting transport-enabled python rnsd"
echo "Command: PYTHONPATH=$ROOT/python ${RNSD_CMD[*]}"
PYTHONPATH="$ROOT/python" "${RNSD_CMD[@]}" >"$LOGFILE" 2>&1 &
RNSD_PID=$!
echo "$RNSD_PID" > "$PIDFILE"

echo "[2/2] Waiting for probe responder log"
started=0
for _ in {1..150}; do
  if [[ -f "$LOGFILE" ]] && grep -q "Transport Instance will respond to probe requests on" "$LOGFILE" 2>/dev/null; then
    started=1
    break
  fi
  if ! kill -0 "$RNSD_PID" 2>/dev/null; then
    echo "python rnsd exited unexpectedly"
    [[ -f "$LOGFILE" ]] && cat "$LOGFILE"
    exit 1
  fi
  sleep 0.1
done

if [[ "$started" -ne 1 ]]; then
  echo "Timed out waiting for probe responder log"
  [[ -f "$LOGFILE" ]] && cat "$LOGFILE"
  exit 1
fi

echo "transport-enabled python rnsd started"
echo "pid: $RNSD_PID"
echo "log: $LOGFILE"
echo "next: ./tests/hand/rnsd/test2/step2_rnprobe_py.sh"
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
