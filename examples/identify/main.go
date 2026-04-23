package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
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
	)
	flag.BoolVar(&server, "server", false, "wait for incoming link requests from clients")
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

	if err := runClient(configArg, destHex); err != nil {
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

	serverDest, err := rns.NewDestination(serverID, rns.DestinationIN, rns.DestinationSINGLE, appName, "identifyexample")
	if err != nil {
		return fmt.Errorf("NewDestination(server): %w", err)
	}

	var (
		latestMu sync.Mutex
		latest   *rns.Link
	)

	serverDest.SetLinkEstablishedCallback(func(link *rns.Link) {
		if link == nil {
			return
		}
		rns.Log("Client connected", rns.LogInfo)
		link.SetLinkClosedCallback(func(*rns.Link) { rns.Log("Client disconnected", rns.LogInfo) })
		link.SetRemoteIdentifiedCallback(func(_ *rns.Link, id *rns.Identity) {
			if id != nil {
				rns.Log("Remote identified as: "+id.String(), rns.LogInfo)
			}
		})
		link.SetPacketCallback(func(message []byte, packet *rns.Packet) {
			latestMu.Lock()
			l := latest
			latestMu.Unlock()
			if l == nil {
				l = link
			}

			remote := "unidentified peer"
			if rid := l.GetRemoteIdentity(); rid != nil {
				remote = rid.String()
			}

			text := string(message)
			rns.Log("Received data from "+remote+": "+text, rns.LogInfo)

			reply := "I received \"" + text + "\" over the link from " + remote
			_ = rns.NewPacket(l, []byte(reply), rns.PacketTypeData, rns.PacketCtxNone, rns.Broadcast, rns.HeaderType1, nil, nil, true, rns.FlagUnset).Send()
		})

		latestMu.Lock()
		latest = link
		latestMu.Unlock()
	})

	rns.Log("Link identification example "+rns.PrettyHexRep(serverDest.Hash)+" running, waiting for a connection.", rns.LogInfo)
	rns.Log("Hit enter to manually send an announce (Ctrl-C to quit)", rns.LogInfo)

	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		serverDest.Announce(nil, false, nil, nil, true)
		rns.Log("Sent announce from "+rns.PrettyHexRep(serverDest.Hash), rns.LogInfo)
	}
	return in.Err()
}

// ---------------- client ----------------

func runClient(configDir *string, destinationHexHash string) error {
	destinationHash, err := parseTruncatedHashHex(destinationHexHash)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	_, err = rns.NewReticulum(configDir, nil, nil, nil, false, nil)
	if err != nil {
		return fmt.Errorf("NewReticulum: %w", err)
	}

	clientID, err := rns.NewIdentity()
	if err != nil {
		return fmt.Errorf("NewIdentity(client): %w", err)
	}
	rns.Log("Client created new identity "+clientID.String(), rns.LogInfo)

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
	serverDest, err := rns.NewDestination(serverID, rns.DestinationOUT, rns.DestinationSINGLE, appName, "identifyexample")
	if err != nil {
		return fmt.Errorf("NewDestination(server out): %w", err)
	}

	var (
		serverLinkMu sync.Mutex
		serverLink   *rns.Link
	)

	link, err := rns.NewLink(serverDest, nil, rns.LinkModeDefault, func(l *rns.Link) {
		serverLinkMu.Lock()
		serverLink = l
		serverLinkMu.Unlock()

		rns.Log("Link established with server, identifying to remote peer...", rns.LogInfo)
		l.Identify(clientID)
	}, func(l *rns.Link) {
		rns.Log("Link closed, exiting now", rns.LogInfo)
		os.Exit(0)
	})
	if err != nil || link == nil {
		return fmt.Errorf("NewOutgoingLink: %v", err)
	}

	link.SetPacketCallback(func(message []byte, packet *rns.Packet) {
		rns.Log("Received data on the link: "+string(message), rns.LogInfo)
		fmt.Print("> ")
	})

	// Wait for the link to become active (serverLink set).
	for {
		serverLinkMu.Lock()
		active := serverLink != nil
		serverLinkMu.Unlock()
		if active {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return clientLoop(func() *rns.Link {
		serverLinkMu.Lock()
		defer serverLinkMu.Unlock()
		return serverLink
	})
}

func clientLoop(getLink func() *rns.Link) error {
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		text, err := r.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				if l := getLink(); l != nil {
					l.Teardown()
				}
				return nil
			}
			if l := getLink(); l != nil {
				l.Teardown()
			}
			return err
		}
		text = strings.TrimSpace(text)

		if text == "quit" || text == "q" || text == "exit" {
			if l := getLink(); l != nil {
				l.Teardown()
			}
			return nil
		}

		if text == "" {
			continue
		}

		l := getLink()
		if l == nil {
			return errors.New("link not active")
		}

		data := []byte(text)
		if len(data) > l.MDU {
			rns.Log(fmt.Sprintf("Cannot send this packet, the data size of %d bytes exceeds the link packet MDU of %d bytes", len(data), l.MDU), rns.LogError)
			continue
		}
		_ = rns.NewPacket(l, data, rns.PacketTypeData, rns.PacketCtxNone, rns.Broadcast, rns.HeaderType1, nil, nil, true, rns.FlagUnset).Send()
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
