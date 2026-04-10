#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PYTHON="${PYTHON:-python3}"

STATUS_CMD_TIMEOUT_SECS="${STATUS_CMD_TIMEOUT_SECS:-2}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-3}"
RNSD_READY_TIMEOUT_SECS="${RNSD_READY_TIMEOUT_SECS:-8}"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache" "$ROOT/tests/artifacts/logs"
export GOCACHE="$ROOT/.gocache"
export GOTMPDIR="$ROOT/.gotmp"
export GOPATH="$ROOT/.gopath"
export GOMODCACHE="$ROOT/.gomodcache"

export PYTHONPATH="${PYTHONPATH:-"$ROOT/python"}"
export PYTHONUNBUFFERED=1

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnsd"
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

run_capture_sh() {
  local out="$1"
  shift
  local code=0
  set +e
  bash -c "$*" >"$out" 2>&1
  code=$?
  set -e
  echo "$code"
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

require_file_contains() {
  local label="$1"
  local path="$2"
  local needle="$3"
  if ! rg -q --fixed-strings "$needle" "$path"; then
    echo "[cmp] $label: expected '$path' to contain: $needle"
    return 1
  fi
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

wait_for_ok() {
  local timeout="$1"
  shift

  local start
  start="$(date +%s)"
  while true; do
    if "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STATUS_CMD_TIMEOUT_SECS" -- "$@" >/dev/null 2>&1; then
      return 0
    fi
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.1
  done
}

new_run_dir_from_template() {
  local template_dir="$1"
  local rpc_key="${2:-}"
  local shared_instance_type="${3:-}"
  local run_dir
  run_dir="$(mktemp -d)"
  cp "$template_dir/config" "$run_dir/config"

  local base=$(( (RANDOM % 10000) + 45000 ))
  local sip="$base"
  local cip=$((base+1))
  local if_base=$(( (RANDOM % 10000) + 52000 ))

  local -a patch_args=(
    --path "$run_dir/config" \
    --shared-instance-port "$sip" \
    --instance-control-port "$cip" \
    --interfaces-port-base "$if_base"
  )
  if [[ -n "$rpc_key" ]]; then
    patch_args+=(--rpc-key "$rpc_key")
  fi
  if [[ -n "$shared_instance_type" ]]; then
    patch_args+=(--shared-instance-type "$shared_instance_type")
  fi

  "$PYTHON" "$ROOT/tests/support/tools/patch_reticulum_config_ports.py" \
    "${patch_args[@]}"

  echo "$run_dir"
}

overall=0

echo
echo "[cmp] rnsd --exampleconfig"
py_out="$OUT_DIR/exampleconfig.python.out"
go_out="$OUT_DIR/exampleconfig.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --exampleconfig)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rnsd" --exampleconfig)"
if ! require_eq "python exit" "$py_code" 0 || ! require_eq "go exit" "$go_code" 0; then
  overall=1
else
  if diff -u "$py_out" "$go_out" >"$OUT_DIR/exampleconfig.diff"; then
    echo "[cmp] exampleconfig OK"
  else
    echo "[cmp] exampleconfig DIFF: $OUT_DIR/exampleconfig.diff"
    overall=1
  fi
fi

echo
echo "[cmp] rnsd --version"
py_out="$OUT_DIR/version.python.out"
go_out="$OUT_DIR/version.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --version)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rnsd" --version)"
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
echo "[cmp] rnsd --quickchecks (should be rejected like Python)"
py_out="$OUT_DIR/quickchecks.python.out"
go_out="$OUT_DIR/quickchecks.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --quickchecks 1)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rnsd" --quickchecks 1)"
if ! require_eq "python exit" "$py_code" 2 || ! require_eq "go exit" "$go_code" 2; then
  overall=1
else
  if ! require_file_contains "python output" "$py_out" "quickchecks" || ! require_file_contains "go output" "$go_out" "quickchecks"; then
    overall=1
  else
    echo "[cmp] quickchecks OK"
  fi
