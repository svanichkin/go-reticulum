#!/usr/bin/env bash
set -euo pipefail

ANNOUNCE_RNSD_LABEL="${ANNOUNCE_RNSD_LABEL:-${ANNOUNCE_INTERFACE_LABEL:-}}"

if [[ -z "${ANNOUNCE_INTERFACE_NAME:-}" || -z "${ANNOUNCE_RNSD_LABEL:-}" || -z "${ANNOUNCE_OBSERVE_PATTERN:-}" ]]; then
  echo "ANNOUNCE_INTERFACE_NAME, ANNOUNCE_RNSD_LABEL and ANNOUNCE_OBSERVE_PATTERN are required" >&2
  exit 2
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
PYTHON="${PYTHON:-python3}"
PY_RNS=("$PYTHON" "$ROOT/tests/runners/parity/announce/common/python_no_epoll.py")

START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-20}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-5}"

PARITY_GOCACHE_ROOT="${PARITY_GOCACHE_ROOT:-$ROOT/.gocache}"
PARITY_GOTMP_ROOT="${PARITY_GOTMP_ROOT:-$ROOT/.gotmp}"
PARITY_GOPATH_ROOT="${PARITY_GOPATH_ROOT:-$ROOT/.gopath}"
PARITY_GOMODCACHE_ROOT="${PARITY_GOMODCACHE_ROOT:-$ROOT/.gomodcache}"
PARITY_BUILD_ROOT="${PARITY_BUILD_ROOT:-$PARITY_GOTMP_ROOT}"
PARITY_WORK_ROOT="${PARITY_WORK_ROOT:-$PARITY_GOTMP_ROOT}"
PARITY_LOG_ROOT="${PARITY_LOG_ROOT:-$ROOT/tests/artifacts/logs}"

mkdir -p \
  "$PARITY_GOCACHE_ROOT" \
  "$PARITY_GOTMP_ROOT" \
  "$PARITY_GOPATH_ROOT" \
  "$PARITY_GOMODCACHE_ROOT" \
  "$PARITY_BUILD_ROOT" \
  "$PARITY_WORK_ROOT" \
  "$PARITY_LOG_ROOT"
export GOCACHE="$PARITY_GOCACHE_ROOT"
GOTMPDIR="${GOTMPDIR:-$(mktemp -d "$PARITY_GOTMP_ROOT/run.XXXXXX")}"
export GOTMPDIR
export GOPATH="$PARITY_GOPATH_ROOT"
export GOMODCACHE="$PARITY_GOMODCACHE_ROOT"
export PYTHONPATH="${PYTHONPATH:-"$ROOT/python"}"
export PYTHONUNBUFFERED=1

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$PARITY_LOG_ROOT/$TS/parity_announce_${ANNOUNCE_INTERFACE_NAME}_rnsd"
mkdir -p "$OUT_DIR"

BUILD_DIR=""
if [[ -n "${PARITY_BIN_DIR:-}" ]] && [[ -x "${PARITY_BIN_DIR}/rnsd" ]]; then
  GO_RNSD="${PARITY_BIN_DIR}/rnsd"
else
  BUILD_DIR="$(mktemp -d "$PARITY_BUILD_ROOT/parity-rnsd.XXXXXX")"
  GO_RNSD="$BUILD_DIR/rnsd"
  go build -o "$GO_RNSD" ./cmd/rnsd
fi

WORK_DIR="$(mktemp -d "$PARITY_WORK_ROOT/parity-announce-${ANNOUNCE_INTERFACE_NAME}-rnsd.XXXXXX")"
PY_DIR="$WORK_DIR/py"
GO_DIR="$WORK_DIR/go"
mkdir -p "$PY_DIR" "$GO_DIR"

PY_LOG="$OUT_DIR/py.rnsd.log"
GO_LOG="$OUT_DIR/go.rnsd.log"
py_pid=""
go_pid=""

stop_proc() {
  local pid="${1:-}"
  if [[ -z "$pid" ]]; then
    return 0
  fi
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    return 0
  fi

  kill -INT "$pid" >/dev/null 2>&1 || true
  local i
  for i in $(seq 1 $((STOP_TIMEOUT_SECS * 10))); do
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      wait "$pid" 2>/dev/null || true
      return 0
    fi
    sleep 0.1
  done

  kill -TERM "$pid" >/dev/null 2>&1 || true
  for i in $(seq 1 $((STOP_TIMEOUT_SECS * 10))); do
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      wait "$pid" 2>/dev/null || true
      return 0
    fi
    sleep 0.1
  done

  kill -KILL "$pid" >/dev/null 2>&1 || true
  wait "$pid" 2>/dev/null || true
}

