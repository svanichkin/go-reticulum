#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PYTHON="${PYTHON:-python3}"
CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-90}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-30}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-6}"
TWO_NODE_START_ATTEMPTS="${TWO_NODE_START_ATTEMPTS:-4}"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache" "$ROOT/tests/artifacts/logs"
export GOCACHE="$ROOT/.gocache"
GOTMPDIR="${GOTMPDIR:-$(mktemp -d "$ROOT/.gotmp/run.XXXXXX")}"
export GOTMPDIR
export GOPATH="$ROOT/.gopath"
export GOMODCACHE="$ROOT/.gomodcache"

export PYTHONPATH="${PYTHONPATH:-"$ROOT/python"}"
export PYTHONUNBUFFERED=1

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnid"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$GO_BIN_DIR" || true
  rm -rf "$GOTMPDIR" || true
}
trap cleanup EXIT

echo "[cmp] out=$OUT_DIR"
echo "[cmp] building go rnid..."
go build -o "$GO_BIN_DIR/rnid" ./cmd/rnid
go build -o "$GO_BIN_DIR/rnsd" ./cmd/rnsd
go build -o "$GO_BIN_DIR/rnstatus" ./cmd/rnstatus

run_capture() {
  local out="$1"
  shift
  local code=0
  set +e
  "$@" >"$out" 2>&1
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
    local now
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.2
  done
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
  rg "destination for this Identity is" "$path" \
    | sed -E 's/^.*<([0-9a-fA-F]+)>.*/\1/' \
    | head -n 1
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

run_two_node_request() {
  local label="$1"
  local rnsd_cmd_a="$2"
  local rnstatus_cmd_a="$3"
  local rnid_cmd_a="$4"
  local cfg_flag_a="$5"
  local rnsd_cmd_b="$6"
  local rnstatus_cmd_b="$7"
  local rnid_cmd_b="$8"
  local cfg_flag_b="$9"

  local node_a_template="$ROOT/configs/testing/two_nodes_udp/node_a/config"
  local node_b_template="$ROOT/configs/testing/two_nodes_udp/node_b/config"
  local node_a_dir node_b_dir home_a home_b log_a log_b pid_a pid_b
  local start_attempt=1
  while [[ "$start_attempt" -le "$TWO_NODE_START_ATTEMPTS" ]]; do
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

    log_a="$OUT_DIR/${label}.two_node.rnsd.node_a.log"
    log_b="$OUT_DIR/${label}.two_node.rnsd.node_b.log"
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

    if startup_had_addr_in_use "$log_a" || startup_had_addr_in_use "$log_b"; then
      stop_proc "$pid_a"
      stop_proc "$pid_b"
      if [[ "$start_attempt" -lt "$TWO_NODE_START_ATTEMPTS" ]]; then
        echo "[cmp] $label: startup port collision, retrying ($start_attempt/$TWO_NODE_START_ATTEMPTS)"
        start_attempt=$((start_attempt+1))
        continue
      fi
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

  local status_a="$OUT_DIR/${label}.two_node.rnstatus.node_a.out"
  local status_b="$OUT_DIR/${label}.two_node.rnstatus.node_b.out"
  local status_a_code status_b_code
  status_a_code="$(run_capture "$status_a" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnstatus_cmd_a $cfg_flag_a "$node_a_dir" -a)"
  status_b_code="$(run_capture "$status_b" env HOME="$home_b" USERPROFILE="$home_b" \
    $rnstatus_cmd_b $cfg_flag_b "$node_b_dir" -a)"
  if [[ "$status_a_code" != "0" || "$status_b_code" != "0" ]]; then
    echo "[cmp] $label: rnstatus failed"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local py_identity="$node_b_dir/node_b_python.id"
  local py_generate="$OUT_DIR/${label}.two_node.generate_python_identity.out"
  local py_gen_code
  py_gen_code="$(run_capture "$py_generate" env HOME="$home_b" USERPROFILE="$home_b" \
    "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" --config "$node_b_dir" --generate "$py_identity" -q)"
  if [[ "$py_gen_code" != "0" ]]; then
    echo "[cmp] $label: python identity generation failed; see $py_generate"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local hash_out="$OUT_DIR/${label}.two_node.hash.out"
  local hash_code
  hash_code="$(run_capture "$hash_out" env HOME="$home_b" USERPROFILE="$home_b" \
    $rnid_cmd_b $cfg_flag_b "$node_b_dir" -i "$py_identity" -H app.aspect)"
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

  local announce_out="$OUT_DIR/${label}.two_node.announce.out"
  local announce_code
  announce_code="$(run_capture_retry "$announce_out" 3 1 env HOME="$home_b" USERPROFILE="$home_b" \
    $rnid_cmd_b $cfg_flag_b "$node_b_dir" -i "$py_identity" -a app.aspect -q)"
  if [[ "$announce_code" != "0" ]]; then
    echo "[cmp] $label: announce failed; see $announce_out"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  # Give the announce a short moment to propagate before the reciprocal
  # request starts spinning. This keeps the cross-language network path stable
  # across slower CI environments and avoids timing-sensitive false negatives.
  sleep 5

  local request_out="$OUT_DIR/${label}.two_node.request.out"
  local request_code
  request_code="$(run_capture_retry "$request_out" 5 2 env HOME="$home_a" USERPROFILE="$home_a" \
    $rnid_cmd_a $cfg_flag_a "$node_a_dir" -i "$dest_hash" -R -t 20 -p -q)"

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
  } >"$OUT_DIR/${label}.two_node.summary.txt"

  echo "[cmp] $label: two-node request OK"
  return 0
}

