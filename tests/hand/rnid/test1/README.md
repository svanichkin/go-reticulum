# rnid hand test 4: cross-language identity and crypto parity

This scenario checks `rnid` compatibility between the Go and Python
implementations without relying on the public Reticulum network.

The test uses isolated runtime artifacts under:

```sh
tests/hand/rnid/test4/.artifacts/run/
```

Run the six steps in order:

```sh
./tests/hand/rnid/test4/step1_generate_print_go.sh
./tests/hand/rnid/test4/step1_generate_print_py.sh
./tests/hand/rnid/test4/step2_export_import_go.sh
./tests/hand/rnid/test4/step2_export_import_py.sh
./tests/hand/rnid/test4/step3_sign_validate_go.sh
./tests/hand/rnid/test4/step3_sign_validate_py.sh
```

What is covered:

- Go `rnid` generate -> Python `rnid` print
- Python `rnid` generate -> Go `rnid` print
- Go export -> Python import
- Python export -> Go import
- Go sign -> Python validate
- Python sign -> Go validate

Each step prints the final identity hash and an `OK` cross-check summary when
the Go and Python outputs agree.
