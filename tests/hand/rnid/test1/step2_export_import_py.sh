#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnid/test4"
ARTIFACTS_DIR="$TEST_DIR/.artifacts"
RUN_DIR="$ARTIFACTS_DIR/run/step4"
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

extract_exported_identity() {
  local file="$1"
  sed -nE 's/.*Exported Identity[[:space:]]*:[[:space:]]*([A-Za-z0-9_=-]+).*/\1/p' "$file" | tail -n 1
}

echo "[1/5] Building Go rnid"
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNID_BIN" ./cmd/rnid

echo "[2/5] Python rnid generates identity"
SRC_ID="$RUN_DIR/py.id"
DST_ID="$RUN_DIR/imported-by-go.id"
EXPORT_TXT="$RUN_DIR/py.export.txt"
rm -f "$SRC_ID" "$DST_ID" "$EXPORT_TXT"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNID_PY" --config "$CFG_DIR" --generate "$SRC_ID" >"$RUN_DIR/py.generate.out" 2>&1

echo "[3/5] Python rnid exports identity"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNID_PY" --config "$CFG_DIR" --identity "$SRC_ID" --export >"$RUN_DIR/py.export.out" 2>&1
extract_exported_identity "$RUN_DIR/py.export.out" >"$EXPORT_TXT"

if [[ ! -s "$EXPORT_TXT" ]]; then
  echo "Could not extract exported identity from Python output"
  cat "$RUN_DIR/py.export.out"
  exit 1
fi

echo "[4/5] Go rnid imports exported identity"
"$RNID_BIN" --config "$CFG_DIR" --import "$(tr -d '\n' <"$EXPORT_TXT")" --write "$DST_ID" >"$RUN_DIR/go.import.out" 2>&1

echo "[5/5] Go rnid prints imported identity"
"$RNID_BIN" --config "$CFG_DIR" --identity "$DST_ID" --print-identity >"$RUN_DIR/go.print.out" 2>&1

SRC_PRINT="$RUN_DIR/py.print.out"
PYTHONPATH="$ROOT/python" "$PYTHON" "$RNID_PY" --config "$CFG_DIR" --identity "$SRC_ID" --print-identity >"$SRC_PRINT" 2>&1

SRC_HASH="$(extract_identity_hash "$SRC_PRINT")"
IMPORTED_HASH="$(extract_identity_hash "$RUN_DIR/go.print.out")"

if [[ -z "$SRC_HASH" || -z "$IMPORTED_HASH" ]]; then
  echo "Could not extract identity hash from rnid output"
  echo "source hash: ${SRC_HASH:-<empty>}"
  echo "imported hash: ${IMPORTED_HASH:-<empty>}"
  exit 1
fi

if [[ "$SRC_HASH" != "$IMPORTED_HASH" ]]; then
  echo "Identity hash mismatch after Python export -> Go import"
  echo "source hash: $SRC_HASH"
  echo "imported hash: $IMPORTED_HASH"
  exit 1
fi

echo "identity hash: $SRC_HASH"
echo "cross-check: Python export -> Go import OK"
echo "artifacts: $RUN_DIR"
