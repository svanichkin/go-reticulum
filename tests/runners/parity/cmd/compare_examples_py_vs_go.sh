#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"

LOG_DIR="${LOG_DIR:-"$ROOT/tests/artifacts/logs"}"
TS="$(date +"%Y%m%d-%H%M%S")"
OUT_DIR="$LOG_DIR/$TS/compare_examples"
mkdir -p "$OUT_DIR"

PYTHON="${PYTHON:-python3}"

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-90}"
READY_TIMEOUT_SECS="${READY_TIMEOUT_SECS:-25}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-6}"

# The Python Reticulum sources live in "$ROOT/python".
export PYTHONPATH="${PYTHONPATH:-"$ROOT/python"}"

# Ensure subprocess calls to "python" work if the examples spawn anything.
export PATH="$ROOT/tests/support/bin:$PATH"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache"
export GOCACHE="$ROOT/.gocache"
GOTMPDIR="${GOTMPDIR:-$(mktemp -d "$ROOT/.gotmp/run.XXXXXX")}"
export GOTMPDIR
export GOPATH="$ROOT/.gopath"
export GOMODCACHE="$ROOT/.gomodcache"

declare -a CHILD_PIDS=()
declare -a CHILD_FIFOS=()
declare -a CHILD_KEEPS=()
declare -a CHILD_TMPDIRS=()

cleanup() {
  local pid
  for pid in "${CHILD_PIDS[@]:-}"; do
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" >/dev/null 2>&1; then
      kill -TERM "$pid" >/dev/null 2>&1 || true
      wait_gone "$STOP_TIMEOUT_SECS" "$pid" || true
      kill -KILL "$pid" >/dev/null 2>&1 || true
    fi
  done

  for pid in "${CHILD_KEEPS[@]:-}"; do
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" >/dev/null 2>&1; then
      kill -TERM "$pid" >/dev/null 2>&1 || true
      wait_gone "$STOP_TIMEOUT_SECS" "$pid" || true
      kill -KILL "$pid" >/dev/null 2>&1 || true
    fi
  done

  local fifo
  for fifo in "${CHILD_FIFOS[@]:-}"; do
    [[ -n "${fifo:-}" ]] && rm -f "$fifo" || true
  done

  local tmpdir
  for tmpdir in "${CHILD_TMPDIRS[@]:-}"; do
    [[ -n "${tmpdir:-}" ]] && rm -rf "$tmpdir" || true
  done

  rm -rf "$GOTMPDIR" || true
}
trap cleanup EXIT

GO_EXAMPLES_BIN="$ROOT/tests/support/bin/go_examples"
mkdir -p "$GO_EXAMPLES_BIN"

echo "[cmp] root=$ROOT"
echo "[cmp] out=$OUT_DIR"
echo "[cmp] PYTHON=$PYTHON"
echo "[cmp] PYTHONPATH=$PYTHONPATH"
echo "[cmp] GO_EXAMPLES_BIN=$GO_EXAMPLES_BIN"
echo "[cmp] CMD_TIMEOUT_SECS=$CMD_TIMEOUT_SECS READY_TIMEOUT_SECS=$READY_TIMEOUT_SECS STOP_TIMEOUT_SECS=$STOP_TIMEOUT_SECS"
echo

pick_free_port() {
  "$PYTHON" - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
}

new_smoke_config_dir() {
  local name="$1"
  local run_dir
  run_dir="$(mktemp -d)"
  CHILD_TMPDIRS+=("$run_dir")
  local sip cip
  sip="$(pick_free_port)"
  cip="$(pick_free_port)"

  cat >"$run_dir/config" <<EOF
[reticulum]
  share_instance = Yes
  instance_name = ${name}
  shared_instance_type = tcp
  shared_instance_port = ${sip}
  instance_control_port = ${cip}
  enable_transport = No

[logging]
  loglevel = 4

[interfaces]
  # No physical interfaces for example tests. The shared instance (LocalInterface IPC)
  # is enough for local end-to-end example flows.
  [[Noop]]
    type = UDPInterface
    enabled = no
    listen_ip = 127.0.0.1
    listen_port = 0
    forward_ip = 127.0.0.1
    forward_port = 0
EOF

  echo "$run_dir"
}

