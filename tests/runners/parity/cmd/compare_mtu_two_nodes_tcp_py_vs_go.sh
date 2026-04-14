#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PYTHON="${PYTHON:-python3}"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache" "$ROOT/tests/artifacts/logs"
export GOCACHE="$ROOT/.gocache"
GOTMPDIR="${GOTMPDIR:-$(mktemp -d "$ROOT/.gotmp/run.XXXXXX")}"
export GOTMPDIR
export GOPATH="$ROOT/.gopath"
export GOMODCACHE="$ROOT/.gomodcache"
export PYTHONPATH="${PYTHONPATH:-"$ROOT/python"}"
export PYTHONUNBUFFERED=1

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-180}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-30}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-6}"
LINK_WAIT_SECS="${LINK_WAIT_SECS:-20}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_mtu_two_nodes_tcp"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$GO_BIN_DIR" || true
  rm -rf "$GOTMPDIR" || true
}
trap cleanup EXIT

echo "[cmp] out=$OUT_DIR"
echo "[cmp] building go binaries..."
go build -o "$GO_BIN_DIR/rnsd" ./cmd/rnsd
go build -o "$GO_BIN_DIR/rnstatus" ./cmd/rnstatus
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
    if [[ -f "$path" ]] && rg --fixed-strings "$needle" "$path" >/dev/null 2>&1; then
      return 0
    fi
    now="$(date +%s)"
    [[ $((now - start)) -ge "$timeout" ]] && return 1
    sleep 0.2
  done
}

wait_for_tcp_status() {
  local timeout="$1" out="$2" iface_name="$3"; shift 3
  local start now code
  start="$(date +%s)"
  while true; do
    code="$(run_capture "$out" "$@")"
    if [[ "$code" == "0" ]] && awk -v iface="$iface_name" '
      index($0, iface) { in_block=1; next }
      in_block && /^[[:space:]]*$/ { in_block=0 }
      in_block && /Status[[:space:]]*:[[:space:]]*Up/ { found=1; exit }
      END { exit(found ? 0 : 1) }
    ' "$out"; then
      return 0
    fi
    now="$(date +%s)"
    [[ $((now - start)) -ge "$timeout" ]] && return 1
    sleep 0.5
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

maybe_skip_env() {
  local log="$1"
  if rg -q -i "operation not permitted|permission denied|bind:|listen .*:.*(denied|not permitted)" "$log"; then
    echo "[cmp] SKIP: environment does not permit binding sockets (see $log)"
    exit 0
  fi
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

extract_mtu_pair() {
  local path="$1"
  sed -nE 's/^EVENT mtu=([0-9]+) mdu=([0-9]+)$/\1 \2/p' "$path" | tail -n 1
}

run_pair() {
  local label="$1"
  local rnsd_cmd_a="$2" rnstatus_cmd_a="$3" helper_cmd_a="$4" cfg_flag_a="$5"
  local rnsd_cmd_b="$6" rnstatus_cmd_b="$7" helper_cmd_b="$8" cfg_flag_b="$9"

  echo
  echo "[cmp] $label: start tcp server/client nodes"

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

  local log_a="$OUT_DIR/${label}.rnsd.node_a.log"
  local log_b="$OUT_DIR/${label}.rnsd.node_b.log"
  env HOME="$home_a" USERPROFILE="$home_a" $rnsd_cmd_a $cfg_flag_a "$node_a_dir" -q >"$log_a" 2>&1 & local pid_a=$!
  env HOME="$home_b" USERPROFILE="$home_b" $rnsd_cmd_b $cfg_flag_b "$node_b_dir" -q >"$log_b" 2>&1 & local pid_b=$!
  sleep 0.5
  maybe_skip_env "$log_a"; maybe_skip_env "$log_b"

  local status_a="$OUT_DIR/${label}.rnstatus.node_a.out"
  local status_b="$OUT_DIR/${label}.rnstatus.node_b.out"
  wait_for_tcp_status "$START_TIMEOUT_SECS" "$status_a" 'TCP Client/' env HOME="$home_a" USERPROFILE="$home_a" $rnstatus_cmd_a $cfg_flag_a "$node_a_dir" -a || { echo "[cmp] $label: node_a not ready"; stop_proc "$pid_a"; stop_proc "$pid_b"; return 1; }
  wait_for_tcp_status "$START_TIMEOUT_SECS" "$status_b" 'TCP Server/' env HOME="$home_b" USERPROFILE="$home_b" $rnstatus_cmd_b $cfg_flag_b "$node_b_dir" -a || { echo "[cmp] $label: node_b not ready"; stop_proc "$pid_a"; stop_proc "$pid_b"; return 1; }

  local listener_out="$OUT_DIR/${label}.listener.out"
  env HOME="$home_b" USERPROFILE="$home_b" $helper_cmd_b $cfg_flag_b "$node_b_dir" --listen --wait-seconds "$LINK_WAIT_SECS" >"$listener_out" 2>&1 &
  local listener_pid=$!
  wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_out" "LISTEN_HASH " || { echo "[cmp] $label: listener not ready"; stop_proc "$listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"; return 1; }
  local listen_hash
  listen_hash="$(extract_listen_hash "$listener_out")"

  local client_out="$OUT_DIR/${label}.client.out"
  local code
  code="$(run_capture "$client_out" env HOME="$home_a" USERPROFILE="$home_a" $helper_cmd_a $cfg_flag_a "$node_a_dir" --destination "$listen_hash" --teardown --wait-seconds "$LINK_WAIT_SECS")"
  if [[ "$code" != "0" ]]; then
    echo "[cmp] $label: client failed; see $client_out"
    stop_proc "$listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_out" "EVENT closed " || { echo "[cmp] $label: listener did not close"; stop_proc "$listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"; return 1; }

  local mtu_client mtu_listener
  mtu_client="$(extract_mtu_pair "$client_out")"
  mtu_listener="$(extract_mtu_pair "$listener_out")"
  [[ -n "$mtu_client" && -n "$mtu_listener" ]] || { echo "[cmp] $label: missing mtu output"; stop_proc "$listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"; return 1; }
  [[ "$mtu_client" == "$mtu_listener" ]] || { echo "[cmp] $label: mtu mismatch client='$mtu_client' listener='$mtu_listener'"; stop_proc "$listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"; return 1; }

  echo "[cmp] $label: mtu_ok=$mtu_client"
  stop_proc "$listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
  echo "[cmp] $label OK"
}

overall=0
if ! run_pair "go_node_a_python_node_b" \
  "$GO_BIN_DIR/rnsd" "$GO_BIN_DIR/rnstatus" "$GO_BIN_DIR/go_link_helper" "-config" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" "$PYTHON $ROOT/tests/support/tools/py_link_helper.py" "--config"; then
  overall=1
fi
if ! run_pair "python_node_a_go_node_b" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" "$PYTHON $ROOT/tests/support/tools/py_link_helper.py" "--config" \
  "$GO_BIN_DIR/rnsd" "$GO_BIN_DIR/rnstatus" "$GO_BIN_DIR/go_link_helper" "-config"; then
  overall=1
fi
if [[ "$overall" != "0" ]]; then
  echo "[cmp] FAIL"
  exit 1
fi
echo "[cmp] OK"
