#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
export ANNOUNCE_INTERFACE_NAME="tcp/bootstrap"
export ANNOUNCE_INTERFACE_LABEL="TCP bootstrap"
export ANNOUNCE_INTERFACE_DIR="$ROOT/tests/runners/parity/path/tcp/bootstrap"
export ANNOUNCE_ENV_PREFIX="PARITY_PATH_TCP_BOOTSTRAP"
export PARITY_PATH_TCP_BOOTSTRAP_SCRIPTS="rnpath.sh rnprobe.sh rnx.sh"

echo "[parity-path-tcp/bootstrap] note: bootstrap configs intentionally use two TCP bootstrap interfaces"
echo "[parity-path-tcp/bootstrap] note: one of the two bootstrap interfaces may fail by DNS/reachability and that alone is not a test failure"
echo "[parity-path-tcp/bootstrap] note: only total lack of bootstrap readiness or broken path flow counts as a real failure"

exec "$ROOT/tests/runners/parity/announce/common/run.sh" "$@"