wait_for_rg() {
  local timeout="$1"
  local file="$2"
  local pattern="$3"

  local start now
  start="$(date +%s)"
  while true; do
    if rg -q --no-messages -- "$pattern" "$file" 2>/dev/null; then
      return 0
    fi
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.1
  done
}

extract_hash() {
  local file="$1"
  rg -o "<[0-9a-f]{32}>" "$file" | head -n 1 | tr -d '<>'
}

normalize_log_to_events() {
  local in="$1"
  local out="$2"

  # Keep only semantically relevant lines and strip dynamic prefixes/hashes.
  sed -E \
    -e 's/^\\[[0-9]{4}-[0-9]{2}-[0-9]{2}[^]]*\\]\\s*\\[[^]]+\\]\\s*//g' \
    -e 's/<[0-9a-f]{32}>/<HASH>/g' \
    -e 's/[[:space:]]+$//g' \
    "$in" \
    | rg -n --no-messages \
      "running|Sent announce|Client connected|Client disconnected|Link established|Received data|Valid reply received|Got response for request|Ready!|Statistics|download|Resource|The resource" \
      | sed -E 's/^[0-9]+://g' \
      >"$out" || true
}

start_proc_with_fifo() {
  local log="$1"
  local fifo="$2"
  shift 2

  rm -f "$fifo"
  mkfifo "$fifo"
  # Keep a writer attached so opening the read side for the child does not deadlock.
  tail -f /dev/null >"$fifo" 2>/dev/null &
  local keep_pid=$!

  "$@" <"$fifo" >"$log" 2>&1 &
  local pid=$!

  CHILD_PIDS+=("$pid")
  CHILD_KEEPS+=("$keep_pid")
  CHILD_FIFOS+=("$fifo")

  echo "$pid $fifo $keep_pid"
}

send_line_fd() {
  local fifo="$1"
  shift
  printf "%s\n" "$*" >"$fifo"
}

wait_gone() {
  local timeout="$1"
  local pid="$2"
  local start now
  start="$(date +%s)"
  while kill -0 "$pid" >/dev/null 2>&1; do
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.1
  done
  return 0
}

stop_proc() {
  local pid="$1"
  local _fifo_handle="$2"
  local fifo="$3"

  if kill -0 "$pid" >/dev/null 2>&1; then
    kill -INT "$pid" >/dev/null 2>&1 || true
    if ! wait_gone "$STOP_TIMEOUT_SECS" "$pid"; then
      kill -TERM "$pid" >/dev/null 2>&1 || true
      if ! wait_gone "$STOP_TIMEOUT_SECS" "$pid"; then
        kill -KILL "$pid" >/dev/null 2>&1 || true
        wait_gone "$STOP_TIMEOUT_SECS" "$pid" || true
      fi
    fi
  fi

  # Terminate the helper that keeps the FIFO write side open.
  pkill -f "tail -f /dev/null >$fifo" >/dev/null 2>&1 || true

  rm -f "$fifo"
}

run_simple_enter_smoke() {
  local label="$1"
  local out_dir="$2"
  local cfg_dir="$3"
  shift 3

  local log="$out_dir/$label.log"
  local fifo="$out_dir/$label.stdin"
  local meta pid fd
  meta="$(start_proc_with_fifo "$log" "$fifo" "$@")"
  pid="$(echo "$meta" | awk '{print $1}')"
  fd="$(echo "$meta" | awk '{print $2}')"

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$log" "running"; then
    echo "[cmp] $label did not start; log: $log"
    stop_proc "$pid" "$fd" "$fifo"
    return 1
  fi

  send_line_fd "$fd" ""
  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$log" "Sent announce"; then
    echo "[cmp] $label did not send an announce; log: $log"
    stop_proc "$pid" "$fd" "$fifo"
    return 1
  fi

  stop_proc "$pid" "$fd" "$fifo"
  normalize_log_to_events "$log" "$out_dir/$label.events"
  return 0
}

