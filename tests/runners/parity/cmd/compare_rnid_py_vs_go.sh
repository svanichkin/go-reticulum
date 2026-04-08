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

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnid"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$GO_BIN_DIR" || true
}
trap cleanup EXIT

echo "[cmp] out=$OUT_DIR"
echo "[cmp] building go rnid..."
go build -o "$GO_BIN_DIR/rnid" ./cmd/rnid

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
echo "[cmp] done (out=$OUT_DIR)"
exit "$overall"
