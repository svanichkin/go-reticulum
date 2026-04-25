# TCP Local Announce Parity

This branch verifies `ANNOUNCE` parity over a direct local TCP path.

Topology:
- shared-client path: `nodeA local client → nodeA shared rnsd(TCP client) → nodeB rnsd(TCP server) → nodeB observe announce`
- standalone sender path: `nodeA tool(TCP client) → nodeB rnsd(TCP server) → nodeB observe announce`

Directions:
- `go → announce → py`
- `py → announce → go`

Script coverage:
- `rncp.sh`: `nodeA rncp listener → periodic announce → nodeB`
- `rnid.sh`: `nodeA rnid -a → announce + reannounce → nodeB`
- `rnsd.sh`: `nodeA rnsd transport announce → nodeB`, then `nodeB restarted rnsd → announce → nodeA`
- `rnx.sh`: `nodeA rnx listener → announce → nodeB`

Transport notes:
- the receiver side is the `TCPServerInterface` side
- the sender side is the `TCPClientInterface` side
- `shared_client` means `nodeA tool(local client) → nodeA shared rnsd`
- current runner labels and artifacts still use `local` for `shared_client`
- all scripts run both `standalone` and `shared_client`