run_timeout_cmd_smoke() {
  local label="$1"
  local out_dir="$2"
  local cfg_dir="$3"
  local ready_re="$4"
  local success_re="$5"
  local cmd_str="$6"

  local log="$out_dir/$label.log"
  local fifo="$out_dir/$label.stdin"
  local meta pid fd
  meta="$(start_proc_with_fifo "$log" "$fifo" bash -c "timeout $CMD_TIMEOUT_SECS $cmd_str")"
  pid="$(echo "$meta" | awk '{print $1}')"
  fd="$(echo "$meta" | awk '{print $2}')"

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$log" "$ready_re"; then
    echo "[cmp] $label did not start; log: $log"
    stop_proc "$pid" "$fd" "$fifo"
    return 1
  fi

  send_line_fd "$fd" "hello"
  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$log" "$success_re"; then
    echo "[cmp] $label did not reach expected output; log: $log"
    stop_proc "$pid" "$fd" "$fifo"
    return 1
  fi

  stop_proc "$pid" "$fd" "$fifo"
  normalize_log_to_events "$log" "$out_dir/$label.events"
  return 0
}

run_server_client_smoke() {
  local name="$1"
  local out_dir="$2"
  local cfg_dir="$3"
  local server_ready_re="$4"
  local client_ready_re="$5"
  local client_success_re="$6"
  local server_cmd_str="$7"
  local client_cmd_prefix_str="$8"

  local server_log="$out_dir/${name}.server.log"
  local server_fifo="$out_dir/${name}.server.stdin"
  local client_log="$out_dir/${name}.client.log"
  local client_fifo="$out_dir/${name}.client.stdin"

  local server_meta server_pid server_fd
  server_meta="$(start_proc_with_fifo "$server_log" "$server_fifo" bash -c "$server_cmd_str")"
  server_pid="$(echo "$server_meta" | awk '{print $1}')"
  server_fd="$(echo "$server_meta" | awk '{print $2}')"

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$server_log" "$server_ready_re"; then
    echo "[cmp] $name server did not become ready; log: $server_log"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  local hash
  hash="$(extract_hash "$server_log")"
  if [[ -z "$hash" ]]; then
    echo "[cmp] $name server did not print a destination hash; log: $server_log"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  # Trigger one announce to avoid waiting for path discovery.
  send_line_fd "$server_fd" ""

  local client_meta client_pid client_fd
  client_meta="$(start_proc_with_fifo "$client_log" "$client_fifo" bash -c "$client_cmd_prefix_str \"$hash\"")"
  client_pid="$(echo "$client_meta" | awk '{print $1}')"
  client_fd="$(echo "$client_meta" | awk '{print $2}')"

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$client_log" "$client_ready_re"; then
    echo "[cmp] $name client did not become ready; log: $client_log"
    stop_proc "$client_pid" "$client_fd" "$client_fifo"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  # One action. Only quit after success is observed (some clients exit on "quit").
  send_line_fd "$client_fd" "hello"

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$client_log" "$client_success_re"; then
    send_line_fd "$client_fd" ""
  fi

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$client_log" "$client_success_re"; then
    echo "[cmp] $name client did not reach expected output; log: $client_log"
    stop_proc "$client_pid" "$client_fd" "$client_fifo"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  send_line_fd "$client_fd" "quit"

  stop_proc "$client_pid" "$client_fd" "$client_fifo"
  stop_proc "$server_pid" "$server_fd" "$server_fifo"

  normalize_log_to_events "$server_log" "$out_dir/${name}.server.events"
  normalize_log_to_events "$client_log" "$out_dir/${name}.client.events"
  return 0
}

build_go_examples() {
  local examples=(
    announce
    broadcast
    buffer
    channel
    echo
    filetransfer
    identify
    link
    minimal
    ratchets
    request
    resource
    speedtest
  )

  echo "[cmp] building go examples..."
  for ex in "${examples[@]}"; do
    echo "[cmp]   go build $ex"
    go build -o "$GO_EXAMPLES_BIN/$ex" "./examples/$ex"
  done
  echo "[cmp] go examples built"
}

