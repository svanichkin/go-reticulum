# rnsd test2: rnprobe

Manual scenario for `rnprobe` against a local transport probe responder.

This test intentionally enables transport:

- `enable_transport = True`
- `respond_to_probes = Yes`

Do not use this as the baseline shared-instance/interface smoke test. `test1` keeps `enable_transport = False` to avoid transport rebroadcast/forwarding behaviour.

## Go

In one terminal:

```bash
./tests/hand/rnsd/test2/step1_rnsd_go.sh
```

In another terminal:

```bash
./tests/hand/rnsd/test2/step2_rnprobe_go.sh
```

## Python

In one terminal:

```bash
./tests/hand/rnsd/test2/step1_rnsd_py.sh
```

In another terminal:

```bash
./tests/hand/rnsd/test2/step2_rnprobe_py.sh
```

## Expected Result

- `step1` logs `Transport Instance will respond to probe requests on ...`.
- `step2` sends three probes to `rnstransport.probe`.
- `step2` reports `Sent 3, received 3, packet loss 0.0%`.

## Runtime Artifacts

Все runtime-артефакты этого теста складываются в:

```sh
tests/hand/rnsd/test2/.artifacts/run/
```
