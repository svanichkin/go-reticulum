#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
export ANNOUNCE_INTERFACE_NAME="tcp/bootstrap"
export ANNOUNCE_INTERFACE_LABEL="TCP bootstrap"
export ANNOUNCE_INTERFACE_DIR="$ROOT/tests/runners/parity/announce/tcp/bootstrap"
export ANNOUNCE_ENV_PREFIX="PARITY_ANNOUNCE_TCP_BOOTSTRAP"

echo "[parity-announce-tcp/bootstrap] note: bootstrap configs intentionally use two TCP bootstrap interfaces"
echo "[parity-announce-tcp/bootstrap] note: one of the two bootstrap interfaces may fail by DNS/reachability and that alone is not a test failure"
echo "[parity-announce-tcp/bootstrap] note: only total lack of bootstrap readiness or broken announce flow counts as a real failure"

exec "$ROOT/tests/runners/parity/announce/common/run.sh" "$@"