run_broadcast_two_procs() {
  local label="$1"
  local out_dir="$2"
  local cfg_dir="$3"
  local cmd_str="$4"

  local recv_log="$out_dir/${label}.recv.log"
  local recv_fifo="$out_dir/${label}.recv.stdin"
  local send_log="$out_dir/${label}.send.log"
  local send_fifo="$out_dir/${label}.send.stdin"

  local recv_meta recv_pid recv_fd
  recv_meta="$(start_proc_with_fifo "$recv_log" "$recv_fifo" bash -c "$cmd_str")"
  recv_pid="$(echo "$recv_meta" | awk '{print $1}')"
  recv_fd="$(echo "$recv_meta" | awk '{print $2}')"

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$recv_log" "Broadcast example"; then
    echo "[cmp] $label receiver did not start; log: $recv_log"
    stop_proc "$recv_pid" "$recv_fd" "$recv_fifo"
    return 1
  fi

  local send_meta send_pid send_fd
  send_meta="$(start_proc_with_fifo "$send_log" "$send_fifo" bash -c "$cmd_str")"
  send_pid="$(echo "$send_meta" | awk '{print $1}')"
  send_fd="$(echo "$send_meta" | awk '{print $2}')"

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$send_log" "Broadcast example"; then
    echo "[cmp] $label sender did not start; log: $send_log"
    stop_proc "$send_pid" "$send_fd" "$send_fifo"
    stop_proc "$recv_pid" "$recv_fd" "$recv_fifo"
    return 1
  fi

  send_line_fd "$send_fd" "hello"

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$recv_log" "Received data: hello"; then
    echo "[cmp] $label did not receive broadcast; recv log: $recv_log"
    stop_proc "$send_pid" "$send_fd" "$send_fifo"
    stop_proc "$recv_pid" "$recv_fd" "$recv_fifo"
    return 1
  fi

  stop_proc "$send_pid" "$send_fd" "$send_fifo"
  stop_proc "$recv_pid" "$recv_fd" "$recv_fifo"

  normalize_log_to_events "$recv_log" "$out_dir/${label}.recv.events"
  normalize_log_to_events "$send_log" "$out_dir/${label}.send.events"
  return 0
}

run_speedtest() {
  local label="$1"
  local out_dir="$2"
  local cfg_dir="$3"
  local server_cmd_str="$4"
  local client_cmd_prefix_str="$5"

  local server_log="$out_dir/${label}.server.log"
  local server_fifo="$out_dir/${label}.server.stdin"
  local client_log="$out_dir/${label}.client.log"
  local client_fifo="$out_dir/${label}.client.stdin"

  local server_meta server_pid server_fd
  server_meta="$(start_proc_with_fifo "$server_log" "$server_fifo" bash -c "$server_cmd_str")"
  server_pid="$(echo "$server_meta" | awk '{print $1}')"
  server_fd="$(echo "$server_meta" | awk '{print $2}')"

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$server_log" "Speedtest"; then
    echo "[cmp] $label server did not start; log: $server_log"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  local hash
  hash="$(extract_hash "$server_log")"
  if [[ -z "$hash" ]]; then
    echo "[cmp] $label server did not print destination hash; log: $server_log"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  send_line_fd "$server_fd" ""

  local client_meta client_pid client_fd
  client_meta="$(start_proc_with_fifo "$client_log" "$client_fifo" bash -c "$client_cmd_prefix_str \"$hash\"")"
  client_pid="$(echo "$client_meta" | awk '{print $1}')"
  client_fd="$(echo "$client_meta" | awk '{print $2}')"

  # Speedtest is not interactive on the client side.
  if ! wait_for_rg "$CMD_TIMEOUT_SECS" "$server_log" "--- Statistics -----"; then
    echo "[cmp] $label did not print statistics; server log: $server_log"
    stop_proc "$client_pid" "$client_fd" "$client_fifo"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  stop_proc "$client_pid" "$client_fd" "$client_fifo"
  stop_proc "$server_pid" "$server_fd" "$server_fifo"

  normalize_log_to_events "$server_log" "$out_dir/${label}.server.events"
  normalize_log_to_events "$client_log" "$out_dir/${label}.client.events"
  return 0
}

