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

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-20}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnir"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$GO_BIN_DIR" || true
}
trap cleanup EXIT

echo "[cmp] out=$OUT_DIR"
echo "[cmp] building go rnir..."
go build -o "$GO_BIN_DIR/rnir" ./cmd/rnir

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

normalize_output() {
  local src="$1"
  local dst="$2"
  tr -d '\r' <"$src" \
    | sed -E '/^\[[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\] \[[^]]+\][[:space:]]*/d' \
    | sed -E 's/[[:space:]]+$//' \
    >"$dst"
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

overall=0

echo
echo "[cmp] rnir --version"
py_out="$OUT_DIR/version.python.out"
go_out="$OUT_DIR/version.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rnir.py" --version)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rnir" --version)"
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
echo "[cmp] rnir --exampleconfig"
py_out="$OUT_DIR/exampleconfig.python.out"
go_out="$OUT_DIR/exampleconfig.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rnir.py" --exampleconfig)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rnir" --exampleconfig)"
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
echo "[cmp] rnir startup with UDP interface config (bind sockets)"
base=$(( (RANDOM % 10000) + 52000 ))
base=$(( base / 2 * 2 ))
sip=$(( (RANDOM % 10000) + 38000 ))
cip=$(( sip + 1 ))

template="$ROOT/configs/testing/two_nodes_udp/node_a/config"
py_cfg="$(new_run_dir_from_template "$template" "$sip" "$cip" "$base" "$((base+1))")"
go_cfg="$(new_run_dir_from_template "$template" "$((sip+2))" "$((cip+2))" "$((base+2))" "$((base+3))")"

py_home="$py_cfg/home"
go_home="$go_cfg/home"
mkdir -p "$py_home" "$go_home"

py_out="$OUT_DIR/startup.python.out"
go_out="$OUT_DIR/startup.go.out"
py_code="$(run_capture "$py_out" env HOME="$py_home" USERPROFILE="$py_home" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rnir.py" --config "$py_cfg" -q)"
go_code="$(run_capture "$go_out" env HOME="$go_home" USERPROFILE="$go_home" \
  "$GO_BIN_DIR/rnir" -config "$go_cfg" -q)"

if ! require_eq "python startup exit" "$py_code" 0 || ! require_eq "go startup exit" "$go_code" 0; then
  echo "[cmp] startup outputs: $py_out $go_out"
  overall=1
else
  normalize_output "$py_out" "$OUT_DIR/startup.python.norm"
  normalize_output "$go_out" "$OUT_DIR/startup.go.norm"
  echo "[cmp] startup OK (exit codes)"
fi

echo
if [[ "$overall" -eq 0 ]]; then
  echo "[cmp] OK"
  exit 0
else
  echo "[cmp] FAIL (see $OUT_DIR)"
  exit 1
fi
