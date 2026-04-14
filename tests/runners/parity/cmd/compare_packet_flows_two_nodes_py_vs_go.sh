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

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-120}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-30}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-6}"
PAIR_START_ATTEMPTS="${PAIR_START_ATTEMPTS:-4}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_packet_flows_two_nodes"
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
go build -o "$GO_BIN_DIR/rnid" ./cmd/rnid
go build -o "$GO_BIN_DIR/rnpath" ./cmd/rnpath
go build -o "$GO_BIN_DIR/rnprobe" ./cmd/rnprobe
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

run_capture_retry() {
  local out="$1"
  local attempts="$2"
  local delay="$3"
  shift 3
  local code=0
  local i=1
  while [[ "$i" -le "$attempts" ]]; do
    code="$(run_capture "$out" "$@")"
    if [[ "$code" == "0" ]]; then
      echo "$code"
      return 0
    fi
    if [[ "$i" -lt "$attempts" ]]; then
      sleep "$delay"
    fi
    i=$((i+1))
  done
  echo "$code"
  return 0
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
  "$PYTHON" "$ROOT/tests/support/tools/patch_reticulum_config_ports.py" \
    --path "$run_dir/config" \
    --shared-instance-port "$sip" \
    --instance-control-port "$cip" \
    --listen-port "$listen" \
    --forward-port "$forward"
  echo "$run_dir"
}

extract_dest_hash() {
  local path="$1"
  rg "destination for this Identity is" "$path" | sed -E 's/^.*<([0-9a-fA-F]+)>.*/\1/' | head -n 1
}