run_filetransfer_go() {
  local out_dir="$1"
  local cfg_dir="$2"

  local serve_dir="$out_dir/serve_dir"
  local dl_dir="$out_dir/downloads"
  mkdir -p "$serve_dir" "$dl_dir"
  printf "hello filetransfer\n" >"$serve_dir/hello.txt"

  local server_log="$out_dir/filetransfer.server.log"
  local server_fifo="$out_dir/filetransfer.server.stdin"
  local client_log="$out_dir/filetransfer.client.log"
  local client_fifo="$out_dir/filetransfer.client.stdin"

  local server_meta server_pid server_fd
  server_meta="$(start_proc_with_fifo "$server_log" "$server_fifo" "$GO_EXAMPLES_BIN/filetransfer" -config "$cfg_dir" -serve "$serve_dir")"
  server_pid="$(echo "$server_meta" | awk '{print $1}')"
  server_fd="$(echo "$server_meta" | awk '{print $2}')"

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$server_log" "File server"; then
    echo "[cmp] filetransfer.go server did not start; log: $server_log"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  local hash
  hash="$(extract_hash "$server_log")"
  if [[ -z "$hash" ]]; then
    echo "[cmp] filetransfer.go server did not print destination hash; log: $server_log"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  send_line_fd "$server_fd" ""

  local client_meta client_pid client_fd
  client_meta="$(start_proc_with_fifo "$client_log" "$client_fifo" "$GO_EXAMPLES_BIN/filetransfer" -config "$cfg_dir" -destination "$hash" -out "$dl_dir")"
  client_pid="$(echo "$client_meta" | awk '{print $1}')"
  client_fd="$(echo "$client_meta" | awk '{print $2}')"

  if ! wait_for_rg "$CMD_TIMEOUT_SECS" "$client_log" "Ready!"; then
    echo "[cmp] filetransfer.go client did not become ready; log: $client_log"
    stop_proc "$client_pid" "$client_fd" "$client_fifo"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  # Select the first file, then press enter to return, then quit.
  send_line_fd "$client_fd" "0"
  if ! wait_for_rg "$CMD_TIMEOUT_SECS" "$client_log" "Press enter to return to the menu"; then
    # Python prints "The download completed! Press enter to return to the menu."
    if ! wait_for_rg "$CMD_TIMEOUT_SECS" "$client_log" "download.*Press enter to return to the menu"; then
      echo "[cmp] filetransfer.go download did not conclude; log: $client_log"
      stop_proc "$client_pid" "$client_fd" "$client_fifo"
      stop_proc "$server_pid" "$server_fd" "$server_fifo"
      return 1
    fi
  fi
  send_line_fd "$client_fd" ""
  send_line_fd "$client_fd" "q"

  if [[ ! -s "$dl_dir/hello.txt" ]]; then
    echo "[cmp] filetransfer.go expected downloaded file at $dl_dir/hello.txt"
    stop_proc "$client_pid" "$client_fd" "$client_fifo"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  stop_proc "$client_pid" "$client_fd" "$client_fifo"
  stop_proc "$server_pid" "$server_fd" "$server_fifo"

  normalize_log_to_events "$server_log" "$out_dir/filetransfer.server.events"
  normalize_log_to_events "$client_log" "$out_dir/filetransfer.client.events"
  return 0
}

