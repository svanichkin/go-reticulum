package main

import (
	"bufio"
	"crypto/rand"
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
		sizeMB    int
	)
	flag.BoolVar(&server, "server", false, "wait for incoming resources from clients")
	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.StringVar(&destHex, "destination", "", "hexadecimal hash of the server destination (client mode)")
	flag.IntVar(&sizeMB, "size-mb", 32, "resource size in megabytes (client mode)")
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
	if sizeMB <= 0 {
		sizeMB = 32
	}

	if err := runClient(configArg, destHex, sizeMB); err != nil {
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

	serverDest, err := rns.NewDestination(serverID, rns.DestinationIN, rns.DestinationSINGLE, appName, "resourceexample")
	if err != nil {
		return fmt.Errorf("NewDestination(server): %w", err)
	}

	serverDest.SetLinkEstablishedCallback(func(link *rns.Link) {
		if link == nil {
			return
		}
		rns.Log("Client connected", rns.LogInfo)

		link.SetResourceStrategy(rns.LinkAcceptAll)
		link.SetResourceConcludedCallback(func(res *rns.Resource) {
			if res == nil {
				return
			}
			if res.Status() != rns.ResourceComplete {
				rns.Log(fmt.Sprintf("Receiving resource %s failed", res), rns.LogError)
				return
			}

			rns.Log(fmt.Sprintf("Resource %s received", res), rns.LogInfo)
			rns.Log(fmt.Sprintf("Metadata: %#v", res.Metadata()), rns.LogInfo)
			rns.Log(fmt.Sprintf("Data can be moved or copied from: %s", res.DataFile()), rns.LogInfo)

			if first, err := readFirstN(res.DataFile(), 32); err == nil {
				rns.Log("First 32 bytes of data: "+rns.HexRep(first), rns.LogInfo)
			}
		})

		link.SetLinkClosedCallback(func(*rns.Link) { rns.Log("Client disconnected", rns.LogInfo) })
	})

	rns.Log("Resource example "+rns.PrettyHexRep(serverDest.Hash)+" running, waiting for a connection.", rns.LogInfo)
	rns.Log("Hit enter to manually send an announce (Ctrl-C to quit)", rns.LogInfo)

	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		serverDest.Announce(nil, false, nil, nil, true)
		rns.Log("Sent announce from "+rns.PrettyHexRep(serverDest.Hash), rns.LogInfo)
	}
	return in.Err()
}

func readFirstN(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, n)
	r, err := io.ReadFull(f, buf)
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return buf[:r], nil
	}
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// ---------------- client ----------------

func runClient(configDir *string, destinationHexHash string, sizeMB int) error {
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
	serverDest, err := rns.NewDestination(serverID, rns.DestinationOUT, rns.DestinationSINGLE, appName, "resourceexample")
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
		rns.Log("Link established with server, hit enter to send a resource, or type in \"quit\" to quit", rns.LogInfo)
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

	in := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !in.Scan() {
			if l := getLink(&serverLinkMu, &serverLink); l != nil {
				l.Teardown()
			}
			return in.Err()
		}
		text := strings.TrimSpace(in.Text())
		if text == "quit" || text == "q" || text == "exit" {
			if l := getLink(&serverLinkMu, &serverLink); l != nil {
				l.Teardown()
			}
			return nil
		}

		l := getLink(&serverLinkMu, &serverLink)
		if l == nil {
			return errors.New("link not active")
		}

		data := make([]byte, sizeMB*1024*1024)
		_, _ = rand.Read(data)
		rns.Log(fmt.Sprintf("Data length: %d", len(data)), rns.LogInfo)
		rns.Log("First 32 bytes of data: "+rns.HexRep(data[:min(32, len(data))]), rns.LogInfo)

		metadata := map[string]any{
			"text":    randomTexts[int(data[0])%len(randomTexts)],
			"numbers": []any{1, 2, 3, 4},
			"blob":    data[1:17],
		}

		timeoutSeconds := 300.0
		res, err := rns.NewResource(
			data,
			nil,
			l,
			metadata,
			true,
			false, // Python: auto_compress=False
			func(res *rns.Resource) {
				if res == nil {
					return
				}
				if res.Status() == rns.ResourceComplete {
					rns.Log(fmt.Sprintf("The resource %s was sent successfully", res), rns.LogInfo)
				} else {
					rns.Log(fmt.Sprintf("Sending the resource %s failed", res), rns.LogError)
				}
			},
			nil,
			&timeoutSeconds,
			0,
			nil,
			nil,
			false,
			0,
		)
		if err != nil || res == nil {
			return fmt.Errorf("NewResource: %v", err)
		}
	}
}

func getLink(mu *sync.Mutex, l **rns.Link) *rns.Link {
	mu.Lock()
	defer mu.Unlock()
	return *l
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseTruncatedHashHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	destLen := (rns.TRUNCATED_HASHLENGTH / 8) * 2
	if len(s) != destLen {
		return nil, fmt.Errorf("invalid destination length: got %d want %d", len(s), destLen)
	}
	return hex.DecodeString(s)
}
