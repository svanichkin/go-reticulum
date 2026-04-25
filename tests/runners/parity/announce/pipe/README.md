# Pipe Announce Parity

This branch verifies `ANNOUNCE` parity over `PipeInterface`.

Topology:
- shared-client path: `nodeA local client → nodeA shared rnsd → PipeInterface(FIFO bridge) → nodeB rnsd`
- standalone sender path: `nodeA tool → PipeInterface(FIFO bridge) → nodeB rnsd`

Directions:
- `go → announce → py`
- `py → announce → go`

Script coverage:
- `rncp.sh`: `nodeA rncp listener → periodic announce → nodeB`
- `rnid.sh`: `nodeA rnid -a → announce + reannounce → nodeB`
- `rnsd.sh`: `nodeA rnsd transport announce → nodeB`, then `nodeB restarted rnsd → announce → nodeA`
- `rnx.sh`: `nodeA rnx listener → announce → nodeB`

Transport notes:
- each scenario creates `a_to_b` and `b_to_a` FIFOs
- the pipe bridge process maps FIFO traffic into the `PipeInterface` stdio transport
- `shared_client` means `nodeA tool(local client) → nodeA shared rnsd`
- current runner labels and artifacts still use `local` for `shared_client`
- all scripts run both `standalone` and `shared_client`
