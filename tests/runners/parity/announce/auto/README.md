# Auto Announce Parity

This branch verifies `ANNOUNCE` parity over `AutoInterface`.

Topology:
- shared-client path: `nodeA local client → nodeA shared rnsd → AutoInterface(auto_tx → auto_rx) → nodeB rnsd`
- standalone sender path: `nodeA tool → AutoInterface(auto_tx → auto_rx) → nodeB rnsd`

Directions:
- `go → announce → py`
- `py → announce → go`

Script coverage:
- `rncp.sh`: `nodeA rncp listener → periodic announce → nodeB`
- `rnid.sh`: `nodeA rnid -a → announce + reannounce → nodeB`
- `rnsd.sh`: `nodeA rnsd transport announce → nodeB`, then `nodeB restarted rnsd → announce → nodeA`
- `rnx.sh`: `nodeA rnx listener → announce → nodeB`

Mode notes:
- `shared_client` means `nodeA tool(local client) → nodeA shared rnsd`
- current runner labels and artifacts still use `local` for `shared_client`
- `rncp.sh` and `rnsd.sh` run `standalone` and `shared_client`
- `rnid.sh` and `rnx.sh` run `shared_client` only
- the interface setup uses the `auto_tx` / `auto_rx` veth pair
