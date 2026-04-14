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

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnid_two_nodes_udp"
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

maybe_skip_env() {
  local log="$1"
  if rg -q -i "operation not permitted|permission denied|bind:|listen .*:.*(denied|not permitted)" "$log"; then
    echo "[cmp] SKIP: environment does not permit binding sockets (see $log)"
    exit 0
  fi
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

extract_dest_hash() {
  local path="$1"
  rg "destination for this Identity is" "$path" \
    | sed -E 's/^.*<([0-9a-fA-F]+)>.*/\1/' \
    | head -n 1
}

identity_output_ok() {
  local path="$1"
  rg -q "(Identity[[:space:]]*:|Public Key[[:space:]]*:)" "$path"
}

request_identity() {
  local out="$1"
  local home_dir="$2"
  local rnid_cmd="$3"
  local cfg_flag="$4"
  local config_dir="$5"
  local dest_hash="$6"
  local timeout_secs="${7:-15}"
  run_capture "$out" env HOME="$home_dir" USERPROFILE="$home_dir" \
    $rnid_cmd $cfg_flag "$config_dir" -i "$dest_hash" -R -t "$timeout_secs" -p -q
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

  local log_a="$OUT_DIR/${label}.rnsd.node_a.log"
  local log_b="$OUT_DIR/${label}.rnsd.node_b.log"
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rnsd_cmd_b $cfg_flag_b "$node_b_dir" -q >"$log_b" 2>&1 &
  local pid_b=$!
  env HOME="$home_a" USERPROFILE="$home_a" \
    $rnsd_cmd_a $cfg_flag_a "$node_a_dir" -q >"$log_a" 2>&1 &
  local pid_a=$!

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
  request_code="$(request_identity "$request_out" "$home_a" "$rnid_cmd_a" "$cfg_flag_a" "$node_a_dir" "$dest_hash" 15)"
  if [[ "$request_code" != "0" ]] || ! identity_output_ok "$request_out"; then
    echo "[cmp] $label: request failed; see $request_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local hash_one_out="$OUT_DIR/${label}.hash.one.out"
  local hash_one_code
  hash_one_code="$(run_capture "$hash_one_out" env HOME="$home_b" USERPROFILE="$home_b" \
    $rnid_cmd_b $cfg_flag_b "$node_b_dir" -i "$remote_identity" -H app.one)"
  if [[ "$hash_one_code" != "0" ]]; then
    echo "[cmp] $label: hash app.one failed; see $hash_one_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  local hash_one
  hash_one="$(extract_dest_hash "$hash_one_out")"
  if [[ -z "$hash_one" ]]; then
    echo "[cmp] $label: could not parse app.one hash; see $hash_one_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local hash_two_out="$OUT_DIR/${label}.hash.two.out"
  local hash_two_code
  hash_two_code="$(run_capture "$hash_two_out" env HOME="$home_b" USERPROFILE="$home_b" \
    $rnid_cmd_b $cfg_flag_b "$node_b_dir" -i "$remote_identity" -H app.two)"
  if [[ "$hash_two_code" != "0" ]]; then
    echo "[cmp] $label: hash app.two failed; see $hash_two_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  local hash_two
  hash_two="$(extract_dest_hash "$hash_two_out")"
  if [[ -z "$hash_two" ]]; then
    echo "[cmp] $label: could not parse app.two hash; see $hash_two_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if [[ "$hash_one" == "$hash_two" ]]; then
    echo "[cmp] $label: app.one and app.two hashes unexpectedly match"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local announce_one_out="$OUT_DIR/${label}.announce.one.out"
  local announce_one_code
  announce_one_code="$(run_capture_retry "$announce_one_out" 3 1 env HOME="$home_b" USERPROFILE="$home_b" \
    $rnid_cmd_b $cfg_flag_b "$node_b_dir" -i "$remote_identity" -a app.one -q)"
  if [[ "$announce_one_code" != "0" ]]; then
    echo "[cmp] $label: announce app.one failed; see $announce_one_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local announce_two_out="$OUT_DIR/${label}.announce.two.out"
  local announce_two_code
  announce_two_code="$(run_capture_retry "$announce_two_out" 3 1 env HOME="$home_b" USERPROFILE="$home_b" \
    $rnid_cmd_b $cfg_flag_b "$node_b_dir" -i "$remote_identity" -a app.two -q)"
  if [[ "$announce_two_code" != "0" ]]; then
    echo "[cmp] $label: announce app.two failed; see $announce_two_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local request_one_out="$OUT_DIR/${label}.request.one.out"
  local request_one_code
  request_one_code="$(request_identity "$request_one_out" "$home_a" "$rnid_cmd_a" "$cfg_flag_a" "$node_a_dir" "$hash_one" 15)"
  if [[ "$request_one_code" != "0" ]] || ! identity_output_ok "$request_one_out"; then
    echo "[cmp] $label: request app.one failed; see $request_one_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local request_two_out="$OUT_DIR/${label}.request.two.out"
  local request_two_code
  request_two_code="$(request_identity "$request_two_out" "$home_a" "$rnid_cmd_a" "$cfg_flag_a" "$node_a_dir" "$hash_two" 15)"
  if [[ "$request_two_code" != "0" ]] || ! identity_output_ok "$request_two_out"; then
    echo "[cmp] $label: request app.two failed; see $request_two_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local reannounce_out="$OUT_DIR/${label}.reannounce.out"
  local reannounce_code
  reannounce_code="$(run_capture_retry "$reannounce_out" 3 1 env HOME="$home_b" USERPROFILE="$home_b" \
    $rnid_cmd_b $cfg_flag_b "$node_b_dir" -i "$remote_identity" -a app.aspect -q)"
  if [[ "$reannounce_code" != "0" ]]; then
    echo "[cmp] $label: re-announce failed; see $reannounce_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local request_after_reannounce_out="$OUT_DIR/${label}.request_after_reannounce.out"
  local request_after_reannounce_code
  request_after_reannounce_code="$(request_identity "$request_after_reannounce_out" "$home_a" "$rnid_cmd_a" "$cfg_flag_a" "$node_a_dir" "$dest_hash" 15)"
  if [[ "$request_after_reannounce_code" != "0" ]] || ! identity_output_ok "$request_after_reannounce_out"; then
    echo "[cmp] $label: request after re-announce failed; see $request_after_reannounce_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local repeat_request_out="$OUT_DIR/${label}.repeat_request.out"
  local repeat_request_code
  repeat_request_code="$(request_identity "$repeat_request_out" "$home_a" "$rnid_cmd_a" "$cfg_flag_a" "$node_a_dir" "$dest_hash" 15)"
  if [[ "$repeat_request_code" != "0" ]] || ! identity_output_ok "$repeat_request_out"; then
    echo "[cmp] $label: repeated request while peer online failed; see $repeat_request_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  stop_proc "$pid_b"

  local offline_request_out="$OUT_DIR/${label}.offline_request.out"
  local offline_request_code
  offline_request_code="$(request_identity "$offline_request_out" "$home_a" "$rnid_cmd_a" "$cfg_flag_a" "$node_a_dir" "$dest_hash" 1)"
  stop_proc "$pid_a"
  if [[ "$offline_request_code" != "0" ]] || ! identity_output_ok "$offline_request_out"; then
    echo "[cmp] $label: offline recall request failed; see $offline_request_out"
    return 1
  fi

  {
    echo "dest_hash=$dest_hash"
    echo "request_ok=yes"
    echo "multi_aspect_ok=yes"
    echo "reannounce_ok=yes"
    echo "repeat_request_ok=yes"
    echo "offline_recall_ok=yes"
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