fi

echo
echo "[cmp] rnsd --service (log to file, no stdout)"
template="$ROOT/configs/testing/single_shared_tcp"
py_dir="$(new_run_dir_from_template "$template")"
go_dir="$(new_run_dir_from_template "$template")"

py_stdout="$OUT_DIR/service.python.stdout"
go_stdout="$OUT_DIR/service.go.stdout"

"$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --config "$py_dir" --service -q >"$py_stdout" 2>&1 &
py_pid=$!

if ! wait_for_ok "$RNSD_READY_TIMEOUT_SECS" test -f "$py_dir/logfile"; then
  echo "[cmp] service python did not create logfile; stdout: $py_stdout"
  stop_proc "$py_pid"
  overall=1
else
  stop_proc "$py_pid"
  if [[ -s "$py_stdout" ]]; then
    echo "[cmp] service python wrote to stdout; see $py_stdout"
    overall=1
  fi
  if [[ ! -s "$py_dir/logfile" ]]; then
    echo "[cmp] service python logfile is missing/empty"
    overall=1
  fi
  if ! require_file_contains "service python logfile" "$py_dir/logfile" "Started rnsd version"; then
    overall=1
  fi
fi

RNS_EXIT_WAIT_TIMEOUT=1 "$GO_BIN_DIR/rnsd" --config "$go_dir" --service -q >"$go_stdout" 2>&1 &
go_pid=$!

if ! wait_for_ok "$RNSD_READY_TIMEOUT_SECS" test -f "$go_dir/logfile"; then
  echo "[cmp] service go did not create logfile; stdout: $go_stdout"
  stop_proc "$go_pid"
  overall=1
else
  stop_proc "$go_pid"
  if [[ -s "$go_stdout" ]]; then
    echo "[cmp] service go wrote to stdout; see $go_stdout"
    overall=1
  fi
  if [[ ! -s "$go_dir/logfile" ]]; then
    echo "[cmp] service go logfile is missing/empty"
    overall=1
  fi
  if ! require_file_contains "service go logfile" "$go_dir/logfile" "Started rnsd version"; then
    overall=1
  fi
fi

rm -rf "$py_dir" "$go_dir"
if [[ "$overall" -eq 0 ]]; then
  echo "[cmp] service OK"
fi

echo
echo "[cmp] rnsd --interactive (should start a repl and exit on exit())"
run_dir="$(new_run_dir_from_template "$template")"
py_out="$OUT_DIR/interactive.python.out"
go_out="$OUT_DIR/interactive.go.out"

py_code="$(run_capture_sh "$py_out" "printf 'exit()\\n' | \"$PYTHON\" \"$ROOT/python/RNS/Utilities/rnsd.py\" --config \"$run_dir\" --interactive -q")"
go_code="$(run_capture_sh "$go_out" "printf 'exit()\\n' | \"$GO_BIN_DIR/rnsd\" --config \"$run_dir\" --interactive -q")"

if ! require_eq "python exit" "$py_code" 0 || ! require_eq "go exit" "$go_code" 0; then
  overall=1
else
  if ! require_file_contains "python interactive output" "$py_out" ">>> " || ! require_file_contains "go interactive output" "$go_out" ">>> "; then
    overall=1
  else
    echo "[cmp] interactive OK"
  fi
fi

rm -rf "$run_dir"

echo
echo "[cmp] rnsd rpc_key handling (invalid key should fallback, stats should match)"
run_dir="$(new_run_dir_from_template "$template" "nothex")"

py_log="$OUT_DIR/rpc_key_invalid.python.rnsd.log"
go_log="$OUT_DIR/rpc_key_invalid.go.rnsd.log"
py_json="$OUT_DIR/rpc_key_invalid.python.rnstatus.json"
go_json="$OUT_DIR/rpc_key_invalid.go.rnstatus.json"
py_norm="$OUT_DIR/rpc_key_invalid.python.rnstatus.norm"
go_norm="$OUT_DIR/rpc_key_invalid.go.rnstatus.norm"

