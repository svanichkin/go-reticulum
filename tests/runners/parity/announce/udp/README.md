# UDP Announce Parity

This branch verifies `ANNOUNCE` parity over `UDPInterface`.

Topology:
- shared-client path: `nodeA local client → nodeA shared rnsd(UDP sender) → nodeB rnsd(UDP receiver) → nodeB observe announce`
- standalone sender path: `nodeA tool(UDP sender) → nodeB rnsd(UDP receiver) → nodeB observe announce`

Directions:
- `go → announce → py`
- `py → announce → go`

Script coverage:
- `rncp.sh`: `nodeA rncp listener → periodic announce → nodeB`
- `rnid.sh`: `nodeA rnid -a → announce + reannounce → nodeB`
- `rnsd.sh`: `nodeA rnsd transport announce → nodeB`, then `nodeB restarted rnsd → announce → nodeA`
- `rnx.sh`: `nodeA rnx listener → announce → nodeB`

Transport notes:
- sender and receiver use paired localhost UDP ports
- `forward_port` points at the opposite node for each scenario
- `shared_client` means `nodeA tool(local client) → nodeA shared rnsd`
- current runner labels and artifacts still use `local` for `shared_client`
- all scripts run both `standalone` and `shared_client`
