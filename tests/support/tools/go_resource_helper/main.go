package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const (
	resAppName = "parityresource"
	resAspect  = "blob"
)

func main() {
	var (
		configDir      string
		identityPath   string
		destinationHex string
		listenMode     bool
		mode           string
		traceMode      bool
		incompressible bool
		waitSeconds    float64
		smallPayload   string
		largeSize      int
	)

	flag.StringVar(&configDir, "config", "", "reticulum config dir")
	flag.StringVar(&identityPath, "identity", "", "identity path")
	flag.StringVar(&destinationHex, "destination", "", "listener destination hash")
	flag.BoolVar(&listenMode, "listen", false, "listen mode")
	flag.StringVar(&mode, "mode", "normal", "normal, reject, cancel")
	flag.BoolVar(&traceMode, "trace", false, "enable resource trace logs")
	flag.BoolVar(&incompressible, "incompressible-large", false, "use incompressible large payload")
	flag.Float64Var(&waitSeconds, "wait-seconds", 45, "wait seconds")
	flag.StringVar(&smallPayload, "small-payload", "parity-small", "small payload")
	flag.IntVar(&largeSize, "large-size", 24576, "large payload size")
	flag.Parse()

	logLevel := 2
	var configPtr *string
	if configDir != "" {
		configPtr = &configDir
	}
	if _, err := rns.NewReticulum(configPtr, &logLevel, nil, nil, false, nil); err != nil {
		fatalf("reticulum init failed: %v", err)
	}
	if traceMode {
		rns.SetCompactLogFormat(true)
		rns.SetLogLevel(rns.LOG_EXTREME)
	} else {
		rns.SetCompactLogFormat(true)
		rns.SetLogLevel(-1)
	}

	id, err := loadOrCreateResourceIdentity(identityPath)
	if err != nil {
		fatalf("identity failed: %v", err)
	}

	if listenMode {
		if err := runResourceListener(id, mode, waitSeconds, []byte(smallPayload), largeSize, incompressible); err != nil {
			fatalf("listener failed: %v", err)
		}
		return
	}

	if destinationHex == "" {
		fatalf("destination is required")
	}
	if err := runResourceClient(id, mode, destinationHex, waitSeconds, []byte(smallPayload), largeSize, incompressible); err != nil {
		fatalf("client failed: %v", err)
	}
}

func runResourceListener(id *rns.Identity, mode string, waitSeconds float64, smallPayload []byte, largeSize int, incompressible bool) error {
	dest, err := rns.NewDestination(id, rns.DestinationIN, rns.DestinationSINGLE, resAppName, resAspect)
	if err != nil {
		return err
	}

	expectedLarge := makeLargePayload(largeSize)
	if incompressible {
		expectedLarge = makeCancelPayload(largeSize)
	}
	doneCh := make(chan string, 4)
	seen := map[string]bool{}

	dest.SetLinkEstablishedCallback(func(l *rns.Link) {
		l.SetResourceStrategy(rns.LinkAcceptApp)
		l.SetResourceCallback(func(adv *rns.ResourceAdvertisement) bool {
			if adv == nil {
				return false
			}
			fmt.Printf("EVENT adv transfer=%d data=%d parts=%d segments=%d compressed=%v\n", adv.T, adv.D, adv.N, adv.L, adv.C)
			if mode == "reject" {
				fmt.Println("EVENT rejected kind=small")
				return false
			}
			return true
		})
		l.SetResourceStartedCallback(func(res *rns.Resource) {
			if res == nil {
				return
			}
			fmt.Printf("EVENT started transfer=%d data=%d parts=%d segments=%d compressed=%v\n", res.GetTransferSize(), res.GetDataSize(), res.GetParts(), res.GetSegments(), res.IsCompressed())
		})
		l.SetResourceConcludedCallback(func(res *rns.Resource) {
			if res == nil {
				return
			}
			if mode == "cancel" && res.Status() == rns.ResourceFailed {
				fmt.Println("EVENT canceled_by_initiator")
				doneCh <- "canceled"
				return
			}
			if res.Status() != rns.ResourceComplete {
				fmt.Printf("EVENT resource_failed status=%d\n", res.Status())
				return
			}
			data, err := os.ReadFile(res.DataFile())
			if err != nil {
				fmt.Printf("EVENT resource_read_failed %v\n", err)
				return
			}
			metaKind := metadataKind(res.Metadata())
			sum := sha256.Sum256(data)
			fmt.Printf("EVENT concluded kind=%s bytes=%d sha256=%s\n", metaKind, len(data), hex.EncodeToString(sum[:]))
			switch metaKind {
			case "small":
				if bytes.Equal(data, smallPayload) {
					doneCh <- "small"
				} else {
					fmt.Println("EVENT mismatch kind=small")
				}
			case "large":
				if bytes.Equal(data, expectedLarge) {
					doneCh <- "large"
				} else {
					fmt.Println("EVENT mismatch kind=large")
				}
			}
		})
	})

	fmt.Printf("LISTEN_HASH %s\n", hex.EncodeToString(dest.Hash))
	time.Sleep(time.Second)

	deadline := time.Now().Add(durationResource(waitSeconds))
	for time.Now().Before(deadline) {
		if (mode == "cancel" && seen["canceled"]) || (mode != "cancel" && seen["small"] && seen["large"]) {
			return nil
		}
		dest.Announce(nil, false, nil, nil, true)
		select {
		case kind := <-doneCh:
			seen[kind] = true
		case <-time.After(2 * time.Second):
		}
	}

	if mode == "cancel" {
		if !seen["canceled"] {
			return fmt.Errorf("cancel not observed")
		}
		return nil
	}
	if !seen["small"] || !seen["large"] {
		return fmt.Errorf("resources not fully received: small=%v large=%v", seen["small"], seen["large"])
	}
	return nil
}

