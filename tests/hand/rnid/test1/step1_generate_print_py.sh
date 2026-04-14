#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnid/test4"
ARTIFACTS_DIR="$TEST_DIR/.artifacts"
RUN_DIR="$ARTIFACTS_DIR/run/step2"
CFG_DIR="$RUN_DIR/config"
BIN_DIR="$ROOT/bin"
RNID_BIN="$BIN_DIR/rnid"
PYTHON="${PYTHON:-python3}"
RNID_PY="$ROOT/python/RNS/Utilities/rnid.py"

mkdir -p "$ARTIFACTS_DIR" "$RUN_DIR" "$CFG_DIR" "$BIN_DIR" /tmp/go-cache /tmp/go-tmp
cp "$TEST_DIR/config" "$CFG_DIR/config"

extract_identity_hash() {
  local file="$1"
  sed -nE 's/.*Loaded Identity <?([0-9a-fA-F]+)>?.*/\1/p' "$file" | tail -n 1
}

echo "[1/4] Building Go rnid"
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNID_BIN" ./cmd/rnid

echo "[2/4] Python rnid generates identity"
ID_PATH="$RUN_DIR/py.id"
rm -f "$ID_PATH"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNID_PY" --config "$CFG_DIR" --generate "$ID_PATH" >"$RUN_DIR/py.generate.out" 2>&1

echo "[3/4] Go rnid prints generated identity"
"$RNID_BIN" --config "$CFG_DIR" --identity "$ID_PATH" --print-identity >"$RUN_DIR/go.print.out" 2>&1

echo "[4/4] Done"
PY_PRINT_OUT="$RUN_DIR/py.print.out"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNID_PY" --config "$CFG_DIR" --identity "$ID_PATH" --print-identity >"$PY_PRINT_OUT" 2>&1

GO_HASH="$(extract_identity_hash "$RUN_DIR/go.print.out")"
PY_HASH="$(extract_identity_hash "$PY_PRINT_OUT")"

if [[ -z "$GO_HASH" || -z "$PY_HASH" ]]; then
  echo "Could not extract identity hash from rnid output"
  echo "go hash: ${GO_HASH:-<empty>}"
  echo "py hash: ${PY_HASH:-<empty>}"
  exit 1
fi

if [[ "$GO_HASH" != "$PY_HASH" ]]; then
  echo "Identity hash mismatch"
  echo "go hash: $GO_HASH"
  echo "py hash: $PY_HASH"
  exit 1
fi

echo "identity hash: $GO_HASH"
echo "cross-check: Python generate -> Go print OK"
echo "artifacts: $RUN_DIR"
