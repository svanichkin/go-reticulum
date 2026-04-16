#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
export ANNOUNCE_INTERFACE_NAME="tcp_local"
export ANNOUNCE_INTERFACE_LABEL="TCP local"
export ANNOUNCE_INTERFACE_DIR="$ROOT/tests/runners/parity/path/tcp/local"
export ANNOUNCE_ENV_PREFIX="PARITY_PATH_TCP_LOCAL"
export PARITY_PATH_TCP_LOCAL_SCRIPTS="rnpath.sh rnprobe.sh rnx.sh"

exec "$ROOT/tests/runners/parity/announce/common/run.sh" "$@"
