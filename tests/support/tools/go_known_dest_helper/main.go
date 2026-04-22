package main

import (
	"encoding/hex"
	"fmt"
	"os"

	rns "github.com/svanichkin/go-reticulum/rns"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: go_known_dest_helper <config-dir> <destination-hash-hex> <public-key-hex>")
		os.Exit(2)
	}

	configDir := os.Args[1]
	destinationHash, err := hex.DecodeString(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode destination hash: %v\n", err)
		os.Exit(1)
	}
	publicKey, err := hex.DecodeString(os.Args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode public key: %v\n", err)
		os.Exit(1)
	}

	logLevel := 0
	if _, err := rns.NewReticulum(&configDir, &logLevel, nil, nil, false, nil); err != nil {
		fmt.Fprintf(os.Stderr, "start reticulum: %v\n", err)
		os.Exit(1)
	}
	rns.IdentityRemember([]byte("pkt"), destinationHash, publicKey, nil)
	if err := rns.IdentitySaveKnownDestinations(); err != nil {
		fmt.Fprintf(os.Stderr, "save known destinations: %v\n", err)
		os.Exit(1)
	}
}
