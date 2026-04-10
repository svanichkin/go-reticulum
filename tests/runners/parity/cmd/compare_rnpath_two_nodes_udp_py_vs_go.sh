#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PYTHON="${PYTHON:-python3}"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache" "$ROOT/tests/artifacts/logs"
export GOCACHE="$ROOT/.gocache"
export GOTMPDIR="$ROOT/.gotmp"
export GOPATH="$ROOT/.gopath"
export GOMODCACHE="$ROOT/.gomodcache"

export PYTHONPATH="${PYTHONPATH:-"$ROOT/python"}"
export PYTHONUNBUFFERED=1

CMD_TIMEOUT_SECS="${CMD_TIMEOUT_SECS:-120}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-30}"
STOP_TIMEOUT_SECS="${STOP_TIMEOUT_SECS:-6}"

TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
OUT_DIR="$ROOT/tests/artifacts/logs/$TS/compare_rnpath_two_nodes_udp"
mkdir -p "$OUT_DIR"

GO_BIN_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$GO_BIN_DIR" || true
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

wait_for_ok() {
  local timeout="$1"
  shift
  local start
  start="$(date +%s)"
  while true; do
    if "$PYTHON" "$ROOT/tests/support/tools/timeout_exec.py" --timeout 2 -- "$@" >/dev/null 2>&1; then
      return 0
    fi
    local now
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$timeout" ]]; then
      return 1
    fi
    sleep 0.2
  done
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
}

maybe_skip_env() {
  local log="$1"
  if rg -q -i "operation not permitted|permission denied|bind:|listen .*:.*(denied|not permitted)" "$log"; then
    echo "[cmp] SKIP: environment does not permit binding sockets (see $log)"
    exit 0
  fi
}

enable_probe_responder() {
  local config_path="$1"
  "$PYTHON" - "$config_path" <<'PY'
import pathlib, re, sys
path = pathlib.Path(sys.argv[1])
lines = path.read_text(encoding="utf-8").splitlines(keepends=True)
start = end = None
for i, line in enumerate(lines):
    if line.strip().lower() == "[reticulum]":
        start = i
        break
if start is None:
    raise SystemExit("missing [reticulum] section")
end = len(lines)
for j in range(start+1, len(lines)):
    s = lines[j].strip()
    if s.startswith("[") and s.endswith("]") and not s.startswith("[["):
        end = j
        break
pat = re.compile(r"^(\s*)respond_to_probes\s*=\s*.*$", re.IGNORECASE)
for i in range(start+1, end):
    m = pat.match(lines[i])
    if m:
        lines[i] = f"{m.group(1)}respond_to_probes = yes\n"
        break
else:
    lines.insert(end, "  respond_to_probes = yes\n")
path.write_text("".join(lines), encoding="utf-8")
PY
}

new_udp_run_dir() {
  local template="$1"
  local sip="$2"
  local cip="$3"
  local listen_port="$4"
  local forward_port="$5"
  local run_dir
  run_dir="$(mktemp -d)"
  cp "$template" "$run_dir/config"
  "$PYTHON" "$ROOT/tests/support/tools/patch_reticulum_config_ports.py" \
    --path "$run_dir/config" \
    --shared-instance-port "$sip" \
    --instance-control-port "$cip" \
    --listen-port "$listen_port" \
    --forward-port "$forward_port"
  enable_probe_responder "$run_dir/config"
  echo "$run_dir"
}

extract_probe_hash() {
  local path="$1"
  local from_status
  from_status="$(rg -o "Probe responder at <[0-9a-fA-F]+> active" "$path" | head -n 1 || true)"
  if [[ -n "$from_status" ]]; then
    echo "$from_status" | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>'
    return 0
  fi
  local from_notice
  from_notice="$(rg -o "respond to probe requests on <[^>]*:<[0-9a-fA-F]+>>" "$path" | head -n 1 || true)"
  if [[ -n "$from_notice" ]]; then
    echo "$from_notice" | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>'
    return 0
  fi
  local transport_id
  transport_id="$(rg -o "Transport Instance <[0-9a-fA-F]+> running" "$path" | head -n 1 | rg -o "<[0-9a-fA-F]+>" | head -n 1 | tr -d '<>' || true)"
  if [[ -n "$transport_id" ]]; then
    "$PYTHON" -c 'import sys, RNS; print(RNS.Destination.hash_from_name_and_identity("rnstransport.probe", bytes.fromhex(sys.argv[1])).hex())' "$transport_id"
  fi
}

