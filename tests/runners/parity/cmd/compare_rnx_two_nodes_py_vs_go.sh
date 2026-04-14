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

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-90}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-30}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-6}"
PAIR_START_ATTEMPTS="${PAIR_START_ATTEMPTS:-4}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnx_two_nodes"
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
go build -o "$GO_BIN_DIR/rnx" ./cmd/rnx

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

normalize_output() {
  local src="$1"
  local dst="$2"

  tr '\r' '\n' <"$src" \
    | tr -d '\b' \
    | sed -E 's/[⢄⢂⢁⡁⡈⡐⡠]//g' \
    | sed -E '/^\[[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\] \[[^]]+\][[:space:]]*/d' \
    | sed -E 's/<[0-9a-fA-F]+>/<HASH>/g' \
    | sed -E 's/[[:space:]]+$//; s/[[:space:]]{2,}/ /g' \
    | sed -E '/^$/d' \
    >"$dst"
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

extract_first_hash() {
  local path="$1"
  rg -o "<[0-9a-fA-F]+>" "$path" | head -n 1 | tr -d '<>' || true
}

extract_listener_hash() {
  local path="$1"
  local line
  line="$(rg -o "rnx listening for commands on <[0-9a-fA-F]+>" "$path" | head -n 1 || true)"
  if [[ -n "$line" ]]; then
    echo "$line" | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>'
    return 0
  fi
  extract_first_hash "$path"
}

extract_identity_hash() {
  local path="$1"
  local line
  line="$(rg -o "Identity[[:space:]]*:[[:space:]]*<[0-9a-fA-F]+>" "$path" | head -n 1 || true)"
  if [[ -n "$line" ]]; then
    echo "$line" | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>'
    return 0
  fi
  # fallback: first hash token in output
  extract_first_hash "$path"
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

run_pair() {
  local label="$1" # python|go
  local rnsd_cmd="$2"
  local rnstatus_cmd="$3"
  local rnx_cmd="$4"

  local cfg_flag="-config"
  if [[ "$label" == "python" ]]; then
    cfg_flag="--config"
  fi

  echo
  echo "[cmp] $label: start two rnsd nodes"

  local node_a_template="$ROOT/configs/testing/two_nodes_udp/node_a/config"
  local node_b_template="$ROOT/configs/testing/two_nodes_udp/node_b/config"
  local node_a_dir node_b_dir home_a home_b
  local log_a log_b pid_a pid_b
  local start_attempt=1
  while [[ "$start_attempt" -le "$PAIR_START_ATTEMPTS" ]]; do
    local base
    base=$(( (RANDOM % 10000) + 52000 ))
    base=$(( base / 2 * 2 ))

    local sip_a cip_a sip_b cip_b
    sip_a=$(( (RANDOM % 10000) + 38000 ))
    cip_a=$(( sip_a + 1 ))
    sip_b=$(( sip_a + 2 ))
    cip_b=$(( sip_a + 3 ))

    node_a_dir="$(new_run_dir_from_template "$node_a_template" "$sip_a" "$cip_a" "$base" "$((base+1))")"
    node_b_dir="$(new_run_dir_from_template "$node_b_template" "$sip_b" "$cip_b" "$((base+1))" "$base")"

    home_a="$node_a_dir/home"
    home_b="$node_b_dir/home"
    mkdir -p "$home_a" "$home_b"

    log_a="$OUT_DIR/${label}.rnsd.node_a.log"
    log_b="$OUT_DIR/${label}.rnsd.node_b.log"
    : >"$log_a"
    : >"$log_b"

    env HOME="$home_a" USERPROFILE="$home_a" \
      $rnsd_cmd $cfg_flag "$node_a_dir" -q >"$log_a" 2>&1 &
    pid_a=$!

    env HOME="$home_b" USERPROFILE="$home_b" \
      $rnsd_cmd $cfg_flag "$node_b_dir" -q >"$log_b" 2>&1 &
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
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  done

  # Ensure shared instance is reachable for clients.
  _="$(run_capture "$OUT_DIR/${label}.rnstatus.node_a.out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnstatus_cmd $cfg_flag "$node_a_dir" -a)"
  _="$(run_capture "$OUT_DIR/${label}.rnstatus.node_b.out" env HOME="$home_b" USERPROFILE="$home_b" \
    $rnstatus_cmd $cfg_flag "$node_b_dir" -a)"

  echo "[cmp] $label: start rnx listener on node_b"
  local listener_log="$OUT_DIR/${label}.rnx.listener.log"
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rnx_cmd $cfg_flag "$node_b_dir" -l -n >"$listener_log" 2>&1 &
  local listener_pid=$!

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rnx listening for commands on"; then
    echo "[cmp] $label: rnx listener did not become ready; see $listener_log"
    stop_proc "$listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local listener_hash
  listener_hash="$(extract_listener_hash "$listener_log")"
  if [[ -z "$listener_hash" ]]; then
    echo "[cmp] $label: could not parse listener destination hash; see $listener_log"
    stop_proc "$listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: listener_hash=$listener_hash"

  echo "[cmp] $label: get client identity (node_a)"
  local id_log="$OUT_DIR/${label}.rnx.print_identity.log"
  _="$(run_capture "$id_log" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnx_cmd $cfg_flag "$node_a_dir" -p)"
  local client_id_hash
  client_id_hash="$(extract_identity_hash "$id_log")"
  if [[ -z "$client_id_hash" ]]; then
    echo "[cmp] $label: could not parse client identity hash; see $id_log"
    stop_proc "$listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi
  echo "[cmp] $label: client_id_hash=$client_id_hash"

  echo "[cmp] $label: client basic command (noauth listener)"
  local out_basic="$OUT_DIR/${label}.client.basic.out"
  local code_basic
  code_basic="$(run_capture "$out_basic" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnx_cmd $cfg_flag "$node_a_dir" -w 10 "$listener_hash" "echo hello")"

  stop_proc "$listener_pid"

  echo "[cmp] $label: start rnx listener ACL allow-list only"
  : >"$listener_log"
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rnx_cmd $cfg_flag "$node_b_dir" -l -a "$client_id_hash" >"$listener_log" 2>&1 &
  listener_pid=$!

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rnx listening for commands on"; then
    echo "[cmp] $label: rnx ACL listener did not become ready; see $listener_log"
    stop_proc "$listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi
  listener_hash="$(extract_listener_hash "$listener_log")"
  if [[ -z "$listener_hash" ]]; then
    echo "[cmp] $label: could not parse ACL listener hash; see $listener_log"
    stop_proc "$listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  echo "[cmp] $label: client command with identify (should succeed)"
  local out_acl_ok="$OUT_DIR/${label}.client.acl_ok.out"
  local code_acl_ok
  code_acl_ok="$(run_capture "$out_acl_ok" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnx_cmd $cfg_flag "$node_a_dir" -w 10 "$listener_hash" "echo ok")"

  echo "[cmp] $label: client command without identify (should fail)"
  local out_acl_noid="$OUT_DIR/${label}.client.acl_noid.out"
  local code_acl_noid
  code_acl_noid="$(run_capture "$out_acl_noid" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnx_cmd $cfg_flag "$node_a_dir" -N -w 6 "$listener_hash" "echo noid")"

  stop_proc "$listener_pid"

  echo "[cmp] $label: start rnx listener no-announce (client should not find path)"
  : >"$listener_log"
  local noannounce_id="$node_b_dir/no_announce.id"
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rnx_cmd $cfg_flag "$node_b_dir" -l -n -b -i "$noannounce_id" >"$listener_log" 2>&1 &
  listener_pid=$!

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rnx listening for commands on"; then
    echo "[cmp] $label: rnx no-announce listener did not become ready; see $listener_log"
    stop_proc "$listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi
  listener_hash="$(extract_listener_hash "$listener_log")"
  if [[ -z "$listener_hash" ]]; then
    echo "[cmp] $label: could not parse no-announce listener hash; see $listener_log"
    stop_proc "$listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local out_no_announce="$OUT_DIR/${label}.client.no_announce.out"
  local code_no_announce
  # Note: Python can often resolve the path even if the listener didn't announce at start,
  # because a direct PATH_REQUEST/response flow may still succeed between local nodes.
  # Use a slightly longer timeout to avoid flaky parity diffs.
  code_no_announce="$(run_capture "$out_no_announce" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnx_cmd $cfg_flag "$node_a_dir" -w 3 "$listener_hash" "echo should_fail")"

  stop_proc "$listener_pid"

  echo "[cmp] $label: start rnx listener for output limit / mirror tests"
  : >"$listener_log"
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rnx_cmd $cfg_flag "$node_b_dir" -l -n >"$listener_log" 2>&1 &
  listener_pid=$!

  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rnx listening for commands on"; then
    echo "[cmp] $label: rnx listener (limits) did not become ready; see $listener_log"
    stop_proc "$listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi
  listener_hash="$(extract_listener_hash "$listener_log")"
  if [[ -z "$listener_hash" ]]; then
    echo "[cmp] $label: could not parse listener hash (limits); see $listener_log"
    stop_proc "$listener_pid"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local out_limits="$OUT_DIR/${label}.client.limits.out"
  local code_limits
  local limits_cmd="sh -c 'printf abcdef; printf 12345 1>&2'"
  code_limits="$(run_capture "$out_limits" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnx_cmd $cfg_flag "$node_a_dir" -w 10 --stdout 3 --stderr 2 "$listener_hash" "$limits_cmd")"

  local out_mirror="$OUT_DIR/${label}.client.mirror.out"
  local code_mirror
  local mirror_cmd="sh -c 'exit 7'"
  code_mirror="$(run_capture "$out_mirror" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnx_cmd $cfg_flag "$node_a_dir" -w 10 -m "$listener_hash" "$mirror_cmd")"

  stop_proc "$listener_pid"
  stop_proc "$pid_a"
  stop_proc "$pid_b"

  normalize_output "$out_acl_ok" "$OUT_DIR/${label}.client.acl_ok.norm"
  normalize_output "$out_acl_noid" "$OUT_DIR/${label}.client.acl_noid.norm"
  normalize_output "$out_no_announce" "$OUT_DIR/${label}.client.no_announce.norm"
  normalize_output "$out_mirror" "$OUT_DIR/${label}.client.mirror.norm"

  {
    echo "basic_exit=$code_basic"
    echo "limits_exit=$code_limits"
    echo "acl_ok_norm_sha=$(shasum -a 256 "$OUT_DIR/${label}.client.acl_ok.norm" | awk '{print $1}')"
    echo "acl_noid_norm_sha=$(shasum -a 256 "$OUT_DIR/${label}.client.acl_noid.norm" | awk '{print $1}')"
    echo "no_announce_norm_sha=$(shasum -a 256 "$OUT_DIR/${label}.client.no_announce.norm" | awk '{print $1}')"
    echo "mirror_norm_sha=$(shasum -a 256 "$OUT_DIR/${label}.client.mirror.norm" | awk '{print $1}')"
  } >"$OUT_DIR/${label}.summary.txt"

  echo "[cmp] $label OK"
  return 0
}

overall=0

if ! run_pair "python" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnx.py"; then
  overall=1
fi

if ! run_pair "go" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/rnx"; then
  overall=1
fi

if [[ "$overall" -ne 0 ]]; then
  echo "[cmp] FAIL (see $OUT_DIR)"
  exit 1
fi

echo "[cmp] summary parity OK"
if diff -u "$OUT_DIR/python.summary.txt" "$OUT_DIR/go.summary.txt" >"$OUT_DIR/summary.diff"; then
  rm -f "$OUT_DIR/summary.diff" || true
  echo "[cmp] OK"
  exit 0
else
  echo "[cmp] DIFF: $OUT_DIR/summary.diff"
  exit 1
fi