extract_probe_hash() {
  local path="$1"
  local from_status
  from_status="$(rg -o "Probe responder at <[0-9a-fA-F]+> active" "$path" | head -n 1 || true)"
  if [[ -n "$from_status" ]]; then
    echo "$from_status" | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>'
    return 0
  fi
  local from_notice
  from_notice="$(rg -o "respond to probe requests on <[^>]*:<[0-9a-fA-F]+>>" "$path" | head -n 1 || true)"
  if [[ -n "$from_notice" ]]; then
    echo "$from_notice" | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>'
  fi
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

startup_had_addr_in_use() {
  local log="$1"
  [[ -f "$log" ]] && rg -q -i "address already in use|errno 48|bind: address already in use" "$log"
}

run_pair() {
  local label="$1"
  local rnsd_cmd_a="$2"
  local rnstatus_cmd_a="$3"
  local rnid_cmd_a="$4"
  local rnpath_cmd_a="$5"
  local rnprobe_cmd_a="$6"
  local rncp_cmd_a="$7"
  local cfg_flag_a="$8"
  local rnsd_cmd_b="$9"
  local rnstatus_cmd_b="${10}"
  local rnid_cmd_b="${11}"
  local rncp_cmd_b="${12}"
  local cfg_flag_b="${13}"

  echo
  echo "[cmp] $label: start packet-flow nodes"

  local node_a_dir node_b_dir home_a home_b remote_identity gen_identity_log
  local log_a log_b pid_a pid_b
  local start_attempt=1
  while [[ "$start_attempt" -le "$PAIR_START_ATTEMPTS" ]]; do
    local base sip_a cip_a sip_b cip_b
    base=$(( (RANDOM % 10000) + 52000 ))
    base=$(( base / 2 * 2 ))
    sip_a=$(( (RANDOM % 10000) + 38000 ))
    cip_a=$(( sip_a + 1 ))
    sip_b=$(( sip_a + 2 ))
    cip_b=$(( sip_a + 3 ))

    node_a_dir="$(new_run_dir_from_template "$ROOT/configs/testing/two_nodes_udp/node_a/config" "$sip_a" "$cip_a" "$base" "$((base+1))")"
    node_b_dir="$(new_run_dir_from_template "$ROOT/configs/testing/two_nodes_udp/node_b/config" "$sip_b" "$cip_b" "$((base+1))" "$base")"

    home_a="$node_a_dir/home"
    home_b="$node_b_dir/home"
    mkdir -p "$home_a" "$home_b"

    remote_identity="$node_b_dir/remote_python.id"
    gen_identity_log="$OUT_DIR/${label}.identity.generate.out"
    : >"$gen_identity_log"
    if ! generate_python_identity "$node_b_dir" "$home_b" "$remote_identity" "$gen_identity_log"; then
      if startup_had_addr_in_use "$gen_identity_log" && [[ "$start_attempt" -lt "$PAIR_START_ATTEMPTS" ]]; then
        echo "[cmp] $label: identity generation port collision, retrying ($start_attempt/$PAIR_START_ATTEMPTS)"
        start_attempt=$((start_attempt+1))
        continue
      fi
      echo "[cmp] $label: failed to generate remote identity; see $gen_identity_log"
      return 1
    fi

    log_a="$OUT_DIR/${label}.rnsd.node_a.log"
    log_b="$OUT_DIR/${label}.rnsd.node_b.log"
    : >"$log_a"
    : >"$log_b"
    env HOME="$home_a" USERPROFILE="$home_a" \
      $rnsd_cmd_a $cfg_flag_a "$node_a_dir" -q >"$log_a" 2>&1 &
    pid_a=$!
    env HOME="$home_b" USERPROFILE="$home_b" \
      $rnsd_cmd_b $cfg_flag_b "$node_b_dir" -q >"$log_b" 2>&1 &
    pid_b=$!

    sleep 0.5
    maybe_skip_env "$log_a"
    maybe_skip_env "$log_b"
    if wait_for_file_contains "$START_TIMEOUT_SECS" "$log_a" "Started rnsd version" \
      && wait_for_file_contains "$START_TIMEOUT_SECS" "$log_b" "Started rnsd version"; then
      break
    fi

    if (startup_had_addr_in_use "$log_a" || startup_had_addr_in_use "$log_b") && [[ "$start_attempt" -lt "$PAIR_START_ATTEMPTS" ]]; then
      stop_proc "$pid_a"
      stop_proc "$pid_b"
      echo "[cmp] $label: startup port collision, retrying ($start_attempt/$PAIR_START_ATTEMPTS)"
      start_attempt=$((start_attempt+1))
      continue
    fi

    if ! wait_for_file_contains 0 "$log_a" "Started rnsd version"; then
      echo "[cmp] $label: node_a did not start; see $log_a"
    else
      echo "[cmp] $label: node_b did not start; see $log_b"
    fi
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  done

  if ! wait_for_file_contains 0 "$log_b" "Started rnsd version"; then
    echo "[cmp] $label: node_b did not start; see $log_b"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local status_a="$OUT_DIR/${label}.rnstatus.node_a.out"
  local status_b="$OUT_DIR/${label}.rnstatus.node_b.out"
  _="$(run_capture "$status_a" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnstatus_cmd_a $cfg_flag_a "$node_a_dir" -a)"
  _="$(run_capture "$status_b" env HOME="$home_b" USERPROFILE="$home_b" \
    $rnstatus_cmd_b $cfg_flag_b "$node_b_dir" -a)"

  echo "[cmp] $label: link/probe"
  local probe_hash
  probe_hash="$(extract_probe_hash "$status_b")"
  if [[ -z "$probe_hash" ]]; then
    echo "[cmp] $label: could not parse probe hash; see $status_b"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  local probe_out="$OUT_DIR/${label}.probe.out"
  local probe_code
  probe_code="$(run_capture "$probe_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnprobe_cmd_a $cfg_flag_a "$node_a_dir" -n 1 -t 15 -w 0.2 rnstransport.probe "$probe_hash")"
  if [[ "$probe_code" != "0" ]] || ! rg -q "Valid reply from" "$probe_out"; then
    echo "[cmp] $label: probe failed; see $probe_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: probe_ok=yes"

  echo "[cmp] $label: announce + path"
  local hash_out="$OUT_DIR/${label}.hash.out"
  local hash_code
  hash_code="$(run_capture "$hash_out" env HOME="$home_b" USERPROFILE="$home_b" \
    $rnid_cmd_b $cfg_flag_b "$node_b_dir" -i "$remote_identity" -H app.aspect)"
  if [[ "$hash_code" != "0" ]]; then
    echo "[cmp] $label: hash failed; see $hash_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  local dest_hash
  dest_hash="$(extract_dest_hash "$hash_out")"
  if [[ -z "$dest_hash" ]]; then
    echo "[cmp] $label: could not parse destination hash; see $hash_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local announce_out="$OUT_DIR/${label}.announce.out"
  local announce_code
  announce_code="$(run_capture_retry "$announce_out" 3 1 env HOME="$home_b" USERPROFILE="$home_b" \
    $rnid_cmd_b $cfg_flag_b "$node_b_dir" -i "$remote_identity" -a app.aspect -q)"
  if [[ "$announce_code" != "0" ]]; then
    echo "[cmp] $label: announce failed; see $announce_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local path_out="$OUT_DIR/${label}.path.out"
  local path_code
  path_code="$(run_capture "$path_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnpath_cmd_a $cfg_flag_a "$node_a_dir" -w 15 "$dest_hash")"
  if [[ "$path_code" != "0" ]] || ! rg -q "Path found, destination" "$path_out"; then
    echo "[cmp] $label: path discovery failed; see $path_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: announce_ok=yes path_ok=yes"

  echo "[cmp] $label: identity request/response"
  local request_out="$OUT_DIR/${label}.request.out"
  local request_code
  request_code="$(run_capture "$request_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnid_cmd_a $cfg_flag_a "$node_a_dir" -i "$dest_hash" -R -t 15 -p -q)"
  if [[ "$request_code" != "0" ]] || ! rg -q "(Identity[[:space:]]*:|Public Key[[:space:]]*:)" "$request_out"; then
    echo "[cmp] $label: request failed; see $request_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: request_ok=yes"

  echo "[cmp] $label: resource transfer"
  local recv_dir="$OUT_DIR/${label}.recv"
  mkdir -p "$recv_dir"
  local listener_log="$OUT_DIR/${label}.listener.log"
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rncp_cmd_b $cfg_flag_b "$node_b_dir" -i "$remote_identity" -l -n -b 0 -S -s "$recv_dir" >"$listener_log" 2>&1 &
  local rncp_listener_pid=$!
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rncp listening on"; then
    echo "[cmp] $label: rncp listener did not become ready; see $listener_log"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  local listen_hash
  listen_hash="$(extract_listen_hash "$listener_log")"
  local send_src="$OUT_DIR/${label}.send.src"
  printf "packet-flow-send-%s-%s\n" "$label" "$(date +%s%N)" >"$send_src"
  local send_sha
  send_sha="$(sha256_file "$send_src")"
  local send_out="$OUT_DIR/${label}.send.out"
  local send_code
  send_code="$(run_capture "$send_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rncp_cmd_a $cfg_flag_a "$node_a_dir" -S -w 30 "$send_src" "$listen_hash")"
  if [[ "$send_code" != "0" ]]; then
    echo "[cmp] $label: resource send failed; see $send_out"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  local recv_file="$recv_dir/$(basename "$send_src")"
  if ! wait_for_file_nonempty "$START_TIMEOUT_SECS" "$recv_file" || [[ "$(sha256_file "$recv_file")" != "$send_sha" ]]; then
    echo "[cmp] $label: resource receive mismatch; see $send_out"
    stop_proc "$rncp_listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: resource_ok=yes"

  stop_proc "$rncp_listener_pid"
  stop_proc "$pid_a"
  stop_proc "$pid_b"
  echo "[cmp] $label OK"
}

overall=0

if ! run_pair "go_node_a_python_node_b" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/rnid" \
  "$GO_BIN_DIR/rnpath" \
  "$GO_BIN_DIR/rnprobe" \
  "$GO_BIN_DIR/rncp" \
  "-config" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnid.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rncp.py" \
  "--config"; then
  overall=1
fi

if ! run_pair "python_node_a_go_node_b" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnid.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnpath.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnprobe.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rncp.py" \
  "--config" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/rnid" \
  "$GO_BIN_DIR/rncp" \
  "-config"; then
  overall=1
fi

if [[ "$overall" -ne 0 ]]; then
  echo "[cmp] FAIL (see $OUT_DIR)"
  exit 1
fi

echo "[cmp] OK"
exit 0
