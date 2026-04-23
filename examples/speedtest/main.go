package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const appName = "example_utilities"

func main() {
	var (
		server    bool
		destHex   string
		configDir string
		capMB     int
	)
	flag.BoolVar(&server, "server", false, "wait for incoming requests from clients")
	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.StringVar(&destHex, "destination", "", "hexadecimal hash of the server destination (client mode)")
	flag.IntVar(&capMB, "cap-mb", 2, "how much data to send/receive before printing stats")
	flag.Parse()

	var configArg *string
	if strings.TrimSpace(configDir) != "" {
		configArg = &configDir
	}
	if capMB <= 0 {
		capMB = 2
	}
	dataCap := capMB * 1024 * 1024

	if server {
		if err := runServer(configArg, dataCap); err != nil {
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

	if err := runClient(configArg, destHex, dataCap); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ---------------- server ----------------

func runServer(configDir *string, dataCap int) error {
	_, err := rns.NewReticulum(configDir, nil, nil, nil, false, nil)
	if err != nil {
		return fmt.Errorf("NewReticulum: %w", err)
	}

	serverID, err := rns.NewIdentity()
	if err != nil {
		return fmt.Errorf("NewIdentity: %w", err)
	}

	serverDest, err := rns.NewDestination(serverID, rns.DestinationIN, rns.DestinationSINGLE, appName, "speedtest")
	if err != nil {
		return fmt.Errorf("NewDestination(server): %w", err)
	}

	serverDest.SetLinkEstablishedCallback(func(link *rns.Link) {
		if link == nil {
			return
		}
		rns.Log("Client connected", rns.LogInfo)
		started := time.Now()

		var (
			mu           sync.Mutex
			received     int
			printCounter int
		)

		link.SetPacketCallback(func(message []byte, packet *rns.Packet) {
			mu.Lock()
			defer mu.Unlock()

			received += len(message)
			printCounter++
			if printCounter >= 50 {
				rns.Log(sizeStr(float64(received)), rns.LogInfo)
				printCounter = 0
			}

			if received > dataCap {
				dur := time.Since(started)
				if dur <= 0 {
					dur = time.Millisecond
				}
				fmt.Println("")
				fmt.Println("")
				fmt.Println("--- Statistics -----")
				fmt.Printf("\tTime taken       : %s\n", formatDuration(dur))
				fmt.Printf("\tData transferred : %s\n", sizeStr(float64(received)))
				fmt.Printf("\tTransfer rate    : %s/s\n", sizeStr(float64(received)/dur.Seconds(), "b"))
				fmt.Println("")
				_ = os.Stdout.Sync()

				received = 0
				printCounter = 0
				link.Teardown()
			}
		})
		link.SetLinkClosedCallback(func(*rns.Link) { rns.Log("Client disconnected", rns.LogInfo) })
	})

	rns.Log("Speedtest "+rns.PrettyHexRep(serverDest.Hash)+" running, waiting for a connection.", rns.LogInfo)
	rns.Log("Hit enter to manually send an announce (Ctrl-C to quit)", rns.LogInfo)

	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		serverDest.Announce(nil, false, nil, nil, true)
		rns.Log("Sent announce from "+rns.PrettyHexRep(serverDest.Hash), rns.LogInfo)
	}
	return in.Err()
}

// ---------------- client ----------------

func runClient(configDir *string, destinationHexHash string, dataCap int) error {
	destinationHash, err := parseTruncatedHashHex(destinationHexHash)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	_, err = rns.NewReticulum(configDir, nil, nil, nil, false, nil)
	if err != nil {
		return fmt.Errorf("NewReticulum: %w", err)
	}

	if !rns.HasPath(destinationHash) {
		rns.Log("Destination is not yet known. Requesting path and waiting for announce to arrive...", rns.LogInfo)
		rns.RequestPath(destinationHash, nil, nil, false)
		for !rns.HasPath(destinationHash) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	serverID := rns.IdentityRecall(destinationHash)
	if serverID == nil {
		return errors.New("could not recall server identity")
	}

	rns.Log("Establishing link with server...", rns.LogInfo)
	serverDest, err := rns.NewDestination(serverID, rns.DestinationOUT, rns.DestinationSINGLE, appName, "speedtest")
	if err != nil {
		return fmt.Errorf("NewDestination(server out): %w", err)
	}

	link, err := rns.NewLink(serverDest, nil, rns.LinkModeDefault, func(l *rns.Link) {
		rns.Log("Link established with server, sending...", rns.LogInfo)

		mdu := l.MDU
		if mdu <= 0 {
			mdu = 1
		}
		payload := make([]byte, mdu)
		_, _ = rand.Read(payload)

		started := time.Now()
		printed := false
		dataSent := 0

		for l.Status == rns.LinkActive && dataSent < int(float64(dataCap)*1.25) {
			p := rns.NewPacket(l, payload, rns.PacketTypeData, rns.PacketCtxNone, rns.Broadcast, rns.HeaderType1, nil, nil, false, rns.FlagUnset)
			if p == nil {
				break
			}
			_ = p.Send()
			dataSent += len(payload)

			if dataSent > dataCap && !printed {
				printed = true
				dur := time.Since(started)
				if dur <= 0 {
					dur = time.Millisecond
				}
				fmt.Println("")
				fmt.Println("")
				fmt.Println("--- Statistics -----")
				fmt.Printf("\tTime taken       : %s\n", formatDuration(dur))
				fmt.Printf("\tData transferred : %s\n", sizeStr(float64(dataSent)))
				fmt.Printf("\tTransfer rate    : %s/s\n", sizeStr(float64(dataSent)/dur.Seconds(), "b"))
				fmt.Println("")
				_ = os.Stdout.Sync()
				time.Sleep(100 * time.Millisecond)
			}
		}
	}, func(l *rns.Link) {
		rns.Log("Link closed, exiting now", rns.LogInfo)
		os.Exit(0)
	})
	if err != nil || link == nil {
		return fmt.Errorf("NewOutgoingLink: %v", err)
	}

	_ = link
	for {
		time.Sleep(200 * time.Millisecond)
	}
}

func parseTruncatedHashHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	destLen := (rns.TRUNCATED_HASHLENGTH / 8) * 2
	if len(s) != destLen {
		return nil, fmt.Errorf("invalid destination length: got %d want %d", len(s), destLen)
	}
	return hex.DecodeString(s)
}

func formatDuration(d time.Duration) string {
	// Match Python's H:M:S formatting, with seconds as float.
	secs := d.Seconds()
	h := int(secs) / 3600
	m := (int(secs) % 3600) / 60
	s := secs - float64(h*3600+m*60)
	return fmt.Sprintf("%02d:%02d:%05.2f", h, m, s)
}

func sizeStr(num float64, suffix ...string) string {
	suf := "B"
	if len(suffix) > 0 {
		suf = suffix[0]
	}
	units := []string{"", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi"}
	lastUnit := "Yi"

	if suf == "b" {
		num *= 8
		units = []string{"", "K", "M", "G", "T", "P", "E", "Z"}
		lastUnit = "Y"
	}

	for _, u := range units {
		if num < 1024.0 {
			return fmt.Sprintf("%3.2f %s%s", num, u, suf)
		}
		num /= 1024.0
	}
	return fmt.Sprintf("%.2f %s%s", num, lastUnit, suf)
}
