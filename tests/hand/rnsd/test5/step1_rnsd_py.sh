#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test5"
ARTIFACTS_DIR="$TEST_DIR/.artifacts"
RUN_DIR="$ARTIFACTS_DIR/run/py"
CFG="$RUN_DIR"
CONFIG_FILE="$RUN_DIR/config"
PYTHON="${PYTHON:-python3}"
LOGFILE="$RUN_DIR/logfile.py"
PIDFILE="$RUN_DIR/python-rnsd.pid"
SHARED_PORT=37438
CONTROL_PORT=37439
INSTANCE_NAME="test5-py"
RNSD_CMD=("$PYTHON" -u "$ROOT/python/RNS/Utilities/rnsd.py" --config "$CFG" -vvvvvvv)

mkdir -p "$ARTIFACTS_DIR" "$RUN_DIR"
cp "$TEST_DIR/config" "$CONFIG_FILE"
perl -0pi -e "s/(\\[reticulum\\]\\n)/\\1shared_instance_port = $SHARED_PORT\\ninstance_control_port = $CONTROL_PORT\\n/" "$CONFIG_FILE"
perl -0pi -e "s/^[[:space:]]*instance_name[[:space:]]*=.*$/  instance_name = $INSTANCE_NAME/m" "$CONFIG_FILE"

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

echo "[1/2] Starting python rnsd"
echo "Command: PYTHONUNBUFFERED=1 PYTHONPATH=$ROOT/python ${RNSD_CMD[*]}"
PYTHONUNBUFFERED=1 PYTHONPATH="$ROOT/python" "${RNSD_CMD[@]}" >"$LOGFILE" 2>&1 &
RNSD_PID=$!
echo "$RNSD_PID" > "$PIDFILE"

echo "[2/2] Waiting for startup logs"
started=0
ready=0
for _ in {1..200}; do
  if [[ -f "$LOGFILE" ]]; then
    if grep -q "Started rnsd version" "$LOGFILE" 2>/dev/null; then
      started=1
    fi
    if grep -q "System interfaces are ready" "$LOGFILE" 2>/dev/null; then
      ready=1
    fi
  fi
  if [[ "$started" -eq 1 && "$ready" -eq 1 ]]; then
    break
  fi
  if ! kill -0 "$RNSD_PID" 2>/dev/null; then
    echo "python rnsd exited unexpectedly"
    [[ -f "$LOGFILE" ]] && cat "$LOGFILE"
    exit 1
  fi
  sleep 0.1
done

if [[ "$started" -ne 1 || "$ready" -ne 1 ]]; then
  echo "Timed out waiting for python rnsd startup"
  [[ -f "$LOGFILE" ]] && cat "$LOGFILE"
  exit 1
fi

echo "python rnsd started successfully"
echo "config: $CFG/config"
echo "log: $LOGFILE"
echo "manual step: open Reticulum MeshChat and watch announces in this log stream"
echo "Press Ctrl+C to stop python rnsd"
echo
echo "===== logfile stream ====="

tail -n +1 -f "$LOGFILE" &
TAIL_PID=$!

set +e
wait "$RNSD_PID"
RNSD_EXIT=$?
set -e
kill "$TAIL_PID" 2>/dev/null || true

if [[ "$RNSD_EXIT" -ne 0 ]]; then
  echo "python rnsd exited with code $RNSD_EXIT"
  exit "$RNSD_EXIT"
fi
