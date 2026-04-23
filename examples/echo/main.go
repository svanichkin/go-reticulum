package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const appName = "example_utilities"

func main() {
	var (
		configDir string
		server    bool
		timeout   float64
	)
	flag.BoolVar(&server, "server", false, "wait for incoming packets from clients")
	flag.BoolVar(&server, "s", false, "wait for incoming packets from clients")
	flag.Float64Var(&timeout, "timeout", 0, "set a reply timeout in seconds")
	flag.Float64Var(&timeout, "t", 0, "set a reply timeout in seconds")
	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.Parse()

	var configArg *string
	if configDir != "" {
		configArg = &configDir
	}

	if server {
		runServer(configArg)
		return
	}

	if flag.NArg() < 1 {
		fmt.Println("Usage:")
		fmt.Println("  echo --server [--config DIR]")
		fmt.Println("  echo [--config DIR] [--timeout s] <destination_hash_hex>")
		os.Exit(2)
	}
	runClient(flag.Arg(0), configArg, timeout)
}

func runServer(configDir *string) {
	_, err := rns.NewReticulum(configDir, nil, nil, nil, false, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewReticulum:", err)
		os.Exit(1)
	}

	serverIdentity, err := rns.NewIdentity()
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewIdentity:", err)
		os.Exit(1)
	}

	dest, err := rns.NewDestination(serverIdentity, rns.DestinationIN, rns.DestinationSINGLE, appName, "echo", "request")
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewDestination:", err)
		os.Exit(1)
	}
	dest.SetProofStrategy(rns.DestinationPROVE_ALL)

	dest.SetPacketCallback(func(_ []byte, packet *rns.Packet) {
		rns.Log("Received packet from echo client, proof sent", rns.LogInfo)
		_ = packet
	})

	rns.Log(
		"Echo server "+rns.PrettyHexRep(dest.Hash)+" running, hit enter to manually send an announce (Ctrl-C to quit)",
		rns.LogInfo,
	)

	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		dest.Announce(nil, false, nil, nil, true)
		rns.Log("Sent announce from "+rns.PrettyHexRep(dest.Hash), rns.LogInfo)
	}
}

func runClient(destinationHex string, configDir *string, timeout float64) {
	destLen := (rns.ReticulumTruncatedHashLength / 8) * 2
	if len(destinationHex) != destLen {
		fmt.Fprintln(os.Stderr, "Invalid destination hash length")
		os.Exit(2)
	}
	destinationHash, err := hex.DecodeString(destinationHex)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Invalid destination hash hex")
		os.Exit(2)
	}

	_, err = rns.NewReticulum(configDir, nil, nil, nil, false, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewReticulum:", err)
		os.Exit(1)
	}

	rns.Log(
		"Echo client ready, hit enter to send echo request to "+destinationHex+" (Ctrl-C to quit)",
		rns.LogInfo,
	)

	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		if rns.HasPath(destinationHash) {
			serverIdentity := rns.IdentityRecall(destinationHash)
			if serverIdentity == nil {
				rns.Log("Destination is known but identity could not be recalled", rns.LogError)
				continue
			}
			requestDest, err := rns.NewDestination(serverIdentity, rns.DestinationOUT, rns.DestinationSINGLE, appName, "echo", "request")
			if err != nil {
				rns.Log("Could not create request destination: "+err.Error(), rns.LogError)
				continue
			}
			echoReq := rns.NewPacket(requestDest, rns.IdentityGetRandomHash(), rns.PacketTypeData, rns.PacketCtxNone, rns.Broadcast, rns.HeaderType1, nil, nil, true, rns.FlagUnset)
			if echoReq == nil {
				continue
			}
			receipt := echoReq.Send()
			if receipt == nil {
				rns.Log("Echo request could not be sent", rns.LogError)
				continue
			}
			if timeout > 0 {
				receipt.SetTimeout(timeout)
				receipt.SetTimeoutCallback(func(r *rns.PacketReceipt) {
					if r.Status == rns.ReceiptFailed {
						rns.Log("Packet "+rns.PrettyHexRep(r.Hash)+" timed out", rns.LogError)
					}
				})
			}
			receipt.SetDeliveryCallback(func(r *rns.PacketReceipt) {
				if r.Status != rns.ReceiptDelivered {
					return
				}
				rtt := r.GetRTT()
				if rtt >= 1 {
					rns.Log(fmt.Sprintf("Valid reply received from %s, round-trip time is %.3f seconds", rns.PrettyHexRep(r.Destination.Hash), rtt), rns.LogInfo)
				} else {
					rns.Log(fmt.Sprintf("Valid reply received from %s, round-trip time is %.3f milliseconds", rns.PrettyHexRep(r.Destination.Hash), rtt*1000), rns.LogInfo)
				}
			})
			rns.Log("Sent echo request to "+rns.PrettyHexRep(requestDest.Hash), rns.LogInfo)
		} else {
			rns.Log("Destination is not yet known. Requesting path...", rns.LogInfo)
			rns.RequestPath(destinationHash, nil, nil, false)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
