#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
PYTHON="${PYTHON:-python3}"
PY_RNS=("$PYTHON" "$ROOT/tests/runners/parity/announce/common/python_no_epoll.py")

START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-25}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-5}"
CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-30}"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache" "$ROOT/tests/artifacts/logs"
export GOCACHE="$ROOT/.gocache"
GOTMPDIR="${GOTMPDIR:-$(mktemp -d "$ROOT/.gotmp/run.XXXXXX")}"
export GOTMPDIR
export GOPATH="$ROOT/.gopath"
export GOMODCACHE="$ROOT/.gomodcache"
export PYTHONPATH="${PYTHONPATH:-"$ROOT/python"}"
export PYTHONUNBUFFERED=1

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/parity_announce_${ANNOUNCE_INTERFACE_NAME}_${ANNOUNCE_TOOL_NAME}"
mkdir -p "$OUT_DIR"

BUILD_DIR=""
WORK_DIR="$(mktemp -d "$ROOT/.gotmp/parity-announce-${ANNOUNCE_INTERFACE_NAME}-${ANNOUNCE_TOOL_NAME}.XXXXXX")"
declare -a PIDS=()

announce_prefix() {
  printf '[parity-announce-%s]' "$ANNOUNCE_INTERFACE_NAME"
}

announce_init_bins() {
  local bins=("$@")
  local use_prebuilt=0
  if [[ -n "${PARITY_BIN_DIR:-}" ]]; then
    use_prebuilt=1
    local bin
    for bin in "${bins[@]}"; do
      if [[ ! -x "${PARITY_BIN_DIR}/$bin" ]]; then
        use_prebuilt=0
        break
      fi
    done
  fi

  if [[ "$use_prebuilt" -eq 1 ]]; then
    local bin var
    for bin in "${bins[@]}"; do
      var="GO_${bin^^}"
      printf -v "$var" '%s' "${PARITY_BIN_DIR}/$bin"
    done
    return 0
  fi

  BUILD_DIR="$(mktemp -d "$ROOT/.gotmp/parity-${ANNOUNCE_TOOL_NAME}.XXXXXX")"
  local bin var
  for bin in "${bins[@]}"; do
    var="GO_${bin^^}"
    printf -v "$var" '%s' "$BUILD_DIR/$bin"
    go build -o "$BUILD_DIR/$bin" "./cmd/$bin"
  done
}

stop_proc() {
  local pid="${1:-}"
  if [[ -z "$pid" ]]; then
    return 0
  fi
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    return 0
  fi

  pid_is_zombie() {
    local stat
    stat="$(ps -p "$pid" -o stat= 2>/dev/null || true)"
    [[ "$stat" == Z* ]]
  }

  kill -INT "$pid" >/dev/null 2>&1 || true
  local i
  for i in $(seq 1 $((STOP_TIMEOUT_SECS * 10))); do
    if pid_is_zombie; then
      wait "$pid" 2>/dev/null || true
      return 0
    fi
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      wait "$pid" 2>/dev/null || true
      return 0
    fi
    sleep 0.1
  done

  kill -TERM "$pid" >/dev/null 2>&1 || true
  for i in $(seq 1 $((STOP_TIMEOUT_SECS * 10))); do
    if pid_is_zombie; then
      wait "$pid" 2>/dev/null || true
      return 0
    fi
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      wait "$pid" 2>/dev/null || true
      return 0
    fi
    sleep 0.1
  done

  kill -KILL "$pid" >/dev/null 2>&1 || true
  wait "$pid" 2>/dev/null || true
}

announce_cleanup() {
  local pid
  for pid in "${PIDS[@]}"; do
    stop_proc "$pid"
  done
  pkill -TERM -f '(/rnsd( |$)|RNS/Utilities/rnsd.py)' >/dev/null 2>&1 || true
  sleep 0.2
  pkill -KILL -f '(/rnsd( |$)|RNS/Utilities/rnsd.py)' >/dev/null 2>&1 || true
  rm -rf "$BUILD_DIR" "$WORK_DIR"
}
trap announce_cleanup EXIT

run_capture() {
  local out="$1"
  shift
  local code=0
  set +e
  "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$CMD_TIMEOUT_SECS" -- "$@" >"$out" 2>&1
  code=$?
  set -e
  echo "$code"
}

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

announce_write_local_client_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local shared_port="$3"
  local control_port="$4"
  cat >"$cfg_dir/config" <<EOF
[reticulum]
  share_instance = Yes
  instance_name = $instance_name
  shared_instance_type = tcp
  shared_instance_port = $shared_port
  instance_control_port = $control_port
  enable_transport = No

[logging]
  loglevel = 7
EOF
}

announce_start_go_rnsd() {
  local cfg_dir="$1"
  local log_path="$2"
  : >"$log_path"
  env HOME="$cfg_dir" USERPROFILE="$cfg_dir" RNS_EXIT_WAIT_TIMEOUT=1 \
    "$GO_RNSD" --config "$cfg_dir" -v >"$log_path" 2>&1 &
  local pid=$!
  PIDS+=("$pid")
  echo "$pid"
}

announce_start_py_rnsd() {
  local cfg_dir="$1"
  local log_path="$2"
  : >"$log_path"
  env HOME="$cfg_dir" USERPROFILE="$cfg_dir" \
    "${PY_RNS[@]}" "$ROOT/python/RNS/Utilities/rnsd.py" --config "$cfg_dir" -v >"$log_path" 2>&1 &
  local pid=$!
  PIDS+=("$pid")
  echo "$pid"
}

announce_run_generate_identity() {
  local impl="$1"
  local cfg_dir="$2"
  local identity_path="$3"
  local out="$4"
  local args=(--config "$cfg_dir" --generate "$identity_path")
  if [[ "${ANNOUNCE_GENERATE_QUIET:-0}" == "1" ]]; then
    args+=(-q)
  fi
  if [[ "$impl" == "go" ]]; then
    run_capture "$out" env HOME="$cfg_dir" USERPROFILE="$cfg_dir" "$GO_RNID" "${args[@]}"
  else
    run_capture "$out" env HOME="$cfg_dir" USERPROFILE="$cfg_dir" \
      "${PY_RNS[@]}" "$ROOT/python/RNS/Utilities/rnid.py" "${args[@]}"
  fi
}

if ! declare -F announce_wait_sender_ready >/dev/null; then
  announce_wait_sender_ready() {
    return 0
  }
fi

announce_run_four_directions() {
  local modes="standalone local"
  if declare -F announce_modes_for_tool >/dev/null; then
    modes="$(announce_modes_for_tool "$ANNOUNCE_TOOL_NAME")"
  fi

  overall=0
  local sender mode
  for sender in go py; do
    for mode in $modes; do
      if ! run_direction "$sender" "$mode"; then
        overall=1
      fi
    done
  done
}

announce_prepare_run() {
  echo "$(announce_prefix) out=$OUT_DIR"
  echo "$(announce_prefix) killing stale rnsd processes"
  pkill -TERM -f '(/rnsd( |$)|RNS/Utilities/rnsd.py)' >/dev/null 2>&1 || true
  sleep 0.2
  pkill -KILL -f '(/rnsd( |$)|RNS/Utilities/rnsd.py)' >/dev/null 2>&1 || true

  if declare -F announce_interface_setup >/dev/null; then
    announce_interface_setup
  fi
}
