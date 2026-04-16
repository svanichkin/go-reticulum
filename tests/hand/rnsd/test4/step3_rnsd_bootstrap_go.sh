#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test4"
ARTIFACTS_DIR="$TEST_DIR/.artifacts"
RUN_DIR="$ARTIFACTS_DIR/run/bootstrap/go"
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

START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-60}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-10}"

mkdir -p "$ARTIFACTS_DIR" "$RUN_DIR" "$ORIGIN_DIR" "$RELAY_B_DIR" "$RELAY_C_DIR" "$BIN_DIR"

write_bootstrap_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local shared_port="$3"
  local control_port="$4"
  local share_instance="$5"
  cat >"$cfg_dir/config" <<EOF
[reticulum]
enable_transport = True
share_instance = ${share_instance}
shared_instance_port = ${shared_port}
instance_control_port = ${control_port}
instance_name = ${instance_name}

[logging]
loglevel = 7

[interfaces]

  [[TCP bootstrap dead]]
    type = TCPClientInterface
    enabled = Yes
    target_host = 212.233.88.164
    target_port = 4242

  [[TCP bootstrap dublin]]
    type = TCPClientInterface
    enabled = Yes
    target_host = dublin.connect.reticulum.network
    target_port = 4965
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
  stop_pidfile "$RELAY_C_PIDFILE"
  stop_pidfile "$RELAY_B_PIDFILE"
  stop_pidfile "$ORIGIN_PIDFILE"
}
trap cleanup EXIT INT TERM

echo "[1/5] Building rnsd"
mkdir -p /tmp/go-cache /tmp/go-tmp
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNSD_BIN" ./cmd/rnsd

echo "[2/5] Writing runtime configs"
write_bootstrap_config "$ORIGIN_DIR" test4-go-bootstrap-origin 38501 38502 Yes
write_bootstrap_config "$RELAY_B_DIR" test4-go-bootstrap-relay-b 38511 38512 No
write_bootstrap_config "$RELAY_C_DIR" test4-go-bootstrap-relay-c 38521 38522 No

rm -f "$ORIGIN_LOG" "$RELAY_B_LOG" "$RELAY_C_LOG"

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

echo "[5/5] Waiting for bootstrap TCP startup"
ready=0
for _ in {1..600}; do
  if [[ -f "$ORIGIN_LOG" ]] && [[ -f "$RELAY_B_LOG" ]] && [[ -f "$RELAY_C_LOG" ]] &&
     grep -q "Started rnsd version" "$ORIGIN_LOG" 2>/dev/null &&
     grep -q "Started rnsd version" "$RELAY_B_LOG" 2>/dev/null &&
     grep -q "Started rnsd version" "$RELAY_C_LOG" 2>/dev/null &&
     grep -q "TCP" "$ORIGIN_LOG" 2>/dev/null &&
     grep -q "TCP" "$RELAY_B_LOG" 2>/dev/null &&
     grep -q "TCP" "$RELAY_C_LOG" 2>/dev/null; then
    ready=1
    break
  fi
  sleep 0.1
done

if [[ "$ready" -ne 1 ]]; then
  echo "Timed out waiting for bootstrap TCP startup"
  exit 1
fi

echo "go bootstrap announce ring started"
echo "origin pid: $ORIGIN_PID"
echo "relay-b pid: $RELAY_B_PID"
echo "relay-c pid: $RELAY_C_PID"
echo "origin log: $ORIGIN_LOG"
echo "relay-b log: $RELAY_B_LOG"
echo "relay-c log: $RELAY_C_LOG"
echo "next: ./tests/hand/rnsd/test4/step4_rnid_bootstrap_go.sh"
echo "Press Ctrl+C to stop nodes"
wait "$ORIGIN_PID" "$RELAY_B_PID" "$RELAY_C_PID"