require_eq() {
  local label="$1"
  local got="$2"
  local want="$3"
  if [[ "$got" != "$want" ]]; then
    echo "[cmp] $label: expected '$want', got '$got'"
    return 1
  fi
  return 0
}

extract_key_lines() {
  # Print only Public/Private key lines and strip timestamps/log level prefixes.
  rg "Public Key[[:space:]]*:|Private Key[[:space:]]*:" "$1" \
    | sed -E 's/^.*(Public Key[[:space:]]*:.*)$/\\1/; s/^.*(Private Key[[:space:]]*:.*)$/\\1/' \
    | sed -E 's/\\s+$//'
}

extract_export_data() {
  # Extract the exported identity payload (hex/base32/base64) from both Python and Go outputs.
  rg "Exported Identity[[:space:]]*:" "$1" | sed -E 's/^.*Exported Identity[[:space:]]*:[[:space:]]*//' | head -n 1
}

make_offline_config() {
  local dir="$1"
  mkdir -p "$dir"
  cat >"$dir/config" <<'CFG'
[reticulum]
  enable_transport = No
  share_instance = No

[logging]
  loglevel = 0

[interfaces]
CFG
}

new_run_dir() {
  local d
  d="$(mktemp -d)"
  make_offline_config "$d"
  echo "$d"
}

overall=0

echo
echo "[cmp] rnid --version"
py_out="$OUT_DIR/version.python.out"
go_out="$OUT_DIR/version.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" --version)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rnid" --version)"
if ! require_eq "python exit" "$py_code" 0 || ! require_eq "go exit" "$go_code" 0; then
  overall=1
else
  if diff -u "$py_out" "$go_out" >"$OUT_DIR/version.diff"; then
    echo "[cmp] version OK"
  else
    echo "[cmp] version DIFF: $OUT_DIR/version.diff"
    overall=1
  fi
fi

echo
echo "[cmp] identity generate + print (cross-load)"
run_dir="$(new_run_dir)"
home_dir="$run_dir/home"
mkdir -p "$home_dir"

py_id="$run_dir/py.id"
go_id="$run_dir/go.id"

py_code="$(run_capture "$OUT_DIR/gen.python.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" --config "$run_dir" -g "$py_id" -q)"
go_code="$(run_capture "$OUT_DIR/gen.go.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rnid" -config "$run_dir" -g "$go_id" -q)"
if ! require_eq "python generate exit" "$py_code" 0 || ! require_eq "go generate exit" "$go_code" 0; then
  overall=1
fi

py_print_py="$OUT_DIR/print.python_from_pyid.out"
go_print_py="$OUT_DIR/print.go_from_pyid.out"
py_print_go="$OUT_DIR/print.python_from_goid.out"
go_print_go="$OUT_DIR/print.go_from_goid.out"

py_code="$(run_capture "$py_print_py" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" --config "$run_dir" -i "$py_id" -p -q)"
go_code="$(run_capture "$go_print_py" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rnid" -config "$run_dir" -i "$py_id" -p -q)"
if ! require_eq "python print(py.id) exit" "$py_code" 0 || ! require_eq "go print(py.id) exit" "$go_code" 0; then
  overall=1
else
  extract_key_lines "$py_print_py" >"$OUT_DIR/print.python_from_pyid.keys"
  extract_key_lines "$go_print_py" >"$OUT_DIR/print.go_from_pyid.keys"
  if diff -u "$OUT_DIR/print.python_from_pyid.keys" "$OUT_DIR/print.go_from_pyid.keys" >"$OUT_DIR/print_pyid.diff"; then
    echo "[cmp] print(py.id) OK"
  else
    echo "[cmp] print(py.id) DIFF: $OUT_DIR/print_pyid.diff"
    overall=1
  fi
fi

py_code="$(run_capture "$py_print_go" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" --config "$run_dir" -i "$go_id" -p -q)"
go_code="$(run_capture "$go_print_go" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rnid" -config "$run_dir" -i "$go_id" -p -q)"
if ! require_eq "python print(go.id) exit" "$py_code" 0 || ! require_eq "go print(go.id) exit" "$go_code" 0; then
  overall=1
else
  extract_key_lines "$py_print_go" >"$OUT_DIR/print.python_from_goid.keys"
  extract_key_lines "$go_print_go" >"$OUT_DIR/print.go_from_goid.keys"
  if diff -u "$OUT_DIR/print.python_from_goid.keys" "$OUT_DIR/print.go_from_goid.keys" >"$OUT_DIR/print_goid.diff"; then
    echo "[cmp] print(go.id) OK"
  else
    echo "[cmp] print(go.id) DIFF: $OUT_DIR/print_goid.diff"
    overall=1
  fi
