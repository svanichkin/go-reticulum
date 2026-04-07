# go-reticulum

Go port (parity-focused) of the [original Reticulum project](https://github.com/markqvist/Reticulum).

This repository is maintained as a practical parity port. The current parity target is Python Reticulum `1.1.4`, and the Go implementation is aligned against that version across the core library, bundled CLI utilities, examples, and parity/integration checks currently tracked in this repository. A significant part of the work was/is done with AI assistance (Codex/LLMs) for reading the reference implementation, creating parity TODOs, writing tests, and porting examples/CLI tooling.

The port is developed by a single maintainer with assistance from ChatGPT 5.1/5.2. Even though the project is covered with unit tests, integration tests, and smoke checks, unstable areas may still exist; if you run into one, please open an issue or contact the maintainer.

## Goals

- Port Reticulum to Go with maximum behavioural parity.
- Keep the project testable: unit tests and selected integration tests.
- Provide CLI utilities compatible in spirit with the original ones (`rnstatus`, `rnpath`, `rnid`, etc.).

## Repository layout

- `rns/` — core library (network stack and protocols).
- `cmd/` — CLI utilities (the Go equivalents of Reticulum’s “official” tools).
- `examples/` — ports of Python examples (small demo programs).

## How to run

Requires Go (recommended: current stable version).

## How to use

Prefer using prebuilt binaries from [Releases](https://github.com/svanichkin/go-reticulum/releases) if you don’t want to build from source.

### Quick start (local shared instance)

1) Start the daemon (shared instance):

```bash
go run ./cmd/rnsd -v
```

2) In another terminal, view status:

```bash
go run ./cmd/rnstatus -a
```

If you use a non-default config directory, pass it to every tool:

```bash
CFG="$HOME/.reticulum-go"
go run ./cmd/rnsd -config "$CFG" -v
go run ./cmd/rnstatus -config "$CFG" -a
```

### CLI utilities (`cmd/*`)

Quick run without building:

```bash
go run ./cmd/rnsd --help
go run ./cmd/rnstatus --help
```

#### Tools overview

- `rnsd` — Reticulum daemon / shared instance runner. See [`cmd/rnsd/README.md`](cmd/rnsd/README.md).
- `rnstatus` — status display for interfaces and transport. See [`cmd/rnstatus/README.md`](cmd/rnstatus/README.md).
- `rnpath` — path discovery and path table management. See [`cmd/rnpath/README.md`](cmd/rnpath/README.md).
- `rnprobe` — probe/RTT utility using receipts. See [`cmd/rnprobe/README.md`](cmd/rnprobe/README.md).
- `rnid` — identity and crypto utility (generate/announce/encrypt/sign). See [`cmd/rnid/README.md`](cmd/rnid/README.md).
- `rnx` — remote execution (listener/client, ACL, interactive mode). See [`cmd/rnx/README.md`](cmd/rnx/README.md).
- `rncp` — file transfer (send/receive/fetch, ACL). See [`cmd/rncp/README.md`](cmd/rncp/README.md).
- `rnir` — identity resolver stub / Reticulum initialiser (parity placeholder). See [`cmd/rnir/README.md`](cmd/rnir/README.md).
- `rnodeconf` — RNode firmware/config tool (hardware-dependent). See [`cmd/rnodeconf/README.md`](cmd/rnodeconf/README.md).

Build a single tool:

```bash
go build -o ./bin/rnstatus ./cmd/rnstatus
./bin/rnstatus --help
```

#### Examples overview

- `minimal` — smallest possible init + announce loop. See [`examples/minimal/README.md`](examples/minimal/README.md).
- `announce` — destination announces + announce handler. See [`examples/announce/README.md`](examples/announce/README.md).
- `broadcast` — plain destination “channel” broadcast. See [`examples/broadcast/README.md`](examples/broadcast/README.md).
- `link` — basic link establishment + packet exchange. See [`examples/link/README.md`](examples/link/README.md).
- `identify` — link identification (`Link.Identify`) demo. See [`examples/identify/README.md`](examples/identify/README.md).
- `echo` — receipts/proofs + RTT measurement. See [`examples/echo/README.md`](examples/echo/README.md).
- `ratchets` — destination ratchets demo. See [`examples/ratchets/README.md`](examples/ratchets/README.md).
- `channel` — message-based Channel over Link. See [`examples/channel/README.md`](examples/channel/README.md).
- `buffer` — buffered read/write over Link Channel. See [`examples/buffer/README.md`](examples/buffer/README.md).
- `request` — request/response handlers over Link. See [`examples/request/README.md`](examples/request/README.md).
- `resource` — large transfers via `Resource`. See [`examples/resource/README.md`](examples/resource/README.md).
- `filetransfer` — directory listing + file download over `Resource`. See [`examples/filetransfer/README.md`](examples/filetransfer/README.md).
- `speedtest` — link throughput measurement. See [`examples/speedtest/README.md`](examples/speedtest/README.md).
- `interface` — add a custom interface programmatically (serial). See [`examples/interface/README.md`](examples/interface/README.md).

Quick run:

```bash
go run ./examples/minimal --help
go run ./examples/link --help
```

## Parity & docs

- Current parity target: Python Reticulum `1.1.4`.
- `PARITY_*.md` files track any remaining or newly discovered parity drift vs the Python reference (no unrelated wishlists).
- Ideally, each parity item is closed by a code change plus a test/verification step.
- Note: the Python Reticulum supports “external interfaces” by loading `<Type>.py` modules from `interfacepath`. The Go port does not execute/load those Python modules; if such a file exists, startup will error to avoid silently running with a placeholder interface.

## Disclaimer

This is a work-in-progress port: APIs, protocols, and CLI behaviour can change as parity improves.

## Support & Donate

- Monero: `41uoDd1PNKm7j4LaBHHZ77ZPbEwEJzaRHhjEqFtKLZeWjd4sNfs3mtpbw1mcQrnNLBKWSJgui9ELEUz217Ui6kF13SmF4t5`

## License

MIT – see [LICENSE](LICENSE).
