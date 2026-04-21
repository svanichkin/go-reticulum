package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const appName = "example_utilities"

var fruits = []string{"Peach", "Quince", "Date", "Tangerine", "Pomelo", "Carambola", "Grape"}
var nobleGases = []string{"Helium", "Neon", "Argon", "Krypton", "Xenon", "Radon", "Oganesson"}

type exampleAnnounceHandler struct {
	aspectFilter string
}

func (h *exampleAnnounceHandler) AspectFilter() string { return h.aspectFilter }

func (h *exampleAnnounceHandler) ReceivedAnnounce(destinationHash []byte, announcedIdentity *rns.Identity, appData []byte) {
	rns.Log("Received an announce from "+rns.PrettyHexRep(destinationHash), rns.LogInfo)
	if len(appData) > 0 {
		rns.Log("The announce contained the following app data: "+string(appData), rns.LogInfo)
	}
}

func main() {
	var configDir string
	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.Parse()

	var configArg *string
	if configDir != "" {
		configArg = &configDir
	}

	_, err := rns.NewReticulum(configArg, nil, nil, nil, false, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewReticulum:", err)
		os.Exit(1)
	}

	identity, err := rns.NewIdentity()
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewIdentity:", err)
		os.Exit(1)
	}

	destFruits, err := rns.NewDestination(identity, rns.DestinationIN, rns.DestinationSINGLE, appName, "announcesample", "fruits")
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewDestination(fruits):", err)
		os.Exit(1)
	}
	destGases, err := rns.NewDestination(identity, rns.DestinationIN, rns.DestinationSINGLE, appName, "announcesample", "noble_gases")
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewDestination(gases):", err)
		os.Exit(1)
	}

	destFruits.SetProofStrategy(rns.DestinationPROVE_ALL)
	destGases.SetProofStrategy(rns.DestinationPROVE_ALL)

	rns.RegisterAnnounceHandler(&exampleAnnounceHandler{
		aspectFilter: "example_utilities.announcesample.fruits",
	})

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rns.Log("Announce example running, hit enter to manually send an announce (Ctrl-C to quit)", rns.LogInfo)

	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		fruit := fruits[rng.Intn(len(fruits))]
		destFruits.Announce([]byte(fruit), false, nil, nil, true)
		rns.Log("Sent announce from "+rns.PrettyHexRep(destFruits.Hash), rns.LogInfo)

		gas := nobleGases[rng.Intn(len(nobleGases))]
		destGases.Announce([]byte(gas), false, nil, nil, true)
		rns.Log("Sent announce from "+rns.PrettyHexRep(destGases.Hash), rns.LogInfo)
	}
}
