# PARITY: rnir.go

## Findings

- `--exampleconfig` is blocked by a bug in the local Python reference at [python/RNS/Utilities/rnir.py](/Users/alien/Vault/Projects/Self/Golang/Reticulum/go-reticulum/python/RNS/Utilities/rnir.py): it references `__example_rns_config__`, but that symbol is not defined, so Python currently exits with `NameError` before any Go/Python parity comparison is meaningful for that flag.
