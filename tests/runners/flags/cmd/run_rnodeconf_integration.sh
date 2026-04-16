#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

go test -tags=integration ./cmd/rnodeconf -run RNodeconfIntegration -count=1
