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
LINK_WAIT_SECS="${LINK_WAIT_SECS:-20}"
LINK_HOLD_SECS="${LINK_HOLD_SECS:-12}"
LINK_KEEPALIVE_SECS="${LINK_KEEPALIVE_SECS:-5}"
LINK_STALE_HOLD_SECS="${LINK_STALE_HOLD_SECS:-45}"
LINK_STALE_WAIT_SECS="${LINK_STALE_WAIT_SECS:-60}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_link_level_two_nodes_tcp"
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
go build -o "$GO_BIN_DIR/go_link_helper" ./tests/support/tools/go_link_helper

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
  return 0
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

maybe_skip_env() {
  local log="$1"
  if rg -q -i "operation not permitted|permission denied|bind:|listen .*:.*(denied|not permitted)" "$log"; then
    echo "[cmp] SKIP: environment does not permit binding sockets (see $log)"
    exit 0
  fi
}

extract_listen_hash() {
  local path="$1"
  awk '/^LISTEN_HASH / { print $2; exit }' "$path"
}

assert_log_contains() {
  local path="$1"
  local needle="$2"
  local label="$3"
  if ! rg --fixed-strings "$needle" "$path" >/dev/null 2>&1; then
    echo "[cmp] $label missing '$needle'; see $path"
    return 1
  fi
}

