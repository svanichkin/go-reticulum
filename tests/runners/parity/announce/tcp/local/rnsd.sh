#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
source "$ROOT/tests/runners/parity/announce/tcp/local/interface.sh"
source "$ROOT/tests/runners/parity/announce/common/rnsd.sh"
