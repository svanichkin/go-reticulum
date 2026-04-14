# rnstatus

Reticulum status utility for the Go port. It queries a local shared instance for interface and transport status.

## Overview

- Shows configured interfaces, their state, and basic traffic counters.
- Can print totals, announce stats, link stats, discovered interfaces, and JSON output.
- Supports remote status queries via Remote Management (`-R` + `-i`), if enabled in the target.

## Usage

Local status from the shared instance:

`rnstatus [options] [filter]`

Remote management mode:

`rnstatus -R <transport_identity_hash> -i <identity_path> [options]`

## Flags

| Flag(s) | Type | Default | Description |
|---|---:|---:|---|
| `-config` | string | `""` | Alternative Reticulum config directory (defaults to `~/.reticulum`). |
| `-a`, `-all` | bool | `false` | Show all interfaces (including normally hidden ones). |
| `-A`, `-announce-stats` | bool | `false` | Show announce counters/stats. |
| `-l`, `-link-stats` | bool | `false` | Show link stats (requires local shared instance, or remote management). |
| `-d`, `-discovered` | bool | `false` | List discovered interfaces. |
| `-D` | bool | `false` | Show detailed discovered-interface output and config entries. |
| `-t`, `-totals` | bool | `false` | Print traffic totals. |
| `-s`, `-sort` | string | `""` | Sort by one of: `rate`, `traffic`, `rx`, `tx`, `rxs`, `txs`, `announces`, `arx`, `atx`, `held`. |
| `-r`, `-reverse` | bool | `false` | Reverse sorting. |
| `-j`, `-json` | bool | `false` | Output JSON. |
| `-R` | string | `""` | Remote mode: transport identity hash of remote instance. |
| `-i` | string | `""` | Remote mode: path to identity used for remote management. |
| `-w` | float | `15` | Remote mode: timeout before giving up (seconds). |
| `-v`, `-verbose` | count | `0` | Increase verbosity (repeatable). |
| `-version` | bool | `false` | Print version and exit. |
| `-h`, `-help` | bool | `false` | Print help and exit. |

## Arguments

- `filter` (optional): a substring filter for interface names.

## Examples

All interfaces:

`go run ./cmd/rnstatus -a`

Sorted by traffic:

`go run ./cmd/rnstatus -a -s traffic`

JSON output:

`go run ./cmd/rnstatus -a -j`

Discovered interfaces:

`go run ./cmd/rnstatus -d`

Detailed discovered interfaces and config entries:

`go run ./cmd/rnstatus -D`

Remote management status:

`go run ./cmd/rnstatus -R <REMOTE_TRANSPORT_ID> -i ./management.id -a`

## Exit codes

- `0`: Success.
- `1`: No shared instance available (local mode).
- `2`: Could not get status / missing stats.
- `20`: Remote management / remote query errors.
- `2` (process exit): CLI usage / unknown flag.
