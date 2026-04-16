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

found=0
for script in "$ANNOUNCE_DIR"/*.sh; do
  name="$(basename "$script")"
  if [[ "$name" == "run.sh" || "$name" == _* ]]; then
    continue
  fi
  [[ -f "$script" ]] || continue
  found=1
  echo "[parity-announce] $name"
  bash "$script"
done

if [[ "$found" -eq 0 ]]; then
  echo "[parity-announce] no announce parity scripts yet"
fi

