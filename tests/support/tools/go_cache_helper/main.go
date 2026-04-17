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

const (
	cacheApp    = "paritycache"
	cacheAspect = "replay"
)

func main() {
	var (
		configDir      string
		identityPath   string
		destinationHex string
		hashLogPath    string
		listenMode     bool
		payload        string
		waitSeconds    float64
	)

	flag.StringVar(&configDir, "config", "", "reticulum config dir")
	flag.StringVar(&identityPath, "identity", "", "identity path")
	flag.StringVar(&destinationHex, "destination", "", "listener destination hash")
	flag.StringVar(&hashLogPath, "hash-log", "", "listener log path to extract cache hash from")
	flag.BoolVar(&listenMode, "listen", false, "listen mode")
	flag.StringVar(&payload, "payload", "cache-payload", "packet payload")
	flag.Float64Var(&waitSeconds, "wait-seconds", 30, "wait timeout")
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
	rns.SetLogLevel(rns.LOG_DEBUG)

	id, err := loadOrCreateIdentity(identityPath)
	if err != nil {
		fatalf("identity failed: %v", err)
	}

	if listenMode {
		if err := runListener(id, payload, waitSeconds); err != nil {
			fatalf("listener failed: %v", err)
		}
		return
	}

	if destinationHex == "" {
		fatalf("destination is required")
	}
	if err := runClient(id, destinationHex, hashLogPath, payload, waitSeconds); err != nil {
		fatalf("client failed: %v", err)
	}
}

func runListener(id *rns.Identity, payload string, waitSeconds float64) error {
	dest, err := rns.NewDestination(id, rns.DestinationIN, rns.DestinationSINGLE, cacheApp, cacheAspect)
	if err != nil {
		return err
	}

	doneCh := make(chan struct{}, 1)
	dest.SetLinkEstablishedCallback(func(l *rns.Link) {
		p := rns.NewPacket(l, []byte(payload), rns.WithoutReceipt())
		if p == nil {
			return
		}
		if err := p.Pack(); err != nil {
			return
		}
		rns.Cache(p, true)
		fmt.Printf("EVENT cached_ready hash=%s payload=%s\n", hex.EncodeToString(p.PacketHash), payload)
		l.SetLinkClosedCallback(func(_ *rns.Link) {
			select {
			case doneCh <- struct{}{}:
			default:
			}
		})
	})

	fmt.Printf("LISTEN_HASH %s\n", hex.EncodeToString(dest.Hash()))
	time.Sleep(time.Second)

	deadline := time.Now().Add(duration(waitSeconds))
	for time.Now().Before(deadline) {
		dest.Announce(nil, false, nil, nil, true)
		select {
		case <-doneCh:
			return nil
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("timeout waiting for cache flow")
}

func runClient(id *rns.Identity, destinationHex, hashLogPath, payload string, waitSeconds float64) error {
	destHash, err := hex.DecodeString(destinationHex)
	if err != nil {
		return err
	}
	if !awaitPath(destHash, duration(waitSeconds)) {
		return fmt.Errorf("path not found")
	}
	remoteID := rns.IdentityRecall(destHash)
	if remoteID == nil {
		return fmt.Errorf("could not recall remote identity")
	}
	remoteDest, err := rns.NewDestination(remoteID, rns.DestinationOUT, rns.DestinationSINGLE, cacheApp, cacheAspect)
	if err != nil {
		return err
	}

	link, err := rns.NewOutgoingLink(remoteDest, rns.LinkModeDefault, nil, nil)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(duration(waitSeconds))
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

	firstHash, err := awaitHashFromLog(hashLogPath, duration(waitSeconds))
	if err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	rns.CacheRequest(firstHash, link)
	fmt.Printf("EVENT cache_request hash=%s\n", hex.EncodeToString(firstHash))
	time.Sleep(1500 * time.Millisecond)
	fmt.Printf("EVENT cache_request_sent hash=%s\n", hex.EncodeToString(firstHash))
	link.Teardown()
	return nil
}

func awaitHashFromLog(path string, timeout time.Duration) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("hash log path is required")
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			for _, line := range splitLines(string(data)) {
				var hexHash string
				if _, err := fmt.Sscanf(line, "EVENT cached_ready hash=%s", &hexHash); err == nil {
					return hex.DecodeString(hexHash)
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("cached hash not found in log")
}

func splitLines(s string) []string {
	out := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start <= len(s) {
		out = append(out, s[start:])
	}
	return out
}

func awaitPath(destHash []byte, timeout time.Duration) bool {
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

func loadOrCreateIdentity(path string) (*rns.Identity, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".reticulum", "storage", "identities", cacheApp)
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

func duration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
