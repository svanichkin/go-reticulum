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

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-20}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnpath"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$GO_BIN_DIR" || true
}
trap cleanup EXIT

echo "[cmp] out=$OUT_DIR"
echo "[cmp] building go rnpath..."
go build -o "$GO_BIN_DIR/rnpath" ./cmd/rnpath

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

  # Strip Reticulum log lines and normalize CR/LF to stable diffs.
  tr -d '\r' <"$src" \
    | sed -E '/^\\[[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\\] \\[[^]]+\\][[:space:]]*/d' \
    | sed -E 's/[[:space:]]+$//' \
    >"$dst"
}

make_offline_config() {
  local dir="$1"
  mkdir -p "$dir"
  cat >"$dir/config" <<'CFG'
[reticulum]
  enable_transport = False
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

dest_len_hex="$("$PYTHON" -c 'import RNS; print((RNS.Reticulum.TRUNCATED_HASHLENGTH//8)*2)')"
valid_dest="$(printf '%*s' "$dest_len_hex" '' | tr ' ' 'a')"
invalid_len_dest="aa"
invalid_hex_dest="${valid_dest:0:$((dest_len_hex-1))}z"

overall=0

compare_case() {
  local label="$1"
  shift

  local py_out="$OUT_DIR/${label}.python.out"
  local go_out="$OUT_DIR/${label}.go.out"
  local py_norm="$OUT_DIR/${label}.python.norm"
  local go_norm="$OUT_DIR/${label}.go.norm"
  local diff_out="$OUT_DIR/${label}.diff"

  local py_home go_home py_cfg go_cfg
  py_cfg="$(new_run_dir)"
  go_cfg="$(new_run_dir)"
  py_home="$py_cfg/home"
  go_home="$go_cfg/home"
  mkdir -p "$py_home" "$go_home"

  local -a args=("$@")

  local py_code go_code
  py_code="$(run_capture "$py_out" env HOME="$py_home" USERPROFILE="$py_home" \
    "$PYTHON" "$ROOT/python/RNS/Utilities/rnpath.py" --config "$py_cfg" "${args[@]}")"
  go_code="$(run_capture "$go_out" env HOME="$go_home" USERPROFILE="$go_home" \
    "$GO_BIN_DIR/rnpath" --config "$go_cfg" "${args[@]}")"

  normalize_output "$py_out" "$py_norm"
  normalize_output "$go_out" "$go_norm"

  if ! require_eq "$label python exit" "$py_code" "$go_code"; then
    echo "[cmp] $label: exit code mismatch; outputs: $py_out $go_out"
    overall=1
    return 0
  fi

  if diff -u "$py_norm" "$go_norm" >"$diff_out"; then
    echo "[cmp] $label OK"
    rm -f "$diff_out" || true
  else
    echo "[cmp] $label DIFF: $diff_out"
    overall=1
  fi
}

echo
echo "[cmp] rnpath --version"
py_out="$OUT_DIR/version.python.out"
go_out="$OUT_DIR/version.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rnpath.py" --version)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rnpath" --version)"
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
echo "[cmp] rnpath -v --version (flag parsing parity)"
py_out="$OUT_DIR/version_v.python.out"
go_out="$OUT_DIR/version_v.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rnpath.py" -v --version)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rnpath" -v --version)"
if ! require_eq "python exit" "$py_code" 0 || ! require_eq "go exit" "$go_code" 0; then
  overall=1
else
  if diff -u "$py_out" "$go_out" >"$OUT_DIR/version_v.diff"; then
    echo "[cmp] version_v OK"
  else
    echo "[cmp] version_v DIFF: $OUT_DIR/version_v.diff"
    overall=1
  fi
fi

echo
compare_case "table_empty_t" -t
compare_case "table_empty_tj" -t -j
compare_case "table_empty_tjm" -t -j -m 1
compare_case "table_unknown_dest" -t "$valid_dest"

echo
compare_case "rates_empty_r" -r
compare_case "rates_empty_rj" -r -j
compare_case "rates_invalid_len" -r "$invalid_len_dest"

echo
compare_case "drop_invalid_len" -d "$invalid_len_dest"
compare_case "drop_invalid_hex" -d "$invalid_hex_dest"
compare_case "drop_valid_missing" -d "$valid_dest"

echo
compare_case "drop_via_valid_missing_x" -x "$valid_dest"
compare_case "drop_announces_D" -D

echo
if [[ "$overall" -eq 0 ]]; then
  echo "[cmp] OK"
  exit 0
else
  echo "[cmp] FAIL (see $OUT_DIR)"
  exit 1
fi