run_pair() {
  local label="$1"
  local rnsd_cmd_a="$2"
  local rnstatus_cmd_a="$3"
  local helper_cmd_a="$4"
  local cfg_flag_a="$5"
  local rnsd_cmd_b="$6"
  local rnstatus_cmd_b="$7"
  local helper_cmd_b="$8"
  local cfg_flag_b="$9"

  echo
  echo "[cmp] $label: bootstrap phase"

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

  local log_a="$OUT_DIR/${label}.bootstrap.node_a.log"
  local log_b="$OUT_DIR/${label}.bootstrap.node_b.log"
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
    echo "[cmp] $label: bootstrap node_a did not start; see $log_a"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$log_b" "Started rnsd version"; then
    echo "[cmp] $label: bootstrap node_b did not start; see $log_b"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local status_a="$OUT_DIR/${label}.bootstrap.status.node_a.out"
  local status_b="$OUT_DIR/${label}.bootstrap.status.node_b.out"
  if ! wait_for_tcp_status "$START_TIMEOUT_SECS" "$status_a" 'TCP Client/' \
    env HOME="$home_a" USERPROFILE="$home_a" $rnstatus_cmd_a $cfg_flag_a "$node_a_dir" -a; then
    echo "[cmp] $label: bootstrap node_a TCP client not ready"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if ! wait_for_tcp_status "$START_TIMEOUT_SECS" "$status_b" 'TCP Server/' \
    env HOME="$home_b" USERPROFILE="$home_b" $rnstatus_cmd_b $cfg_flag_b "$node_b_dir" -a; then
    echo "[cmp] $label: bootstrap node_b TCP server not ready"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if ! wait_for_any_bootstrap_up "$START_TIMEOUT_SECS" "$status_a" "$status_b" \
    "env HOME='$home_a' USERPROFILE='$home_a' $rnstatus_cmd_a $cfg_flag_a '$node_a_dir' -a >'$status_a' 2>&1" \
    "env HOME='$home_b' USERPROFILE='$home_b' $rnstatus_cmd_b $cfg_flag_b '$node_b_dir' -a >'$status_b' 2>&1"; then
    echo "[cmp] $label: no bootstrap became ready"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: bootstrap_ok=yes"

  stop_proc "$pid_a"
  stop_proc "$pid_b"

  echo "[cmp] $label: local TCP mixed link-level phase"

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

  local listener_log="$OUT_DIR/${label}.listener.out"
  local client_log="$OUT_DIR/${label}.client.out"
  local listener_identity="$node_b_dir/listener.id"
  local client_identity="$node_a_dir/client.id"

  env HOME="$home_b" USERPROFILE="$home_b" \
    $helper_cmd_b $cfg_flag_b "$node_b_dir" \
    --identity "$listener_identity" \
    --listen \
    --wait-seconds "$LINK_WAIT_SECS" \
    --keepalive-seconds "$LINK_KEEPALIVE_SECS" >"$listener_log" 2>&1 &
  local listener_pid=$!

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "LISTEN_HASH "; then
    echo "[cmp] $label: listener helper did not become ready; see $listener_log"
    stop_proc "$listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local listen_hash
  listen_hash="$(extract_listen_hash "$listener_log")"
  if [[ -z "$listen_hash" ]]; then
    echo "[cmp] $label: failed to parse listener hash; see $listener_log"
    stop_proc "$listener_pid"
    return 1
  fi

  sleep 1

  local client_code
  local attempt
  for attempt in 1 2 3; do
    client_code="$(run_capture "$client_log" env HOME="$home_a" USERPROFILE="$home_a" \
      $helper_cmd_a $cfg_flag_a "$node_a_dir" \
      --identity "$client_identity" \
      --destination "$listen_hash" \
      --identify \
      --hold-seconds "$LINK_HOLD_SECS" \
      --wait-seconds "$LINK_WAIT_SECS" \
      --keepalive-seconds "$LINK_KEEPALIVE_SECS" \
      --teardown)"
    if [[ "$client_code" == "0" ]]; then
      break
    fi
    if ! rg -q "path not found|connect: connection refused" "$client_log"; then
      break
    fi
    if [[ "$attempt" -lt 3 ]]; then
      sleep 2
    fi
  done
  if [[ "$client_code" != "0" ]]; then
    echo "[cmp] $label: client helper failed; see $client_log"
    stop_proc "$listener_pid"
    return 1
  fi

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "EVENT closed"; then
    echo "[cmp] $label: listener did not observe link close; see $listener_log"
    stop_proc "$listener_pid"
    return 1
  fi

  assert_log_contains "$client_log" "EVENT established" "$label client" || { stop_proc "$listener_pid"; return 1; }
  assert_log_contains "$client_log" "EVENT identify_sent" "$label client" || { stop_proc "$listener_pid"; return 1; }
  assert_log_contains "$client_log" "EVENT still_active" "$label client" || { stop_proc "$listener_pid"; return 1; }
  assert_log_contains "$listener_log" "EVENT established" "$label listener" || { stop_proc "$listener_pid"; return 1; }
  assert_log_contains "$listener_log" "EVENT identified" "$label listener" || { stop_proc "$listener_pid"; return 1; }
  assert_log_contains "$listener_log" "EVENT closed" "$label listener" || { stop_proc "$listener_pid"; return 1; }

  stop_proc "$listener_pid"

  echo "[cmp] $label: stale/timeout phase"
  local stale_listener_log="$OUT_DIR/${label}.stale.listener.out"
  local stale_client_log="$OUT_DIR/${label}.stale.client.out"

  env HOME="$home_b" USERPROFILE="$home_b" \
    $helper_cmd_b $cfg_flag_b "$node_b_dir" \
    --identity "$listener_identity" \
    --listen \
    --wait-seconds "$LINK_WAIT_SECS" \
    --keepalive-seconds "$LINK_KEEPALIVE_SECS" >"$stale_listener_log" 2>&1 &
  listener_pid=$!

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$stale_listener_log" "LISTEN_HASH "; then
    echo "[cmp] $label: stale listener did not become ready; see $stale_listener_log"
    stop_proc "$listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  listen_hash="$(extract_listen_hash "$stale_listener_log")"
  if [[ -z "$listen_hash" ]]; then
    echo "[cmp] $label: failed to parse stale listener hash; see $stale_listener_log"
    stop_proc "$listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  env HOME="$home_a" USERPROFILE="$home_a" \
    $helper_cmd_a $cfg_flag_a "$node_a_dir" \
    --identity "$client_identity" \
    --destination "$listen_hash" \
    --identify \
    --expect-close \
    --hold-seconds "$LINK_STALE_HOLD_SECS" \
    --wait-seconds "$LINK_STALE_WAIT_SECS" \
    --keepalive-seconds "$LINK_KEEPALIVE_SECS" >"$stale_client_log" 2>&1 &
  local stale_client_pid=$!

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$stale_client_log" "EVENT established"; then
    echo "[cmp] $label: stale client did not establish link; see $stale_client_log"
    stop_proc "$stale_client_pid"; stop_proc "$listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$stale_listener_log" "EVENT established"; then
    echo "[cmp] $label: stale listener did not observe establishment; see $stale_listener_log"
    stop_proc "$stale_client_pid"; stop_proc "$listener_pid"; stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  stop_proc "$pid_b"

  if ! wait_for_file_contains "$LINK_STALE_WAIT_SECS" "$stale_client_log" "EVENT stale_closed"; then
    echo "[cmp] $label: stale client did not observe stale close; see $stale_client_log"
    stop_proc "$stale_client_pid"; stop_proc "$listener_pid"; stop_proc "$pid_a"
    return 1
  fi
  stop_proc "$stale_client_pid"

  assert_log_contains "$stale_client_log" "EVENT closed" "$label stale-client" || { stop_proc "$listener_pid"; stop_proc "$pid_a"; return 1; }
  assert_log_contains "$stale_client_log" "EVENT stale_closed" "$label stale-client" || { stop_proc "$listener_pid"; stop_proc "$pid_a"; return 1; }

  echo "[cmp] $label OK"
  stop_proc "$listener_pid"
  stop_proc "$pid_a"
  stop_proc "$pid_b"
}

overall=0

if ! run_pair "go_node_a_python_node_b" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/go_link_helper" \
  "-config" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/tests/support/tools/py_link_helper.py" \
  "--config"; then
  overall=1
fi

if ! run_pair "python_node_a_go_node_b" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/tests/support/tools/py_link_helper.py" \
  "--config" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/go_link_helper" \
  "-config"; then
  overall=1
fi

if [[ "$overall" -ne 0 ]]; then
  echo "[cmp] FAIL (see $OUT_DIR)"
  exit 1
fi

echo "[cmp] OK"