run_filetransfer_python() {
  local out_dir="$1"
  local cfg_dir="$2"

  local serve_dir="$out_dir/serve_dir"
  mkdir -p "$serve_dir"
  printf "hello filetransfer\n" >"$serve_dir/hello.txt"

  local server_log="$out_dir/filetransfer.server.log"
  local server_fifo="$out_dir/filetransfer.server.stdin"
  local client_log="$out_dir/filetransfer.client.log"
  local client_fifo="$out_dir/filetransfer.client.stdin"

  local server_meta server_pid server_fd
  server_meta="$(start_proc_with_fifo "$server_log" "$server_fifo" "$PYTHON" -u "$ROOT/python/Examples/Filetransfer.py" --config "$cfg_dir" --serve "$serve_dir")"
  server_pid="$(echo "$server_meta" | awk '{print $1}')"
  server_fd="$(echo "$server_meta" | awk '{print $2}')"

  if ! wait_for_rg "$READY_TIMEOUT_SECS" "$server_log" "File server"; then
    echo "[cmp] filetransfer.python server did not start; log: $server_log"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  local hash
  hash="$(extract_hash "$server_log")"
  if [[ -z "$hash" ]]; then
    echo "[cmp] filetransfer.python server did not print destination hash; log: $server_log"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  send_line_fd "$server_fd" ""

  local client_meta client_pid client_fd
  client_meta="$(start_proc_with_fifo "$client_log" "$client_fifo" "$PYTHON" -u "$ROOT/python/Examples/Filetransfer.py" --config "$cfg_dir" "$hash")"
  client_pid="$(echo "$client_meta" | awk '{print $1}')"
  client_fd="$(echo "$client_meta" | awk '{print $2}')"

  if ! wait_for_rg "$CMD_TIMEOUT_SECS" "$client_log" "Ready!"; then
    echo "[cmp] filetransfer.python client did not become ready; log: $client_log"
    stop_proc "$client_pid" "$client_fd" "$client_fifo"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  send_line_fd "$client_fd" "0"
  if ! wait_for_rg "$CMD_TIMEOUT_SECS" "$client_log" "download.*Press enter to return to the menu"; then
    echo "[cmp] filetransfer.python download did not conclude; log: $client_log"
    stop_proc "$client_pid" "$client_fd" "$client_fifo"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi
  send_line_fd "$client_fd" ""
  send_line_fd "$client_fd" "q"

  # Python writes the downloaded file to CWD; run from out_dir so it stays contained.
  if [[ ! -s "$out_dir/hello.txt" ]]; then
    echo "[cmp] filetransfer.python expected downloaded file at $out_dir/hello.txt"
    stop_proc "$client_pid" "$client_fd" "$client_fifo"
    stop_proc "$server_pid" "$server_fd" "$server_fifo"
    return 1
  fi

  stop_proc "$client_pid" "$client_fd" "$client_fifo"
  stop_proc "$server_pid" "$server_fd" "$server_fifo"

  normalize_log_to_events "$server_log" "$out_dir/filetransfer.server.events"
  normalize_log_to_events "$client_log" "$out_dir/filetransfer.client.events"
  return 0
}

overall=0

build_go_examples

examples=(
  minimal
  announce
  broadcast
  link
  identify
  echo
  ratchets
  channel
  buffer
  request
  resource
  filetransfer
  speedtest
)

