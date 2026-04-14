#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PYTHON="${PYTHON:-python3}"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache" "$ROOT/tests/artifacts/logs"
export GOCACHE="$ROOT/.gocache"
GOTMPDIR="${GOTMPDIR:-$(mktemp -d "$ROOT/.gotmp/run.XXXXXX")}"
export GOTMPDIR
export GOPATH="$ROOT/.gopath"
export GOMODCACHE="$ROOT/.gomodcache"

export PYTHONPATH="${PYTHONPATH:-"$ROOT/python"}"
export PYTHONUNBUFFERED=1

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-35}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-20}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-5}"
PROBES="${PROBES:-3}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnprobe_two_nodes"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$GO_BIN_DIR" || true
  rm -rf "$GOTMPDIR" || true
}
trap cleanup EXIT

echo "[cmp] out=$OUT_DIR"
echo "[cmp] building go binaries..."
go build -o "$GO_BIN_DIR/rnsd" ./cmd/rnsd
go build -o "$GO_BIN_DIR/rnstatus" ./cmd/rnstatus
go build -o "$GO_BIN_DIR/rnprobe" ./cmd/rnprobe

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

stop_proc() {
  local pid="$1"
  if [[ -z "${pid}" ]]; then
    return 0
  fi
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    return 0
  fi

  kill -INT "$pid" >/dev/null 2>&1 || true
  if "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1; then
    return 0
  fi

  kill -TERM "$pid" >/dev/null 2>&1 || true
  if "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1; then
    return 0
  fi

  kill -KILL "$pid" >/dev/null 2>&1 || true
  "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout "$STOP_TIMEOUT_SECS" -- bash -c "wait $pid" >/dev/null 2>&1 || true
  return 0
}

wait_for_ok() {
  local timeout="$1"
  shift

  local start
  start="$(date +%s)"
  while true; do
    if "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout 2 -- "$@" >/dev/null 2>&1; then
      return 0
    fi
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.2
  done
}

new_run_dir_from_template() {
  local template="$1"
  local sip="$2"
  local cip="$3"
  local listen="$4"
  local forward="$5"

  local run_dir
  run_dir="$(mktemp -d)"
  cp "$template" "$run_dir/config"
  "$PYTHON" "$ROOT/tests/support/tools/patch_reticulum_config_ports.py" \
    --path "$run_dir/config" \
    --shared-instance-port "$sip" \
    --instance-control-port "$cip" \
    --listen-port "$listen" \
    --forward-port "$forward"
  echo "$run_dir"
}

extract_probe_hash() {
  local path="$1"
  # Prefer the explicit rnstatus line:
  # " Probe responder at <...> active"
  local from_status
  from_status="$(rg -o "Probe responder at <[0-9a-fA-F]+> active" "$path" | head -n 1 || true)"
  if [[ -n "$from_status" ]]; then
    echo "$from_status" | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>'
    return 0
  fi

  # Fallback: parse the rnsd startup log line embedded in rnstatus output on Go:
  # "Transport Instance will respond to probe requests on <...:<HEX>>"
  local from_notice
  from_notice="$(rg -o "respond to probe requests on <[^>]*:<[0-9a-fA-F]+>>" "$path" | head -n 1 || true)"
  if [[ -n "$from_notice" ]]; then
    echo "$from_notice" | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>'
    return 0
  fi

  return 0
}

make_summary() {
  local in="$1"
  local out="$2"

  # Normalize carriage-return updates and control chars.
  tr '\r' '\n' <"$in" \
    | tr -d '\b' \
    | sed -E 's/[⢄⢂⢁⡁⡈⡐⡠]//g' \
    | sed -E '/^\[[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\] \[[^]]+\][[:space:]]*/d' \
    | sed -E 's/[[:space:]]+$//; s/[[:space:]]{2,}/ /g' \
    >"$out.tmp"

  local replies
  replies="$(rg -c "Valid reply from" "$out.tmp" || true)"

  local stats
  stats="$(rg -m 1 "Sent [0-9]+, received [0-9]+, packet loss" "$out.tmp" || true)"
  stats="$(echo "$stats" | sed -E 's/([0-9]+), received ([0-9]+)/<N>, received <N>/; s/packet loss [0-9.]+%/packet loss <LOSS>%/')"

  {
    echo "valid_replies=$replies"
    echo "$stats"
  } >"$out"

  rm -f "$out.tmp" || true
}

require_eq() {
  local label="$1"
  local got="$2"
  local want="$3"
  if [[ "$got" != "$want" ]]; then
    echo "[cmp] $label: expected '$want', got '$got'"
    return 1
  fi
  return 0
}

maybe_skip_env() {
  local log="$1"
  if rg -q -i "operation not permitted|permission denied|bind:|listen .*:.*(denied|not permitted)" "$log"; then
    echo "[cmp] SKIP: environment does not permit binding sockets (see $log)"
    exit 0
  fi
}

