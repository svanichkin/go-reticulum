#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnid/test4"
ARTIFACTS_DIR="$TEST_DIR/.artifacts"
RUN_DIR="$ARTIFACTS_DIR/run/step6"
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

echo "[1/5] Building Go rnid"
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNID_BIN" ./cmd/rnid

echo "[2/5] Python rnid generates identity"
ID_PATH="$RUN_DIR/py.id"
DATA_PATH="$RUN_DIR/message.txt"
SIG_PATH="$RUN_DIR/message.rsg"
rm -f "$ID_PATH" "$SIG_PATH"
printf "py-sign-%s\n" "$(date +%s%N)" >"$DATA_PATH"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNID_PY" --config "$CFG_DIR" --generate "$ID_PATH" >"$RUN_DIR/py.generate.out" 2>&1

echo "[3/5] Python rnid signs payload"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNID_PY" --config "$CFG_DIR" --identity "$ID_PATH" --sign "$DATA_PATH" --write "$SIG_PATH" --force >"$RUN_DIR/py.sign.out" 2>&1

echo "[4/5] Go rnid validates signature"
"$RNID_BIN" --config "$CFG_DIR" --identity "$ID_PATH" --validate "$SIG_PATH" --read "$DATA_PATH" >"$RUN_DIR/go.validate.out" 2>&1

echo "[5/5] Done"
ID_PRINT="$RUN_DIR/py.print.out"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNID_PY" --config "$CFG_DIR" --identity "$ID_PATH" --print-identity >"$ID_PRINT" 2>&1
ID_HASH="$(extract_identity_hash "$ID_PRINT")"

if ! rg -q "is valid" "$RUN_DIR/go.validate.out"; then
  echo "Go validation output does not confirm a valid signature"
  cat "$RUN_DIR/go.validate.out"
  exit 1
fi

echo "identity hash: $ID_HASH"
echo "cross-check: Python sign -> Go validate OK"
echo "artifacts: $RUN_DIR"
