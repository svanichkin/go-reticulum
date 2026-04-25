package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
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

	var (
		linksMu sync.Mutex
		links   []*rns.Link
	)
	dest.SetLinkEstablishedCallback(func(link *rns.Link) {
		linksMu.Lock()
		links = append(links, link)
		linksMu.Unlock()
	})

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

	if err := dest.RegisterRequestHandler("timeout", func(path string, data any, requestID []byte, linkID []byte, remoteIdentity *rns.Identity, requestedAt time.Time) any {
		return nil
	}, rns.DestinationALLOW_ALL, nil); err != nil {
		return err
	}

	if err := dest.RegisterRequestHandler("denied", func(path string, data any, requestID []byte, linkID []byte, remoteIdentity *rns.Identity, requestedAt time.Time) any {
		return "unexpected-denied-response"
	}, rns.DestinationALLOW_NONE, nil); err != nil {
		return err
	}

	if err := dest.RegisterRequestHandler("malformed", func(path string, data any, requestID []byte, linkID []byte, remoteIdentity *rns.Identity, requestedAt time.Time) any {
		linksMu.Lock()
		activeLinks := append([]*rns.Link(nil), links...)
		linksMu.Unlock()
		for _, link := range activeLinks {
			if link != nil && string(link.LinkID) == string(linkID) {
				packet := rns.NewPacket(
					link,
					[]byte("bad-response-payload"),
					rns.PacketTypeData,
					rns.PacketCtxResponse,
					rns.Broadcast,
					rns.HeaderType1,
					nil,
					nil,
					false,
					rns.FlagUnset,
				)
				if packet != nil {
					_ = packet.Send()
					fmt.Println("EVENT malformed_sent")
				}
				break
			}
		}
		return nil
	}, rns.DestinationALLOW_ALL, nil); err != nil {
		return err
	}

	fmt.Printf("LISTEN_HASH %s\n", hex.EncodeToString(dest.Hash))
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
	link, err := rns.NewLink(remoteDest, nil, rns.LinkModeDefault, nil, nil)
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
	result := link.Request("echo", "hello", func(rr *rns.RequestReceipt) {
		respCh <- rr
	}, func(rr *rns.RequestReceipt) {
		failCh <- rr
	}, nil, 5)
	rr, _ := result.(*rns.RequestReceipt)
	if rr == nil {
		return fmt.Errorf("echo request not sent")
	}
	select {
	case got := <-respCh:
		if got == nil || got.GetResponse() == nil {
			return fmt.Errorf("empty echo response")
		}
		fmt.Printf("EVENT echo_response %v\n", got.GetResponse())
	case <-failCh:
		return fmt.Errorf("echo request failed")
	case <-time.After(8 * time.Second):
		return fmt.Errorf("echo response timeout")
	}

	failCh = make(chan *rns.RequestReceipt, 1)
	respCh = make(chan *rns.RequestReceipt, 1)
	result = link.Request("sleep", 3, func(rr *rns.RequestReceipt) {
		respCh <- rr
	}, func(rr *rns.RequestReceipt) {
		failCh <- rr
	}, nil, 1)
	rr, _ = result.(*rns.RequestReceipt)
	if rr == nil {
		return fmt.Errorf("sleep request not sent")
	}
	select {
	case <-failCh:
		return fmt.Errorf("sleep request unexpectedly failed")
	case got := <-respCh:
		if got == nil || got.GetResponse() == nil {
			return fmt.Errorf("empty sleep response")
		}
		fmt.Printf("EVENT sleep_response %v\n", got.GetResponse())
	case <-time.After(6 * time.Second):
		return fmt.Errorf("sleep response timeout")
	}

	denyResult, err := requestExpectNoResponse(link, "denied", "nope", 1, 4*time.Second)
	if err != nil {
		return err
	}
	fmt.Printf("EVENT denied_%s\n", denyResult)

	timeoutResult, err := requestExpectNoResponse(link, "timeout", "hang", 1, 3*time.Second)
	if err != nil {
		return err
	}
	fmt.Printf("EVENT timeout_%s\n", timeoutResult)

	malformedResult, err := requestExpectNoResponse(link, "malformed", "bad", 1, 3*time.Second)
	if err != nil {
		return err
	}
	fmt.Printf("EVENT malformed_%s\n", malformedResult)

	link.Teardown()
	return nil
}

func requestExpectNoResponse(link *rns.Link, path string, data any, timeout float64, wait time.Duration) (string, error) {
	respCh := make(chan *rns.RequestReceipt, 1)
	failCh := make(chan *rns.RequestReceipt, 1)
	result := link.Request(path, data, func(rr *rns.RequestReceipt) {
		respCh <- rr
	}, func(rr *rns.RequestReceipt) {
		failCh <- rr
	}, nil, timeout)
	rr, _ := result.(*rns.RequestReceipt)
	if rr == nil {
		return "", fmt.Errorf("%s request not sent", path)
	}

	select {
	case got := <-respCh:
		if got != nil {
			return "", fmt.Errorf("%s request unexpectedly got response: %v", path, got.GetResponse())
		}
		return "", fmt.Errorf("%s request unexpectedly got empty response", path)
	case <-failCh:
		return "failed", nil
	case <-time.After(wait):
		return "silent", nil
	}
}

func awaitReqPath(destHash []byte, timeout time.Duration) bool {
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
