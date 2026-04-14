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
go test -count=1 -tags=integration ./cmd/rnodeconf -run '^TestRNodeconfIntegration_(HelpAndVersion|UnknownFlagExit2|FlashMissingParamsExit68|RNodeconfDirNoAccessExit99|TrustKeyStoresFile|TrustKeyInvalidDERExit1|ListModelsAndShowModel|KeyAndShowSigningKey|ShowSigningKey_NoKeysDoesNotCreateKeys|SimPort_InfoAndConfig|SimPort_ConfigReflectsWiFiAndMasksPSK|SimPort_WiFiAPModeConfig|ValidationErrors|SimPort_ModesAndROMBootstrap|SimPort_ExtractBranches|SimPort_FlashBranchExit1|SimPort_EEPROMWipe|SimPort_NotProvisionedExit77_79|SimPort_HiddenHashFlagsWork|SimPort_UpdateAndAutoinstall|SimPort_SettingsAndEEPROMBackup|SimPort_MissingExternalFlashTools)$'
