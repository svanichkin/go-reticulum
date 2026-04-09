package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const (
	reqAppName = "parityreq"
	reqAspect  = "rpc"
)

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
	flag.Float64Var(&waitSeconds, "wait-seconds", 30, "wait seconds")
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

	id, err := loadOrCreateReqIdentity(identityPath)
	if err != nil {
		fatalf("identity failed: %v", err)
	}

	if listenMode {
		if err := runReqListener(id, waitSeconds); err != nil {
			fatalf("listener failed: %v", err)
		}
		return
	}

	if destinationHex == "" {
		fatalf("destination is required")
	}
	if err := runReqClient(id, destinationHex, waitSeconds); err != nil {
		fatalf("client failed: %v", err)
	}
}

func runReqListener(id *rns.Identity, waitSeconds float64) error {
	dest, err := rns.NewDestination(id, rns.DestinationIN, rns.DestinationSINGLE, reqAppName, reqAspect)
	if err != nil {
		return err
	}

	if err := dest.RegisterRequestHandler("echo", func(path string, data any, requestID []byte, linkID []byte, remoteIdentity *rns.Identity, requestedAt time.Time) any {
		return fmt.Sprintf("echo:%v", data)
	}, rns.DestinationALLOW_ALL, nil); err != nil {
		return err
	}

	if err := dest.RegisterRequestHandler("sleep", func(path string, data any, requestID []byte, linkID []byte, remoteIdentity *rns.Identity, requestedAt time.Time) any {
		secs := 3.0
		switch v := data.(type) {
		case int64:
			secs = float64(v)
		case int:
			secs = float64(v)
		case float64:
			secs = v
		case string:
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				secs = parsed
			}
		}
		time.Sleep(time.Duration(secs * float64(time.Second)))
		return fmt.Sprintf("slept:%v", secs)
	}, rns.DestinationALLOW_ALL, nil); err != nil {
		return err
	}

	fmt.Printf("LISTEN_HASH %s\n", hex.EncodeToString(dest.Hash()))
	time.Sleep(time.Second)
	deadline := time.Now().Add(durationReq(waitSeconds))
	for time.Now().Before(deadline) {
		dest.Announce(nil, false, nil, nil, true)
		time.Sleep(3 * time.Second)
	}
	fmt.Println("EVENT timeout")
	return nil
}

func runReqClient(id *rns.Identity, destinationHex string, waitSeconds float64) error {
	destHash, err := hex.DecodeString(destinationHex)
	if err != nil {
		return err
	}
	if !awaitReqPath(destHash, durationReq(waitSeconds)) {
		return fmt.Errorf("path not found")
	}

	remoteID := rns.IdentityRecall(destHash)
	if remoteID == nil {
		return fmt.Errorf("could not recall remote identity")
	}
	remoteDest, err := rns.NewDestination(remoteID, rns.DestinationOUT, rns.DestinationSINGLE, reqAppName, reqAspect)
	if err != nil {
		return err
	}
	link, err := rns.NewOutgoingLink(remoteDest, rns.LinkModeDefault, nil, nil)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(durationReq(waitSeconds))
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
	time.Sleep(200 * time.Millisecond)

	respCh := make(chan *rns.RequestReceipt, 1)
	failCh := make(chan *rns.RequestReceipt, 1)
	rr := rns.RequestReceiptFrom(link.Request("echo", "hello", func(rr *rns.RequestReceipt) {
		respCh <- rr
	}, func(rr *rns.RequestReceipt) {
		failCh <- rr
	}, nil, 5))
	if rr == nil {
		return fmt.Errorf("echo request not sent")
	}
	select {
	case got := <-respCh:
		if got == nil || got.Response() == nil {
			return fmt.Errorf("empty echo response")
		}
		fmt.Printf("EVENT echo_response %v\n", got.Response())
	case <-failCh:
		return fmt.Errorf("echo request failed")
	case <-time.After(8 * time.Second):
		return fmt.Errorf("echo response timeout")
	}

	failCh = make(chan *rns.RequestReceipt, 1)
	respCh = make(chan *rns.RequestReceipt, 1)
	rr = rns.RequestReceiptFrom(link.Request("sleep", 3, func(rr *rns.RequestReceipt) {
		respCh <- rr
	}, func(rr *rns.RequestReceipt) {
		failCh <- rr
	}, nil, 1))
	if rr == nil {
		return fmt.Errorf("sleep request not sent")
	}
	select {
	case <-failCh:
		return fmt.Errorf("sleep request unexpectedly failed")
	case got := <-respCh:
		if got == nil || got.Response() == nil {
			return fmt.Errorf("empty sleep response")
		}
		fmt.Printf("EVENT sleep_response %v\n", got.Response())
	case <-time.After(6 * time.Second):
		return fmt.Errorf("sleep response timeout")
	}

	link.Teardown()
	return nil
}

func awaitReqPath(destHash []byte, timeout time.Duration) bool {
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

func loadOrCreateReqIdentity(path string) (*rns.Identity, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".reticulum", "storage", "identities", reqAppName)
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

func durationReq(sec float64) time.Duration {
	return time.Duration(sec * float64(time.Second))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
