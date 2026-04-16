#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"

path_run_compare() {
  local script_rel="$1"
  shift || true
  exec "$ROOT/tests/runners/parity/cmd/$script_rel" "$@"
}