cleanup() {
  stop_proc "$py_pid"
  stop_proc "$go_pid"
  pkill -TERM -f '(/rnsd( |$)|RNS/Utilities/rnsd.py)' >/dev/null 2>&1 || true
  sleep 0.2
  pkill -KILL -f '(/rnsd( |$)|RNS/Utilities/rnsd.py)' >/dev/null 2>&1 || true
  rm -rf "$BUILD_DIR" "$WORK_DIR"
}
trap cleanup EXIT

wait_for_file_contains() {
  local timeout="$1"
  local path="$2"
  local pattern="$3"
  local start
  start="$(date +%s)"
  while true; do
    if [[ -f "$path" ]] && rg -q "$pattern" "$path"; then
      return 0
    fi
    local now
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.1
  done
}

wait_for_new_log_match() {
  local timeout="$1"
  local path="$2"
  local from_line="$3"
  local pattern="$4"
  local start
  start="$(date +%s)"
  while true; do
    if [[ -f "$path" ]] && tail -n +"$from_line" "$path" | rg -q "$pattern"; then
      return 0
    fi
    local now
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.1
  done
}

start_py_rnsd() {
  local mode="${ANNOUNCE_PY_RNSD_MODE:-direct}"
  if [[ "$mode" == "wrapper" ]]; then
    "${PY_RNS[@]}" "$ROOT/python/RNS/Utilities/rnsd.py" --config "$PY_DIR" -v >>"$PY_LOG" 2>&1 &
  else
    "$PYTHON" "$ROOT/python/RNS/Utilities/rnsd.py" --config "$PY_DIR" -v >>"$PY_LOG" 2>&1 &
  fi
  echo "$!"
}

echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] out=$OUT_DIR"
if declare -F announce_setup_interface >/dev/null; then
  announce_setup_interface
elif declare -F announce_interface_setup >/dev/null; then
  announce_interface_setup
fi
echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] killing stale rnsd processes"
pkill -TERM -f '(/rnsd( |$)|RNS/Utilities/rnsd.py)' >/dev/null 2>&1 || true
sleep 0.2
pkill -KILL -f '(/rnsd( |$)|RNS/Utilities/rnsd.py)' >/dev/null 2>&1 || true

announce_write_rnsd_configs "$PY_DIR" "$GO_DIR"

echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] start python rnsd ($ANNOUNCE_RNSD_LABEL only)"
: >"$PY_LOG"
py_pid="$(start_py_rnsd)"

if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$PY_LOG" "Started rnsd version"; then
  echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] python rnsd did not start; see $PY_LOG"
  exit 1
fi

echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] start go rnsd ($ANNOUNCE_RNSD_LABEL only)"
: >"$GO_LOG"
RNS_EXIT_WAIT_TIMEOUT=1 "$GO_RNSD" --config "$GO_DIR" -v >"$GO_LOG" 2>&1 &
go_pid=$!

if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$GO_LOG" "Started rnsd version"; then
  echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] go rnsd did not start; see $GO_LOG"
  exit 1
fi

echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] wait for go announce to appear in python logs"
if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$PY_LOG" "$ANNOUNCE_OBSERVE_PATTERN"; then
  echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] python did not observe go announce; see $PY_LOG"
  exit 1
fi

echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] restart python and wait for python announce in go logs"
go_log_from_line=$(( $(wc -l <"$GO_LOG") + 1 ))
stop_proc "$py_pid"
py_pid=""
sleep 1

py_pid="$(start_py_rnsd)"

if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$PY_LOG" "Started rnsd version"; then
  echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] restarted python rnsd did not start; see $PY_LOG"
  exit 1
fi

if ! wait_for_new_log_match "$START_TIMEOUT_SECS" "$GO_LOG" "$go_log_from_line" "$ANNOUNCE_OBSERVE_PATTERN"; then
  echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] go did not observe announce from restarted python rnsd; see $GO_LOG"
  exit 1
fi

echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] rnsd $ANNOUNCE_RNSD_LABEL announce checks passed"
