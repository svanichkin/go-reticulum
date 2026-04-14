# rnsd hand test 5: MeshChat announce watch

Manual scenario for watching announce traffic from Reticulum MeshChat against
the same base `rnsd` config in Go and Python.

Use the Go step in one terminal:

```sh
./tests/hand/rnsd/test5/step1_rnsd_go.sh
```

Use the Python step in another terminal:

```sh
./tests/hand/rnsd/test5/step1_rnsd_py.sh
```

The scripts start from the same `config` file in this directory and write
runtime artifacts under:

```sh
tests/hand/rnsd/test5/.artifacts/run/
```

For isolation, each script rewrites the local shared-instance name and ports in
its runtime copy so the Go and Python runs do not collide with each other or
with an already running default instance.

The TCP server interface itself still listens on port `4242`, so that port must
be free before you start the step script.

After the daemon reports startup and the TCP server interface is listening,
open Reticulum MeshChat manually and watch the daemon log stream for announce
activity.

Stop the daemon with `Ctrl+C` in the step terminal.
