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

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rncp_two_nodes_udp"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$GO_BIN_DIR" || true
}
trap cleanup EXIT

echo "[cmp] out=$OUT_DIR"
echo "[cmp] building go binaries..."
go build -o "$GO_BIN_DIR/rnsd" ./cmd/rnsd
go build -o "$GO_BIN_DIR/rnstatus" ./cmd/rnstatus
go build -o "$GO_BIN_DIR/rncp" ./cmd/rncp

run_capture() {
  local out="$1"
  shift
  local code=0
  set +e
  "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$CMD_TIMEOUT_SECS" -- "$@" >"$out" 2>&1
  code=$?
  set -e
  echo "$code"
}

run_capture_with_timeout() {
  local timeout="$1"
  local out="$2"
  shift 2
  local code=0
  set +e
  "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$timeout" -- "$@" >"$out" 2>&1
  code=$?
  set -e
  echo "$code"
}

wait_for_ok() {
  local timeout="$1"
  shift
  local start
  start="$(date +%s)"
  while true; do
    if "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout 2 -- "$@" >/dev/null 2>&1; then
      return 0
    fi
    local now
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.2
  done
}

stop_proc() {
  local pid="$1"
  if [[ -z "${pid}" ]]; then
    return 0
  fi
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    return 0
  fi
  kill -INT "$pid" >/dev/null 2>&1 || true
  if "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1; then
    return 0
  fi
  kill -TERM "$pid" >/dev/null 2>&1 || true
  if "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1; then
    return 0
  fi
  kill -KILL "$pid" >/dev/null 2>&1 || true
  "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1 || true
}

wait_for_file_contains() {
  local timeout="$1"
  local path="$2"
  local needle="$3"
  local start
  start="$(date +%s)"
  while true; do
    if [[ -f "$path" ]] && rg --fixed-strings "$needle" "$path" >/dev/null 2>&1; then
      return 0
    fi
    local now
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.2
  done
}

wait_for_file_nonempty() {
  local timeout="$1"
  local path="$2"
  local start
  start="$(date +%s)"
  while true; do
    if [[ -f "$path" ]] && [[ -s "$path" ]]; then
      return 0
    fi
    local now
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.2
  done
}

wait_for_path_absent() {
  local timeout="$1"
  local path="$2"
  local start
  start="$(date +%s)"
  while true; do
    if [[ ! -e "$path" ]]; then
      return 0
    fi
    local now
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.2
  done
}

sha256_file() {
  local path="$1"
  shasum -a 256 "$path" | awk '{print $1}'
}

new_udp_run_dir() {
  local template="$1"
  local sip="$2"
  local cip="$3"
  local listen_port="$4"
  local forward_port="$5"
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
  local line
  line="$(rg -o "rncp listening on <[0-9a-fA-F]+>" "$path" | head -n 1 || true)"
  if [[ -n "$line" ]]; then
    echo "$line" | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>'
    return 0
  fi
  rg -o "<[0-9a-fA-F]+>" "$path" | head -n 1 | tr -d '<>' || true
}

generate_python_identity() {
  local config_dir="$1"
  local home_dir="$2"
  local identity_path="$3"
  local out_path="$4"
  local code
  code="$(run_capture "$out_path" env HOME="$home_dir" USERPROFILE="$home_dir" \
    "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" --config "$config_dir" --generate "$identity_path" -q)"
  [[ "$code" == "0" ]]
}

maybe_skip_env() {
  local log="$1"
  if rg -q -i "operation not permitted|permission denied|bind:|listen .*:.*(denied|not permitted)" "$log"; then
    echo "[cmp] SKIP: environment does not permit binding sockets (see $log)"
    exit 0
  fi
}

