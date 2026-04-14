#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PYTHON="${PYTHON:-python3}"
SHARED_INSTANCE_TYPE="${SHARED_INSTANCE_TYPE:-}"
STATUS_CMD_TIMEOUT_SECS="${STATUS_CMD_TIMEOUT_SECS:-2}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-3}"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache" "$ROOT/tests/artifacts/logs"
export GOCACHE="$ROOT/.gocache"
GOTMPDIR="${GOTMPDIR:-$(mktemp -d "$ROOT/.gotmp/run.XXXXXX")}"
export GOTMPDIR
export GOPATH="$ROOT/.gopath"
export GOMODCACHE="$ROOT/.gomodcache"

export PYTHONPATH="${PYTHONPATH:-"$ROOT/python"}"
export PYTHONUNBUFFERED=1

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnstatus"
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

wait_for_status() {
  local cmd=("$@")
  local tries=80
  for _ in $(seq 1 "$tries"); do
    if "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STATUS_CMD_TIMEOUT_SECS" -- "${cmd[@]}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  return 1
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

run_status_text() {
  local out_file="$1"
  shift
  if ! "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STATUS_CMD_TIMEOUT_SECS" -- "$@" >"$out_file" 2>&1; then
    return 1
  fi
  return 0
}

normalize_text() {
  local in_file="$1"
  local out_file="$2"
  "$PYTHON" "$ROOT/tests/support/tools/normalize_rnstatus_text.py" <"$in_file" >"$out_file"
}

run_one() {
  local label="$1"
  local template_dir="$2"

  local base=$(( (RANDOM % 10000) + 45000 ))
  local sip="$base"
  local cip=$((base+1))
  local if_base=$(( (RANDOM % 10000) + 52000 ))

  local run_dir
  run_dir="$(mktemp -d)"

  cp "$template_dir/config" "$run_dir/config"

  local -a patch_args=(
    --path "$run_dir/config"
    --shared-instance-port "$sip"
    --instance-control-port "$cip"
    --interfaces-port-base "$if_base"
  )
  if [[ -n "$SHARED_INSTANCE_TYPE" ]]; then
    patch_args+=(--shared-instance-type "$SHARED_INSTANCE_TYPE")
  fi
  "$PYTHON" "$ROOT/tests/support/tools/patch_reticulum_config_ports.py" "${patch_args[@]}"

  local py_log="$OUT_DIR/$label.python.rnsd.log"
  local go_log="$OUT_DIR/$label.go.rnsd.log"
  local py_json="$OUT_DIR/$label.python.rnstatus.json"
  local go_json="$OUT_DIR/$label.go.rnstatus.json"
  local py_norm="$OUT_DIR/$label.python.rnstatus.norm"
  local go_norm="$OUT_DIR/$label.go.rnstatus.norm"
  local diff_out="$OUT_DIR/$label.diff"

  echo "[cmp] $label (ports: shared=$sip/$cip ifaces~$if_base)"

  # --- python ---
  "$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --config "$run_dir" -q >"$py_log" 2>&1 &
  local py_pid=$!
  # Avoid a race in upstream Python between concurrent Reticulum initialisation (rnsd + rnstatus).
  for _ in $(seq 1 80); do
    if [[ -d "$run_dir/storage/cache/announces" ]]; then
      break
    fi
    sleep 0.05
  done
  if ! wait_for_status "$PYTHON" "$ROOT/python/RNS/Utilities/rnstatus.py" --config "$run_dir" -j -a; then
    echo "[cmp] $label python did not become ready; log: $py_log"
    stop_proc "$py_pid"
    rm -rf "$run_dir"
    return 1
  fi
  if ! "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STATUS_CMD_TIMEOUT_SECS" -- \
    "$PYTHON" "$ROOT/python/RNS/Utilities/rnstatus.py" --config "$run_dir" -j -a >"$py_json"; then
    echo "[cmp] $label python rnstatus timed out; log: $py_log"
    stop_proc "$py_pid"
    rm -rf "$run_dir"
    return 1
  fi
  "$PYTHON" "$ROOT/tests/support/tools/normalize_rnstatus_json.py" <"$py_json" >"$py_norm"

  # Text-mode parity cases (normalised).
  local -a TEXT_CASES=(
    "default::"
    "all::-a"
    "announce_stats::-A"
    "link_stats::-l"
    "totals::-t"
    "sort_rate::-s rate"
    "sort_rate_rev::-s rate -r"
    "filter_default::Default"
  )

  for case in "${TEXT_CASES[@]}"; do
    local case_label="${case%%::*}"
    local case_args="${case#*::}"
    local py_txt="$OUT_DIR/$label.$case_label.python.rnstatus.txt"
    local py_txt_norm="$OUT_DIR/$label.$case_label.python.rnstatus.txt.norm"

    # shellcheck disable=SC2086
    if ! run_status_text "$py_txt" "$PYTHON" "$ROOT/python/RNS/Utilities/rnstatus.py" --config "$run_dir" $case_args; then
      echo "[cmp] $label python rnstatus ($case_label) timed out; log: $py_log"
      stop_proc "$py_pid"
      rm -rf "$run_dir"
      return 1
    fi
    normalize_text "$py_txt" "$py_txt_norm"
  done
  stop_proc "$py_pid"

  # --- go ---
  "$GO_BIN_DIR/rnsd" -config "$run_dir" -q -q >"$go_log" 2>&1 &
  local go_pid=$!
  if ! wait_for_status "$GO_BIN_DIR/rnstatus" -config "$run_dir" -j -a; then
    echo "[cmp] $label go did not become ready; log: $go_log"
    stop_proc "$go_pid"
    rm -rf "$run_dir"
    return 1
  fi
  if ! "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STATUS_CMD_TIMEOUT_SECS" -- \
    "$GO_BIN_DIR/rnstatus" -config "$run_dir" -j -a >"$go_json"; then
    echo "[cmp] $label go rnstatus timed out; log: $go_log"
    stop_proc "$go_pid"
    rm -rf "$run_dir"
    return 1
  fi
  "$PYTHON" "$ROOT/tests/support/tools/normalize_rnstatus_json.py" <"$go_json" >"$go_norm"

  for case in "${TEXT_CASES[@]}"; do
    local case_label="${case%%::*}"
    local case_args="${case#*::}"
    local go_txt="$OUT_DIR/$label.$case_label.go.rnstatus.txt"
    local go_txt_norm="$OUT_DIR/$label.$case_label.go.rnstatus.txt.norm"

    # shellcheck disable=SC2086
    if ! run_status_text "$go_txt" "$GO_BIN_DIR/rnstatus" -config "$run_dir" $case_args; then
      echo "[cmp] $label go rnstatus ($case_label) timed out; log: $go_log"
      stop_proc "$go_pid"
      rm -rf "$run_dir"
      return 1
    fi
    normalize_text "$go_txt" "$go_txt_norm"
  done
  stop_proc "$go_pid"

  # --- compare ---
  local ok=1
  if diff -u "$py_norm" "$go_norm" >"$diff_out"; then
    ok=0
  else
    echo "[cmp] $label JSON DIFF: $diff_out"
    ok=1
  fi

  for case in "${TEXT_CASES[@]}"; do
    local case_label="${case%%::*}"
    local py_txt_norm="$OUT_DIR/$label.$case_label.python.rnstatus.txt.norm"
    local go_txt_norm="$OUT_DIR/$label.$case_label.go.rnstatus.txt.norm"
    local tdiff="$OUT_DIR/$label.$case_label.text.diff"
    if diff -u "$py_txt_norm" "$go_txt_norm" >"$tdiff"; then
      :
    else
      echo "[cmp] $label TEXT($case_label) DIFF: $tdiff"
      ok=1
    fi
  done

  if [[ "$ok" -eq 0 ]]; then
    echo "[cmp] $label OK"
  fi

  rm -rf "$run_dir"
}

CONFIG_ROOT="${CONFIG_ROOT:-"$ROOT/configs/testing"}"

mapfile -t CFG_FILES < <(find "$CONFIG_ROOT" -type f -name config | sort)
if [[ "${#CFG_FILES[@]}" -eq 0 ]]; then
  echo "[cmp] no configs found under $CONFIG_ROOT"
  exit 2
fi

overall=0
for cfg in "${CFG_FILES[@]}"; do
  template_dir="$(dirname "$cfg")"
  rel="${template_dir#"$ROOT"/}"
  label="$(echo "$rel" | tr '/.' '__')"
  if ! run_one "$label" "$template_dir"; then
    overall=1
  fi
done

echo "[cmp] done (out=$OUT_DIR)"
exit "$overall"
