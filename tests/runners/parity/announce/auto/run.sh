#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
export ANNOUNCE_INTERFACE_NAME="auto"
export ANNOUNCE_INTERFACE_LABEL="AutoInterface"
export ANNOUNCE_INTERFACE_DIR="$ROOT/tests/runners/parity/announce/auto"
export ANNOUNCE_ENV_PREFIX="PARITY_ANNOUNCE_AUTO"
export ANNOUNCE_DOCKER_ARGS="--cap-add NET_ADMIN"
export ANNOUNCE_DOCKER_ENV="GOFLAGS=-buildvcs=false"
export ANNOUNCE_DOCKER_ROOT=1

exec "$ROOT/tests/runners/parity/announce/common/run.sh" "$@"
