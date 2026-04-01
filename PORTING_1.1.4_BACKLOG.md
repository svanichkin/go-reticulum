# Reticulum 1.1.4 Porting Backlog

This document maps the Python `1.0.4 -> 1.1.4` delta to the current Go port and sizes the work needed to reach comparable behaviour.

## Scope Snapshot

- Python diff in `python/`: `102` files changed between `1.0.4` and `1.1.4`
- Executable Python code only: `26` files changed, `2729` insertions, `1286` deletions
- Biggest Python deltas:
  - `python/RNS/Discovery.py`: `+734/-0`
  - `python/RNS/Transport.py`: `+609/-374`
  - `python/RNS/Reticulum.py`: `+587/-354`
  - `python/RNS/Utilities/rnpath.py`: `+224/-255`
  - `python/RNS/Utilities/rnstatus.py`: `+218/-147`
  - `python/RNS/Interfaces/AutoInterface.py`: `+82/-55`

## Executive Read

This is not a single upgrade patch. It is a multi-stream port:

1. Interface discovery and network identity are new subsystems in Python 1.1.x.
2. Blackhole management adds new transport state, persistence, RPC/API surface and CLI flows.
3. `Transport` and `Reticulum` changed substantially around announce processing and control-plane behaviour.
4. CLI parity is now materially behind for `rnstatus`, `rnpath`, and `rnpkg`.

## Size Legend

- `S`: under 1 day
- `M`: 1-3 days
- `L`: 3-7 days
- `XL`: 1-2+ weeks

Effort below is for implementation plus focused tests, not full parity validation on real networks.

## Priority Streams

### Stream 1: Discovery + Network Identity

Status: mostly missing in Go
Size: `XL`

Python sources:
- `python/RNS/Discovery.py`
- `python/RNS/Reticulum.py`
- `python/RNS/Transport.py`
- `python/RNS/Interfaces/AutoInterface.py`
- `python/RNS/Interfaces/WeaveInterface.py`
- `python/RNS/Utilities/rnstatus.py`

Go targets:
- `rns/reticulum.go`
- `rns/transport.go`
- `rns/interfaces/auto_interface.go`
- `rns/interfaces/weave_interface.go`
- likely new files: `rns/discovery.go`, `rns/discovery_test.go`
- `cmd/rnstatus/rnstatus.go`
- possibly `cmd/rnsd/rnsd.go`

Main features:
- Global on-network interface discovery
- Discovery announcer and discovery handler
- Network identity loading/generation and transport registration
- Discovery source filtering by network identity
- Discovery announce encryption
- Auto-connect of discovered interfaces
- `bootstrap_only` interface handling
- `discovered_interfaces()` API exposure
- Discovery status output in `rnstatus`
- `reachable_on` handling for discovered interfaces

Current Go state:
- `AutoInterface` already has peer discovery for its own multicast logic in `rns/interfaces/auto_interface.go`
- There is no dedicated `Discovery` subsystem in Go
- No Go hits for `network_identity`, `discovered_interfaces`, `autoconnect`, `reachable_on`, `bootstrap_only`

Recommended breakdown:
- `1A` Discovery data model and persistence hooks: `L`
- `1B` Network identity lifecycle in `Reticulum` and `Transport`: `L`
- `1C` Discovery announce publish/receive path: `XL`
- `1D` Auto-connect policy and discovered interface materialisation: `L`
- `1E` CLI/API exposure in `rnstatus` and RPC: `M`

### Stream 2: Blackhole Management

Status: missing in Go
Size: `L` to `XL`

Python sources:
- `python/RNS/Transport.py`
- `python/RNS/Reticulum.py`
- `python/RNS/Utilities/rnpath.py`
- `python/RNS/Discovery.py` for remote updater interaction

Go targets:
- `rns/transport.go`
- `rns/reticulum.go`
- `cmd/rnpath/rnpath.go`
- possibly new persistence helpers in `rns/reticulum.go` or a new `rns/blackhole.go`

Main features:
- Local blackhole table
- Blackhole entry duration and reason
- Persist/reload blackhole state
- Remove blackholed paths from path table
- Publish blackhole list over management destination
- Remote blackhole list updater
- Local and remote blackhole list viewing in `rnpath`
- `blackhole_identity()` and `unblackhole_identity()` RPC/API methods

Current Go state:
- No Go hits for `blackhole`
- `rnpath` has no blackhole commands
- Shared-instance/RPC plumbing exists, so the API layer can reuse that foundation