run_pair() {
  local label="$1" # python|go
  local rnsd_cmd="$2"
  local rnstatus_cmd="$3"
  local rnprobe_cmd="$4"
  local cfg_flag
  cfg_flag="-config"
  if [[ "$label" == "python" ]]; then
    cfg_flag="--config"
  fi

  echo
  echo "[cmp] $label: start two nodes"

  local base
  base=$(( (RANDOM % 10000) + 52000 ))
  base=$(( base / 2 * 2 ))

  local node_a_template="$ROOT/configs/testing/two_nodes_udp/node_a/config"
  local node_b_template="$ROOT/configs/testing/two_nodes_udp/node_b/config"

  local sip_a cip_a sip_b cip_b
  sip_a=$(( (RANDOM % 10000) + 38000 ))
  cip_a=$(( sip_a + 1 ))
  sip_b=$(( sip_a + 2 ))
  cip_b=$(( sip_a + 3 ))

  local node_a_dir node_b_dir
  node_a_dir="$(new_run_dir_from_template "$node_a_template" "$sip_a" "$cip_a" "$base" "$((base+1))")"
  node_b_dir="$(new_run_dir_from_template "$node_b_template" "$sip_b" "$cip_b" "$((base+1))" "$base")"

  local home_a home_b
  home_a="$node_a_dir/home"
  home_b="$node_b_dir/home"
  mkdir -p "$home_a" "$home_b"

  local log_a="$OUT_DIR/${label}.rnsd.node_a.log"
  local log_b="$OUT_DIR/${label}.rnsd.node_b.log"

  env HOME="$home_a" USERPROFILE="$home_a" \
    $rnsd_cmd $cfg_flag "$node_a_dir" -q >"$log_a" 2>&1 &
  local pid_a=$!

  env HOME="$home_b" USERPROFILE="$home_b" \
    $rnsd_cmd $cfg_flag "$node_b_dir" -q >"$log_b" 2>&1 &
  local pid_b=$!

  sleep 0.5
  maybe_skip_env "$log_a"
  maybe_skip_env "$log_b"

  local status_a="$OUT_DIR/${label}.rnstatus.node_a.out"
  local status_b="$OUT_DIR/${label}.rnstatus.node_b.out"

  if ! wait_for_ok "$START_TIMEOUT_SECS" bash -c "env HOME='$home_a' USERPROFILE='$home_a' $rnstatus_cmd $cfg_flag '$node_a_dir' -a >'$status_a' 2>&1"; then
    if rg -q -i "address already in use|errno 48" "$log_a" "$status_a"; then
      echo "[cmp] $label: startup port collision on node_a, retry"
      stop_proc "$pid_a"
      stop_proc "$pid_b"
      return 2
    fi
    if rg -q -i "operation not permitted|permission denied" "$log_a" "$status_a"; then
      echo "[cmp] SKIP: environment does not permit socket operations (see $log_a)"
      stop_proc "$pid_a"
      stop_proc "$pid_b"
      exit 0
    fi
    echo "[cmp] $label: node_a did not become ready; see $log_a and $status_a"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi
  if ! wait_for_ok "$START_TIMEOUT_SECS" bash -c "env HOME='$home_b' USERPROFILE='$home_b' $rnstatus_cmd $cfg_flag '$node_b_dir' -a >'$status_b' 2>&1"; then
    if rg -q -i "address already in use|errno 48" "$log_b" "$status_b"; then
      echo "[cmp] $label: startup port collision on node_b, retry"
      stop_proc "$pid_a"
      stop_proc "$pid_b"
      return 2
    fi
    if rg -q -i "operation not permitted|permission denied" "$log_b" "$status_b"; then
      echo "[cmp] SKIP: environment does not permit socket operations (see $log_b)"
      stop_proc "$pid_a"
      stop_proc "$pid_b"
      exit 0
    fi
    echo "[cmp] $label: node_b did not become ready; see $log_b and $status_b"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  local probe_hash
  probe_hash="$(extract_probe_hash "$status_b")"
  if [[ -z "$probe_hash" ]]; then
    echo "[cmp] $label: could not extract probe responder hash from rnstatus output; see $status_b"
    stop_proc "$pid_a"
    stop_proc "$pid_b"
    return 1
  fi

  echo "[cmp] $label: probe responder hash=$probe_hash"

  local probe_out="$OUT_DIR/${label}.rnprobe.out"
  local code
  code="$(run_capture "$probe_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnprobe_cmd $cfg_flag "$node_a_dir" -n "$PROBES" -w 0.2 -t 15 rnstransport.probe "$probe_hash")"

  stop_proc "$pid_a"
  stop_proc "$pid_b"

  if ! require_eq "$label rnprobe exit" "$code" 0; then
    echo "[cmp] $label: rnprobe failed; see $probe_out"
    return 1
  fi
  if ! rg -q "packet loss 0" "$probe_out"; then
    echo "[cmp] $label: expected 0%% loss; see $probe_out"
    return 1
  fi

  make_summary "$probe_out" "$OUT_DIR/${label}.summary.txt"
  echo "[cmp] $label OK"
  return 0
}

run_pair_with_retry() {
  local label="$1"
  shift
  local attempts=3
  local i=1
  local code=0
  while [[ "$i" -le "$attempts" ]]; do
    set +e
    run_pair "$label" "$@"
    code=$?
    set -e
    if [[ "$code" == "0" ]]; then
      return 0
    fi
    if [[ "$code" != "2" ]]; then
      return "$code"
    fi
    if [[ "$i" -lt "$attempts" ]]; then
      sleep 1
    fi
    i=$((i+1))
  done
  return 1
}

overall=0

if ! run_pair_with_retry "python" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnprobe.py"; then
  overall=1
fi

if ! run_pair_with_retry "go" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/rnprobe"; then
  overall=1
fi

if [[ "$overall" -ne 0 ]]; then
  echo "[cmp] FAIL (see $OUT_DIR)"
  exit 1
fi

if diff -u "$OUT_DIR/python.summary.txt" "$OUT_DIR/go.summary.txt" >"$OUT_DIR/summary.diff"; then
  echo "[cmp] summary parity OK"
  rm -f "$OUT_DIR/summary.diff" || true
else
  echo "[cmp] summary parity DIFF: $OUT_DIR/summary.diff"
  overall=1
fi

if [[ "$overall" -eq 0 ]]; then
  echo "[cmp] OK"
  exit 0
fi

echo "[cmp] FAIL (see $OUT_DIR)"
exit 1
