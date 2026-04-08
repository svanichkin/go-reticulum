#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"

mkdir -p "$ROOT/.gocache" "$ROOT/.gotmp" "$ROOT/.gopath" "$ROOT/.gomodcache"
export GOCACHE="${GOCACHE:-$ROOT/.gocache}"
export GOTMPDIR="${GOTMPDIR:-$ROOT/.gotmp}"
export GOPATH="${GOPATH:-$ROOT/.gopath}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.gomodcache}"

cd "$ROOT"
go test -count=1 -tags=integration ./cmd/rnsd -run '^TestRNSDIntegration_(Help|Version|ExampleConfig|UnknownFlagExit2|UnexpectedPositionalExit2|ServiceCreatesLogfile|InteractiveQuit)$'
