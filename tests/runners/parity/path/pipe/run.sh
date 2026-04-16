#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
export ANNOUNCE_INTERFACE_NAME="pipe"
export ANNOUNCE_INTERFACE_LABEL="PipeInterface"
export ANNOUNCE_INTERFACE_DIR="$ROOT/tests/runners/parity/path/pipe"
export ANNOUNCE_ENV_PREFIX="PARITY_PATH_PIPE"
export PARITY_PATH_PIPE_SCRIPTS="rnpath.sh rnprobe.sh rnx.sh"

exec "$ROOT/tests/runners/parity/announce/common/run.sh" "$@"