json_len() {
  local path="$1"
  "$PYTHON" - "$path" <<'PY' 2>/dev/null || echo "0"
import json, sys
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

normalize_line() {
  sed -E \
    -e 's/<[0-9a-fA-F]+>/<HEX>/g' \
    -e 's/is [0-9]+ hop(s?)[[:space:]]+away/is <N> hops away/g' \
    -e 's/expires [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}/expires <TS>/g' \
    -e 's/on .* expires/on <IF> expires/g' \
    -e 's/on .*/on <IF>/g' \
    -e 's/[[:space:]]+$//; s/[[:space:]]{2,}/ /g'
}

run_pair() {
  local label="$1"
  local rnsd_cmd_a="$2"
  local rnstatus_cmd_a="$3"
  local rnpath_cmd_a="$4"
  local cfg_flag_a="$5"
  local rnsd_cmd_b="$6"
  local rnstatus_cmd_b="$7"
  local cfg_flag_b="$8"

  echo
  echo "[cmp] $label: start two UDP nodes"

  local base
  base=$(( (RANDOM % 10000) + 52000 ))
  base=$(( base / 2 * 2 ))
  local sip_a cip_a sip_b cip_b
  sip_a=$(( (RANDOM % 10000) + 38000 ))
  cip_a=$(( sip_a + 1 ))
  sip_b=$(( sip_a + 2 ))
  cip_b=$(( sip_a + 3 ))

  local node_a_dir node_b_dir
  node_a_dir="$(new_udp_run_dir "$ROOT/configs/testing/two_nodes_udp/node_a/config" "$sip_a" "$cip_a" "$base" "$((base+1))")"
  node_b_dir="$(new_udp_run_dir "$ROOT/configs/testing/two_nodes_udp/node_b/config" "$sip_b" "$cip_b" "$((base+1))" "$base")"

  local home_a="$node_a_dir/home"
  local home_b="$node_b_dir/home"
  mkdir -p "$home_a" "$home_b"

  local log_a="$OUT_DIR/${label}.rnsd.node_a.log"
  local log_b="$OUT_DIR/${label}.rnsd.node_b.log"
  env HOME="$home_a" USERPROFILE="$home_a" \
    $rnsd_cmd_a $cfg_flag_a "$node_a_dir" -q >"$log_a" 2>&1 &
  local pid_a=$!
  env HOME="$home_b" USERPROFILE="$home_b" \
    $rnsd_cmd_b $cfg_flag_b "$node_b_dir" -q >"$log_b" 2>&1 &
  local pid_b=$!

  sleep 0.5
  maybe_skip_env "$log_a"
  maybe_skip_env "$log_b"

  local status_a="$OUT_DIR/${label}.rnstatus.node_a.out"
  local status_b="$OUT_DIR/${label}.rnstatus.node_b.out"
  if ! wait_for_ok "$START_TIMEOUT_SECS" bash -c "env HOME='$home_a' USERPROFILE='$home_a' $rnstatus_cmd_a $cfg_flag_a '$node_a_dir' -a >'$status_a' 2>&1"; then
    echo "[cmp] $label: node_a did not become ready; see $log_a and $status_a"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi
  if ! wait_for_ok "$START_TIMEOUT_SECS" bash -c "env HOME='$home_b' USERPROFILE='$home_b' $rnstatus_cmd_b $cfg_flag_b '$node_b_dir' -a >'$status_b' 2>&1"; then
    echo "[cmp] $label: node_b did not become ready; see $log_b and $status_b"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local probe_hash
  probe_hash="$(extract_probe_hash "$status_b")"
  if [[ -z "$probe_hash" ]]; then
    echo "[cmp] $label: could not extract probe responder hash; see $status_b"
    stop_proc "$pid_a"; stop_proc "$pid_b"
    return 1
  fi

  local discover_out="$OUT_DIR/${label}.rnpath.discover.out"
  local table_out="$OUT_DIR/${label}.rnpath.table.out"
  local json_out="$OUT_DIR/${label}.rnpath.table.json"
  local drop_out="$OUT_DIR/${label}.rnpath.drop.out"
  local code_discover code_table code_json code_drop
  code_discover="$(run_capture "$discover_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnpath_cmd_a $cfg_flag_a "$node_a_dir" -w 15 "$probe_hash")"
  code_table="$(run_capture "$table_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnpath_cmd_a $cfg_flag_a "$node_a_dir" -t "$probe_hash")"
  code_json="$(run_capture "$json_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnpath_cmd_a $cfg_flag_a "$node_a_dir" -t -j "$probe_hash")"
  code_drop="$(run_capture "$drop_out" env HOME="$home_a" USERPROFILE="$home_a" \
    $rnpath_cmd_a $cfg_flag_a "$node_a_dir" -d "$probe_hash")"

  stop_proc "$pid_a"
  stop_proc "$pid_b"

  if [[ "$code_discover" != "0" ]] || ! rg -q "Path found, destination|Path to .* is now" "$discover_out"; then
    echo "[cmp] $label: rnpath discover failed; see $discover_out"
    return 1
  fi
  if [[ "$code_table" != "0" ]] || ! rg -q "expires " "$table_out"; then
    echo "[cmp] $label: rnpath table failed; see $table_out"
    return 1
  fi
  if [[ "$code_json" != "0" ]] || [[ "$(json_len "$json_out")" == "0" ]]; then
    echo "[cmp] $label: rnpath json table failed; see $json_out"
    return 1
  fi
  if [[ "$code_drop" != "0" ]] || ! rg -q "Dropped path to|Path to .* dropped" "$drop_out"; then
    echo "[cmp] $label: rnpath drop failed; see $drop_out"
    return 1
  fi

  {
    echo "discover=$(tr '\r' '\n' <"$discover_out" | rg -m 1 'Path (found|not found)|Path to .* is now' | normalize_line)"
    echo "table=$(rg -m 1 'expires ' "$table_out" | normalize_line)"
    echo "json_len=$(json_len "$json_out")"
    echo "drop_ok=yes"
  } >"$OUT_DIR/${label}.summary.txt"

  echo "[cmp] $label OK"
  return 0
}

overall=0

if ! run_pair "go_node_a_python_node_b" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "$GO_BIN_DIR/rnpath" \
  "-config" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "--config"; then
  overall=1
fi

if ! run_pair "python_node_a_go_node_b" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnsd.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnstatus.py" \
  "$PYTHON $ROOT/python/RNS/Utilities/rnpath.py" \
  "--config" \
  "$GO_BIN_DIR/rnsd" \
  "$GO_BIN_DIR/rnstatus" \
  "-config"; then
  overall=1
fi

if [[ "$overall" -ne 0 ]]; then
  echo "[cmp] FAIL (see $OUT_DIR)"
  exit 1
fi

echo "[cmp] OK"
