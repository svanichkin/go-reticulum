#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LAYER_DIR="${LOG_DIR:-"$ROOT/tests/artifacts/logs/smoke"}"
TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
KEEP_ARTIFACTS="${KEEP_ARTIFACTS:-0}"

if [[ "$KEEP_ARTIFACTS" != "1" ]]; then
  rm -rf "$LAYER_DIR"
fi

mkdir -p "$LAYER_DIR"
OUT_DIR="$LAYER_DIR/$TS"
mkdir -p "$OUT_DIR"
ln -sfn "$OUT_DIR" "$LAYER_DIR/latest"

echo "[cmd-smoke] layer=smoke"
echo "[cmd-smoke] root=$ROOT"
echo "[cmd-smoke] out=$OUT_DIR"
echo "[cmd-smoke] layer_dir=$LAYER_DIR"
echo "[cmd-smoke] KEEP_ARTIFACTS=$KEEP_ARTIFACTS"
echo

scripts=(
  "$ROOT/tests/runners/smoke/cmd/run_rncp_smoke.sh"
  "$ROOT/tests/runners/smoke/cmd/run_rnid_smoke.sh"
  "$ROOT/tests/runners/smoke/cmd/run_rnir_smoke.sh"
  "$ROOT/tests/runners/smoke/cmd/run_rnodeconf_smoke.sh"
  "$ROOT/tests/runners/smoke/cmd/run_rnpath_smoke.sh"
  "$ROOT/tests/runners/smoke/cmd/run_rnprobe_smoke.sh"
  "$ROOT/tests/runners/smoke/cmd/run_rnsd_smoke.sh"
  "$ROOT/tests/runners/smoke/cmd/run_rnstatus_smoke.sh"
  "$ROOT/tests/runners/smoke/cmd/run_rnx_smoke.sh"
)

for s in "${scripts[@]}"; do
  echo "[cmd-smoke] $(basename "$s")"
  bash "$s" 2>&1 | tee "$OUT_DIR/$(basename "$s").log"
  echo
done

echo "[cmd-smoke] done"
