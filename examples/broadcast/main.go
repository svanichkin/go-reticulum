package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const appName = "example_utilities"

func main() {
	var (
		configDir string
		channel   string
	)
	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.StringVar(&channel, "channel", "", "broadcast channel name")
	flag.Parse()

	var configArg *string
	if configDir != "" {
		configArg = &configDir
	}
	if channel == "" {
		channel = "public_information"
	}

	_, err := rns.NewReticulum(configArg, nil, nil, nil, false, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewReticulum:", err)
		os.Exit(1)
	}

	dest, err := rns.NewDestination(nil, rns.DestinationIN, rns.DestinationPLAIN, appName, "broadcast", channel)
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewDestination:", err)
		os.Exit(1)
	}

	dest.SetPacketCallback(func(data []byte, _ *rns.Packet) {
		fmt.Println()
		fmt.Printf("Received data: %s\r\n> ", string(data))
	})

	rns.Log(
		"Broadcast example "+rns.PrettyHexRep(dest.Hash)+" running, enter text and hit enter to broadcast (Ctrl-C to quit)",
		rns.LogInfo,
	)

	in := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !in.Scan() {
			return
		}
		entered := strings.TrimRight(in.Text(), "\r\n")
		if entered == "" {
			continue
		}
		pkt := rns.NewPacket(dest, []byte(entered))
		if pkt == nil {
			continue
		}
		_ = pkt.Send()
	}
}