run_pair() {
  local label="$1"
  local rnsd_cmd_a="$2"
  local rnstatus_cmd_a="$3"
  local rncp_cmd_a="$4"
  local cfg_flag_a="$5"
  local rnsd_cmd_b="$6"
  local rnstatus_cmd_b="$7"
  local rncp_cmd_b="$8"
  local cfg_flag_b="$9"

  echo
  echo "[cmp] $label: start two UDP nodes"

  local base
  base=$(( (RANDOM % 10000) + 52000 ))
  base=$(( base / 2 * 2 ))
  local sip_a cip_a sip_b cip_b
  sip_a=$(( (RANDOM % 10000) + 38000 ))
  cip_a=$(( sip_a + 1 ))
  sip_b=$(( sip_a + 2 ))
  cip_b=$(( sip_a + 3 ))

  local node_a_dir node_b_dir
  node_a_dir="$(new_udp_run_dir "$ROOT/configs/testing/two_nodes_udp/node_a/config" "$sip_a" "$cip_a" "$base" "$((base+1))")"
  node_b_dir="$(new_udp_run_dir "$ROOT/configs/testing/two_nodes_udp/node_b/config" "$sip_b" "$cip_b" "$((base+1))" "$base")"

  local home_a="$node_a_dir/home"
  local home_b="$node_b_dir/home"
  mkdir -p "$home_a" "$home_b"

  local listener_identity="$node_b_dir/node_b_python.id"
  local gen_identity_log="$OUT_DIR/${label}.listener_identity.generate.out"
  if ! generate_python_identity "$node_b_dir" "$home_b" "$listener_identity" "$gen_identity_log"; then
    echo "[cmp] $label: failed to generate listener identity via python; see $gen_identity_log"
    return 1
  fi

  local log_a="$OUT_DIR/${label}.rnsd.node_a.log"
  local log_b="$OUT_DIR/${label}.rnsd.node_b.log"
  env HOME="$home_a" USERPROFILE="$home_a" \
    $rnsd_cmd_a $cfg_flag_a "$node_a_dir" -q >"$log_a" 2>&1 &
  local pid_a=$!
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rnsd_cmd_b $cfg_flag_b "$node_b_dir" -q >"$log_b" 2>&1 &
  local pid_b=$!

  sleep 0.5
  maybe_skip_env "$log_a"
  maybe_skip_env "$log_b"

  local status_a="$OUT_DIR/${label}.rnstatus.node_a.out"
  local status_b="$OUT_DIR/${label}.rnstatus.node_b.out"
  if ! wait_for_ok "$START_TIMEOUT_SECS" bash -c "env HOME='$home_a' USERPROFILE='$home_a' $rnstatus_cmd_a $cfg_flag_a '$node_a_dir' -a >'$status_a' 2>&1"; then
    echo "[cmp] $label: node_a did not become ready; see $log_a and $status_a"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if ! wait_for_ok "$START_TIMEOUT_SECS" bash -c "env HOME='$home_b' USERPROFILE='$home_b' $rnstatus_cmd_b $cfg_flag_b '$node_b_dir' -a >'$status_b' 2>&1"; then
    echo "[cmp] $label: node_b did not become ready; see $log_b and $status_b"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  echo "[cmp] $label: rncp send over UDP (A -> B)"
  local recv_dir="$OUT_DIR/${label}.recv"
  mkdir -p "$recv_dir"
  local listener_log="$OUT_DIR/${label}.listener.log"

  env HOME="$home_b" USERPROFILE="$home_b" \
    $rncp_cmd_b $cfg_flag_b "$node_b_dir" -i "$listener_identity" -l -n -b 0 -S -s "$recv_dir" >"$listener_log" 2>&1 &
  local rncp_listener_pid=$!

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rncp listening on"; then
    echo "[cmp] $label: rncp listener did not become ready; see $listener_log"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local listen_hash
  listen_hash="$(extract_listen_hash "$listener_log")"
  if [[ -z "$listen_hash" ]]; then
    echo "[cmp] $label: could not parse listener hash; see $listener_log"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local send_src="$OUT_DIR/${label}.send.src"
  printf "udp-rncp-send-%s-%s\n" "$label" "$(date +%s%N)" >"$send_src"
  local send_sha
  send_sha="$(sha256_file "$send_src")"
  local send_out="$OUT_DIR/${label}.send.out"
  local send_code
  send_code="$(run_capture "$send_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd_a $cfg_flag_a "$node_a_dir" -S -w 30 "$send_src" "$listen_hash")"
  if [[ "$send_code" != "0" ]]; then
    echo "[cmp] $label: send failed; see $send_out"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local recv_file="$recv_dir/$(basename "$send_src")"
  if ! wait_for_file_nonempty "$START_TIMEOUT_SECS" "$recv_file"; then
    echo "[cmp] $label: did not receive file at $recv_file; see $send_out"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if [[ "$(sha256_file "$recv_file")" != "$send_sha" ]]; then
    echo "[cmp] $label: received file mismatch"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: send_ok=yes"

  echo "[cmp] $label: rncp fetch over UDP (A <- B)"
  stop_proc "$rncp_listener_pid"
  : >"$listener_log"

  local jail_dir="$OUT_DIR/${label}.jail"
  mkdir -p "$jail_dir"
  local fetch_name="fetch.txt"
  printf "udp-rncp-fetch-%s-%s\n" "$label" "$(date +%s%N)" >"$jail_dir/$fetch_name"
  local fetch_sha
  fetch_sha="$(sha256_file "$jail_dir/$fetch_name")"

  env HOME="$home_b" USERPROFILE="$home_b" \
    $rncp_cmd_b $cfg_flag_b "$node_b_dir" -i "$listener_identity" -l -n -F -b 0 -S -j "$jail_dir" >"$listener_log" 2>&1 &
  rncp_listener_pid=$!
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rncp listening on"; then
    echo "[cmp] $label: fetch listener did not become ready; see $listener_log"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  sleep 1
  listen_hash="$(extract_listen_hash "$listener_log")"
  local fetch_save="$OUT_DIR/${label}.fetch_out"
  mkdir -p "$fetch_save"
  local fetch_out="$OUT_DIR/${label}.fetch.out"
  local fetch_code
  fetch_code="$(run_capture "$fetch_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd_a $cfg_flag_a "$node_a_dir" -S -f -s "$fetch_save" -w 30 "$fetch_name" "$listen_hash")"
  if [[ "$fetch_code" != "0" ]]; then
    echo "[cmp] $label: fetch failed; see $fetch_out"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local fetched_file="$fetch_save/$fetch_name"
  if ! wait_for_file_nonempty "$START_TIMEOUT_SECS" "$fetched_file"; then
    echo "[cmp] $label: did not fetch file to $fetched_file; see $fetch_out"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if [[ "$(sha256_file "$fetched_file")" != "$fetch_sha" ]]; then
    echo "[cmp] $label: fetched file mismatch"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: fetch_ok=yes"

  echo "[cmp] $label: repeated fetch over UDP"
  local repeat_a="$OUT_DIR/${label}.repeat_a"
  local repeat_b="$OUT_DIR/${label}.repeat_b"
  mkdir -p "$repeat_a" "$repeat_b"
  local repeat_out_a="$OUT_DIR/${label}.repeat_a.out"
  local repeat_out_b="$OUT_DIR/${label}.repeat_b.out"
  local repeat_code
  repeat_code="$(run_capture "$repeat_out_a" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd_a $cfg_flag_a "$node_a_dir" -S -f -s "$repeat_a" -w 30 "$fetch_name" "$listen_hash")"
  if [[ "$repeat_code" != "0" ]]; then
    echo "[cmp] $label: repeated fetch #1 failed; see $repeat_out_a"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  repeat_code="$(run_capture "$repeat_out_b" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd_a $cfg_flag_a "$node_a_dir" -S -f -s "$repeat_b" -w 30 "$fetch_name" "$listen_hash")"
  if [[ "$repeat_code" != "0" ]]; then
    echo "[cmp] $label: repeated fetch #2 failed; see $repeat_out_b"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  [[ "$(sha256_file "$repeat_a/$fetch_name")" == "$fetch_sha" ]] || {
    echo "[cmp] $label: repeated fetch #1 hash mismatch"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  }
  [[ "$(sha256_file "$repeat_b/$fetch_name")" == "$fetch_sha" ]] || {
    echo "[cmp] $label: repeated fetch #2 hash mismatch"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  }
  echo "[cmp] $label: repeated_fetch_ok=yes"

  echo "[cmp] $label: overwrite fetch over UDP"
  local overwrite_dir="$OUT_DIR/${label}.overwrite"
  mkdir -p "$overwrite_dir"
  printf "old-content-%s\n" "$label" >"$overwrite_dir/$fetch_name"
  local overwrite_out="$OUT_DIR/${label}.overwrite.out"
  local overwrite_code
  overwrite_code="$(run_capture "$overwrite_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd_a $cfg_flag_a "$node_a_dir" -S -f -O -s "$overwrite_dir" -w 30 "$fetch_name" "$listen_hash")"
  if [[ "$overwrite_code" != "0" ]]; then
    echo "[cmp] $label: overwrite fetch failed; see $overwrite_out"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  [[ "$(sha256_file "$overwrite_dir/$fetch_name")" == "$fetch_sha" ]] || {
    echo "[cmp] $label: overwrite fetch did not replace file"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  }
  echo "[cmp] $label: overwrite_ok=yes"

  echo "[cmp] $label: missing-file fetch rejection"
  local missing_out="$OUT_DIR/${label}.missing.out"
  local missing_name="missing.txt"
  local missing_code
  missing_code="$(run_capture "$missing_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd_a $cfg_flag_a "$node_a_dir" -S -f -s "$fetch_save" -w 30 "$missing_name" "$listen_hash")"
  if ! rg -F "was not found on the remote" "$missing_out" >/dev/null 2>&1; then
    echo "[cmp] $label: missing-file fetch output mismatch; see $missing_out"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  wait_for_path_absent 1 "$fetch_save/$missing_name" || {
    echo "[cmp] $label: missing-file unexpectedly created local file"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  }
  echo "[cmp] $label: missing_ok=yes"

  echo "[cmp] $label: jail traversal rejection"
  local traversal_out="$OUT_DIR/${label}.traversal.out"
  local traversal_code
  traversal_code="$(run_capture_with_timeout 12 "$traversal_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd_a $cfg_flag_a "$node_a_dir" -S -f -s "$fetch_save" -w 5 "../nope" "$listen_hash")"
  wait_for_path_absent 1 "$fetch_save/nope" || {
    echo "[cmp] $label: traversal fetch unexpectedly created local file"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  }
  echo "[cmp] $label: jail_ok=yes"

  stop_proc "$rncp_listener_pid"
  : >"$listener_log"

  echo "[cmp] $label: fetch denied"
  local deny_save="$OUT_DIR/${label}.deny_save"
  mkdir -p "$deny_save"
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rncp_cmd_b $cfg_flag_b "$node_b_dir" -i "$listener_identity" -l -n -b 0 -S -s "$deny_save" >"$listener_log" 2>&1 &
  rncp_listener_pid=$!
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rncp listening on"; then
    echo "[cmp] $label: deny listener did not become ready; see $listener_log"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  listen_hash="$(extract_listen_hash "$listener_log")"
  local deny_out="$OUT_DIR/${label}.deny.out"
  local deny_fetch_save="$OUT_DIR/${label}.deny_fetch_out"
  rm -rf "$deny_fetch_save"
  mkdir -p "$deny_fetch_save"
  local deny_code
  deny_code="$(run_capture_with_timeout 15 "$deny_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd_a $cfg_flag_a "$node_a_dir" -S -f -s "$deny_fetch_save" -w 30 "$fetch_name" "$listen_hash")"
  if ! rg -F "was not allowed by the remote" "$deny_out" >/dev/null 2>&1 \
    && ! rg -F "unknown error" "$deny_out" >/dev/null 2>&1 \
    && [[ "$deny_code" != "124" ]]; then
    echo "[cmp] $label: denied fetch output mismatch; see $deny_out"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  wait_for_path_absent 1 "$deny_fetch_save/$fetch_name" || {
    echo "[cmp] $label: denied fetch unexpectedly created local file"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  }
  echo "[cmp] $label: deny_ok=yes"
  stop_proc "$rncp_listener_pid"
  : >"$listener_log"

  echo "[cmp] $label: large send over UDP"
  local large_recv_dir="$OUT_DIR/${label}.large_recv"
  mkdir -p "$large_recv_dir"
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rncp_cmd_b $cfg_flag_b "$node_b_dir" -v -v -v -v -i "$listener_identity" -l -n -b 0 -S -s "$large_recv_dir" >"$listener_log" 2>&1 &
  rncp_listener_pid=$!
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rncp listening on"; then
    echo "[cmp] $label: large-send listener did not become ready; see $listener_log"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  listen_hash="$(extract_listen_hash "$listener_log")"
  local large_src="$OUT_DIR/${label}.large.bin"
  python3 - <<'PY' >"$large_src"
