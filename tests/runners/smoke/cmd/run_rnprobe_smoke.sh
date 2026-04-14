#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache"
export GOCACHE="${GOCACHE:-$ROOT/.gocache}"
GOTMPDIR="${GOTMPDIR:-$(mktemp -d "$ROOT/.gotmp/run.XXXXXX")}"
export GOTMPDIR
export GOPATH="${GOPATH:-$ROOT/.gopath}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.gomodcache}"

cleanup() { rm -rf "$GOTMPDIR"; }
trap cleanup EXIT

cd "$ROOT"
go test -count=1 -tags=integration ./cmd/rnprobe -run '^TestRNProbeIntegration_InvalidArgsExit0$'
