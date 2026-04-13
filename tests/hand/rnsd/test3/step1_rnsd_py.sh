#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test3"
ARTIFACTS_DIR="$TEST_DIR/.artifacts"
RUN_DIR="$ARTIFACTS_DIR/run/py"
SERVER_DIR="$RUN_DIR/server"
CLIENT_DIR="$RUN_DIR/client"
PYTHON="${PYTHON:-python3}"
SERVER_LOG="$SERVER_DIR/logfile.py"
CLIENT_LOG="$CLIENT_DIR/logfile.py"
SERVER_PIDFILE="$SERVER_DIR/python-rnsd.pid"
CLIENT_PIDFILE="$CLIENT_DIR/python-rnsd.pid"
SERVER_SHARED_PORT=37444
SERVER_CONTROL_PORT=37445
CLIENT_SHARED_PORT=37446
CLIENT_CONTROL_PORT=37447

mkdir -p "$ARTIFACTS_DIR" "$SERVER_DIR" "$CLIENT_DIR"

inject_shared_instance() {
  local cfg="$1"
  local shared_port="$2"
  local control_port="$3"
  perl -0pi -e "s/(\\[reticulum\\]\\n)/\\1share_instance = Yes\nshared_instance_port = ${shared_port}\\ninstance_control_port = ${control_port}\\n/" "$cfg"
}

stop_pidfile() {
  local pidfile="$1"
  if [[ -f "$pidfile" ]]; then
    local pid
    pid="$(cat "$pidfile" 2>/dev/null || true)"
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      for _ in {1..50}; do
        if ! kill -0 "$pid" 2>/dev/null; then
          break
        fi
        sleep 0.1
      done
      kill -9 "$pid" 2>/dev/null || true
    fi
    rm -f "$pidfile"
  fi
}

cleanup() {
  [[ -n "${TAIL_SERVER_PID:-}" ]] && kill "$TAIL_SERVER_PID" 2>/dev/null || true
  [[ -n "${TAIL_CLIENT_PID:-}" ]] && kill "$TAIL_CLIENT_PID" 2>/dev/null || true
  stop_pidfile "$CLIENT_PIDFILE"
  stop_pidfile "$SERVER_PIDFILE"
}

trap cleanup EXIT INT TERM

echo "[1/4] Writing runtime configs"
cp "$TEST_DIR/config" "$SERVER_DIR/config"
cp "$TEST_DIR/config" "$CLIENT_DIR/config"
inject_shared_instance "$SERVER_DIR/config" "$SERVER_SHARED_PORT" "$SERVER_CONTROL_PORT"
inject_shared_instance "$CLIENT_DIR/config" "$CLIENT_SHARED_PORT" "$CLIENT_CONTROL_PORT"
rm -f "$SERVER_LOG" "$CLIENT_LOG" "$SERVER_LOG.1" "$CLIENT_LOG.1"

echo "[2/4] Starting Python server node"
PYTHONPATH="$ROOT/python" "$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --config "$SERVER_DIR" -vvvvvvv >"$SERVER_LOG" 2>&1 &
SERVER_PID=$!
echo "$SERVER_PID" >"$SERVER_PIDFILE"

echo "[3/4] Starting Python client node"
PYTHONPATH="$ROOT/python" "$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --config "$CLIENT_DIR" -vvvvvvv >"$CLIENT_LOG" 2>&1 &
CLIENT_PID=$!
echo "$CLIENT_PID" >"$CLIENT_PIDFILE"

echo "[4/4] Waiting for startup"
for _ in {1..150}; do
  if [[ -f "$SERVER_LOG" ]] && [[ -f "$CLIENT_LOG" ]] &&
     grep -q "Started rnsd version" "$SERVER_LOG" 2>/dev/null &&
     grep -q "Started rnsd version" "$CLIENT_LOG" 2>/dev/null; then
    break
  fi
  sleep 0.1
done

echo "python rncp TCP nodes started"
echo "server pid: $SERVER_PID"
echo "client pid: $CLIENT_PID"
echo "server log: $SERVER_LOG"
echo "client log: $CLIENT_LOG"
echo "next: ./tests/hand/rnsd/test3/step2_rncp_py.sh"
echo "Press Ctrl+C to stop nodes"
echo
echo "===== server logfile stream ====="
tail -n +1 -f "$SERVER_LOG" &
TAIL_SERVER_PID=$!
echo
echo "===== client logfile stream ====="
tail -n +1 -f "$CLIENT_LOG" &
TAIL_CLIENT_PID=$!

wait "$SERVER_PID" "$CLIENT_PID"
