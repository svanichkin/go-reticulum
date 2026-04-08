#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LAYER_DIR="${LOG_DIR:-"$ROOT/tests/artifacts/logs/go"}"
TS="$(date +"%Y%m%d-%H%M%S")"
KEEP_ARTIFACTS="${KEEP_ARTIFACTS:-0}"

if [[ "$KEEP_ARTIFACTS" != "1" ]]; then
  rm -rf "$LAYER_DIR"
fi

mkdir -p "$LAYER_DIR"
OUT_DIR="$LAYER_DIR/$TS"
mkdir -p "$OUT_DIR"
ln -sfn "$OUT_DIR" "$LAYER_DIR/latest"

export RNS_INTEGRATION="${RNS_INTEGRATION:-1}"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache"
export GOCACHE="$ROOT/.gocache"
export GOTMPDIR="$ROOT/.gotmp"
export GOPATH="$ROOT/.gopath"
export GOMODCACHE="$ROOT/.gomodcache"

echo "[go] layer=go-tests"
echo "[go] root=$ROOT"
echo "[go] out=$OUT_DIR"
echo "[go] layer_dir=$LAYER_DIR"
echo "[go] KEEP_ARTIFACTS=$KEEP_ARTIFACTS"
echo "[go] RNS_INTEGRATION=$RNS_INTEGRATION"
echo "[go] RUN_SLOW_TESTS=${RUN_SLOW_TESTS:-}"
echo "[go] GOCACHE=$GOCACHE"
echo "[go] GOTMPDIR=$GOTMPDIR"
echo "[go] GOPATH=$GOPATH"
echo "[go] GOMODCACHE=$GOMODCACHE"
echo

cd "$ROOT"

set -x
go test ./... -count=1 2>&1 | tee "$OUT_DIR/output.log"
go test ./... -json -count=1 2>/dev/null | python3 "$ROOT/tests/support/tools/go_unittest_report.py" | tee "$OUT_DIR/unittest.log"
set +x

echo
echo "[go] done"
