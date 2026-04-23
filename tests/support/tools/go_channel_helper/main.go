package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

const (
	channelApp    = "paritychannel"
	channelAspect = "messages"
)

type channelMessage struct {
	ID   string
	Data string
}

func (m *channelMessage) MsgType() uint16 { return 0xABCD }
func (m *channelMessage) Pack() ([]byte, error) {
	return umsgpack.Packb([]any{m.ID, m.Data})
}
func (m *channelMessage) Unpack(raw []byte) error {
	var v []any
	if err := umsgpack.Unpackb(raw, &v); err != nil {
		return err
	}
	if len(v) >= 2 {
		if s, ok := v[0].(string); ok {
			m.ID = s
		}
		if s, ok := v[1].(string); ok {
			m.Data = s
		}
	}
	return nil
}

func main() {
	var (
		configDir      string
		identityPath   string
		destinationHex string
		listenMode     bool
		waitSeconds    float64
	)

	flag.StringVar(&configDir, "config", "", "reticulum config dir")
	flag.StringVar(&identityPath, "identity", "", "identity path")
	flag.StringVar(&destinationHex, "destination", "", "listener destination hash")
	flag.BoolVar(&listenMode, "listen", false, "listen mode")
	flag.Float64Var(&waitSeconds, "wait-seconds", 45, "wait seconds")
	flag.Parse()

	logLevel := 2
	var configPtr *string
	if configDir != "" {
		configPtr = &configDir
	}
	if _, err := rns.NewReticulum(configPtr, &logLevel, nil, nil, false, nil); err != nil {
		fatalf("reticulum init failed: %v", err)
	}
	rns.SetCompactLogFormat(true)
	rns.SetLogLevel(-1)

	id, err := loadOrCreateChannelIdentity(identityPath)
	if err != nil {
		fatalf("identity failed: %v", err)
	}

	if listenMode {
		if err := runChannelListener(id, waitSeconds); err != nil {
			fatalf("listener failed: %v", err)
		}
		return
	}

	if destinationHex == "" {
		fatalf("destination is required")
	}
	if err := runChannelClient(id, destinationHex, waitSeconds); err != nil {
		fatalf("client failed: %v", err)
	}
}

func runChannelListener(id *rns.Identity, waitSeconds float64) error {
	dest, err := rns.NewDestination(id, rns.DestinationIN, rns.DestinationSINGLE, channelApp, channelAspect)
	if err != nil {
		return err
	}

	got := make(chan string, 8)
	dest.SetLinkEstablishedCallback(func(l *rns.Link) {
		ch := l.Channel()
		if err := ch.RegisterMessageType(&channelMessage{}); err != nil {
			fmt.Printf("EVENT register_failed %v\n", err)
			return
		}
		ch.AddMessageHandler(func(m rns.MessageBase) bool {
			msg, ok := m.(*channelMessage)
			if !ok {
				return false
			}
			fmt.Printf("EVENT received %s %s\n", msg.ID, msg.Data)
			got <- msg.ID
			return true
		})
	})

	fmt.Printf("LISTEN_HASH %s\n", hex.EncodeToString(dest.Hash))
	time.Sleep(time.Second)
	deadline := time.Now().Add(durationChannel(waitSeconds))
	ids := ""
	for time.Now().Before(deadline) {
		if ids == "123" {
			return nil
		}
		dest.Announce(nil, false, nil, nil, true)
		select {
		case id := <-got:
			ids += id
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("timeout ids=%q", ids)
}

func runChannelClient(id *rns.Identity, destinationHex string, waitSeconds float64) error {
	destHash, err := hex.DecodeString(destinationHex)
	if err != nil {
		return err
	}
	if !awaitChannelPath(destHash, durationChannel(waitSeconds)) {
		return fmt.Errorf("path not found")
	}

	remoteID := rns.IdentityRecall(destHash)
	if remoteID == nil {
		return fmt.Errorf("could not recall remote identity")
	}
	remoteDest, err := rns.NewDestination(remoteID, rns.DestinationOUT, rns.DestinationSINGLE, channelApp, channelAspect)
	if err != nil {
		return err
	}
	link, err := rns.NewLink(remoteDest, nil, rns.LinkModeDefault, nil, nil)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(durationChannel(waitSeconds))
	for time.Now().Before(deadline) {
		if link.Status == rns.LinkActive {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if link.Status != rns.LinkActive {
		return fmt.Errorf("link not active")
	}

	link.Identify(id)
	time.Sleep(1 * time.Second)

	ch := link.Channel()
	if err := ch.RegisterMessageType(&channelMessage{}); err != nil {
		return err
	}
	messages := []channelMessage{
		{ID: "1", Data: "alpha"},
		{ID: "2", Data: "beta"},
		{ID: "3", Data: "gamma"},
	}
	for _, msg := range messages {
		sendDeadline := time.Now().Add(durationChannel(waitSeconds))
		for time.Now().Before(sendDeadline) {
			if ch.IsReadyToSend() {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if !ch.IsReadyToSend() {
			return fmt.Errorf("channel not ready for message %s", msg.ID)
		}
		if _, err := ch.Send(&channelMessage{ID: msg.ID, Data: msg.Data}); err != nil {
			return err
		}
		fmt.Printf("EVENT sent %s %s\n", msg.ID, msg.Data)
	}

	time.Sleep(2 * time.Second)
	link.Teardown()
	return nil
}

func awaitChannelPath(destHash []byte, timeout time.Duration) bool {
	if !rns.HasPath(destHash) {
		rns.RequestPath(destHash, nil, nil, false)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if rns.HasPath(destHash) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return rns.HasPath(destHash)
}

func loadOrCreateChannelIdentity(path string) (*rns.Identity, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".reticulum", "storage", "identities", channelApp)
	}
	if _, err := os.Stat(path); err == nil {
		return rns.IdentityFromFile(path)
	}
	id, err := rns.NewIdentity()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := id.Save(path); err != nil {
		return nil, err
	}
	return id, nil
}

func durationChannel(seconds float64) time.Duration {
	if seconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(seconds * float64(time.Second))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
