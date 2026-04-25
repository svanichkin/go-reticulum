# TCP Bootstrap Announce Parity

This branch verifies `ANNOUNCE` parity over external TCP bootstrap hops.

Topology:
- shared-client path: `nodeA local client → nodeA shared rnsd → external bootstrap TCP hop → nodeB rnsd`
- standalone sender path: `nodeA tool → external bootstrap TCP hop → nodeB rnsd`

Directions:
- `go → announce → py`
- `py → announce → go`

Script coverage:
- `rncp.sh`: `nodeA rncp listener → periodic announce → nodeB`
- `rnid.sh`: `nodeA rnid -a → announce + reannounce → nodeB`
- `rnsd.sh`: `nodeA rnsd transport announce → nodeB`, then `nodeB restarted rnsd → announce → nodeA`
- `rnx.sh`: `nodeA rnx listener → announce → nodeB`

Transport notes:
- the hop is provided by the public bootstrap endpoints configured in `interface.sh`
- the configured bootstrap list currently contains two addresses:
  `212.233.88.164:4242` and `dublin.connect.reticulum.network:4965`
- either configured bootstrap endpoint may be unreachable at a given moment; this alone is not a product failure
- success depends on at least one configured bootstrap node being reachable
- `shared_client` means `nodeA tool(local client) → nodeA shared rnsd`; it does not describe the external hop
- current runner labels and artifacts still use `local` for `shared_client`
- all scripts run both `standalone` and `shared_client`
