# rnsd hand test 4: TCP announce loop guard

Manual scenario for checking that a single announce does not come back around
a TCP transport ring and get delivered to local clients more than once.

Topology for both implementations:

```text
origin(A) -> relay-b(B) -> relay-c(C) -> origin(A)
```

Each node has:

- one `TCPServerInterface` for the previous hop
- one `TCPClientInterface` to the next hop

Only `origin(A)` exposes a shared instance. In `step2`:

1. an observer utility connects to `origin(A)` and registers an announce handler
2. `rnid` sends exactly one announce from `origin(A)`
3. the observer counts how many times it receives that aspect

Expected result for both Python and Go:

- observer count is exactly `1`
- relay-b and relay-c each log one path update for the announced destination

If announces loop back into `origin(A)` and are re-forwarded to local clients,
the observer count becomes `2+`, which is a failure.

## Bootstrap variant

There is also a bootstrap-specific variant in `step3`/`step4` that uses
external `TCPClientInterface` bootstrap peers instead of a closed local TCP ring.
It follows the same observer idea, but exercises the announce path through the
bootstrap network:

- `step3` starts the bootstrap-configured `rnsd` instances
- `step4` resolves the announce hash, starts the observer, and sends one announce

The bootstrap scripts are intentionally separate because their timeout and
network behavior are different from the local ring test.

## Go

In one terminal:

```bash
./tests/hand/rnsd/test4/step1_rnsd_go.sh
```

In another terminal:

```bash
./tests/hand/rnsd/test4/step2_rnid_go.sh
```

Bootstrap variant:

```bash
./tests/hand/rnsd/test4/step3_rnsd_bootstrap_go.sh
./tests/hand/rnsd/test4/step4_rnid_bootstrap_go.sh
```

## Python

In one terminal:

```bash
./tests/hand/rnsd/test4/step1_rnsd_py.sh
```

In another terminal:

```bash
./tests/hand/rnsd/test4/step2_rnid_py.sh
```

Bootstrap variant:

```bash
./tests/hand/rnsd/test4/step3_rnsd_bootstrap_py.sh
./tests/hand/rnsd/test4/step4_rnid_bootstrap_py.sh
```

## Runtime Artifacts

All runtime artifacts are written to:

```sh
tests/hand/rnsd/test4/.artifacts/run/
```