Recommended breakdown:
- `2A` Transport state + persistence + expiry: `L`
- `2B` Reticulum API/RPC exposure: `M`
- `2C` `rnpath` UX and remote fetch flow: `M`
- `2D` remote updater/background sync: `M`

### Stream 3: Transport Behaviour Changes

Status: partially implemented, partially missing
Size: `L`

Python sources:
- `python/RNS/Transport.py`
- `python/RNS/Reticulum.py`
- `python/RNS/Link.py`
- `python/RNS/Packet.py`

Go targets:
- `rns/transport.go`
- `rns/reticulum.go`
- `rns/link.go`
- `rns/packet.go`

Main features/fixes:
- `await_path()` transport API
- Synchronous announce flow
- Improved announce processing
- Request-timeout race fix
- Discovery path-request bookkeeping changes
- Blackhole-aware announce/path handling

Current Go state:
- Path request support exists in `rns/transport.go`
- No Go hit for `await_path`
- Announce flow has partial parity work already, but not the 1.1.x control-plane additions
- Some race/deadlock fixes landed recently in Go, but they are not the full 1.1.x delta

Recommended breakdown:
- `3A` `await_path` implementation and tests: `M`
- `3B` announce path review vs Python 1.1.4: `L`
- `3C` request timeout / race parity review: `M`

### Stream 4: CLI Parity

Status: mixed
Size: `L`

Python sources:
- `python/RNS/Utilities/rnstatus.py`
- `python/RNS/Utilities/rnpath.py`
- `python/RNS/Utilities/rncp.py`
- `python/RNS/Utilities/rnsd.py`
- `python/RNS/Utilities/rnpkg.py`

Go targets:
- `cmd/rnstatus/rnstatus.go`
- `cmd/rnpath/rnpath.go`
- `cmd/rncp/rncp.go`
- `cmd/rnsd/rnsd.go`
- no Go equivalent yet for `rnpkg`

Current gaps:
- `rnstatus`: no monitor mode, no discovered-interface listing, no network identity display
- `rnpath`: no blackhole management or remote blackhole list UX
- `rncp`: Python added custom identity support; Go already has explicit `-i/-identity`, so this looks largely covered
- `rnsd`: Python gained helper/shim updates; Go needs review for new management surfaces once discovery/blackhole land
- `rnpkg`: entirely missing in Go

Recommended breakdown:
- `4A` `rnstatus` monitor mode and discovery output: `M`
- `4B` `rnpath` blackhole commands and remote list support: `M`
- `4C` `rnpkg` command decision: `S`
  - either implement `cmd/rnpkg`
  - or explicitly declare it out of scope
- `4D` audit `rncp`/`rnsd` for 1.1.x parity: `S` to `M`

## File Mapping