fi

echo
echo "[cmp] export/import parity (cross)"
py_export="$OUT_DIR/export.python.out"
go_export="$OUT_DIR/export.go.out"
py_code="$(run_capture "$py_export" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" --config "$run_dir" -i "$py_id" -x -q)"
go_code="$(run_capture "$go_export" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rnid" -config "$run_dir" -i "$py_id" -x -q)"
if ! require_eq "python export exit" "$py_code" 0 || ! require_eq "go export exit" "$go_code" 0; then
  overall=1
else
  extract_export_data "$py_export" >"$OUT_DIR/export.python.data"
  extract_export_data "$go_export" >"$OUT_DIR/export.go.data"
  if diff -u "$OUT_DIR/export.python.data" "$OUT_DIR/export.go.data" >"$OUT_DIR/export.diff"; then
    echo "[cmp] export OK"
  else
    echo "[cmp] export DIFF: $OUT_DIR/export.diff"
    overall=1
  fi
fi

imported_from_py="$run_dir/imported_from_py.id"
imported_from_go="$run_dir/imported_from_go.id"
py_data="$(cat "$OUT_DIR/export.python.data")"
go_data="$(cat "$OUT_DIR/export.go.data")"

py_code="$(run_capture "$OUT_DIR/import.go_from_py.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rnid" -m "$py_data" -w "$imported_from_py" -f -q)"
go_code="$(run_capture "$OUT_DIR/import.python_from_go.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" -m "$go_data" -w "$imported_from_go" -f -q)"
if ! require_eq "go import(py_export) exit" "$py_code" 0 || ! require_eq "python import(go_export) exit" "$go_code" 0; then
  overall=1
fi

echo
echo "[cmp] sign/validate (cross)"
plain="$run_dir/plain.txt"
printf "hello rnid parity\n" >"$plain"
py_sig="$run_dir/py.sig"
go_sig="$run_dir/go.sig"

py_code="$(run_capture "$OUT_DIR/sign.python.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" --config "$run_dir" -i "$py_id" -s "$plain" -w "$py_sig" -f -q)"
go_code="$(run_capture "$OUT_DIR/sign.go.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rnid" -config "$run_dir" -i "$go_id" -s "$plain" -w "$go_sig" -f -q)"
if ! require_eq "python sign exit" "$py_code" 0 || ! require_eq "go sign exit" "$go_code" 0; then
  overall=1
fi

# validate python signature with go, and go signature with python
py_code="$(run_capture "$OUT_DIR/validate.go_from_py.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rnid" -config "$run_dir" -i "$py_id" -V "$py_sig" -r "$plain" -q)"
go_code="$(run_capture "$OUT_DIR/validate.python_from_go.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" --config "$run_dir" -i "$go_id" -V "$go_sig" -r "$plain" -q)"
if ! require_eq "go validate(py.sig) exit" "$py_code" 0 || ! require_eq "python validate(go.sig) exit" "$go_code" 0; then
  overall=1
else
  echo "[cmp] sign/validate OK"
fi

echo
echo "[cmp] encrypt/decrypt (cross)"
py_enc="$run_dir/py.rfe"
go_enc="$run_dir/go.rfe"
py_dec="$run_dir/py.dec"
go_dec="$run_dir/go.dec"

py_code="$(run_capture "$OUT_DIR/encrypt.python.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" --config "$run_dir" -i "$py_id" -e "$plain" -w "$py_enc" -f -q)"
go_code="$(run_capture "$OUT_DIR/encrypt.go.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rnid" -config "$run_dir" -i "$go_id" -e "$plain" -w "$go_enc" -f -q)"
if ! require_eq "python encrypt exit" "$py_code" 0 || ! require_eq "go encrypt exit" "$go_code" 0; then
  overall=1
fi

py_code="$(run_capture "$OUT_DIR/decrypt.go_from_py.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rnid" -config "$run_dir" -i "$py_id" -d "$py_enc" -w "$go_dec" -f -q)"
go_code="$(run_capture "$OUT_DIR/decrypt.python_from_go.out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rnid.py" --config "$run_dir" -i "$go_id" -d "$go_enc" -w "$py_dec" -f -q)"
if ! require_eq "go decrypt(py.rfe) exit" "$py_code" 0 || ! require_eq "python decrypt(go.rfe) exit" "$go_code" 0; then
  overall=1
else
  if diff -u "$plain" "$go_dec" >"$OUT_DIR/decrypt_go_from_py.diff" && diff -u "$plain" "$py_dec" >"$OUT_DIR/decrypt_python_from_go.diff"; then
    echo "[cmp] encrypt/decrypt OK"
  else
    echo "[cmp] decrypt output mismatch"
    overall=1
  fi
fi

rm -rf "$run_dir"

echo
echo "[cmp] two-node request via python-generated remote identity"
if ! run_two_node_request "go_node_a_python_node_b" \
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

echo
echo "[cmp] two-node request via go remote identity recalled by python"
if ! run_two_node_request "python_node_a_go_node_b" \
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

echo
echo "[cmp] done (out=$OUT_DIR)"
exit "$overall"
