#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test4"
ARTIFACTS_DIR="$TEST_DIR/.artifacts"
RUN_DIR="$ARTIFACTS_DIR/run/go"
ORIGIN_DIR="$RUN_DIR/origin"
RELAY_B_DIR="$RUN_DIR/relay-b"
RELAY_C_DIR="$RUN_DIR/relay-c"
BIN_DIR="$ROOT/bin"
RNSD_BIN="$BIN_DIR/rnsd"

ORIGIN_LOG="$ORIGIN_DIR/logfile"
RELAY_B_LOG="$RELAY_B_DIR/logfile"
RELAY_C_LOG="$RELAY_C_DIR/logfile"

ORIGIN_PIDFILE="$ORIGIN_DIR/rnsd.pid"
RELAY_B_PIDFILE="$RELAY_B_DIR/rnsd.pid"
RELAY_C_PIDFILE="$RELAY_C_DIR/rnsd.pid"

ORIGIN_TCP_PORT=38500
RELAY_B_TCP_PORT=38510
RELAY_C_TCP_PORT=38520
ORIGIN_SHARED_PORT=38501
ORIGIN_CONTROL_PORT=38502

mkdir -p "$ARTIFACTS_DIR" "$RUN_DIR" "$ORIGIN_DIR" "$RELAY_B_DIR" "$RELAY_C_DIR" "$BIN_DIR"

write_origin_config() {
  cat >"$ORIGIN_DIR/config" <<EOF
[reticulum]
enable_transport = True
share_instance = Yes
shared_instance_port = ${ORIGIN_SHARED_PORT}
instance_control_port = ${ORIGIN_CONTROL_PORT}
instance_name = test4-go-origin

[logging]
loglevel = 7

[interfaces]

  [[TCP Origin Server]]
    type = TCPServerInterface
    enabled = Yes
    listen_ip = 127.0.0.1
    listen_port = ${ORIGIN_TCP_PORT}

  [[TCP Origin Client]]
    type = TCPClientInterface
    enabled = Yes
    target_host = 127.0.0.1
    target_port = ${RELAY_B_TCP_PORT}
    reconnect_wait = 1
EOF
}

write_relay_b_config() {
  cat >"$RELAY_B_DIR/config" <<EOF
[reticulum]
enable_transport = True
share_instance = No
instance_name = test4-go-relay-b

[logging]
loglevel = 7

[interfaces]

  [[TCP Relay B Server]]
    type = TCPServerInterface
    enabled = Yes
    listen_ip = 127.0.0.1
    listen_port = ${RELAY_B_TCP_PORT}

  [[TCP Relay B Client]]
    type = TCPClientInterface
    enabled = Yes
    target_host = 127.0.0.1
    target_port = ${RELAY_C_TCP_PORT}
    reconnect_wait = 1
EOF
}

write_relay_c_config() {
  cat >"$RELAY_C_DIR/config" <<EOF
[reticulum]
enable_transport = True
share_instance = No
instance_name = test4-go-relay-c

[logging]
loglevel = 7

[interfaces]

  [[TCP Relay C Server]]
    type = TCPServerInterface
    enabled = Yes
    listen_ip = 127.0.0.1
    listen_port = ${RELAY_C_TCP_PORT}

  [[TCP Relay C Client]]
    type = TCPClientInterface
    enabled = Yes
    target_host = 127.0.0.1
    target_port = ${ORIGIN_TCP_PORT}
    reconnect_wait = 1
EOF
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
  [[ -n "${TAIL_ORIGIN_PID:-}" ]] && kill "$TAIL_ORIGIN_PID" 2>/dev/null || true
  [[ -n "${TAIL_RELAY_B_PID:-}" ]] && kill "$TAIL_RELAY_B_PID" 2>/dev/null || true
  [[ -n "${TAIL_RELAY_C_PID:-}" ]] && kill "$TAIL_RELAY_C_PID" 2>/dev/null || true
  stop_pidfile "$RELAY_C_PIDFILE"
  stop_pidfile "$RELAY_B_PIDFILE"
  stop_pidfile "$ORIGIN_PIDFILE"
}

