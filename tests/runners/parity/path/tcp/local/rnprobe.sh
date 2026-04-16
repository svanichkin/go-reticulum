#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
source "$ROOT/tests/runners/parity/path/common/rnpath.sh"
path_run_compare "compare_rnprobe_py_vs_go.sh"