func runResourceClient(id *rns.Identity, mode, destinationHex string, waitSeconds float64, smallPayload []byte, largeSize int, incompressible bool) error {
	destHash, err := hex.DecodeString(destinationHex)
	if err != nil {
		return err
	}
	if !awaitResourcePath(destHash, durationResource(waitSeconds)) {
		return fmt.Errorf("path not found")
	}

	remoteID := rns.IdentityRecall(destHash)
	if remoteID == nil {
		return fmt.Errorf("could not recall remote identity")
	}
	remoteDest, err := rns.NewDestination(remoteID, rns.DestinationOUT, rns.DestinationSINGLE, resAppName, resAspect)
	if err != nil {
		return err
	}
	link, err := rns.NewLink(remoteDest, nil, rns.LinkModeDefault, nil, nil)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(durationResource(waitSeconds))
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

	if mode == "reject" {
		status, err := sendResourceExpectFailure(link, "small", smallPayload, durationResource(waitSeconds))
		if err != nil {
			return err
		}
		fmt.Printf("EVENT %s\n", status)
		link.Teardown()
		return nil
	}
	if mode == "cancel" {
		cancelSize := largeSize
		if cancelSize < 1<<20 {
			cancelSize = 1 << 20
		}
		status, err := sendResourceAndCancel(link, "large", makeCancelPayload(cancelSize), durationResource(waitSeconds))
		if err != nil {
			return err
		}
		fmt.Printf("EVENT %s\n", status)
		link.Teardown()
		return nil
	}

	if err := sendResource(link, "small", smallPayload, durationResource(waitSeconds)); err != nil {
		return err
	}
	largePayload := makeLargePayload(largeSize)
	if incompressible {
		largePayload = makeCancelPayload(largeSize)
	}
	if err := sendResource(link, "large", largePayload, durationResource(waitSeconds)); err != nil {
		return err
	}

	link.Teardown()
	return nil
}

