#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test2"
RUN_DIR="$TEST_DIR/.run/go"
CFG="$RUN_DIR"
CONFIG_FILE="$RUN_DIR/config"
BIN_DIR="$ROOT/bin"
RNSD_BIN="$BIN_DIR/rnsd"
LOGFILE="$RUN_DIR/logfile"
PIDFILE="$RUN_DIR/rnsd.pid"
SHARED_PORT=37432
CONTROL_PORT=37433
RNSD_FLAGS=(-config "$CFG" -service -vvvvvvv)

mkdir -p "$RUN_DIR"
cp "$TEST_DIR/config" "$CONFIG_FILE"
perl -0pi -e "s/(\\[reticulum\\]\\n)/\\1shared_instance_port = $SHARED_PORT\\ninstance_control_port = $CONTROL_PORT\\n/" "$CONFIG_FILE"

mkdir -p "$BIN_DIR"

cleanup() {
  if [[ -n "${TAIL_PID:-}" ]] && kill -0 "${TAIL_PID}" 2>/dev/null; then
    kill "$TAIL_PID" 2>/dev/null || true
  fi
  if [[ -f "$PIDFILE" ]]; then
    pid="$(cat "$PIDFILE" 2>/dev/null || true)"
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
      echo
      echo "Stopping rnsd pid=$pid"
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

on_interrupt() {
  cleanup
  exit 130
}

trap cleanup EXIT
trap on_interrupt INT TERM

if [[ -f "$PIDFILE" ]]; then
  old_pid="$(cat "$PIDFILE" 2>/dev/null || true)"
  if [[ -n "${old_pid:-}" ]] && kill -0 "$old_pid" 2>/dev/null; then
    echo "rnsd is already running with pid $old_pid"
    exit 1
  fi
  rm -f "$PIDFILE"
fi

echo "[1/4] Building rnsd"
mkdir -p /tmp/go-cache /tmp/go-tmp
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNSD_BIN" ./cmd/rnsd

echo "[2/4] Resetting old logfile"
rm -f "$LOGFILE" "$LOGFILE.1"

echo "[3/4] Starting transport-enabled rnsd in service mode"
echo "Command: $RNSD_BIN ${RNSD_FLAGS[*]}"
"$RNSD_BIN" "${RNSD_FLAGS[@]}" &
RNSD_PID=$!
echo "$RNSD_PID" > "$PIDFILE"

echo "[4/4] Waiting for probe responder log"
started=0
for _ in {1..150}; do
  if [[ -f "$LOGFILE" ]] && grep -q "Transport Instance will respond to probe requests on" "$LOGFILE" 2>/dev/null; then
    started=1
    break
  fi
  if ! kill -0 "$RNSD_PID" 2>/dev/null; then
    echo "rnsd exited unexpectedly"
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

echo "transport-enabled rnsd started successfully"
echo "pid: $RNSD_PID"
echo "log: $LOGFILE"
echo "next: ./tests/hand/rnsd/test2/step2_rnprobe_go.sh"
echo "Press Ctrl+C to stop rnsd"
echo
echo "===== logfile stream ====="

tail -n +1 -f "$LOGFILE" &
TAIL_PID=$!

wait "$RNSD_PID"
RNSD_EXIT=$?
kill "$TAIL_PID" 2>/dev/null || true

if [[ "$RNSD_EXIT" -ne 0 ]]; then
  echo "rnsd exited with code $RNSD_EXIT"
  exit "$RNSD_EXIT"
fi
