#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PYTHON="${PYTHON:-python3}"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache" "$ROOT/tests/artifacts/logs"
export GOCACHE="$ROOT/.gocache"
export GOTMPDIR="$ROOT/.gotmp"
export GOPATH="$ROOT/.gopath"
export GOMODCACHE="$ROOT/.gomodcache"

export PYTHONPATH="${PYTHONPATH:-"$ROOT/python"}"
export PYTHONUNBUFFERED=1

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-90}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-30}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-6}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rncp_two_nodes"
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
  "$PYTHON" "$ROOT/tests/tools/timeout_exec.py" --timeout "$CMD_TIMEOUT_SECS" -- "$@" >"$out" 2>&1
  code=$?
  set -e
  echo "$code"
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
  if "$PYTHON" "$ROOT/tests/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1; then
    return 0
  fi

  kill -TERM "$pid" >/dev/null 2>&1 || true
  if "$PYTHON" "$ROOT/tests/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1; then
    return 0
  fi

  kill -KILL "$pid" >/dev/null 2>&1 || true
  "$PYTHON" "$ROOT/tests/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1 || true
  return 0
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

new_run_dir_from_template() {
  local template="$1"
  local sip="$2"
  local cip="$3"
  local listen="$4"
  local forward="$5"

  local run_dir
  run_dir="$(mktemp -d)"
  cp "$template" "$run_dir/config"
  "$PYTHON" "$ROOT/tests/tools/patch_reticulum_config_ports.py" \
    --path "$run_dir/config" \
    --shared-instance-port "$sip" \
    --instance-control-port "$cip" \
    --listen-port "$listen" \
    --forward-port "$forward"
  echo "$run_dir"
}

extract_listen_hash() {
  local path="$1"
  # Prefer the explicit rncp line:
  # "rncp listening on <...>"
  local line
  line="$(rg -o "rncp listening on <[0-9a-fA-F]+>" "$path" | head -n 1 || true)"
  if [[ -n "$line" ]]; then
    echo "$line" | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>'
    return 0
  fi

  # Fallback: first hash-like token.
  rg -o "<[0-9a-fA-F]+>" "$path" | head -n 1 | tr -d '<>' || true
}

json_escape() {
  "$PYTHON" -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$1"
}

maybe_skip_env() {
  local log="$1"
  if rg -q -i "operation not permitted|permission denied|bind:|listen .*:.*(denied|not permitted)" "$log"; then
    echo "[cmp] SKIP: environment does not permit binding sockets (see $log)"
    exit 0
  fi
}

