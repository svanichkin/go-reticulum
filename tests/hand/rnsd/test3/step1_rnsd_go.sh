#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test3"
ARTIFACTS_DIR="$TEST_DIR/.artifacts"
RUN_DIR="$ARTIFACTS_DIR/run/go"
SERVER_DIR="$RUN_DIR/server"
CLIENT_DIR="$RUN_DIR/client"
BIN_DIR="$ROOT/bin"
RNSD_BIN="$BIN_DIR/rnsd"
SERVER_LOG="$SERVER_DIR/logfile"
CLIENT_LOG="$CLIENT_DIR/logfile"
SERVER_PIDFILE="$SERVER_DIR/rnsd.pid"
CLIENT_PIDFILE="$CLIENT_DIR/rnsd.pid"
SERVER_SHARED_PORT=37440
SERVER_CONTROL_PORT=37441
CLIENT_SHARED_PORT=37442
CLIENT_CONTROL_PORT=37443

mkdir -p "$ARTIFACTS_DIR" "$SERVER_DIR" "$CLIENT_DIR" "$BIN_DIR"

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

echo "[1/5] Building rnsd"
mkdir -p /tmp/go-cache /tmp/go-tmp
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNSD_BIN" ./cmd/rnsd

echo "[2/5] Writing runtime configs"
cp "$TEST_DIR/config" "$SERVER_DIR/config"
cp "$TEST_DIR/config" "$CLIENT_DIR/config"
inject_shared_instance "$SERVER_DIR/config" "$SERVER_SHARED_PORT" "$SERVER_CONTROL_PORT"
inject_shared_instance "$CLIENT_DIR/config" "$CLIENT_SHARED_PORT" "$CLIENT_CONTROL_PORT"
rm -f "$SERVER_LOG" "$CLIENT_LOG" "$SERVER_LOG.1" "$CLIENT_LOG.1"

echo "[3/5] Starting Go server node"
"$RNSD_BIN" -config "$SERVER_DIR" -service -vvvvvvv &
SERVER_PID=$!
echo "$SERVER_PID" >"$SERVER_PIDFILE"

echo "[4/5] Starting Go client node"
"$RNSD_BIN" -config "$CLIENT_DIR" -service -vvvvvvv &
CLIENT_PID=$!
echo "$CLIENT_PID" >"$CLIENT_PIDFILE"

echo "[5/5] Waiting for startup"
for _ in {1..150}; do
  if [[ -f "$SERVER_LOG" ]] && [[ -f "$CLIENT_LOG" ]] &&
     grep -q "Started rnsd version" "$SERVER_LOG" 2>/dev/null &&
     grep -q "Started rnsd version" "$CLIENT_LOG" 2>/dev/null; then
    break
  fi
  sleep 0.1
done

echo "go rncp TCP nodes started"
echo "server pid: $SERVER_PID"
echo "client pid: $CLIENT_PID"
echo "server log: $SERVER_LOG"
echo "client log: $CLIENT_LOG"
echo "next: ./tests/hand/rnsd/test3/step2_rncp_go.sh"
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
