#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
TEST_DIR="$ROOT/tests/hand/rnsd/test3"
ARTIFACTS_DIR="$TEST_DIR/.artifacts"
RUN_GO="$ARTIFACTS_DIR/run/go"
RUN_PY="$ARTIFACTS_DIR/run/py"
GO_SERVER_DIR="$RUN_GO/server"
GO_CLIENT_DIR="$RUN_GO/client"
PY_SERVER_DIR="$RUN_PY/server"
PY_CLIENT_DIR="$RUN_PY/client"
BIN_DIR="$ROOT/bin"
RNCP_BIN="$BIN_DIR/rncp"
PYTHON="${PYTHON:-python3}"
RNCP_PY="$ROOT/python/RNS/Utilities/rncp.py"
WORK_DIR="$ARTIFACTS_DIR/run/cross-go"
PROPAGATION_WAIT_SECS="${PROPAGATION_WAIT_SECS:-5}"
RNCP_RETRY_ATTEMPTS="${RNCP_RETRY_ATTEMPTS:-3}"
RNCP_RETRY_DELAY_SECS="${RNCP_RETRY_DELAY_SECS:-5}"

mkdir -p "$ARTIFACTS_DIR" "$BIN_DIR" "$WORK_DIR"

require_config() {
  local cfg="$1"
  if [[ ! -f "$cfg/config" ]]; then
    echo "Missing runtime config: $cfg/config"
    echo "Run both step1 scripts first:"
    echo "  ./tests/hand/rnsd/test3/step1_rnsd_go.sh"
    echo "  ./tests/hand/rnsd/test3/step1_rnsd_py.sh"
    exit 1
  fi
}

sha256_file() {
  shasum -a 256 "$1" | awk '{print $1}'
}

wait_for_listener_hash() {
  local log="$1"
  local dest=""
  for _ in {1..150}; do
    if [[ -f "$log" ]]; then
      dest="$(sed -nE 's/.*rncp listening on <([0-9a-fA-F]+)>.*/\1/p' "$log" | tail -n 1)"
      [[ -n "$dest" ]] && break
    fi
    sleep 0.1
  done
  if [[ -z "$dest" ]]; then
    echo "Could not parse rncp listener destination from $log" >&2
    cat "$log" 2>/dev/null >&2 || true
    return 1
  fi
  echo "$dest"
}

wait_for_file() {
  local path="$1"
  for _ in {1..150}; do
    [[ -s "$path" ]] && return 0
    sleep 0.1
  done
  return 1
}

wait_for_announce_propagation() {
  local dest="$1"
  if [[ "${PROPAGATION_WAIT_SECS}" =~ ^[0-9]+([.][0-9]+)?$ ]] && [[ "$PROPAGATION_WAIT_SECS" != "0" ]]; then
    echo "Waiting ${PROPAGATION_WAIT_SECS}s for announce propagation: $dest"
    sleep "$PROPAGATION_WAIT_SECS"
  fi
}

is_retryable_rncp_failure() {
  local out="$1"
  rg -q "Path not found|No interfaces could process the outbound packet|Link establishment timed out|link not active|connect: connection refused" "$out"
}

run_with_rncp_retries() {
  local out="$1"
  shift

  local code=0
  local attempt=1
  while [[ "$attempt" -le "$RNCP_RETRY_ATTEMPTS" ]]; do
    set +e
    "$@" >"$out" 2>&1
    code=$?
    set -e

    if [[ "$code" == "0" ]]; then
      return 0
    fi

    if [[ "$attempt" -ge "$RNCP_RETRY_ATTEMPTS" ]] || ! is_retryable_rncp_failure "$out"; then
      return "$code"
    fi

    echo "Transient rncp failure, retrying in ${RNCP_RETRY_DELAY_SECS}s (attempt ${attempt}/${RNCP_RETRY_ATTEMPTS})"
    sleep "$RNCP_RETRY_DELAY_SECS"
    attempt=$((attempt + 1))
  done

  return "$code"
}

stop_listener() {
  if [[ -n "${LISTENER_PID:-}" ]] && kill -0 "$LISTENER_PID" 2>/dev/null; then
    kill "$LISTENER_PID" 2>/dev/null || true
    wait "$LISTENER_PID" 2>/dev/null || true
  fi
  LISTENER_PID=""
}

cleanup() {
  stop_listener
}
trap cleanup EXIT INT TERM

