package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const packetApp = "paritypacket"

func main() {
	var (
		configDir      string
		identityPath   string
		remoteIdentity string
		mode           string
		listenMode     bool
		destinationHex string
		groupKeyHex    string
		payload        string
		waitSeconds    float64
		announce       bool
	)

	flag.StringVar(&configDir, "config", "", "reticulum config dir")
	flag.StringVar(&identityPath, "identity", "", "identity path")
	flag.StringVar(&remoteIdentity, "remote-identity", "", "remote identity path for outbound group mode")
	flag.StringVar(&mode, "mode", "", "plain or group")
	flag.BoolVar(&listenMode, "listen", false, "listen for packet")
	flag.StringVar(&destinationHex, "destination", "", "listener destination hash")
	flag.StringVar(&groupKeyHex, "group-key", "", "hex-encoded group key")
	flag.StringVar(&payload, "payload", "hello", "packet payload")
	flag.Float64Var(&waitSeconds, "wait-seconds", 30, "wait timeout")
	flag.BoolVar(&announce, "announce", false, "announce listener destination periodically")
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

	if mode != "plain" && mode != "group" {
		fatalf("mode must be plain or group")
	}

	id, err := loadOrCreateIdentity(identityPath)
	if err != nil {
		fatalf("identity failed: %v", err)
	}

	if listenMode {
		if err := runListener(id, mode, groupKeyHex, payload, waitSeconds, announce); err != nil {
			fatalf("listener failed: %v", err)
		}
		return
	}

	if destinationHex == "" {
		fatalf("destination is required in sender mode")
	}
	if err := runSender(mode, destinationHex, groupKeyHex, payload, waitSeconds, remoteIdentity); err != nil {
		fatalf("sender failed: %v", err)
	}
}

func runListener(id *rns.Identity, mode, groupKeyHex, expectPayload string, waitSeconds float64, announce bool) error {
	var (
		dest *rns.Destination
		err  error
	)

	switch mode {
	case "plain":
		dest, err = rns.NewDestination(nil, rns.DestinationIN, rns.DestinationPLAIN, packetApp, mode)
	case "group":
		dest, err = rns.NewDestination(id, rns.DestinationIN, rns.DestinationGROUP, packetApp, mode)
	default:
		return fmt.Errorf("unsupported mode %q", mode)
	}
	if err != nil {
		return err
	}

	if mode == "group" {
		key, err := decodeHex(groupKeyHex)
		if err != nil {
			return err
		}
		if err := dest.LoadPrivateKey(key); err != nil {
			return err
		}
	}

	done := make(chan error, 1)
	dest.SetPacketCallback(func(data []byte, _ *rns.Packet) {
		fmt.Printf("EVENT received %s\n", string(data))
		if string(data) != expectPayload {
			done <- fmt.Errorf("payload mismatch: got %q want %q", string(data), expectPayload)
			return
		}
		done <- nil
	})

	fmt.Printf("LISTEN_HASH %s\n", hex.EncodeToString(dest.Hash()))

	deadline := time.Now().Add(durationSeconds(waitSeconds))
	for {
		if announce && mode == "group" {
			dest.Announce(nil, false, nil, nil, true)
		}
		select {
		case err := <-done:
			return err
		case <-time.After(1 * time.Second):
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout")
		}
	}
}

func runSender(mode, destinationHex, groupKeyHex, payload string, waitSeconds float64, remoteIdentityPath string) error {
	destHash, err := hex.DecodeString(destinationHex)
	if err != nil {
		return err
	}

	var dest *rns.Destination
	switch mode {
	case "plain":
		dest, err = rns.NewDestination(nil, rns.DestinationOUT, rns.DestinationPLAIN, packetApp, mode)
		if err != nil {
			return err
		}
	case "group":
		var remoteID *rns.Identity
		if remoteIdentityPath != "" {
			loaded, err := rns.IdentityFromFile(remoteIdentityPath)
			if err != nil {
				return err
			}
			remoteID = loaded
		}
		if remoteID == nil {
			if !awaitPath(destHash, durationSeconds(waitSeconds)) {
				return fmt.Errorf("path not found")
			}
			remoteID = rns.IdentityRecall(destHash)
		}
		if remoteID == nil {
			return fmt.Errorf("could not recall remote identity")
		}
		dest, err = rns.NewDestination(remoteID, rns.DestinationOUT, rns.DestinationGROUP, packetApp, mode)
		if err != nil {
			return err
		}
		key, err := decodeHex(groupKeyHex)
		if err != nil {
			return err
		}
		if err := dest.LoadPrivateKey(key); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported mode %q", mode)
	}

	pkt := rns.NewPacket(dest, []byte(payload))
	if pkt == nil {
		return fmt.Errorf("failed to build packet")
	}
	fmt.Printf("EVENT destination %s\n", hex.EncodeToString(dest.Hash()))
	if pkt.Send() == nil && !pkt.Sent {
		return fmt.Errorf("send failed")
	}
	fmt.Println("EVENT sent")
	return nil
}

func awaitPath(destHash []byte, timeout time.Duration) bool {
	if !rns.TransportHasPath(destHash) {
		rns.TransportRequestPath(destHash)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if rns.TransportHasPath(destHash) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return rns.TransportHasPath(destHash)
}

func loadOrCreateIdentity(path string) (*rns.Identity, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".reticulum", "storage", "identities", packetApp)
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

func decodeHex(s string) ([]byte, error) {
	if s == "" {
		return nil, fmt.Errorf("missing hex input")
	}
	return hex.DecodeString(s)
}

func durationSeconds(sec float64) time.Duration {
	return time.Duration(sec * float64(time.Second))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
