# Announce Parity

This directory is the new packet-family parity layer for `ANNOUNCE` scenarios.

Current intent:
- keep `tests/runners/parity/cmd` as-is for legacy scenario coverage
- add new parity runners grouped by packet family
- start with `announce`

Branch READMEs:
- [auto](auto/README.md)
- [pipe](pipe/README.md)
- [tcp/local](tcp/local/README.md)
- [tcp/bootstrap](tcp/bootstrap/README.md)
- [udp](udp/README.md)

Mode naming:
- `standalone`: the tool owns the transport and opens the interface itself
- `shared_client`: the tool is a local client of a shared `rnsd`
- current runner labels and artifacts still use `local` for `shared_client`

Examples:
- route: `go→shared_client→tcp→py`
  current runner label: `go_to_py_local`
- route: `py→shared_client→tcp→go`
  current runner label: `py_to_go_local`
- route: `go→standalone→bootstrap→py`
  current runner label: `go_to_py_standalone`
- route: `py→standalone→udp→go`
  current runner label: `py_to_go_standalone`

- `go→shared_client→tcp→py`
  means the Go tool sends through `nodeA tool(local client) → nodeA shared rnsd`,
  and Python receives the announce over the tested transport path.
- `go→standalone→bootstrap→py`
  means the Go tool reaches Python through an external bootstrap TCP hop without a shared sender-side `rnsd`.

Role convention:
- For TCP, the receiving side is normally the `TCPServerInterface` side.
- For UDP and Auto, prefer `initiator/responder` wording instead of `client/server`.

Recommended script contract:
- create isolated temp run dirs
- use prebuilt Go binaries from `$PARITY_BIN_DIR` when available
- emit deterministic artifacts into `tests/artifacts/logs/...`
- kill spawned `rnsd` / `rnsd.py` on exit
- fail fast on missing markers
