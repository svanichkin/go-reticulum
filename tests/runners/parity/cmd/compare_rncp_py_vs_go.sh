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

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-25}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rncp"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$GO_BIN_DIR" || true
}
trap cleanup EXIT

echo "[cmp] out=$OUT_DIR"
echo "[cmp] building go rncp..."
go build -o "$GO_BIN_DIR/rncp" ./cmd/rncp

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

require_file_contains() {
  local label="$1"
  local path="$2"
  local needle="$3"
  if ! rg --fixed-strings "$needle" "$path" >/dev/null 2>&1; then
    echo "[cmp] $label: expected '$path' to contain: $needle"
    return 1
  fi
  return 0
}

normalize_output() {
  local src="$1"
  local dst="$2"

  tr '\r' '\n' <"$src" \
    | tr -d '\b' \
    | sed -E 's/[⢄⢂⢁⡁⡈⡐⡠]//g' \
    | sed -E '/^\[[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\] \[[^]]+\][[:space:]]*/d' \
    | awk '/^Traceback / { exit } { print }' \
    | sed -E 's/[[:space:]]+$//; s/[[:space:]]{2,}/ /g' \
    | sed -E '/^$/d' \
    >"$dst"
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

dest_len_hex="$("$PYTHON" -c 'import RNS; print((RNS.Reticulum.TRUNCATED_HASHLENGTH//8)*2)')"
valid_dest="$(printf '%*s' "$dest_len_hex" '' | tr ' ' 'a')"
invalid_len_dest="aa"
invalid_hex_dest="${valid_dest:0:$((dest_len_hex-1))}z"

overall=0

echo
echo "[cmp] rncp --version"
py_out="$OUT_DIR/version.python.out"
go_out="$OUT_DIR/version.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rncp.py" --version)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rncp" --version)"
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
echo "[cmp] rncp -v --version (flag parsing parity)"
py_out="$OUT_DIR/version_v.python.out"
go_out="$OUT_DIR/version_v.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rncp.py" -v --version)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rncp" -v --version)"
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
echo "[cmp] rncp -h (help presence parity)"
py_out="$OUT_DIR/help.python.out"
go_out="$OUT_DIR/help.go.out"
py_code="$(run_capture "$py_out" "$PYTHON" "$ROOT/python/RNS/Utilities/rncp.py" -h)"
go_code="$(run_capture "$go_out" "$GO_BIN_DIR/rncp" -h)"
if ! require_eq "python exit" "$py_code" 0 || ! require_eq "go exit" "$go_code" 0; then
  overall=1
else
  ok=0
  if ! require_file_contains "python help" "$py_out" "Reticulum File Transfer Utility" || ! require_file_contains "go help" "$go_out" "Reticulum File Transfer Utility"; then
    ok=1
  fi
  if ! rg -q -e "Usage:" -e "^usage:" "$py_out"; then
    echo "[cmp] python help: expected help output to contain Usage"
    ok=1
  fi
  if ! rg -q -e "Usage:" -e "^usage:" "$go_out"; then
    echo "[cmp] go help: expected help output to contain Usage"
    ok=1
  fi
  if [[ "$ok" -ne 0 ]]; then
    overall=1
  else
    echo "[cmp] help OK"
  fi
fi