run_pair() {
  local label="$1" # python|go
  local rnsd_cmd="$2"
  local rnstatus_cmd="$3"
  local rncp_cmd="$4"

  local cfg_flag
  cfg_flag="-config"
  if [[ "$label" == "python" ]]; then
    cfg_flag="--config"
  fi

  echo
  echo "[cmp] $label: start two rnsd nodes"

  local base
  base=$(( (RANDOM % 10000) + 52000 ))
  base=$(( base / 2 * 2 ))

  local node_a_template="$ROOT/configs/testing/two_nodes_udp/node_a/config"
  local node_b_template="$ROOT/configs/testing/two_nodes_udp/node_b/config"

  local sip_a cip_a sip_b cip_b
  sip_a=$(( (RANDOM % 10000) + 38000 ))
  cip_a=$(( sip_a + 1 ))
  sip_b=$(( sip_a + 2 ))
  cip_b=$(( sip_a + 3 ))

  local node_a_dir node_b_dir
  node_a_dir="$(new_run_dir_from_template "$node_a_template" "$sip_a" "$cip_a" "$base" "$((base+1))")"
  node_b_dir="$(new_run_dir_from_template "$node_b_template" "$sip_b" "$cip_b" "$((base+1))" "$base")"

  local home_a home_b
  home_a="$node_a_dir/home"
  home_b="$node_b_dir/home"
  mkdir -p "$home_a" "$home_b"

  local log_a="$OUT_DIR/${label}.rnsd.node_a.log"
  local log_b="$OUT_DIR/${label}.rnsd.node_b.log"

  env HOME="$home_a" USERPROFILE="$home_a" \
    $rnsd_cmd $cfg_flag "$node_a_dir" -q >"$log_a" 2>&1 &
  local pid_a=$!

  env HOME="$home_b" USERPROFILE="$home_b" \
    $rnsd_cmd $cfg_flag "$node_b_dir" -q >"$log_b" 2>&1 &
  local pid_b=$!

  sleep 0.5
  maybe_skip_env "$log_a"
  maybe_skip_env "$log_b"

  local status_a="$OUT_DIR/${label}.rnstatus.node_a.out"
  local status_b="$OUT_DIR/${label}.rnstatus.node_b.out"
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$log_a" "Started rnsd version"; then
    echo "[cmp] $label: node_a did not start; see $log_a"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$log_b" "Started rnsd version"; then
    echo "[cmp] $label: node_b did not start; see $log_b"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  # Ensure shared instance is reachable for clients.
  _="$(run_capture "$status_a" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnstatus_cmd $cfg_flag "$node_a_dir" -a)"
  _="$(run_capture "$status_b" env HOME="$home_b" USERPROFILE="$home_b" \
    $rnstatus_cmd $cfg_flag "$node_b_dir" -a)"

  echo "[cmp] $label: rncp send (A -> B)"
  local send_src="$OUT_DIR/${label}.send.src"
  printf "rncp-send-v1-%s-%s\n" "$label" "$(date +%s%N)" >"$send_src"
  local send_src_v1_sha
  send_src_v1_sha="$(sha256_file "$send_src")"

  local recv_dir="$OUT_DIR/${label}.recv"
  mkdir -p "$recv_dir"
  local listener_log="$OUT_DIR/${label}.listener.log"

  # Start listener in background (no-auth, announce once at startup, silent output, save dir).
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rncp_cmd $cfg_flag "$node_b_dir" -l -n -b 0 -S -s "$recv_dir" >"$listener_log" 2>&1 &
  local rncp_listener_pid=$!

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rncp listening on"; then
    echo "[cmp] $label: rncp listener did not become ready; see $listener_log"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local listen_hash
  listen_hash="$(extract_listen_hash "$listener_log")"
  if [[ -z "$listen_hash" ]]; then
    echo "[cmp] $label: could not parse listener destination hash; see $listener_log"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: listener_hash=$listen_hash"

  local send_out="$OUT_DIR/${label}.send.out"
  local send_code
  send_code="$(run_capture "$send_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd $cfg_flag "$node_a_dir" -S -w 30 "$send_src" "$listen_hash")"

  if [[ "$send_code" != "0" ]]; then
    echo "[cmp] $label: send failed; see $send_out"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local recv_file="$recv_dir/$(basename "$send_src")"
  if ! wait_for_file_nonempty "$START_TIMEOUT_SECS" "$recv_file"; then
    echo "[cmp] $label: did not receive file at $recv_file; listener_log=$listener_log send_out=$send_out"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local recv_sha
  recv_sha="$(sha256_file "$recv_file")"
  local send_ok="no"
  if [[ "$recv_sha" == "$send_src_v1_sha" ]]; then
    send_ok="yes"
  fi

  echo "[cmp] $label: send_ok=$send_ok"

  echo "[cmp] $label: rncp send duplicate (no overwrite)"
  printf "rncp-send-v2-%s-%s\n" "$label" "$(date +%s%N)" >"$send_src"
  local send_src_v2_sha
  send_src_v2_sha="$(sha256_file "$send_src")"

  local send2_out="$OUT_DIR/${label}.send2.out"
  local send2_code
  send2_code="$(run_capture "$send2_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd $cfg_flag "$node_a_dir" -S -w 30 "$send_src" "$listen_hash")"

  if [[ "$send2_code" != "0" ]]; then
    echo "[cmp] $label: send2 failed; see $send2_out"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local recv_file_2="$recv_dir/$(basename "$send_src").1"
  if ! wait_for_file_nonempty "$START_TIMEOUT_SECS" "$recv_file_2"; then
    echo "[cmp] $label: did not receive duplicate file at $recv_file_2; listener_log=$listener_log send2_out=$send2_out"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local recv2_sha
  recv2_sha="$(sha256_file "$recv_file_2")"
  local send_dup_ok="no"
  if [[ "$recv2_sha" == "$send_src_v2_sha" ]]; then
    send_dup_ok="yes"
  fi
  echo "[cmp] $label: send_dup_ok=$send_dup_ok"

  echo "[cmp] $label: rncp send duplicate (overwrite enabled)"
  local recv_dir_ovr="$OUT_DIR/${label}.recv_overwrite"
  mkdir -p "$recv_dir_ovr"
  stop_proc "$rncp_listener_pid"
  : >"$listener_log"

  env HOME="$home_b" USERPROFILE="$home_b" \
    $rncp_cmd $cfg_flag "$node_b_dir" -l -n -O -b 0 -S -s "$recv_dir_ovr" >"$listener_log" 2>&1 &
  rncp_listener_pid=$!

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rncp listening on"; then
    echo "[cmp] $label: rncp overwrite-listener did not become ready; see $listener_log"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi
  listen_hash="$(extract_listen_hash "$listener_log")"
  if [[ -z "$listen_hash" ]]; then
    echo "[cmp] $label: could not parse listener destination hash (overwrite listener); see $listener_log"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  printf "rncp-ovr-v1-%s-%s\n" "$label" "$(date +%s%N)" >"$send_src"
  local send_ovr_v1_sha
  send_ovr_v1_sha="$(sha256_file "$send_src")"
  local send_ovr1_out="$OUT_DIR/${label}.send_ovr1.out"
  local send_ovr1_code
  send_ovr1_code="$(run_capture "$send_ovr1_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd $cfg_flag "$node_a_dir" -S -w 30 "$send_src" "$listen_hash")"
  if [[ "$send_ovr1_code" != "0" ]]; then
    echo "[cmp] $label: send_ovr1 failed; see $send_ovr1_out"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  printf "rncp-ovr-v2-%s-%s\n" "$label" "$(date +%s%N)" >"$send_src"
  local send_ovr_v2_sha
  send_ovr_v2_sha="$(sha256_file "$send_src")"
  local send_ovr2_out="$OUT_DIR/${label}.send_ovr2.out"
  local send_ovr2_code
  send_ovr2_code="$(run_capture "$send_ovr2_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd $cfg_flag "$node_a_dir" -S -w 30 "$send_src" "$listen_hash")"
  if [[ "$send_ovr2_code" != "0" ]]; then
    echo "[cmp] $label: send_ovr2 failed; see $send_ovr2_out"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local ovr_recv_file="$recv_dir_ovr/$(basename "$send_src")"
  if ! wait_for_file_nonempty "$START_TIMEOUT_SECS" "$ovr_recv_file"; then
    echo "[cmp] $label: did not receive overwrite file at $ovr_recv_file; listener_log=$listener_log"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local ovr_recv_sha
  ovr_recv_sha="$(sha256_file "$ovr_recv_file")"
  local send_overwrite_ok="no"
  if [[ "$ovr_recv_sha" == "$send_ovr_v2_sha" ]] && [[ ! -f "$ovr_recv_file.1" ]]; then
    send_overwrite_ok="yes"
  fi
  echo "[cmp] $label: send_overwrite_ok=$send_overwrite_ok"

  echo "[cmp] $label: rncp fetch (A <- B)"
  local jail_dir="$OUT_DIR/${label}.jail"
  mkdir -p "$jail_dir"
  local fetch_name="fetch.txt"
  printf "rncp-fetch-%s-%s\n" "$label" "$(date +%s%N)" >"$jail_dir/$fetch_name"
  local fetch_src_sha
  fetch_src_sha="$(sha256_file "$jail_dir/$fetch_name")"

  # Restart listener with fetch enabled + jail.
  stop_proc "$rncp_listener_pid"
  : >"$listener_log"

  env HOME="$home_b" USERPROFILE="$home_b" \
    $rncp_cmd $cfg_flag "$node_b_dir" -l -n -F -b 0 -S -j "$jail_dir" >"$listener_log" 2>&1 &
  rncp_listener_pid=$!

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rncp listening on"; then
    echo "[cmp] $label: rncp fetch-listener did not become ready; see $listener_log"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi
  listen_hash="$(extract_listen_hash "$listener_log")"
  if [[ -z "$listen_hash" ]]; then
    echo "[cmp] $label: could not parse listener destination hash (fetch listener); see $listener_log"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: fetch_listener_hash=$listen_hash"

  local fetch_save="$OUT_DIR/${label}.fetch_out"
  mkdir -p "$fetch_save"
  local fetch_out="$OUT_DIR/${label}.fetch.out"
  local fetch_code
  fetch_code="$(run_capture "$fetch_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd $cfg_flag "$node_a_dir" -S -f -s "$fetch_save" -w 30 "$fetch_name" "$listen_hash")"

  if [[ "$fetch_code" != "0" ]]; then
    echo "[cmp] $label: fetch failed; see $fetch_out"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local fetched_file="$fetch_save/$fetch_name"
  if ! wait_for_file_nonempty "$START_TIMEOUT_SECS" "$fetched_file"; then
    echo "[cmp] $label: did not fetch file to $fetched_file; fetch_out=$fetch_out"
    stop_proc "$rncp_listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local fetched_sha
  fetched_sha="$(sha256_file "$fetched_file")"
  local fetch_ok="no"
  if [[ "$fetched_sha" == "$fetch_src_sha" ]]; then
    fetch_ok="yes"
  fi

  echo "[cmp] $label: fetch_ok=$fetch_ok"

  stop_proc "$rncp_listener_pid"
  stop_proc "$pid_a"
  stop_proc "$pid_b"

  {
    echo "send_exit=$send_code"
    echo "send_ok=$send_ok"
    echo "send_dup_exit=$send2_code"
    echo "send_dup_ok=$send_dup_ok"
    echo "send_overwrite_ok=$send_overwrite_ok"
    echo "fetch_exit=$fetch_code"
    echo "fetch_ok=$fetch_ok"
  } >"$OUT_DIR/${label}.summary.txt"

  if [[ "$send_ok" != "yes" ]] || [[ "$send_dup_ok" != "yes" ]] || [[ "$send_overwrite_ok" != "yes" ]] || [[ "$fetch_ok" != "yes" ]]; then
    echo "[cmp] $label: content mismatch; see $OUT_DIR/${label}.summary.txt"
    return 1
  fi

  echo "[cmp] $label OK"
  return 0
}

overall=0

if ! run_pair "python" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rncp.py"; then
  overall=1
fi

if ! run_pair "go" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/rncp"; then
  overall=1
fi

if [[ "$overall" -ne 0 ]]; then
  echo "[cmp] FAIL (see $OUT_DIR)"
  exit 1
fi

if diff -u "$OUT_DIR/python.summary.txt" "$OUT_DIR/go.summary.txt" >"$OUT_DIR/summary.diff"; then
  echo "[cmp] summary parity OK"
  rm -f "$OUT_DIR/summary.diff" || true
else
  echo "[cmp] summary parity DIFF: $OUT_DIR/summary.diff"
  overall=1
fi

if [[ "$overall" -eq 0 ]]; then
  echo "[cmp] OK"
  exit 0
fi

echo "[cmp] FAIL (see $OUT_DIR)"
exit 1