import sys
data = (b"0123456789abcdef" * (64*1024))
sys.stdout.buffer.write(data)
PY
  local large_sha
  large_sha="$(sha256_file "$large_src")"
  local large_out="$OUT_DIR/${label}.large_send.out"
  local large_code
  large_code="$(run_capture "$large_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd_a $cfg_flag_a "$node_a_dir" -v -v -S -w 60 "$large_src" "$listen_hash")"
  if [[ "$large_code" != "0" ]]; then
    echo "[cmp] $label: large send failed; see $large_out"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  local large_recv="$large_recv_dir/$(basename "$large_src")"
  if ! wait_for_file_nonempty "$START_TIMEOUT_SECS" "$large_recv"; then
    echo "[cmp] $label: large send did not arrive; see $large_out"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  [[ "$(sha256_file "$large_recv")" == "$large_sha" ]] || {
    echo "[cmp] $label: large send hash mismatch"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  }
  echo "[cmp] $label: large_send_ok=yes"
  stop_proc "$rncp_listener_pid"

  stop_proc "$pid_a"
  stop_proc "$pid_b"
  echo "[cmp] $label OK"
}

overall=0

if ! run_pair "go_node_a_python_node_b" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/rncp" \
  "-config" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rncp.py" \
  "--config"; then
  overall=1
fi

if ! run_pair "python_node_a_go_node_b" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rncp.py" \
  "--config" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/rncp" \
  "-config"; then
  overall=1
fi

if [[ "$overall" -ne 0 ]]; then
  echo "[cmp] FAIL (see $OUT_DIR)"
  exit 1
fi

echo "[cmp] OK"