compare_case_exact() {
  local label="$1"
  shift

  local py_out="$OUT_DIR/${label}.python.out"
  local go_out="$OUT_DIR/${label}.go.out"
  local py_norm="$OUT_DIR/${label}.python.norm"
  local go_norm="$OUT_DIR/${label}.go.norm"
  local diff_out="$OUT_DIR/${label}.diff"

  local py_cfg go_cfg py_home go_home
  py_cfg="$(new_run_dir)"
  go_cfg="$(new_run_dir)"
  py_home="$py_cfg/home"
  go_home="$go_cfg/home"
  mkdir -p "$py_home" "$go_home"

  local -a args=("$@")

  local py_code go_code
  py_code="$(run_capture "$py_out" env HOME="$py_home" USERPROFILE="$py_home" \
    "$PYTHON" "$ROOT/python/RNS/Utilities/rncp.py" --config "$py_cfg" "${args[@]}")"
  go_code="$(run_capture "$go_out" env HOME="$go_home" USERPROFILE="$go_home" \
    "$GO_BIN_DIR/rncp" --config "$go_cfg" "${args[@]}")"

  normalize_output "$py_out" "$py_norm"
  normalize_output "$go_out" "$go_norm"

  if ! require_eq "$label exit" "$go_code" "$py_code"; then
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
compare_case_exact "invalid_dest_len" "dummy" "$invalid_len_dest"
compare_case_exact "invalid_dest_hex" "dummy" "$invalid_hex_dest"

echo
echo "[cmp] rncp -p (print identity parity via shared identity file)"
run_dir="$(new_run_dir)"
home_dir="$run_dir/home"
mkdir -p "$home_dir"
id_path="$run_dir/rncp.id"

py_out="$OUT_DIR/print.python.out"
go_out="$OUT_DIR/print.go.out"
py_code="$(run_capture "$py_out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rncp.py" --config "$run_dir" -q -p -i "$id_path")"
go_code="$(run_capture "$go_out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rncp" --config "$run_dir" -q -p -i "$id_path")"
if ! require_eq "python exit" "$py_code" 0 || ! require_eq "go exit" "$go_code" 0; then
  overall=1
else
  normalize_output "$py_out" "$OUT_DIR/print.python.norm"
  normalize_output "$go_out" "$OUT_DIR/print.go.norm"
  if diff -u "$OUT_DIR/print.python.norm" "$OUT_DIR/print.go.norm" >"$OUT_DIR/print.diff"; then
    echo "[cmp] print OK"
    rm -f "$OUT_DIR/print.diff" || true
  else
    echo "[cmp] print DIFF: $OUT_DIR/print.diff"
    overall=1
  fi
fi

echo
echo "[cmp] rncp -l -s (invalid output dir exit code parity)"
run_dir="$(new_run_dir)"
home_dir="$run_dir/home"
mkdir -p "$home_dir"
id_path="$run_dir/rncp.id"
bad_dir="$run_dir/does-not-exist"

py_out="$OUT_DIR/bad_save.python.out"
go_out="$OUT_DIR/bad_save.go.out"
py_code="$(run_capture "$py_out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rncp.py" --config "$run_dir" -q -l -i "$id_path" -s "$bad_dir")"
go_code="$(run_capture "$go_out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rncp" --config "$run_dir" -q -l -i "$id_path" -s "$bad_dir")"
if ! require_eq "python exit" "$py_code" 3 || ! require_eq "go exit" "$go_code" 3; then
  overall=1
else
  if ! require_file_contains "python bad_save" "$py_out" "Output directory not found" || ! require_file_contains "go bad_save" "$go_out" "Output directory not found"; then
    overall=1
  else
    echo "[cmp] bad_save OK"
  fi
fi

echo
echo "[cmp] rncp -l -a (invalid allowed identity exit code parity)"
run_dir="$(new_run_dir)"
home_dir="$run_dir/home"
mkdir -p "$home_dir"
id_path="$run_dir/rncp.id"

py_out="$OUT_DIR/bad_allowed.python.out"
go_out="$OUT_DIR/bad_allowed.go.out"
py_code="$(run_capture "$py_out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rncp.py" --config "$run_dir" -q -l -i "$id_path" -a "$invalid_len_dest")"
go_code="$(run_capture "$go_out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rncp" --config "$run_dir" -q -l -i "$id_path" -a "$invalid_len_dest")"
if ! require_eq "python exit" "$py_code" 1 || ! require_eq "go exit" "$go_code" 1; then
  overall=1
else
  normalize_output "$py_out" "$OUT_DIR/bad_allowed.python.norm"
  normalize_output "$go_out" "$OUT_DIR/bad_allowed.go.norm"
  if diff -u "$OUT_DIR/bad_allowed.python.norm" "$OUT_DIR/bad_allowed.go.norm" >"$OUT_DIR/bad_allowed.diff"; then
    echo "[cmp] bad_allowed OK"
    rm -f "$OUT_DIR/bad_allowed.diff" || true
  else
    echo "[cmp] bad_allowed DIFF: $OUT_DIR/bad_allowed.diff"
    overall=1
  fi
fi

echo
echo "[cmp] rncp -p -i (invalid identity exit code parity)"
run_dir="$(new_run_dir)"
home_dir="$run_dir/home"
mkdir -p "$home_dir"
bad_id="$run_dir/bad.id"
printf "this is not an identity\n" >"$bad_id"

py_out="$OUT_DIR/bad_identity.python.out"
go_out="$OUT_DIR/bad_identity.go.out"
py_code="$(run_capture "$py_out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$PYTHON" "$ROOT/python/RNS/Utilities/rncp.py" --config "$run_dir" -q -p -i "$bad_id")"
go_code="$(run_capture "$go_out" env HOME="$home_dir" USERPROFILE="$home_dir" \
  "$GO_BIN_DIR/rncp" --config "$run_dir" -q -p -i "$bad_id")"
if ! require_eq "python exit" "$py_code" 2 || ! require_eq "go exit" "$go_code" 2; then
  overall=1
else
  if ! require_file_contains "python bad_identity" "$py_out" "Could not load identity for rncp" || ! require_file_contains "go bad_identity" "$go_out" "Could not load identity for rncp"; then
    overall=1
  else
    echo "[cmp] bad_identity OK"
  fi
fi

echo
if [[ "$overall" -eq 0 ]]; then
  echo "[cmp] OK"
  exit 0
else
  echo "[cmp] FAIL (see $OUT_DIR)"
  exit 1
fi
