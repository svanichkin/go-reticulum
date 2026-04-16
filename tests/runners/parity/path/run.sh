#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PATH_DIR="$ROOT/tests/runners/parity/path"

usage() {
  cat <<'EOF'
Usage: tests/runners/parity/path/run.sh

Runs all path-family parity scripts in this directory except:
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

echo "[parity-path] dir=$PATH_DIR"

found=0
while IFS= read -r script; do
  name="$(basename "$script")"
  parent="$(basename "$(dirname "$script")")"
  if [[ "$name" != "run.sh" ]]; then
    continue
  fi
  if [[ "$parent" == "common" || "$parent" == "." ]]; then
    continue
  fi
  found=1
  echo "[parity-path] $parent/$name"
  bash "$script"
done < <(find "$PATH_DIR" -mindepth 2 -maxdepth 3 -name run.sh | sort)

if [[ "$found" -eq 0 ]]; then
  echo "[parity-path] no path parity scripts yet"
fi