for ex in "${examples[@]}"; do
  echo
  echo "[cmp] $ex"

  py_out="$OUT_DIR/$ex/python"
  go_out="$OUT_DIR/$ex/go"
  mkdir -p "$py_out" "$go_out"

  py_cfg="$(new_smoke_config_dir "py_${ex}_${TS}_$RANDOM")"
  go_cfg="$(new_smoke_config_dir "go_${ex}_${TS}_$RANDOM")"

  case "$ex" in
    minimal)
      if ! run_simple_enter_smoke "minimal.python" "$py_out" "$py_cfg" "$PYTHON" -u "$ROOT/python/Examples/Minimal.py" --config "$py_cfg"; then
        overall=1
      fi
      if ! run_simple_enter_smoke "minimal.go" "$go_out" "$go_cfg" "$GO_EXAMPLES_BIN/minimal" -config "$go_cfg"; then
        overall=1
      fi
      ;;
    announce)
      if ! run_simple_enter_smoke "announce.python" "$py_out" "$py_cfg" "$PYTHON" -u "$ROOT/python/Examples/Announce.py" --config "$py_cfg"; then
        overall=1
      fi
      if ! run_simple_enter_smoke "announce.go" "$go_out" "$go_cfg" "$GO_EXAMPLES_BIN/announce" -config "$go_cfg"; then
        overall=1
      fi
      ;;
    broadcast)
      if ! run_broadcast_two_procs "broadcast.python" "$py_out" "$py_cfg" "$PYTHON -u $ROOT/python/Examples/Broadcast.py --config $py_cfg --channel public_information"; then
        overall=1
      fi
      if ! run_broadcast_two_procs "broadcast.go" "$go_out" "$go_cfg" "$GO_EXAMPLES_BIN/broadcast -config $go_cfg -channel public_information"; then
        overall=1
      fi
      ;;
    link)
      if ! run_server_client_smoke \
        "link.python" "$py_out" "$py_cfg" \
        "Link example" \
        "Link established with server" \
        "I received \\\"hello\\\" over the link" \
        "$PYTHON -u $ROOT/python/Examples/Link.py --config $py_cfg --server" \
        "$PYTHON -u $ROOT/python/Examples/Link.py --config $py_cfg"; then
        overall=1
      fi
      if ! run_server_client_smoke \
        "link.go" "$go_out" "$go_cfg" \
        "Link example" \
        "Link established with server" \
        "I received \\\"hello\\\" over the link" \
        "$GO_EXAMPLES_BIN/link -config $go_cfg -server" \
        "$GO_EXAMPLES_BIN/link -config $go_cfg -destination"; then
        overall=1
      fi
      ;;
    identify)
      if ! run_server_client_smoke \
        "identify.python" "$py_out" "$py_cfg" \
        "Link identification example" \
        "Link established with server" \
        "I received \\\"hello\\\" over the link" \
        "$PYTHON -u $ROOT/python/Examples/Identify.py --config $py_cfg --server" \
        "$PYTHON -u $ROOT/python/Examples/Identify.py --config $py_cfg"; then
        overall=1
      fi
      if ! run_server_client_smoke \
        "identify.go" "$go_out" "$go_cfg" \
        "Link identification example" \
        "Link established with server" \
        "I received \\\"hello\\\" over the link" \
        "$GO_EXAMPLES_BIN/identify -config $go_cfg -server" \
        "$GO_EXAMPLES_BIN/identify -config $go_cfg -destination"; then
        overall=1
      fi
      ;;
    echo)
      if ! run_server_client_smoke \
        "echo.python" "$py_out" "$py_cfg" \
        "Echo server" \
        "Echo client ready" \
        "Valid reply received" \
        "$PYTHON -u $ROOT/python/Examples/Echo.py --config $py_cfg --server" \
        "$PYTHON -u $ROOT/python/Examples/Echo.py --config $py_cfg --timeout 5"; then
        overall=1
      fi
      if ! run_server_client_smoke \
        "echo.go" "$go_out" "$go_cfg" \
        "Echo server" \
        "Echo client ready" \
        "Valid reply received" \
        "$GO_EXAMPLES_BIN/echo -config $go_cfg -server" \
        "$GO_EXAMPLES_BIN/echo -config $go_cfg -timeout 5"; then
        overall=1
      fi
      ;;
    ratchets)
      if ! run_timeout_cmd_smoke \
        "ratchets.python" "$py_out" "$py_cfg" \
        "Ratcheted echo server" \
        "Valid reply received" \
        "$PYTHON -u $ROOT/python/Examples/Ratchets.py --config $py_cfg --server"; then
        overall=1
      fi
      if ! run_timeout_cmd_smoke \
        "ratchets.go" "$go_out" "$go_cfg" \
        "Ratcheted echo server" \
        "Valid reply received" \
        "$GO_EXAMPLES_BIN/ratchets -config $go_cfg -server"; then
        overall=1
      fi
      ;;
    channel)
      if ! run_server_client_smoke \
        "channel.python" "$py_out" "$py_cfg" \
        "Channel example" \
        "Establishing link with server|Link established with server" \
        "I received \\\"hello\\\" over the link" \
        "$PYTHON -u $ROOT/python/Examples/Channel.py --config $py_cfg --server" \
        "$PYTHON -u $ROOT/python/Examples/Channel.py --config $py_cfg"; then
        overall=1
      fi
      if ! run_server_client_smoke \
        "channel.go" "$go_out" "$go_cfg" \
        "Channel example" \
        "Link established with server" \
        "I received \\\"hello\\\" over the link" \
        "$GO_EXAMPLES_BIN/channel -config $go_cfg -server" \
        "$GO_EXAMPLES_BIN/channel -config $go_cfg"; then
        overall=1
      fi
      ;;
    buffer)
      if ! run_server_client_smoke \
        "buffer.python" "$py_out" "$py_cfg" \
        "Link buffer example" \
        "Link established with server" \
        "I received \\\"hello\\\" over the buffer" \
        "$PYTHON -u $ROOT/python/Examples/Buffer.py --config $py_cfg --server" \
        "$PYTHON -u $ROOT/python/Examples/Buffer.py --config $py_cfg"; then
        overall=1
      fi
      if ! run_server_client_smoke \
        "buffer.go" "$go_out" "$go_cfg" \
        "Link buffer example" \
        "Link established with server" \
        "I received \\\"hello\\\" over the buffer" \
        "$GO_EXAMPLES_BIN/buffer -config $go_cfg -server" \
        "$GO_EXAMPLES_BIN/buffer -config $go_cfg"; then
        overall=1
      fi
      ;;
    request)
      if ! run_server_client_smoke \
        "request.python" "$py_out" "$py_cfg" \
        "Request example" \
        "Link established with server" \
        "Got response for request" \
        "$PYTHON -u $ROOT/python/Examples/Request.py --config $py_cfg --server" \
        "$PYTHON -u $ROOT/python/Examples/Request.py --config $py_cfg"; then
        overall=1
      fi
      if ! run_server_client_smoke \
        "request.go" "$go_out" "$go_cfg" \
        "Request example" \
        "Link established with server" \
        "Got response for request" \
        "$GO_EXAMPLES_BIN/request -config $go_cfg -server" \
        "$GO_EXAMPLES_BIN/request -config $go_cfg -destination"; then
        overall=1
      fi
      ;;
    resource)
      # Resource transfer is heavy (Python uses a fixed 32 MiB payload).
      # Keep it as a best-effort smoke: send once and only require the client to start sending.
      if ! run_server_client_smoke \
        "resource.python" "$py_out" "$py_cfg" \
        "Resource example" \
        "Link established with server" \
        "Data length:" \
        "$PYTHON -u $ROOT/python/Examples/Resource.py --config $py_cfg --server" \
        "$PYTHON -u $ROOT/python/Examples/Resource.py --config $py_cfg"; then
        overall=1
      fi
      if ! run_server_client_smoke \
        "resource.go" "$go_out" "$go_cfg" \
        "Resource example" \
        "Link established with server" \
        "Data length:" \
        "$GO_EXAMPLES_BIN/resource -config $go_cfg -server" \
        "$GO_EXAMPLES_BIN/resource -config $go_cfg -destination"; then
        overall=1
      fi
      ;;
    filetransfer)
      # Python client writes into CWD; run from its output folder to keep it contained.
      ( cd "$py_out" && run_filetransfer_python "$py_out" "$py_cfg" ) || overall=1
      if ! run_filetransfer_go "$go_out" "$go_cfg"; then
        overall=1
      fi
      ;;
    speedtest)
      if ! run_speedtest \
        "speedtest.python" "$py_out" "$py_cfg" \
        "$PYTHON -u $ROOT/python/Examples/Speedtest.py --config $py_cfg --server" \
        "$PYTHON -u $ROOT/python/Examples/Speedtest.py --config $py_cfg"; then
        overall=1
      fi
      if ! run_speedtest \
        "speedtest.go" "$go_out" "$go_cfg" \
        "$GO_EXAMPLES_BIN/speedtest -config $go_cfg -server" \
        "$GO_EXAMPLES_BIN/speedtest -config $go_cfg -cap-mb 2 -destination"; then
        overall=1
      fi
      ;;
    *)
      echo "[cmp] unknown example $ex"
      overall=1
      ;;
  esac

  rm -rf "$py_cfg" "$go_cfg"
done

echo
if [[ "$overall" -eq 0 ]]; then
  echo "[cmp] OK"
else
  echo "[cmp] FAIL (see $OUT_DIR)"
fi
exit "$overall"
