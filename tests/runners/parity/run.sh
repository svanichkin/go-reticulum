#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LAYER_DIR="${LOG_DIR:-"$ROOT/tests/artifacts/logs/parity"}"
TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
KEEP_ARTIFACTS="${KEEP_ARTIFACTS:-0}"

if [[ "$KEEP_ARTIFACTS" != "1" ]]; then
  rm -rf "$LAYER_DIR"
fi

mkdir -p "$LAYER_DIR"
OUT_DIR="$LAYER_DIR/$TS"
mkdir -p "$OUT_DIR"
ln -sfn "$OUT_DIR" "$LAYER_DIR/latest"

RUN_NETWORK="${RUN_NETWORK:-1}"

usage() {
  cat <<'EOF'
Usage: tests/runners/parity/run.sh [--no-network]

Parity layer:
  - suite-level Go vs Python comparison
  - tests/runners/parity/cmd/compare_*_py_vs_go.sh
  - tests/runners/parity/cmd/compare_*_two_nodes_py_vs_go.sh (unless --no-network)

Env:
  RUN_NETWORK=1|0  (default 1; --no-network sets 0)
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi
if [[ "${1:-}" == "--no-network" ]]; then
  RUN_NETWORK=0
  shift
fi
if [[ "$#" -ne 0 ]]; then
  usage
  exit 2
fi

echo "[parity] layer=parity"
echo "[parity] root=$ROOT"
echo "[parity] RUN_NETWORK=$RUN_NETWORK"
echo "[parity] out=$OUT_DIR"
echo "[parity] layer_dir=$LAYER_DIR"
echo "[parity] KEEP_ARTIFACTS=$KEEP_ARTIFACTS"

echo
echo "[parity] Suite-level Go vs Python summary"
"$ROOT/tests/runners/parity/python/run.sh" LOG_DIR="$OUT_DIR/python" KEEP_ARTIFACTS="$KEEP_ARTIFACTS" >/dev/null
"$ROOT/tests/runners/go/run.sh" LOG_DIR="$OUT_DIR/go" KEEP_ARTIFACTS="$KEEP_ARTIFACTS" >/dev/null

PY_LOG="$OUT_DIR/python/latest/output.log"
GO_LOG="$OUT_DIR/go/latest/unittest.log"

if [[ -z "${PY_LOG}" || -z "${GO_LOG}" ]]; then
  echo "[parity] missing logs (py=$PY_LOG go=$GO_LOG)"
  exit 2
fi

cp -f "$PY_LOG" "$OUT_DIR/python.output.log"
cp -f "$GO_LOG" "$OUT_DIR/go.output.log"

normalize() {
  grep -Ev \
    -e '^\[[0-9]{4}-[0-9]{2}-[0-9]{2} ' \
    -e 'Mbps|Gbps' \
    -e 'timing min/avg/med/max/mdev' \
    -e 'Max deviation from median' \
    -e '^Sign/validate ' \
    -e '^Testing \(random small|large\) chunk encrypt/decrypt' \
    -e '^\s*Encrypt ' \
    -e '^\s*Decrypt '
}

normalize <"$OUT_DIR/python.output.log" >"$OUT_DIR/python.output.norm.log"
normalize <"$OUT_DIR/go.output.log" >"$OUT_DIR/go.output.norm.log"

echo "[parity] python log: $OUT_DIR/python.output.log"
echo "[parity] go log:     $OUT_DIR/go.output.log"
echo

echo "[parity] python summary:"
grep -E "^(OK$|FAILED \\(|Ran [0-9]+ tests)" "$OUT_DIR/python.output.log" || true
echo

echo "[parity] go summary:"
grep -E "^(ok\\s|FAIL\\s|\\?\\s)" "$OUT_DIR/go.output.log" || true
echo

echo "[parity] diff (normalized, first 200 lines):"
diff -u "$OUT_DIR/python.output.norm.log" "$OUT_DIR/go.output.norm.log" | sed -n '1,200p' || true

echo
echo "[parity] Regression scripts (offline)"
offline_scripts=(
  "$ROOT/tests/runners/parity/cmd/compare_rnid_py_vs_go.sh"
  "$ROOT/tests/runners/parity/cmd/compare_rnpath_py_vs_go.sh"
  "$ROOT/tests/runners/parity/cmd/compare_rnprobe_py_vs_go.sh"
  "$ROOT/tests/runners/parity/cmd/compare_rnx_py_vs_go.sh"
  "$ROOT/tests/runners/parity/cmd/compare_rncp_py_vs_go.sh"
  "$ROOT/tests/runners/parity/cmd/compare_examples_py_vs_go.sh"
  "$ROOT/tests/runners/parity/cmd/compare_rnsd_py_vs_go.sh"
)
for s in "${offline_scripts[@]}"; do
  echo "[parity] $(basename "$s")"
  bash "$s"
done

echo
echo "[parity] Regression scripts (local instances)"
echo "[parity] compare_rnstatus_py_vs_go.sh"
bash "$ROOT/tests/runners/parity/cmd/compare_rnstatus_py_vs_go.sh"

if [[ "$RUN_NETWORK" -eq 1 ]]; then
  echo
  echo "[parity] Regression scripts (two nodes)"
  two_node_scripts=(
    "$ROOT/tests/runners/parity/cmd/compare_rnprobe_two_nodes_py_vs_go.sh"
    "$ROOT/tests/runners/parity/cmd/compare_rncp_two_nodes_py_vs_go.sh"
    "$ROOT/tests/runners/parity/cmd/compare_rnpath_two_nodes_py_vs_go.sh"
    "$ROOT/tests/runners/parity/cmd/compare_rnx_two_nodes_py_vs_go.sh"
  )
  for s in "${two_node_scripts[@]}"; do
    echo "[parity] $(basename "$s")"
    bash "$s"
  done
else
  echo
  echo "[parity] Skipping two-node parity scripts (RUN_NETWORK=0)"
fi

echo
echo "[parity] OK"
