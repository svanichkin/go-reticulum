package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const appName = "example_utilities"

func main() {
	var (
		configDir string
		server    bool
	)
	flag.BoolVar(&server, "server", false, "wait for incoming link requests from clients")
	flag.BoolVar(&server, "s", false, "wait for incoming link requests from clients")
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
		fmt.Println("  buffer --server [--config DIR]")
		fmt.Println("  buffer [--config DIR] <destination_hash_hex>")
		os.Exit(2)
	}
	runClient(flag.Arg(0), configArg)
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

	dest, err := rns.NewDestination(serverIdentity, rns.DestinationIN, rns.DestinationSINGLE, appName, "bufferexample")
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewDestination:", err)
		os.Exit(1)
	}

	var latestBuf *rns.BufferedRWPair
	dest.SetLinkEstablishedCallback(func(link *rns.Link) {
		rns.Log("Client connected", rns.LogInfo)
		link.SetLinkClosedCallback(func(*rns.Link) {
			rns.Log("Client disconnected", rns.LogInfo)
		})
		if latestBuf != nil {
			_ = latestBuf.Close()
			latestBuf = nil
		}

		ch := link.Channel()
		latestBuf = rns.CreateBidirectionalBuffer(0, 0, ch, func(readyBytes int) {
			if latestBuf == nil || readyBytes <= 0 {
				return
			}
			data := make([]byte, readyBytes)
			n, rerr := latestBuf.Reader.ReadInto(data)
			if rerr != nil || n == 0 {
				return
			}
			msg := string(data[:n])
			rns.Log("Received data over the buffer: "+msg, rns.LogInfo)
			reply := []byte("I received \"" + msg + "\" over the buffer")
			_, _ = latestBuf.Writer.Write(reply)
			_ = latestBuf.Writer.Flush()
		})
	})

	rns.Log("Link buffer example "+rns.PrettyHexRep(dest.Hash)+" running, waiting for a connection.", rns.LogInfo)
	rns.Log("Hit enter to manually send an announce (Ctrl-C to quit)", rns.LogInfo)

	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		dest.Announce(nil, false, nil, nil, true)
		rns.Log("Sent announce from "+rns.PrettyHexRep(dest.Hash), rns.LogInfo)
	}
}

func runClient(destinationHex string, configDir *string) {
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

	if !rns.HasPath(destinationHash) {
		rns.Log("Destination is not yet known. Requesting path and waiting for announce to arrive...", rns.LogInfo)
		rns.RequestPath(destinationHash, nil, nil, false)
		for !rns.HasPath(destinationHash) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	serverIdentity := rns.IdentityRecall(destinationHash)
	if serverIdentity == nil {
		rns.Log("Could not recall Identity for destination", rns.LogError)
		os.Exit(1)
	}

	rns.Log("Establishing link with server...", rns.LogInfo)
	serverDest, err := rns.NewDestination(serverIdentity, rns.DestinationOUT, rns.DestinationSINGLE, appName, "bufferexample")
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewDestination(out):", err)
		os.Exit(1)
	}

	link, err := rns.NewLink(serverDest, nil, rns.LinkModeDefault, nil, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewOutgoingLink:", err)
		os.Exit(1)
	}

	var buf *rns.BufferedRWPair
	link.SetLinkEstablishedCallback(func(l *rns.Link) {
		ch := l.Channel()
		buf = rns.CreateBidirectionalBuffer(0, 0, ch, func(readyBytes int) {
			if buf == nil || readyBytes <= 0 {
				return
			}
			data := make([]byte, readyBytes)
			n, rerr := buf.Reader.ReadInto(data)
			if rerr != nil || n == 0 {
				return
			}
			rns.Log("Received data over the link buffer: "+string(data[:n]), rns.LogInfo)
			fmt.Print("> ")
		})
		rns.Log("Link established with server, enter some text to send, or \"quit\" to quit", rns.LogInfo)
	})
	link.SetLinkClosedCallback(func(l *rns.Link) {
		rns.Log("Link closed, exiting now", rns.LogInfo)
		os.Exit(0)
	})

	// Wait until link callback has created the buffer.
	for buf == nil {
		time.Sleep(10 * time.Millisecond)
	}

	in := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !in.Scan() {
			return
		}
		text := strings.TrimRight(in.Text(), "\r\n")
		if text == "" {
			continue
		}
		if text == "quit" || text == "q" || text == "exit" {
			link.Teardown()
			return
		}
		_, _ = buf.Writer.Write([]byte(text))
		_ = buf.Writer.Flush()
	}
}
