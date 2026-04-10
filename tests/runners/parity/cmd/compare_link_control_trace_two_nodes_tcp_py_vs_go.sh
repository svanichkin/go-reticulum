#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PYTHON="${PYTHON:-python3}"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache" "$ROOT/tests/artifacts/logs"
export GOCACHE="$ROOT/.gocache"
export GOTMPDIR="$ROOT/.gotmp"
export GOPATH="$ROOT/.gopath"
export GOMODCACHE="$ROOT/.gomodcache"
export PYTHONPATH="${PYTHONPATH:-"$ROOT/python"}"
export PYTHONUNBUFFERED=1

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-180}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-30}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-6}"
WAIT_SECS="${WAIT_SECS:-30}"
HOLD_SECS="${HOLD_SECS:-8}"
KEEPALIVE_SECS="${KEEPALIVE_SECS:-3}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_link_control_trace_two_nodes_tcp"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() { rm -rf "$GO_BIN_DIR" || true; }
trap cleanup EXIT

echo "[cmp] out=$OUT_DIR"
echo "[cmp] building go binaries..."
go build -o "$GO_BIN_DIR/go_link_helper" ./tests/support/tools/go_link_helper

run_capture() {
  local out="$1"; shift
  local code=0
  set +e
  "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$CMD_TIMEOUT_SECS" -- "$@" >"$out" 2>&1
  code=$?
  set -e
  echo "$code"
}

wait_for_file_contains() {
  local timeout="$1" path="$2" needle="$3"
  local start now
  start="$(date +%s)"
  while true; do
    if [[ -f "$path" ]] && rg --fixed-strings "$needle" "$path" >/dev/null 2>&1; then return 0; fi
    now="$(date +%s)"
    [[ $((now - start)) -ge "$timeout" ]] && return 1
    sleep 0.2
  done
}

stop_proc() {
  local pid="$1"
  [[ -z "${pid}" ]] && return 0
  kill -0 "$pid" >/dev/null 2>&1 || return 0
  kill -INT "$pid" >/dev/null 2>&1 || true
  if "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1; then return 0; fi
  kill -TERM "$pid" >/dev/null 2>&1 || true
  if "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1; then return 0; fi
  kill -KILL "$pid" >/dev/null 2>&1 || true
  "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1 || true
}

new_tcp_run_dir() {
  local template="$1" sip="$2" cip="$3" listen_port="$4" target_port="$5"
  local run_dir
  run_dir="$(mktemp -d)"
  cp "$template" "$run_dir/config"
  "$PYTHON" "$ROOT/tests/support/tools/patch_reticulum_config_ports.py" \
    --path "$run_dir/config" \
    --shared-instance-port "$sip" \
    --instance-control-port "$cip" \
    --listen-port "$listen_port" \
    --target-port "$target_port"
  echo "$run_dir"
}

extract_listen_hash() {
  local path="$1"
  awk '/^LISTEN_HASH / { print $2; exit }' "$path"
}

assert_contains() {
  local path="$1" needle="$2" label="$3"
  rg --fixed-strings "$needle" "$path" >/dev/null 2>&1 || {
    echo "[cmp] $label missing '$needle'; see $path"
    return 1
  }
}

assert_trace() {
  local label="$1" client_out="$2" listener_out="$3"
  assert_contains "$listener_out" "LRPROOF sent" "$label" || return 1
  assert_contains "$client_out" "LRPROOF validated" "$label" || return 1
  assert_contains "$client_out" "KEEPALIVE sent 0xFF" "$label" || return 1
  assert_contains "$listener_out" "KEEPALIVE received" "$label" || return 1
  assert_contains "$listener_out" "KEEPALIVE sent 0xFE" "$label" || return 1
  assert_contains "$client_out" "LINKCLOSE sent" "$label" || return 1
  assert_contains "$listener_out" "LINKCLOSE received" "$label" || return 1
}

run_pair() {
  local label="$1"
  local helper_cmd_a="$2" cfg_flag_a="$3"
  local helper_cmd_b="$4" cfg_flag_b="$5"

  echo
  echo "[cmp] $label: link control trace"

  local server_port sip_a cip_a sip_b cip_b
  server_port=$(( (RANDOM % 10000) + 42000 ))
  sip_a=$(( (RANDOM % 10000) + 38000 ))
  cip_a=$(( sip_a + 1 ))
  sip_b=$(( sip_a + 2 ))
  cip_b=$(( sip_a + 3 ))

  local node_a_dir node_b_dir
  node_a_dir="$(new_tcp_run_dir "$ROOT/configs/testing/two_nodes_tcp/client/config" "$sip_a" "$cip_a" 0 "$server_port")"
  node_b_dir="$(new_tcp_run_dir "$ROOT/configs/testing/two_nodes_tcp/server/config" "$sip_b" "$cip_b" "$server_port" 0)"
  local home_a="$node_a_dir/home" home_b="$node_b_dir/home"
  mkdir -p "$home_a" "$home_b"

  local listener_out="$OUT_DIR/${label}.listener.out"
  bash -lc "env HOME=\"$home_b\" USERPROFILE=\"$home_b\" $helper_cmd_b $cfg_flag_b \"$node_b_dir\" --trace --listen --wait-seconds \"$WAIT_SECS\" --keepalive-seconds \"$KEEPALIVE_SECS\"" >"$listener_out" 2>&1 &
  local listener_pid=$!
  wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_out" "LISTEN_HASH " || {
    echo "[cmp] $label: listener not ready"
    stop_proc "$listener_pid"
    return 1
  }
  local listen_hash
  listen_hash="$(extract_listen_hash "$listener_out")"

  local client_out="$OUT_DIR/${label}.client.out"
  local code
  code="$(run_capture "$client_out" bash -lc "env HOME=\"$home_a\" USERPROFILE=\"$home_a\" $helper_cmd_a $cfg_flag_a \"$node_a_dir\" --trace --destination \"$listen_hash\" --identify --teardown --hold-seconds \"$HOLD_SECS\" --wait-seconds \"$WAIT_SECS\" --keepalive-seconds \"$KEEPALIVE_SECS\"")"
  if [[ "$code" != "0" ]]; then
    echo "[cmp] $label: client failed; see $client_out"
    stop_proc "$listener_pid"
    return 1
  fi

  wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_out" "EVENT closed" || {
    echo "[cmp] $label: close event missing"
    stop_proc "$listener_pid"
    return 1
  }

  assert_trace "$label" "$client_out" "$listener_out" || {
    stop_proc "$listener_pid"
    return 1
  }

  stop_proc "$listener_pid"
  echo "[cmp] $label OK"
}

overall=0
if ! run_pair "go_node_a_python_node_b" \
  "$GO_BIN_DIR/go_link_helper" "-config" \
  "$PYTHON $ROOT/tests/support/tools/py_link_helper.py" "--config"; then
  overall=1
fi
if ! run_pair "python_node_a_go_node_b" \
  "$PYTHON $ROOT/tests/support/tools/py_link_helper.py" "--config" \
  "$GO_BIN_DIR/go_link_helper" "-config"; then
  overall=1
fi
if [[ "$overall" != "0" ]]; then
  echo "[cmp] FAIL"
  exit 1
fi
echo "[cmp] OK"