"$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --config "$run_dir" -q >"$py_log" 2>&1 &
py_pid=$!
for _ in $(seq 1 80); do
  if [[ -d "$run_dir/storage/cache/announces" ]]; then
    break
  fi
  sleep 0.05
done
if ! wait_for_ok "$RNSD_READY_TIMEOUT_SECS" "$PYTHON" "$ROOT/python/RNS/Utilities/rnstatus.py" --config "$run_dir" -j -a; then
  echo "[cmp] rpc_key_invalid python did not become ready; log: $py_log"
  stop_proc "$py_pid"
  overall=1
else
  if ! "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STATUS_CMD_TIMEOUT_SECS" -- \
    "$PYTHON" "$ROOT/python/RNS/Utilities/rnstatus.py" --config "$run_dir" -j -a >"$py_json"; then
    echo "[cmp] rpc_key_invalid python rnstatus timed out; log: $py_log"
    stop_proc "$py_pid"
    overall=1
  else
    "$PYTHON" "$ROOT/tests/support/tools/normalize_rnstatus_json.py" <"$py_json" >"$py_norm"
    if ! require_file_contains "python rnsd log" "$py_log" "Invalid shared instance RPC key"; then
      overall=1
    fi
    stop_proc "$py_pid"
  fi
fi

RNS_EXIT_WAIT_TIMEOUT=1 "$GO_BIN_DIR/rnsd" --config "$run_dir" -q >"$go_log" 2>&1 &
go_pid=$!
if ! wait_for_ok "$RNSD_READY_TIMEOUT_SECS" "$GO_BIN_DIR/rnstatus" --config "$run_dir" -j -a; then
  echo "[cmp] rpc_key_invalid go did not become ready; log: $go_log"
  stop_proc "$go_pid"
  overall=1
else
  if ! "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STATUS_CMD_TIMEOUT_SECS" -- \
    "$GO_BIN_DIR/rnstatus" --config "$run_dir" -j -a >"$go_json"; then
    echo "[cmp] rpc_key_invalid go rnstatus timed out; log: $go_log"
    stop_proc "$go_pid"
    overall=1
  else
    "$PYTHON" "$ROOT/tests/support/tools/normalize_rnstatus_json.py" <"$go_json" >"$go_norm"
    if ! require_file_contains "go rnsd log" "$go_log" "Invalid shared instance RPC key"; then
      overall=1
    fi
    stop_proc "$go_pid"
  fi
fi

if [[ -f "$py_norm" && -f "$go_norm" ]]; then
  if diff -u "$py_norm" "$go_norm" >"$OUT_DIR/rpc_key_invalid.diff"; then
    echo "[cmp] rpc_key_invalid OK"
  else
    echo "[cmp] rpc_key_invalid DIFF: $OUT_DIR/rpc_key_invalid.diff"
    overall=1
  fi
fi

rm -rf "$run_dir"

echo
echo "[cmp] share_instance=requireShared (rnstatus without rnsd should match Python behaviour)"
run_dir="$(new_run_dir_from_template "$template")"
py_out="$OUT_DIR/require_shared.python.out"
go_out="$OUT_DIR/require_shared.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rnstatus.py" --config "$run_dir" -j -a)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rnstatus" --config "$run_dir" -j -a)"
if ! require_eq "exit parity" "$go_code" "$py_code"; then
  overall=1
else
  if [[ "$py_code" -eq 0 ]]; then
    py_norm="$OUT_DIR/require_shared.python.norm"
    go_norm="$OUT_DIR/require_shared.go.norm"
    "$PYTHON" "$ROOT/tests/support/tools/normalize_rnstatus_json.py" <"$py_out" >"$py_norm"
    "$PYTHON" "$ROOT/tests/support/tools/normalize_rnstatus_json.py" <"$go_out" >"$go_norm"
    if diff -u "$py_norm" "$go_norm" >"$OUT_DIR/require_shared.diff"; then
      echo "[cmp] require_shared OK"
    else
      echo "[cmp] require_shared DIFF: $OUT_DIR/require_shared.diff"
      overall=1
    fi
  else
    echo "[cmp] require_shared OK"
  fi
