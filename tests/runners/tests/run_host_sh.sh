#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LAYER_DIR="${LOG_DIR:-"$ROOT/tests/artifacts/logs/host-sh"}"
TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
KEEP_ARTIFACTS="${KEEP_ARTIFACTS:-1}"

RUN_SMOKE=1
RUN_FLAGS=1

usage() {
  cat <<'EOF'
Usage: tests/runners/tests/run_host_sh.sh [--smoke-only|--flags-only]

Sequentially runs local shell test runners outside Docker:
  - tests/runners/smoke/run.sh
  - tests/runners/flags/run.sh

Env:
  LOG_DIR=/path/to/log/root
  TS=YYYYmmdd-HHMMSS
  KEEP_ARTIFACTS=1|0
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --smoke-only)
      RUN_SMOKE=1
      RUN_FLAGS=0
      shift
      ;;
    --flags-only)
      RUN_SMOKE=0
      RUN_FLAGS=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage
      exit 2
      ;;
  esac
done

if [[ "$KEEP_ARTIFACTS" != "1" ]]; then
  rm -rf "$LAYER_DIR"
fi

mkdir -p "$LAYER_DIR"
OUT_DIR="$LAYER_DIR/$TS"
mkdir -p "$OUT_DIR"
ln -sfn "$OUT_DIR" "$LAYER_DIR/latest"

echo "[host-sh] root=$ROOT"
echo "[host-sh] out=$OUT_DIR"
echo "[host-sh] KEEP_ARTIFACTS=$KEEP_ARTIFACTS"
echo "[host-sh] RUN_SMOKE=$RUN_SMOKE"
echo "[host-sh] RUN_FLAGS=$RUN_FLAGS"
echo

run_one() {
  local name="$1"
  local script="$2"
  local logfile="$OUT_DIR/$name.log"

  echo "[host-sh] START $name"
  if bash "$script" 2>&1 | tee "$logfile"; then
    echo "[host-sh] PASS  $name"
    echo "PASS $name" >> "$OUT_DIR/summary.txt"
  else
    local status=$?
    echo "[host-sh] FAIL  $name (exit=$status)"
    echo "FAIL $name exit=$status" >> "$OUT_DIR/summary.txt"
    return "$status"
  fi
  echo
}

status=0

if [[ "$RUN_SMOKE" -eq 1 ]]; then
  run_one "smoke" "$ROOT/tests/runners/smoke/run.sh" || status=$?
fi

if [[ "$status" -eq 0 && "$RUN_FLAGS" -eq 1 ]]; then
  run_one "flags" "$ROOT/tests/runners/flags/run.sh" || status=$?
fi

echo "[host-sh] summary:"
cat "$OUT_DIR/summary.txt" 2>/dev/null || true

exit "$status"