func sendResourceAndCancel(link *rns.Link, kind string, payload []byte, timeout time.Duration) (string, error) {
	doneCh := make(chan *rns.Resource, 1)
	timeoutSeconds := timeout.Seconds()
	res, err := rns.NewResource(
		payload,
		nil,
		link,
		map[string]any{"kind": kind},
		true,
		false,
		func(res *rns.Resource) {
			doneCh <- res
		},
		nil,
		&timeoutSeconds,
		0,
		nil,
		nil,
		false,
		0,
	)
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", fmt.Errorf("resource %s not created", kind)
	}
	cancelDeadline := time.Now().Add(minDuration(timeout, 5*time.Second))
	for time.Now().Before(cancelDeadline) {
		if res.Status() >= rns.ResourceTransferring {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	res.Cancel()
	select {
	case concluded := <-doneCh:
		if concluded == nil {
			return "", fmt.Errorf("%s resource concluded nil", kind)
		}
		settleDeadline := time.Now().Add(minDuration(timeout, 2*time.Second))
		for concluded.Status() == rns.ResourceAdvertised || concluded.Status() == rns.ResourceTransferring {
			if time.Now().After(settleDeadline) {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if concluded.Status() == rns.ResourceFailed {
			return "canceled", nil
		}
		return "", fmt.Errorf("%s unexpected status=%d", kind, concluded.Status())
	case <-time.After(timeout):
		return "", fmt.Errorf("%s resource timeout", kind)
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func sendResourceExpectFailure(link *rns.Link, kind string, payload []byte, timeout time.Duration) (string, error) {
	doneCh := make(chan *rns.Resource, 1)
	timeoutSeconds := timeout.Seconds()
	res, err := rns.NewResource(
		payload,
		nil,
		link,
		map[string]any{"kind": kind},
		true,
		true,
		func(res *rns.Resource) {
			doneCh <- res
		},
		nil,
		&timeoutSeconds,
		0,
		nil,
		nil,
		false,
		0,
	)
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", fmt.Errorf("resource %s not created", kind)
	}
	select {
	case concluded := <-doneCh:
		if concluded == nil {
			return "", fmt.Errorf("%s resource concluded nil", kind)
		}
		switch concluded.Status() {
		case rns.ResourceRejected:
			return "rejected", nil
		case rns.ResourceFailed:
			return "canceled", nil
		default:
			return "", fmt.Errorf("%s unexpected status=%d", kind, concluded.Status())
		}
	case <-time.After(timeout):
		return "", fmt.Errorf("%s resource timeout", kind)
	}
}

func sendResource(link *rns.Link, kind string, payload []byte, timeout time.Duration) error {
	doneCh := make(chan *rns.Resource, 1)
	timeoutSeconds := timeout.Seconds()
	res, err := rns.NewResource(
		payload,
		nil,
		link,
		map[string]any{"kind": kind},
		true,
		true,
		func(res *rns.Resource) {
			doneCh <- res
		},
		nil,
		&timeoutSeconds,
		0,
		nil,
		nil,
		false,
		0,
	)
	if err != nil {
		return err
	}
	if res == nil {
		return fmt.Errorf("resource %s not created", kind)
	}

	select {
	case concluded := <-doneCh:
		if concluded == nil {
			return fmt.Errorf("%s resource concluded nil", kind)
		}
		if concluded.Status() != rns.ResourceComplete {
			return fmt.Errorf("%s resource status=%d", kind, concluded.Status())
		}
		sum := sha256.Sum256(payload)
		fmt.Printf("EVENT sent kind=%s bytes=%d sha256=%s segments=%d parts=%d compressed=%v\n", kind, len(payload), hex.EncodeToString(sum[:]), concluded.GetSegments(), concluded.GetParts(), concluded.IsCompressed())
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("%s resource timeout", kind)
	}
}

func makeLargePayload(size int) []byte {
	if size < 1 {
		size = 1
	}
	out := make([]byte, size)
	pattern := []byte("GO-PY-RESOURCE-PARITY-")
	for i := range out {
		out[i] = pattern[i%len(pattern)]
	}
	return out
}

func makeCancelPayload(size int) []byte {
	if size < 1 {
		size = 1
	}
	out := make([]byte, size)
	var x byte = 0x5d
	for i := range out {
		x = byte((uint16(x)*73 + 41) & 0xff)
		out[i] = x ^ byte(i&0xff)
	}
	return out
}

func metadataKind(meta any) string {
	switch m := meta.(type) {
	case map[string]any:
		if kind, ok := m["kind"].(string); ok {
			return kind
		}
	case map[any]any:
		if kind, ok := m["kind"].(string); ok {
			return kind
		}
	}
	return ""
}

func awaitResourcePath(destHash []byte, timeout time.Duration) bool {
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

func loadOrCreateResourceIdentity(path string) (*rns.Identity, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".reticulum", "storage", "identities", resAppName)
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

func durationResource(seconds float64) time.Duration {
	if seconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(seconds * float64(time.Second))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
