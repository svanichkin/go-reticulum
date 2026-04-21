package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const appName = "example_utilities"

func main() {
	var (
		server    bool
		timeoutS  float64
		destHex   string
		configDir string
	)
	flag.BoolVar(&server, "server", false, "wait for incoming packets from clients")
	flag.Float64Var(&timeoutS, "timeout", 0, "set a reply timeout in seconds (client mode)")
	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.StringVar(&destHex, "destination", "", "hexadecimal hash of the server destination (client mode)")
	flag.Parse()

	var configArg *string
	if strings.TrimSpace(configDir) != "" {
		configArg = &configDir
	}

	if server {
		if err := runServer(configArg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if strings.TrimSpace(destHex) == "" {
		fmt.Fprintln(os.Stderr, "missing -destination (or pass -server)")
		flag.Usage()
		os.Exit(2)
	}

	var timeout *time.Duration
	if timeoutS > 0 {
		d := time.Duration(timeoutS * float64(time.Second))
		timeout = &d
	}
	if err := runClient(configArg, destHex, timeout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ---------------- server ----------------

func runServer(configDir *string) error {
	_, err := rns.NewReticulum(configDir, nil, nil, nil, false, nil)
	if err != nil {
		return fmt.Errorf("NewReticulum: %w", err)
	}

	serverID, err := rns.NewIdentity()
	if err != nil {
		return fmt.Errorf("NewIdentity: %w", err)
	}

	dest, err := rns.NewDestination(serverID, rns.DestinationIN, rns.DestinationSINGLE, appName, "ratchet", "echo", "request")
	if err != nil {
		return fmt.Errorf("NewDestination: %w", err)
	}

	// Python example uses /tmp/<hexhash>.ratchets. Use OS temp dir.
	hexHash := rns.HexRep(dest.Hash, false)
	ratchetsPath := filepath.Join(os.TempDir(), hexHash+".ratchets")
	if _, err := dest.EnableRatchets(ratchetsPath); err != nil {
		return fmt.Errorf("EnableRatchets(%s): %w", ratchetsPath, err)
	}
	// Make ratchets rotate frequently for the example.
	_ = dest.SetRatchetInterval(1)

	dest.SetProofStrategy(rns.DestinationPROVE_ALL)
	dest.SetPacketCallback(func(message []byte, packet *rns.Packet) {
		stats := ""
		if packet != nil {
			if rssi := packet.GetRSSI(); rssi != nil {
				stats += fmt.Sprintf(" [RSSI %.1f dBm]", *rssi)
			}
			if snr := packet.GetSNR(); snr != nil {
				stats += fmt.Sprintf(" [SNR %.1f dB]", *snr)
			}
		}
		rns.Log("Received packet from echo client, proof sent"+stats, rns.LogInfo)
	})

	rns.Log("Ratcheted echo server "+rns.PrettyHexRep(dest.Hash)+" running, hit enter to manually send an announce (Ctrl-C to quit)", rns.LogInfo)

	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		dest.Announce(nil, false, nil, nil, true)
		rns.Log("Sent announce from "+rns.PrettyHexRep(dest.Hash), rns.LogInfo)
	}
	return in.Err()
}

// ---------------- client ----------------

func runClient(configDir *string, destinationHexHash string, timeout *time.Duration) error {
	destinationHash, err := parseTruncatedHashHex(destinationHexHash)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	_, err = rns.NewReticulum(configDir, nil, nil, nil, false, nil)
	if err != nil {
		return fmt.Errorf("NewReticulum: %w", err)
	}

	rns.Log("Echo client ready, hit enter to send echo request to "+destinationHexHash+" (Ctrl-C to quit)", rns.LogInfo)

	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		if rns.HasPath(destinationHash) {
			serverID := rns.IdentityRecall(destinationHash)
			if serverID == nil {
				return errors.New("could not recall server identity")
			}

			dest, err := rns.NewDestination(serverID, rns.DestinationOUT, rns.DestinationSINGLE, appName, "ratchet", "echo", "request")
			if err != nil {
				return fmt.Errorf("NewDestination(out): %w", err)
			}

			req := rns.NewPacket(dest, rns.IdentityGetRandomHash())
			rc := req.Send()
			if rc == nil {
				return errors.New("send returned nil receipt")
			}
			if timeout != nil {
				rc.SetTimeout(timeout.Seconds())
				rc.SetTimeoutCallback(func(receipt *rns.PacketReceipt) {
					rns.Log("Packet "+rns.PrettyHexRep(receipt.Hash)+" timed out", rns.LogInfo)
				})
			}
			rc.SetDeliveryCallback(func(receipt *rns.PacketReceipt) {
				if receipt == nil || receipt.Status != rns.ReceiptDelivered {
					return
				}
				rtt := receipt.GetRTT()
				rttString := fmt.Sprintf("%.3f seconds", rtt)
				if rtt < 1 {
					rttString = fmt.Sprintf("%.3f milliseconds", rtt*1000)
				}

				stats := ""
				if pp := receipt.ProofPacket; pp != nil {
					if rssi := pp.GetRSSI(); rssi != nil {
						stats += fmt.Sprintf(" [RSSI %.1f dBm]", *rssi)
					}
					if snr := pp.GetSNR(); snr != nil {
						stats += fmt.Sprintf(" [SNR %.1f dB]", *snr)
					}
				}
				dst := "<unknown>"
				if receipt.Destination != nil {
					dst = rns.PrettyHexRep(receipt.Destination.Hash)
				}
				rns.Log("Valid reply received from "+dst+", round-trip time is "+rttString+stats, rns.LogInfo)
			})

			rns.Log("Sent echo request to "+rns.PrettyHexRep(dest.Hash), rns.LogInfo)
		} else {
			rns.Log("Destination is not yet known. Requesting path...", rns.LogInfo)
			rns.Log("Hit enter to manually retry once an announce is received.", rns.LogInfo)
			rns.RequestPath(destinationHash, nil, nil, false)
		}
	}
	return in.Err()
}

func parseTruncatedHashHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	destLen := (rns.TRUNCATED_HASHLENGTH / 8) * 2
	if len(s) != destLen {
		return nil, fmt.Errorf("invalid destination length: got %d want %d", len(s), destLen)
	}
	return hex.DecodeString(s)
}
