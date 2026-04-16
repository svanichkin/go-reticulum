#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
source "$ROOT/tests/runners/parity/announce/udp/interface.sh"
source "$ROOT/tests/runners/parity/announce/common/rnid.sh"
