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
WAIT_SECS="${WAIT_SECS:-60}"
SMALL_PAYLOAD="${SMALL_PAYLOAD:-resource-small-payload}"
LARGE_SIZE="${LARGE_SIZE:-24576}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_resource_two_nodes_udp"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$GO_BIN_DIR" || true
  rm -rf "$GOTMPDIR" || true
}
trap cleanup EXIT

echo "[cmp] out=$OUT_DIR"
echo "[cmp] building go binaries..."
go build -o "$GO_BIN_DIR/go_resource_helper" ./tests/support/tools/go_resource_helper

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

  local listener_out="$OUT_DIR/${label}.listener.out"
  bash -lc "env HOME=\"$home_b\" USERPROFILE=\"$home_b\" $helper_cmd_b $cfg_flag_b \"$node_b_dir\" --listen --small-payload \"$SMALL_PAYLOAD\" --large-size \"$LARGE_SIZE\" --wait-seconds \"$WAIT_SECS\"" >"$listener_out" 2>&1 &
  local listener_pid=$!
  wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_out" "LISTEN_HASH " || { echo "[cmp] $label: listener not ready"; stop_proc "$listener_pid"; return 1; }
  local listen_hash
  listen_hash="$(extract_listen_hash "$listener_out")"

  local sender_out="$OUT_DIR/${label}.sender.out"
  local code
  code="$(run_capture "$sender_out" bash -lc "env HOME=\"$home_a\" USERPROFILE=\"$home_a\" $helper_cmd_a $cfg_flag_a \"$node_a_dir\" --destination \"$listen_hash\" --small-payload \"$SMALL_PAYLOAD\" --large-size \"$LARGE_SIZE\" --wait-seconds \"$WAIT_SECS\"")"
  if [[ "$code" != "0" ]]; then
    echo "[cmp] $label: sender failed; see $sender_out"
    stop_proc "$listener_pid"
    return 1
  fi

  wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_out" "EVENT concluded kind=small" || { echo "[cmp] $label: small resource missing"; stop_proc "$listener_pid"; return 1; }
  wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_out" "EVENT concluded kind=large" || { echo "[cmp] $label: large resource missing"; stop_proc "$listener_pid"; return 1; }
  rg --fixed-strings "EVENT sent kind=small" "$sender_out" >/dev/null 2>&1 || { echo "[cmp] $label: small send missing"; stop_proc "$listener_pid"; return 1; }
  rg --fixed-strings "EVENT sent kind=large" "$sender_out" >/dev/null 2>&1 || { echo "[cmp] $label: large send missing"; stop_proc "$listener_pid"; return 1; }

  stop_proc "$listener_pid"
  echo "[cmp] $label OK"
}

overall=0
if ! run_pair "go_node_a_python_node_b" \
  "$GO_BIN_DIR/go_resource_helper" "-config" \
  "$PYTHON $ROOT/tests/support/tools/py_resource_helper.py" "--config"; then
  overall=1
fi
if ! run_pair "python_node_a_go_node_b" \
  "$PYTHON $ROOT/tests/support/tools/py_resource_helper.py" "--config" \
  "$GO_BIN_DIR/go_resource_helper" "-config"; then
  overall=1
fi
if [[ "$overall" != "0" ]]; then
  echo "[cmp] FAIL"
  exit 1
fi
echo "[cmp] OK"
