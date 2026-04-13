# rnsd hand test 3: rncp cross-language transfer

This scenario checks `rncp` file transfer between Go and Python Reticulum
nodes connected to the same public Reticulum network via the shared `config`
file.

Start both Go nodes in one terminal:

```sh
./tests/hand/rnsd/test3/step1_rnsd_go.sh
```

Start both Python nodes in another terminal:

```sh
./tests/hand/rnsd/test3/step1_rnsd_py.sh
```

Wait for all four nodes to connect to the network and discover each other
(usually 10–30 seconds after "Started rnsd version" appears in all logs).

Then run either cross-language transfer step:

```sh
./tests/hand/rnsd/test3/step2_rncp_go.sh
./tests/hand/rnsd/test3/step2_rncp_py.sh
```

`step2_rncp_go.sh` sends Go client → Python server, then returns the received
payload Python client → Go server.

`step2_rncp_py.sh` sends Python client → Go server, then returns the received
payload Go client → Python server.

Both steps compare SHA-256 hashes between the original and received files.
Press `Ctrl+C` in both `step1` terminals to stop the nodes.

Runtime artifacts for this hand test are written to:

```sh
tests/hand/rnsd/test3/.artifacts/run/
```

That directory contains generated configs, logs, pid files, storage, and
cross-transfer outputs for both Go and Python nodes.
