#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
ANNOUNCE_DIR="$ROOT/tests/runners/parity/announce"

usage() {
  cat <<'EOF'
Usage: tests/runners/parity/announce/run.sh

Runs all announce-family parity scripts in this directory except:
  - run.sh
  - helper files prefixed with "_"
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ "$#" -ne 0 ]]; then
  usage
  exit 2
fi

echo "[parity-announce] dir=$ANNOUNCE_DIR"

scripts=()
while IFS= read -r script; do
  name="$(basename "$script")"
  parent="$(basename "$(dirname "$script")")"
  if [[ "$name" != "run.sh" ]]; then
    continue
  fi
  if [[ "$parent" == "common" || "$parent" == "." ]]; then
    continue
  fi
  scripts+=("$script")
done < <(find "$ANNOUNCE_DIR" -mindepth 2 -maxdepth 3 -name run.sh | sort)

if [[ "${#scripts[@]}" -eq 0 ]]; then
  echo "[parity-announce] no announce parity scripts yet"
  exit 0
fi

log_dir="$ROOT/tests/artifacts/logs/$(date +"%Y%m%d-%H%M%S")/parity_announce_runner"
mkdir -p "$log_dir"
status=0
pids=()
names=()
logs=()

for script in "${scripts[@]}"; do
  name="$(basename "$(dirname "$script")")"
  log="$log_dir/${name}.log"
  echo "[parity-announce] start $name/run.sh -> $log"
  (bash "$script") >"$log" 2>&1 &
  pids+=("$!")
  names+=("$name")
  logs+=("$log")
done

for i in "${!pids[@]}"; do
  if wait "${pids[$i]}"; then
    code=0
  else
    code="$?"
    status=1
  fi
  echo "[parity-announce] ${names[$i]}/run.sh exit=$code log=${logs[$i]}"
done

exit "$status"