| Python file | Delta | Go counterpart(s) | Status in Go | Porting size | Notes |
| --- | ---: | --- | --- | --- | --- |
| `python/RNS/Discovery.py` | `+734/-0` | none yet, likely new `rns/discovery.go`; plus `rns/reticulum.go`, `rns/transport.go` | Missing | `XL` | Biggest single missing subsystem |
| `python/RNS/Reticulum.py` | `+587/-354` | `rns/reticulum.go`, `rns/reticulum_net.go` | Partial | `XL` | Discovery, network identity, blackhole API, config handling |
| `python/RNS/Transport.py` | `+609/-374` | `rns/transport.go` | Partial | `XL` | Core control-plane parity risk |
| `python/RNS/Interfaces/AutoInterface.py` | `+82/-55` | `rns/interfaces/auto_interface.go` | Partial | `L` | Reverse unicast discovery and discovered-interface integration |
| `python/RNS/Interfaces/WeaveInterface.py` | `+6/-6` | `rns/interfaces/weave_interface.go` | Partial | `M` | Discovery support review needed |
| `python/RNS/Interfaces/TCPInterface.py` | `+10/-5` | `rns/interfaces/tcp_interface.go`, `rns/reticulum.go` | Mostly present | `S` | `fixed_mtu` appears already parsed in Go |
| `python/RNS/Link.py` | `+22/-22` | `rns/link.go` | Partial | `M` | Includes timeout/race parity review |
| `python/RNS/Packet.py` | `+2/-4` | `rns/packet.go` | Likely minor gap | `S` | Validate during transport pass |
| `python/RNS/Resource.py` | `+3/-2` | `rns/resource.go` | Likely minor gap | `S` | Check 1.1.2 resource-transfer regression fix |
| `python/RNS/Identity.py` | `+5/-0` | `rns/identity.go` | Likely minor gap | `S` | May relate to network identity helpers |
| `python/RNS/Destination.py` | `+4/-0` | `rns/destination.go` | Likely minor gap | `S` | Review while adding management destinations |
| `python/RNS/Utilities/rnstatus.py` | `+218/-147` | `cmd/rnstatus/rnstatus.go` | Missing major features | `M` | Monitor mode, discovered interfaces, network identity |
| `python/RNS/Utilities/rnpath.py` | `+224/-255` | `cmd/rnpath/rnpath.go` | Missing major features | `M` to `L` | Blackhole UX, remote blackhole list, await-path adjacencies |
| `python/RNS/Utilities/rncp.py` | `+30/-38` | `cmd/rncp/rncp.go` | Mostly present | `S` | Go already supports explicit identity path |
| `python/RNS/Utilities/rnsd.py` | `+69/-0` | `cmd/rnsd/rnsd.go` | Partial | `M` | Needs follow-up once discovery/blackhole exist |
| `python/RNS/Utilities/rnpkg.py` | `+78/-0` | none | Missing | `M` | New command, no Go equivalent |
| `python/RNS/Utilities/rnodeconf.py` | `+21/-14` | `cmd/rnodeconf/rnodeconf.go` | Partial | `S` | Mostly version/URL and maintenance updates |
| `python/RNS/Utilities/rnir.py` | `+1/-4` | `cmd/rnir/rnir.go` | Likely fine | `S` | Low risk |
| `python/RNS/Interfaces/BackboneInterface.py` | `+10/-3` | `rns/interfaces/backbone_interface.go` | Review needed | `S` | Likely touched by discovery/autoconnect plumbing |
| `python/RNS/Interfaces/Interface.py` | `+4/-0` | `rns/interfaces/interface.go` | Review needed | `S` | Interface capability flags for discovery likely required |
| `python/RNS/Interfaces/RNodeInterface.py` | `+2/-1` | `rns/interfaces/rnode_Interface.go` | Review needed | `S` | Low risk |
| `python/RNS/Interfaces/I2PInterface.py` | `+1/-0` | `rns/interfaces/i2p_interface.go` | Review needed | `S` | Low risk |
| `python/RNS/__init__.py` | `+4/-1` | `rns/rns.go` | Review needed | `S` | Usually version/export changes |
| `python/RNS/_version.py` | `+1/-1` | version strings in Go commands/package | Missing version bump only | `S` | Trivial after feature work |

## What Already Seems Covered In Go

- `TCPInterface` fixed MTU config appears present via `fixed_mtu` parsing in `rns/reticulum.go`
- `AutoInterface` already implements multicast peer discovery of its own
- Some transport locking/race fixes have already landed in Go
- `rncp` already accepts explicit identity path via `-i` / `-identity`
- `rnodeconf` already has EEPROM bootstrap support in Go

These should be revalidated, but they are not the biggest blockers.

## What Is Clearly Missing Today

- Discovery subsystem
- Network identity lifecycle
- Interface discovery source filtering
- Discovery announce encryption
- Discovered-interface registry
- Auto-connect for discovered interfaces
- `bootstrap_only` handling
- `discovered_interfaces()` API
- Blackhole storage, RPC and CLI
- `await_path()`
- `rnstatus` monitor mode
- `rnstatus` discovered-interface output
- `rnpkg`

## Suggested Implementation Order

1. `Transport` and `Reticulum` foundations
   - add network identity
   - add blackhole state
   - add `await_path`
2. New discovery subsystem
   - announce builder/parser
   - discovered-interface registry
   - source filtering and encryption
3. Interface integration
   - `AutoInterface`
   - `WeaveInterface`
   - `bootstrap_only`
   - auto-connect
4. CLI and RPC
   - `rnstatus`
   - `rnpath`
   - `rnsd`
   - decide on `rnpkg`
5. Real parity pass and integration tests

## Rough Total Effort

If done carefully with tests:

- Minimum viable 1.1.x parity on the important control-plane features: `3-5 weeks`
- Broader parity including discovery UX, CLI polish and `rnpkg`: `5-8 weeks`

This estimate assumes one engineer working serially and does not include long real-network interoperability soak time.
