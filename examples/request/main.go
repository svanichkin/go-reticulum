package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const appName = "example_utilities"

var randomTexts = []string{
	"They looked up",
	"On each full moon",
	"Becky was upset",
	"I’ll stay away from it",
	"The pet shop stocks everything",
}

func main() {
	var (
		server    bool
		destHex   string
		configDir string
	)
	flag.BoolVar(&server, "server", false, "wait for incoming requests from clients")
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

	serverDest, err := rns.NewDestination(serverID, rns.DestinationIN, rns.DestinationSINGLE, appName, "requestexample")
	if err != nil {
		return fmt.Errorf("NewDestination(server): %w", err)
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	if err := serverDest.RegisterRequestHandler(
		"/random/text",
		func(path string, data any, requestID []byte, linkID []byte, remoteIdentity *rns.Identity, requestedAt time.Time) any {
			rns.Log("Generating response to request "+rns.PrettyHexRep(requestID)+" on link "+rns.PrettyHexRep(linkID), rns.LogInfo)
			return randomTexts[rng.Intn(len(randomTexts))]
		},
		rns.DestinationALLOW_ALL,
		nil,
	); err != nil {
		return fmt.Errorf("RegisterRequestHandler: %w", err)
	}

	serverDest.SetLinkEstablishedCallback(func(link *rns.Link) {
		if link == nil {
			return
		}
		rns.Log("Client connected", rns.LogInfo)
		link.SetLinkClosedCallback(func(*rns.Link) { rns.Log("Client disconnected", rns.LogInfo) })
	})

	rns.Log("Request example "+rns.PrettyHexRep(serverDest.Hash)+" running, waiting for a connection.", rns.LogInfo)
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
	serverDest, err := rns.NewDestination(serverID, rns.DestinationOUT, rns.DestinationSINGLE, appName, "requestexample")
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
		rns.Log("Link established with server, hit enter to perform a request, or type in \"quit\" to quit", rns.LogInfo)
	}, func(l *rns.Link) {
		rns.Log("Link closed, exiting now", rns.LogInfo)
		os.Exit(0)
	})
	if err != nil || link == nil {
		return fmt.Errorf("NewOutgoingLink: %v", err)
	}

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
	in := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !in.Scan() {
			if l := getLink(); l != nil {
				l.Teardown()
			}
			return in.Err()
		}
		text := strings.TrimSpace(in.Text())

		if text == "quit" || text == "q" || text == "exit" {
			if l := getLink(); l != nil {
				l.Teardown()
			}
			return nil
		}

		l := getLink()
		if l == nil {
			return errors.New("link not active")
		}

		l.Request(
			"/random/text",
			nil,
			func(rr *rns.RequestReceipt) {
				if rr == nil {
					return
				}
				rns.Log("Got response for request "+rns.PrettyHexRep(rr.GetRequestID())+": "+fmt.Sprint(rr.GetResponse()), rns.LogInfo)
			},
			func(rr *rns.RequestReceipt) {
				if rr == nil {
					return
				}
				rns.Log("The request "+rns.PrettyHexRep(rr.GetRequestID())+" failed.", rns.LogInfo)
			},
			nil,
			0,
		)
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
