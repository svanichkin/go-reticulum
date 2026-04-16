#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
export ANNOUNCE_INTERFACE_NAME="tcp/bootstrap"
export ANNOUNCE_INTERFACE_LABEL="TCP bootstrap"
export ANNOUNCE_INTERFACE_DIR="$ROOT/tests/runners/parity/path/tcp/bootstrap"
export ANNOUNCE_ENV_PREFIX="PARITY_PATH_TCP_BOOTSTRAP"
export PARITY_PATH_TCP_BOOTSTRAP_SCRIPTS="rnpath.sh rnprobe.sh rnx.sh"

exec "$ROOT/tests/runners/parity/announce/common/run.sh" "$@"
