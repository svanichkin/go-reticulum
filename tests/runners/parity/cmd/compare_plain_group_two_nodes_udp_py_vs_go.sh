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

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-120}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-30}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-6}"
WAIT_SECS="${WAIT_SECS:-20}"
GROUP_KEY_HEX="${GROUP_KEY_HEX:-00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_plain_group_two_nodes_udp"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() { rm -rf "$GO_BIN_DIR" || true; }
trap cleanup EXIT

echo "[cmp] out=$OUT_DIR"
echo "[cmp] building go binaries..."
go build -o "$GO_BIN_DIR/go_packet_helper" ./tests/support/tools/go_packet_helper

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

maybe_skip_env() {
  local log="$1"
  if rg -q -i "operation not permitted|permission denied|bind:|listen .*:.*(denied|not permitted)" "$log"; then
    echo "[cmp] SKIP: environment does not permit binding sockets (see $log)"
    exit 0
  fi
}

new_udp_run_dir() {
  local template="$1" sip="$2" cip="$3" listen_port="$4" forward_port="$5"
  local run_dir
  run_dir="$(mktemp -d)"
  cp "$template" "$run_dir/config"
  "$PYTHON" "$ROOT/tests/support/tools/patch_reticulum_config_ports.py" \
    --path "$run_dir/config" \
    --shared-instance-port "$sip" \
    --instance-control-port "$cip" \
    --listen-port "$listen_port" \
    --forward-port "$forward_port"
  echo "$run_dir"
}

extract_listen_hash() {
  local path="$1"
  awk '/^LISTEN_HASH / { print $2; exit }' "$path"
}

run_mode() {
  local label="$1" mode="$2" payload="$3"
  local helper_cmd_a="$4" helper_cmd_b="$5"
  local cfg_flag_a="$6" cfg_flag_b="$7"
  local home_a="$8" home_b="$9"
  local node_a_dir="${10}" node_b_dir="${11}"

  local listener_out="$OUT_DIR/${label}.${mode}.listener.out"
  local listener_identity="$node_b_dir/${label}.${mode}.identity"
  local listener_cmd
  if [[ "$mode" == "group" ]]; then
    listener_cmd="env HOME=\"$home_b\" USERPROFILE=\"$home_b\" $helper_cmd_b $cfg_flag_b \"$node_b_dir\" --identity \"$listener_identity\" --listen --mode group --group-key \"$GROUP_KEY_HEX\" --payload \"$payload\" --wait-seconds \"$WAIT_SECS\""
  else
    listener_cmd="env HOME=\"$home_b\" USERPROFILE=\"$home_b\" $helper_cmd_b $cfg_flag_b \"$node_b_dir\" --listen --mode plain --payload \"$payload\" --wait-seconds \"$WAIT_SECS\""
  fi
  bash -lc "$listener_cmd" >"$listener_out" 2>&1 &
  local listener_pid=$!
  wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_out" "LISTEN_HASH " || { echo "[cmp] $label/$mode: listener not ready"; stop_proc "$listener_pid"; return 1; }
  local listen_hash
  listen_hash="$(extract_listen_hash "$listener_out")"

  local sender_out="$OUT_DIR/${label}.${mode}.sender.out"
  local sender_cmd
  if [[ "$mode" == "group" ]]; then
    sender_cmd="$helper_cmd_a $cfg_flag_a \"$node_a_dir\" --mode group --group-key \"$GROUP_KEY_HEX\" --destination \"$listen_hash\" --remote-identity \"$listener_identity\" --payload \"$payload\" --wait-seconds \"$WAIT_SECS\""
  else
    sender_cmd="$helper_cmd_a $cfg_flag_a \"$node_a_dir\" --mode plain --destination \"$listen_hash\" --payload \"$payload\" --wait-seconds \"$WAIT_SECS\""
  fi
  local code
  code="$(run_capture "$sender_out" bash -lc "env HOME=\"$home_a\" USERPROFILE=\"$home_a\" $sender_cmd")"
  if [[ "$code" != "0" ]]; then
    echo "[cmp] $label/$mode: sender failed; see $sender_out"
    stop_proc "$listener_pid"
    return 1
  fi

  wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_out" "EVENT received $payload" || { echo "[cmp] $label/$mode: payload not received; see $listener_out"; stop_proc "$listener_pid"; return 1; }
  stop_proc "$listener_pid"
  echo "[cmp] $label/$mode: ok"
}

run_pair() {
  local label="$1"
  local helper_cmd_a="$2" cfg_flag_a="$3"
  local helper_cmd_b="$4" cfg_flag_b="$5"

  echo
  echo "[cmp] $label: start two UDP nodes"
  local base=$(( (RANDOM % 10000) + 52000 ))
  base=$(( base / 2 * 2 ))
  local sip_a=$(( (RANDOM % 10000) + 38000 ))
  local cip_a=$(( sip_a + 1 ))
  local sip_b=$(( sip_a + 2 ))
  local cip_b=$(( sip_a + 3 ))

  local node_a_dir node_b_dir
  node_a_dir="$(new_udp_run_dir "$ROOT/configs/testing/two_nodes_udp/node_a/config" "$sip_a" "$cip_a" "$base" "$((base+1))")"
  node_b_dir="$(new_udp_run_dir "$ROOT/configs/testing/two_nodes_udp/node_b/config" "$sip_b" "$cip_b" "$((base+1))" "$base")"
  local home_a="$node_a_dir/home" home_b="$node_b_dir/home"
  mkdir -p "$home_a" "$home_b"

  local ready_a="$OUT_DIR/${label}.node_a.ready.log"
  local ready_b="$OUT_DIR/${label}.node_b.ready.log"
  : >"$ready_a"
  : >"$ready_b"
  maybe_skip_env "$ready_a"; maybe_skip_env "$ready_b"

  run_mode "$label" plain "${label}-plain" "$helper_cmd_a" "$helper_cmd_b" "$cfg_flag_a" "$cfg_flag_b" "$home_a" "$home_b" "$node_a_dir" "$node_b_dir" || { return 1; }
  run_mode "$label" group "${label}-group" "$helper_cmd_a" "$helper_cmd_b" "$cfg_flag_a" "$cfg_flag_b" "$home_a" "$home_b" "$node_a_dir" "$node_b_dir" || { return 1; }

  echo "[cmp] $label OK"
}

overall=0
if ! run_pair "go_node_a_python_node_b" \
  "$GO_BIN_DIR/go_packet_helper" "-config" \
  "$PYTHON $ROOT/tests/support/tools/py_packet_helper.py" "--config"; then
  overall=1
fi
if ! run_pair "python_node_a_go_node_b" \
  "$PYTHON $ROOT/tests/support/tools/py_packet_helper.py" "--config" \
  "$GO_BIN_DIR/go_packet_helper" "-config"; then
  overall=1
fi
if [[ "$overall" != "0" ]]; then
  echo "[cmp] FAIL"
  exit 1
fi
echo "[cmp] OK"