fi
rm -rf "$run_dir"

echo
echo "[cmp] shared_instance_type unix/tcp (stats should match where supported)"
for shared_type in tcp unix; do
  echo "[cmp]  type=$shared_type"
  run_dir="$(new_run_dir_from_template "$template" "" "$shared_type")"

  py_log="$OUT_DIR/shared_type_${shared_type}.python.rnsd.log"
  go_log="$OUT_DIR/shared_type_${shared_type}.go.rnsd.log"
  py_json="$OUT_DIR/shared_type_${shared_type}.python.rnstatus.json"
  go_json="$OUT_DIR/shared_type_${shared_type}.go.rnstatus.json"
  py_norm="$OUT_DIR/shared_type_${shared_type}.python.rnstatus.norm"
  go_norm="$OUT_DIR/shared_type_${shared_type}.go.rnstatus.norm"

  "$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --config "$run_dir" -q >"$py_log" 2>&1 &
  py_pid=$!
  for _ in $(seq 1 80); do
    if [[ -d "$run_dir/storage/cache/announces" ]]; then
      break
    fi
    sleep 0.05
  done
  if ! wait_for_ok "$RNSD_READY_TIMEOUT_SECS" "$PYTHON" "$ROOT/python/RNS/Utilities/rnstatus.py" --config "$run_dir" -j -a; then
    echo "[cmp] shared_type=$shared_type python did not become ready; log: $py_log"
    stop_proc "$py_pid"
    overall=1
    continue
  fi
  if ! "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STATUS_CMD_TIMEOUT_SECS" -- \
    "$PYTHON" "$ROOT/python/RNS/Utilities/rnstatus.py" --config "$run_dir" -j -a >"$py_json"; then
    echo "[cmp] shared_type=$shared_type python rnstatus timed out; log: $py_log"
    stop_proc "$py_pid"
    overall=1
    continue
  fi
  "$PYTHON" "$ROOT/tests/support/tools/normalize_rnstatus_json.py" <"$py_json" >"$py_norm"
  stop_proc "$py_pid"

  RNS_EXIT_WAIT_TIMEOUT=1 "$GO_BIN_DIR/rnsd" --config "$run_dir" -q >"$go_log" 2>&1 &
  go_pid=$!
  if ! wait_for_ok "$RNSD_READY_TIMEOUT_SECS" "$GO_BIN_DIR/rnstatus" --config "$run_dir" -j -a; then
    echo "[cmp] shared_type=$shared_type go did not become ready; log: $go_log"
    stop_proc "$go_pid"
    overall=1
    continue
  fi
  if ! "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STATUS_CMD_TIMEOUT_SECS" -- \
    "$GO_BIN_DIR/rnstatus" --config "$run_dir" -j -a >"$go_json"; then
    echo "[cmp] shared_type=$shared_type go rnstatus timed out; log: $go_log"
    stop_proc "$go_pid"
    overall=1
    continue
  fi
  "$PYTHON" "$ROOT/tests/support/tools/normalize_rnstatus_json.py" <"$go_json" >"$go_norm"
  stop_proc "$go_pid"

  if diff -u "$py_norm" "$go_norm" >"$OUT_DIR/shared_type_${shared_type}.diff"; then
    echo "[cmp]  type=$shared_type OK"
  else
    echo "[cmp]  type=$shared_type DIFF: $OUT_DIR/shared_type_${shared_type}.diff"
    overall=1
  fi

  rm -rf "$run_dir"
done

echo
echo "[cmp] done (out=$OUT_DIR)"
exit "$overall"
