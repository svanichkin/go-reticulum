#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LAYER_DIR="${LOG_DIR:-"$ROOT/tests/artifacts/logs/parity"}"
TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
KEEP_ARTIFACTS="${KEEP_ARTIFACTS:-0}"
PARITY_BIN_DIR="${PARITY_BIN_DIR:-"$ROOT/tests/runners/parity/bin"}"

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
echo "[parity] bin_dir=$PARITY_BIN_DIR"

echo
echo "[parity] Building Go cmd binaries"
mkdir -p "$PARITY_BIN_DIR"
for cmd_dir in "$ROOT"/cmd/*; do
  [[ -d "$cmd_dir" ]] || continue
  cmd_name="$(basename "$cmd_dir")"
  echo "[parity]   go build $cmd_name"
  go build -o "$PARITY_BIN_DIR/$cmd_name" "./cmd/$cmd_name"
done
export PARITY_BIN_DIR
suite_status=0

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
    "$ROOT/tests/runners/parity/cmd/compare_rncp_two_nodes_tcp_py_vs_go.sh"
    "$ROOT/tests/runners/parity/cmd/compare_packet_flows_two_nodes_py_vs_go.sh"
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
exit "$suite_status"