trap cleanup EXIT INT TERM

echo "[1/5] Building rnsd"
mkdir -p /tmp/go-cache /tmp/go-tmp
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNSD_BIN" ./cmd/rnsd

echo "[2/5] Writing runtime configs"
write_origin_config
write_relay_b_config
write_relay_c_config
rm -f "$ORIGIN_LOG" "$RELAY_B_LOG" "$RELAY_C_LOG" \
  "$ORIGIN_LOG.1" "$RELAY_B_LOG.1" "$RELAY_C_LOG.1"

echo "[3/5] Starting origin"
"$RNSD_BIN" -config "$ORIGIN_DIR" -service -vvvvvvv &
ORIGIN_PID=$!
echo "$ORIGIN_PID" >"$ORIGIN_PIDFILE"

echo "[4/5] Starting relay-b and relay-c"
"$RNSD_BIN" -config "$RELAY_B_DIR" -service -vvvvvvv &
RELAY_B_PID=$!
echo "$RELAY_B_PID" >"$RELAY_B_PIDFILE"
"$RNSD_BIN" -config "$RELAY_C_DIR" -service -vvvvvvv &
RELAY_C_PID=$!
echo "$RELAY_C_PID" >"$RELAY_C_PIDFILE"

echo "[5/5] Waiting for TCP ring startup"
ready=0
for _ in {1..300}; do
  if [[ -f "$ORIGIN_LOG" ]] && [[ -f "$RELAY_B_LOG" ]] && [[ -f "$RELAY_C_LOG" ]] &&
     grep -q "Started rnsd version" "$ORIGIN_LOG" 2>/dev/null &&
     grep -q "Started rnsd version" "$RELAY_B_LOG" 2>/dev/null &&
     grep -q "Started rnsd version" "$RELAY_C_LOG" 2>/dev/null &&
     grep -q "TCP connection for" "$ORIGIN_LOG" 2>/dev/null &&
     grep -q "TCP connection for" "$RELAY_B_LOG" 2>/dev/null &&
     grep -q "TCP connection for" "$RELAY_C_LOG" 2>/dev/null &&
     grep -q "Accepting incoming TCP connection" "$ORIGIN_LOG" 2>/dev/null &&
     grep -q "Accepting incoming TCP connection" "$RELAY_B_LOG" 2>/dev/null &&
     grep -q "Accepting incoming TCP connection" "$RELAY_C_LOG" 2>/dev/null; then
    ready=1
    break
  fi
  sleep 0.1
done

if [[ "$ready" -ne 1 ]]; then
  echo "Timed out waiting for TCP ring startup"
  exit 1
fi

echo "go TCP announce ring started"
echo "origin pid: $ORIGIN_PID"
echo "relay-b pid: $RELAY_B_PID"
echo "relay-c pid: $RELAY_C_PID"
echo "origin log: $ORIGIN_LOG"
echo "relay-b log: $RELAY_B_LOG"
echo "relay-c log: $RELAY_C_LOG"
echo "next: ./tests/hand/rnsd/test4/step2_rnid_go.sh"
echo "Press Ctrl+C to stop nodes"
echo
echo "===== origin logfile stream ====="
tail -n +1 -f "$ORIGIN_LOG" &
TAIL_ORIGIN_PID=$!
echo
echo "===== relay-b logfile stream ====="
tail -n +1 -f "$RELAY_B_LOG" &
TAIL_RELAY_B_PID=$!
echo
echo "===== relay-c logfile stream ====="
tail -n +1 -f "$RELAY_C_LOG" &
TAIL_RELAY_C_PID=$!

wait "$ORIGIN_PID" "$RELAY_B_PID" "$RELAY_C_PID"
