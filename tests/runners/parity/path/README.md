# Announce Parity

This directory is the new packet-family parity layer for `ANNOUNCE` scenarios.

Current intent:
- keep `tests/runners/parity/cmd` as-is for legacy scenario coverage
- add new parity runners grouped by packet family
- start with `announce`

Route notation:
- use arrow notation for the canonical scenario description
- encode the same route in filenames with `_to_` and `_`

Examples:
- route: `go→tcp→local→tcp→py`
  file: `go_to_tcp_local_tcp_py.sh`
- route: `py→tcp→local→tcp→go`
  file: `py_to_tcp_local_tcp_go.sh`
- route: `go→tcp→bootstrap→tcp→py`
  file: `go_to_tcp_bootstrap_tcp_py.sh`
- route: `py→tcp→bootstrap→tcp→go`
  file: `py_to_tcp_bootstrap_tcp_go.sh`
- route: `go→udp→py`
  file: `go_to_udp_py.sh`
- route: `py→udp→go`
  file: `py_to_udp_go.sh`

- `go→tcp→local→tcp→py`
  means Go-side initiator sends an announce through a local/shared transport path
  and the Python side is reached over a TCP server-side interface.
- `go→tcp→bootstrap→tcp→py`
  means Go-side initiator reaches Python through an external/bootstrap TCP hop.

Role convention:
- For TCP, the receiving side is normally the `TCPServerInterface` side.
- For UDP and Auto, prefer `initiator/responder` wording instead of `client/server`.

Recommended script contract:
- create isolated temp run dirs
- use prebuilt Go binaries from `$PARITY_BIN_DIR` when available
- emit deterministic artifacts into `tests/artifacts/logs/...`
- kill spawned `rnsd` / `rnsd.py` on exit
- fail fast on missing markers