require_config "$GO_SERVER_DIR"
require_config "$GO_CLIENT_DIR"
require_config "$PY_SERVER_DIR"
require_config "$PY_CLIENT_DIR"

echo "[1/7] Building Go rncp"
mkdir -p /tmp/go-cache /tmp/go-tmp
GOCACHE=/tmp/go-cache GOTMPDIR=/tmp/go-tmp go build -a -o "$RNCP_BIN" ./cmd/rncp

rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR/go_to_py/src" "$WORK_DIR/go_to_py/recv" "$WORK_DIR/py_to_go/src" "$WORK_DIR/py_to_go/recv"

echo "[2/7] Python listener on Python server node"
LISTENER_LOG="$WORK_DIR/go_to_py/listener.py.log"
PYTHONPATH="$ROOT/python" PYTHONUNBUFFERED=1 "$PYTHON" "$RNCP_PY" --config "$PY_SERVER_DIR" -l -n -b 0 -S -s "$WORK_DIR/go_to_py/recv" >"$LISTENER_LOG" 2>&1 &
LISTENER_PID=$!
DEST="$(wait_for_listener_hash "$LISTENER_LOG")"
echo "Python destination: $DEST"
wait_for_announce_propagation "$DEST"

echo "[3/7] Go client sends file to Python server"
SRC="$WORK_DIR/go_to_py/src/go-to-python.txt"
printf "go-to-python-%s\n" "$(date +%s%N)" >"$SRC"
if ! run_with_rncp_retries "$WORK_DIR/go_to_py/send.go.out" \
  "$RNCP_BIN" -config "$GO_CLIENT_DIR" -S -w 30 "$SRC" "$DEST"; then
  echo "Go client could not send $SRC to Python destination $DEST"
  cat "$LISTENER_LOG"
  cat "$WORK_DIR/go_to_py/send.go.out"
  exit 1
fi
RECV="$WORK_DIR/go_to_py/recv/$(basename "$SRC")"
if ! wait_for_file "$RECV"; then
  echo "Python server did not receive $RECV"
  cat "$LISTENER_LOG"
  cat "$WORK_DIR/go_to_py/send.go.out"
  exit 1
fi
if [[ "$(sha256_file "$SRC")" != "$(sha256_file "$RECV")" ]]; then
  echo "Go->Python payload hash mismatch"
  exit 1
fi
stop_listener

echo "[4/7] Go listener on Go server node"
LISTENER_LOG="$WORK_DIR/py_to_go/listener.go.log"
"$RNCP_BIN" -config "$GO_SERVER_DIR" -l -n -b 0 -S -s "$WORK_DIR/py_to_go/recv" >"$LISTENER_LOG" 2>&1 &
LISTENER_PID=$!
DEST="$(wait_for_listener_hash "$LISTENER_LOG")"
echo "Go destination: $DEST"
wait_for_announce_propagation "$DEST"

echo "[5/7] Python client sends received data back to Go server"
SRC_BACK="$WORK_DIR/py_to_go/src/python-back-to-go.txt"
cp "$RECV" "$SRC_BACK"
if ! run_with_rncp_retries "$WORK_DIR/py_to_go/send.py.out" \
  env PYTHONPATH="$ROOT/python" PYTHONUNBUFFERED=1 \
  "$PYTHON" "$RNCP_PY" --config "$PY_CLIENT_DIR" -S -w 30 "$SRC_BACK" "$DEST"; then
  echo "Python client could not send $SRC_BACK to Go destination $DEST"
  cat "$LISTENER_LOG"
  cat "$WORK_DIR/py_to_go/send.py.out"
  exit 1
fi
RECV_BACK="$WORK_DIR/py_to_go/recv/$(basename "$SRC_BACK")"
if ! wait_for_file "$RECV_BACK"; then
  echo "Go server did not receive $RECV_BACK"
  cat "$LISTENER_LOG"
  cat "$WORK_DIR/py_to_go/send.py.out"
  exit 1
fi
if [[ "$(sha256_file "$SRC")" != "$(sha256_file "$RECV_BACK")" ]]; then
  echo "Python->Go return payload hash mismatch"
  exit 1
fi

echo "[6/7] Artifacts"
echo "original: $SRC"
echo "go->python received: $RECV"
echo "python->go returned: $RECV_BACK"

echo "[7/7] Done"
