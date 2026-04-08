#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp"
export GOCACHE="$ROOT/.gocache"
export GOTMPDIR="$ROOT/.gotmp"

cd "$ROOT"
go test -tags=integration ./cmd/rnir -count=1
