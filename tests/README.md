# Tests

Directory layout:

- `tests/runners/go/` - all Go test runners, including library and `cmd/...` integration runners
- `tests/runners/smoke/` - smoke runners for CLI utilities
- `tests/runners/parity/` - parity runners and Python reference runners used by parity
- `tests/support/tools/` - helper scripts used by runners
- `tests/support/bin/` - helper executables placed on `PATH` by runners
- `tests/artifacts/logs/` - logs and run outputs

Artifact policy:

- layer entrypoints under `tests/runners/*/run.sh` write into layer-specific subdirectories under `tests/artifacts/logs/`
- by default, a layer runner clears its own artifact directory before running
- set `KEEP_ARTIFACTS=1` to keep previous artifacts
- each layer updates a `latest` symlink to the most recent run
