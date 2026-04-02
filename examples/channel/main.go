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
	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

const appName = "example_utilities"

type StringMessage struct {
	Data      string
	Timestamp time.Time
}

func (m *StringMessage) MsgType() uint16 { return 0x0101 }

func (m *StringMessage) Pack() ([]byte, error) {
	ts := m.Timestamp
	if ts.IsZero() {
		ts = time.Now()
		m.Timestamp = ts
	}
	// Python example packs (data, datetime). For Go portability, encode timestamp as RFC3339Nano.
	return umsgpack.Packb([]any{m.Data, ts.Format(time.RFC3339Nano)})
}

func (m *StringMessage) Unpack(raw []byte) error {
	var decoded []any
	if err := umsgpack.Unpackb(raw, &decoded); err != nil {
		return err
	}
	if len(decoded) >= 1 {
		if s, ok := decoded[0].(string); ok {
			m.Data = s
		}
	}
	if len(decoded) >= 2 {
		if s, ok := decoded[1].(string); ok {
			if ts, err := time.Parse(time.RFC3339Nano, s); err == nil {
				m.Timestamp = ts
			}
		}
	}
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	return nil
}

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
		fmt.Println("  channel --server [--config DIR]")
		fmt.Println("  channel [--config DIR] <destination_hash_hex>")
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

	dest, err := rns.NewDestination(serverIdentity, rns.DestinationIN, rns.DestinationSINGLE, appName, "channelexample")
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewDestination:", err)
		os.Exit(1)
	}

	dest.SetLinkEstablishedCallback(func(link *rns.Link) {
		rns.Log("Client connected", rns.LogInfo)
		link.SetLinkClosedCallback(func(*rns.Link) {
			rns.Log("Client disconnected", rns.LogInfo)
		})

		ch := link.Channel()
		ch.RegisterMessageType(&StringMessage{})
		ch.AddMessageHandler(func(m rns.MessageBase) bool {
			sm, ok := m.(*StringMessage)
			if !ok {
				return false
			}
			rns.Log("Received data on the link: "+sm.Data+" (message created at "+sm.Timestamp.String()+")", rns.LogInfo)
			reply := &StringMessage{Data: "I received \"" + sm.Data + "\" over the link"}
			_ = link.Channel().Send(reply)
			return true
		})
	})

	rns.Log("Channel example "+rns.PrettyHexRep(dest.Hash())+" running, waiting for a connection.", rns.LogInfo)
	rns.Log("Hit enter to manually send an announce (Ctrl-C to quit)", rns.LogInfo)

	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		dest.Announce(nil, false, nil, nil, true)
		rns.Log("Sent announce from "+rns.PrettyHexRep(dest.Hash()), rns.LogInfo)
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

	if !rns.TransportHasPath(destinationHash) {
		rns.Log("Destination is not yet known. Requesting path and waiting for announce to arrive...", rns.LogInfo)
		rns.TransportRequestPath(destinationHash)
		for !rns.TransportHasPath(destinationHash) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	serverIdentity := rns.IdentityRecall(destinationHash)
	if serverIdentity == nil {
		rns.Log("Could not recall Identity for destination", rns.LogError)
		os.Exit(1)
	}

	rns.Log("Establishing link with server...", rns.LogInfo)
	serverDest, err := rns.NewDestination(serverIdentity, rns.DestinationOUT, rns.DestinationSINGLE, appName, "channelexample")
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewDestination(out):", err)
		os.Exit(1)
	}

	link, err := rns.NewOutgoingLink(serverDest, rns.LinkModeDefault, nil, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewOutgoingLink:", err)
		os.Exit(1)
	}

	var channel *rns.Channel
	link.SetLinkEstablishedCallback(func(l *rns.Link) {
		channel = l.Channel()
		channel.RegisterMessageType(&StringMessage{})
		channel.AddMessageHandler(func(m rns.MessageBase) bool {
			sm, ok := m.(*StringMessage)
			if !ok {
				return false
			}
			rns.Log("Received data on the link: "+sm.Data+" (message created at "+sm.Timestamp.String()+")", rns.LogInfo)
			fmt.Print("> ")
			return true
		})
		rns.Log("Link established with server, enter some text to send, or \"quit\" to quit", rns.LogInfo)
	})

	link.SetLinkClosedCallback(func(_ *rns.Link) {
		rns.Log("Link closed, exiting now", rns.LogInfo)
		os.Exit(0)
	})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if channel != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if channel == nil {
		fmt.Fprintln(os.Stderr, "timeout waiting for link establishment")
		os.Exit(1)
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

		msg := &StringMessage{Data: text, Timestamp: time.Now()}
		packed, _ := msg.Pack()
		if !channel.IsReadyToSend() {
			rns.Log("Channel is not ready to send, please wait for pending messages to complete.", rns.LogError)
			continue
		}
		if len(packed) > channel.Mdu() {
			rns.Log(fmt.Sprintf("Cannot send this packet, the data size of %d bytes exceeds the link packet MDU of %d bytes", len(packed), channel.Mdu()), rns.LogError)
			continue
		}
		_ = channel.Send(msg)
	}
}
