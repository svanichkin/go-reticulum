#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
export ANNOUNCE_INTERFACE_NAME="tcp/bootstrap"
export ANNOUNCE_INTERFACE_LABEL="TCP bootstrap"
export ANNOUNCE_INTERFACE_DIR="$ROOT/tests/runners/parity/announce/tcp/bootstrap"
export ANNOUNCE_ENV_PREFIX="PARITY_ANNOUNCE_TCP_BOOTSTRAP"

exec "$ROOT/tests/runners/parity/announce/common/run.sh" "$@"
