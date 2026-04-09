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

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnid_two_nodes_tcp"
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
go build -o "$GO_BIN_DIR/rnid" ./cmd/rnid

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

wait_for_tcp_status() {
  local timeout="$1"
  local out="$2"
  local iface_name="$3"
  shift 3
  local start
  start="$(date +%s)"
  while true; do
    local code
    code="$(run_capture "$out" "$@")"
    if [[ "$code" == "0" ]] && awk -v iface="$iface_name" '
      index($0, iface) { in_block=1; next }
      in_block && /^[[:space:]]*$/ { in_block=0 }
      in_block && /Status[[:space:]]*:[[:space:]]*Up/ { found=1; exit }
      END { exit(found ? 0 : 1) }
    ' "$out"; then
      return 0
    fi
    local now
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.5
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

append_tcp_bootstraps() {
  local config_path="$1"
  cat >>"$config_path" <<'EOF'

  [[Bootstrap TCP 21223388164]]
    type = TCPClientInterface
    enabled = yes
    target_host = 212.233.88.164
    target_port = 4242
    connect_timeout = 3
    reconnect_wait = 2
    max_reconnect_tries = 0
    bootstrap_only = yes

  [[Bootstrap TCP Dublin]]
    type = TCPClientInterface
    enabled = yes
    target_host = dublin.connect.reticulum.network
    target_port = 4965
    connect_timeout = 3
    reconnect_wait = 2
    max_reconnect_tries = 0
    bootstrap_only = yes
EOF
}

new_tcp_run_dir() {
  local template="$1"
  local sip="$2"
  local cip="$3"
  local listen_port="$4"
  local target_port="$5"
  local mode="${6:-plain}"

  local run_dir
  run_dir="$(mktemp -d)"
  cp "$template" "$run_dir/config"
  "$PYTHON" "$ROOT/tests/support/tools/patch_reticulum_config_ports.py" \
    --path "$run_dir/config" \
    --shared-instance-port "$sip" \
    --instance-control-port "$cip" \
    --listen-port "$listen_port" \
    --target-port "$target_port"
  if [[ "$mode" == "with_bootstraps" ]]; then
    append_tcp_bootstraps "$run_dir/config"
  fi
  echo "$run_dir"
}

extract_dest_hash() {
  local path="$1"
  rg "destination for this Identity is" "$path" \
    | sed -E 's/^.*<([0-9a-fA-F]+)>.*/\1/' \
    | head -n 1
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

wait_for_any_bootstrap_up() {
  local timeout="$1"
  local out_a="$2"
  local out_b="$3"
  shift 3
  local cmd_a="$1"
  local cmd_b="$2"
  local start
  start="$(date +%s)"
  while true; do
    _="$(eval "$cmd_a" || true)"
    _="$(eval "$cmd_b" || true)"
    if rg -F "Bootstrap TCP 21223388164" "$out_a" >/dev/null 2>&1 && rg -F "Status    : Up" "$out_a" >/dev/null 2>&1; then
      return 0
    fi
    if rg -F "Bootstrap TCP 21223388164" "$out_b" >/dev/null 2>&1 && rg -F "Status    : Up" "$out_b" >/dev/null 2>&1; then
      return 0
    fi
    if rg -F "Bootstrap TCP Dublin" "$out_a" >/dev/null 2>&1 && rg -F "Status    : Up" "$out_a" >/dev/null 2>&1; then
      return 0
    fi
    if rg -F "Bootstrap TCP Dublin" "$out_b" >/dev/null 2>&1 && rg -F "Status    : Up" "$out_b" >/dev/null 2>&1; then
      return 0
    fi
    local now
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 1
  done
}

run_pair() {
  local label="$1"
  local rnsd_cmd_a="$2"
  local rnstatus_cmd_a="$3"
  local rnid_cmd_a="$4"
  local cfg_flag_a="$5"
  local rnsd_cmd_b="$6"
  local rnstatus_cmd_b="$7"
  local rnid_cmd_b="$8"
  local cfg_flag_b="$9"

  echo
  echo "[cmp] $label: start tcp server/client nodes"

  local server_port sip_a cip_a sip_b cip_b
  server_port=$(( (RANDOM % 10000) + 42000 ))
  sip_a=$(( (RANDOM % 10000) + 38000 ))
  cip_a=$(( sip_a + 1 ))
  sip_b=$(( sip_a + 2 ))
  cip_b=$(( sip_a + 3 ))

  local node_a_dir node_b_dir
  node_a_dir="$(new_tcp_run_dir "$ROOT/configs/testing/two_nodes_tcp/client/config" "$sip_a" "$cip_a" 0 "$server_port" "with_bootstraps")"
  node_b_dir="$(new_tcp_run_dir "$ROOT/configs/testing/two_nodes_tcp/server/config" "$sip_b" "$cip_b" "$server_port" 0 "with_bootstraps")"

  local home_a="$node_a_dir/home"
  local home_b="$node_b_dir/home"
  mkdir -p "$home_a" "$home_b"

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

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$log_a" "Started rnsd version"; then
    echo "[cmp] $label: node_a did not start; see $log_a"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$log_b" "Started rnsd version"; then
    echo "[cmp] $label: node_b did not start; see $log_b"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local status_a="$OUT_DIR/${label}.rnstatus.node_a.out"
  local status_b="$OUT_DIR/${label}.rnstatus.node_b.out"
  if ! wait_for_tcp_status "$START_TIMEOUT_SECS" "$status_a" 'TCP Client/' \
    env HOME="$home_a" USERPROFILE="$home_a" $rnstatus_cmd_a $cfg_flag_a "$node_a_dir" -a; then
    echo "[cmp] $label: node_a TCP client did not become ready; see $status_a and $log_a"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if ! wait_for_tcp_status "$START_TIMEOUT_SECS" "$status_b" 'TCP Server/' \
    env HOME="$home_b" USERPROFILE="$home_b" $rnstatus_cmd_b $cfg_flag_b "$node_b_dir" -a; then
    echo "[cmp] $label: node_b TCP server did not become ready; see $status_b and $log_b"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  if ! wait_for_any_bootstrap_up "$START_TIMEOUT_SECS" "$status_a" "$status_b" \
    "env HOME='$home_a' USERPROFILE='$home_a' $rnstatus_cmd_a $cfg_flag_a '$node_a_dir' -a >'$status_a' 2>&1" \
    "env HOME='$home_b' USERPROFILE='$home_b' $rnstatus_cmd_b $cfg_flag_b '$node_b_dir' -a >'$status_b' 2>&1"; then
    echo "[cmp] $label: no bootstrap became ready; see $status_a and $status_b"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: bootstrap_ok=yes"

  stop_proc "$pid_a"
  stop_proc "$pid_b"

  server_port=$(( (RANDOM % 10000) + 42000 ))
  sip_a=$(( (RANDOM % 10000) + 38000 ))
  cip_a=$(( sip_a + 1 ))
  sip_b=$(( sip_a + 2 ))
  cip_b=$(( sip_a + 3 ))

  node_a_dir="$(new_tcp_run_dir "$ROOT/configs/testing/two_nodes_tcp/client/config" "$sip_a" "$cip_a" 0 "$server_port")"
  node_b_dir="$(new_tcp_run_dir "$ROOT/configs/testing/two_nodes_tcp/server/config" "$sip_b" "$cip_b" "$server_port" 0)"
  home_a="$node_a_dir/home"
  home_b="$node_b_dir/home"
  mkdir -p "$home_a" "$home_b"

  local remote_identity="$node_b_dir/remote_python.id"
  local gen_identity_log="$OUT_DIR/${label}.identity.generate.out"
  if [[ "$label" == "go_node_a_python_node_b" ]]; then
    if ! generate_python_identity "$node_b_dir" "$home_b" "$remote_identity" "$gen_identity_log"; then
      echo "[cmp] $label: failed to generate remote identity via python; see $gen_identity_log"
      return 1
    fi
  else
    remote_identity="$node_b_dir/remote_go.id"
    local gen_go_code
    gen_go_code="$(run_capture "$gen_identity_log" env HOME="$home_b" USERPROFILE="$home_b" \
      $rnid_cmd_b $cfg_flag_b "$node_b_dir" -g "$remote_identity" -q)"
    if [[ "$gen_go_code" != "0" ]]; then
      echo "[cmp] $label: failed to generate remote identity via go; see $gen_identity_log"
      return 1
    fi
  fi

  log_a="$OUT_DIR/${label}.rnsd.node_a.log"
  log_b="$OUT_DIR/${label}.rnsd.node_b.log"
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rnsd_cmd_b $cfg_flag_b "$node_b_dir" -q >"$log_b" 2>&1 &
  pid_b=$!
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$log_b" "Started rnsd version"; then
    echo "[cmp] $label: transfer node_b did not start; see $log_b"
    stop_proc "$pid_b"
    return 1
  fi
  env HOME="$home_a" USERPROFILE="$home_a" \
    $rnsd_cmd_a $cfg_flag_a "$node_a_dir" -q >"$log_a" 2>&1 &
  pid_a=$!

  sleep 0.5
  maybe_skip_env "$log_a"
  maybe_skip_env "$log_b"

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$log_a" "Started rnsd version"; then
    echo "[cmp] $label: transfer node_a did not start; see $log_a"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  status_a="$OUT_DIR/${label}.rnstatus.node_a.out"
  status_b="$OUT_DIR/${label}.rnstatus.node_b.out"
  if ! wait_for_tcp_status "$START_TIMEOUT_SECS" "$status_a" 'TCP Client/' \
    env HOME="$home_a" USERPROFILE="$home_a" $rnstatus_cmd_a $cfg_flag_a "$node_a_dir" -a; then
    echo "[cmp] $label: transfer node_a TCP client did not become ready; see $status_a and $log_a"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if ! wait_for_tcp_status "$START_TIMEOUT_SECS" "$status_b" 'TCP Server/' \
    env HOME="$home_b" USERPROFILE="$home_b" $rnstatus_cmd_b $cfg_flag_b "$node_b_dir" -a; then
    echo "[cmp] $label: transfer node_b TCP server did not become ready; see $status_b and $log_b"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

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

  local request_out="$OUT_DIR/${label}.request.out"
  local request_code
  request_code="$(run_capture "$request_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnid_cmd_a $cfg_flag_a "$node_a_dir" -i "$dest_hash" -R -t 15 -p -q)"

  stop_proc "$pid_a"
  stop_proc "$pid_b"

  if [[ "$request_code" != "0" ]]; then
    echo "[cmp] $label: request failed; see $request_out"
    return 1
  fi
  if ! rg -q "(Identity[[:space:]]*:|Public Key[[:space:]]*:)" "$request_out"; then
    echo "[cmp] $label: request output missing recalled identity details; see $request_out"
    return 1
  fi

  {
    echo "dest_hash=$dest_hash"
    echo "request_ok=yes"
  } >"$OUT_DIR/${label}.summary.txt"

  echo "[cmp] $label OK"
}

overall=0

if ! run_pair "go_node_a_python_node_b" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/rnid" \
  "-config" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnid.py" \
  "--config"; then
  overall=1
fi

if ! run_pair "python_node_a_go_node_b" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnid.py" \
  "--config" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/rnid" \
  "-config"; then
  overall=1
fi

if [[ "$overall" -ne 0 ]]; then
  echo "[cmp] FAIL (see $OUT_DIR)"
  exit 1
fi

echo "[cmp] OK"
