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

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnpath_two_nodes"
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
go build -o "$GO_BIN_DIR/rnpath" ./cmd/rnpath

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
  # " Probe responder at <...> active"
  local line
  line="$(rg -o "Probe responder at <[0-9a-fA-F]+> active" "$path" | head -n 1 || true)"
  if [[ -n "$line" ]]; then
    echo "$line" | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>'
  fi
}

normalize_line() {
  sed -E \
    -e 's/<[0-9a-fA-F]+>/<HEX>/g' \
    -e 's/is [0-9]+ hop(s?)[[:space:]]+away/is <N> hops away/g' \
    -e 's/expires [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}/expires <TS>/g' \
    -e 's/on .* expires/on <IF> expires/g' \
    -e 's/on .*/on <IF>/g' \
    -e 's/[[:space:]]+$//; s/[[:space:]]{2,}/ /g'
}

json_len() {
  local path="$1"
  "$PYTHON" - "$path" <<'PY' 2>/dev/null || echo "0"
import json
import sys

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as f:
    for line in f:
        s = line.strip()
        if not s.startswith("["):
            continue
        try:
            v = json.loads(s)
        except Exception:
            continue
        if isinstance(v, list):
            print(len(v))
            raise SystemExit(0)

print(0)
PY
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
  local rnpath_cmd="$4"

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

  local discover_out="$OUT_DIR/${label}.rnpath.discover.out"
  local table_out="$OUT_DIR/${label}.rnpath.table.out"
  local json_out="$OUT_DIR/${label}.rnpath.table.json"
  local drop_out="$OUT_DIR/${label}.rnpath.drop.out"

  local code_discover code_table code_json code_drop

  code_discover="$(run_capture "$discover_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnpath_cmd $cfg_flag "$node_a_dir" -w 15 "$probe_hash")"

  code_table="$(run_capture "$table_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnpath_cmd $cfg_flag "$node_a_dir" -t "$probe_hash")"

  code_json="$(run_capture "$json_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnpath_cmd $cfg_flag "$node_a_dir" -t -j "$probe_hash")"

  code_drop="$(run_capture "$drop_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnpath_cmd $cfg_flag "$node_a_dir" -d "$probe_hash")"

  stop_proc "$pid_a"
  stop_proc "$pid_b"

  # Build a stable summary for parity diffing.
  local discover_line
  discover_line="$(tr '\r' '\n' <"$discover_out" | rg -m 1 "Path (found|not found)" || true)"
  discover_line="$(echo "$discover_line" | normalize_line)"

  local table_line
  table_line="$(rg -m 1 "expires " "$table_out" || true)"
  table_line="$(echo "$table_line" | normalize_line)"

  local jl
  jl="$(json_len "$json_out")"

  {
    echo "discover_exit=$code_discover"
    echo "discover_line=$discover_line"
    echo "table_exit=$code_table"
    echo "table_line=$table_line"
    echo "json_exit=$code_json"
    echo "json_len=$jl"
    echo "drop_exit=$code_drop"
  } >"$OUT_DIR/${label}.summary.txt"

  # Expectations for a working two-node environment:
  # - discover should succeed (0)
  # - filtered table should succeed (0) and show at least one entry
  # - JSON should parse and be non-empty
  # - drop should succeed (0)
  if [[ "$code_discover" != "0" ]]; then
    echo "[cmp] $label: rnpath discover failed; see $discover_out"
    return 1
  fi
  if [[ "$code_table" != "0" ]]; then
    echo "[cmp] $label: rnpath table failed; see $table_out"
    return 1
  fi
  if [[ -z "$table_line" ]]; then
    echo "[cmp] $label: rnpath table did not show entry; see $table_out"
    return 1
  fi
  if [[ "$code_json" != "0" ]] || [[ "$jl" == "0" ]]; then
    echo "[cmp] $label: rnpath table JSON failed/empty; see $json_out"
    return 1
  fi
  if [[ "$code_drop" != "0" ]]; then
    echo "[cmp] $label: rnpath drop failed; see $drop_out"
    return 1
  fi

  echo "[cmp] $label OK"
  return 0
}

overall=0

if ! run_pair "python" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnpath.py"; then
  overall=1
fi

if ! run_pair "go" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/rnpath"; then
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
