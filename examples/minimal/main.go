package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const appName = "example_utilities"

func main() {
	var configDir string
	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.Parse()

	var configArg *string
	if strings.TrimSpace(configDir) != "" {
		configArg = &configDir
	}

	if err := programSetup(configArg, os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func programSetup(configDir *string, stdin io.Reader) error {
	_, err := rns.NewReticulum(configDir, nil, nil, nil, false, nil)
	if err != nil {
		return fmt.Errorf("NewReticulum: %w", err)
	}

	identity, err := rns.NewIdentity()
	if err != nil {
		return fmt.Errorf("NewIdentity: %w", err)
	}

	dest, err := rns.NewDestination(identity, rns.DestinationIN, rns.DestinationSINGLE, appName, "minimalsample")
	if err != nil {
		return fmt.Errorf("NewDestination: %w", err)
	}

	dest.SetProofStrategy(rns.DestinationPROVE_ALL)
	announceLoop(dest, stdin)
	return nil
}

func announceLoop(dest *rns.Destination, stdin io.Reader) {
	rns.Log("Minimal example "+rns.PrettyHexRep(dest.Hash)+" running, hit enter to manually send an announce (Ctrl-C to quit)", rns.LogInfo)

	in := bufio.NewScanner(stdin)
	for in.Scan() {
		dest.Announce(nil, false, nil, nil, true)
		rns.Log("Sent announce from "+rns.PrettyHexRep(dest.Hash), rns.LogInfo)
	}
}
