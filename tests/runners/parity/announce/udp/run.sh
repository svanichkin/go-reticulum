#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
export ANNOUNCE_INTERFACE_NAME="udp"
export ANNOUNCE_INTERFACE_LABEL="UDP"
export ANNOUNCE_INTERFACE_DIR="$ROOT/tests/runners/parity/announce/udp"
export ANNOUNCE_ENV_PREFIX="PARITY_ANNOUNCE_UDP"

exec "$ROOT/tests/runners/parity/announce/common/run.sh" "$@"
