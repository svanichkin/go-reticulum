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
	appName = "paritylink"
	aspect  = "hold"
)

func main() {
	var (
		configDir        string
		identityPath     string
		destinationHex   string
		listenMode       bool
		identify         bool
		teardown         bool
		expectClose      bool
		traceMode        bool
		holdSeconds      float64
		waitSeconds      float64
		keepaliveSeconds float64
	)

	flag.StringVar(&configDir, "config", "", "reticulum config dir")
	flag.StringVar(&identityPath, "identity", "", "identity path")
	flag.StringVar(&destinationHex, "destination", "", "listener destination hash")
	flag.BoolVar(&listenMode, "listen", false, "listen for links")
	flag.BoolVar(&identify, "identify", true, "send identify on outgoing link")
	flag.BoolVar(&teardown, "teardown", false, "teardown link after hold")
	flag.BoolVar(&expectClose, "expect-close", false, "expect link to close during hold")
	flag.BoolVar(&traceMode, "trace", false, "enable trace logging")
	flag.Float64Var(&holdSeconds, "hold-seconds", 0, "hold active link for N seconds")
	flag.Float64Var(&waitSeconds, "wait-seconds", 30, "overall wait timeout")
	flag.Float64Var(&keepaliveSeconds, "keepalive-seconds", 0, "override keepalive after link activation")
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
	if traceMode {
		rns.SetLogLevel(rns.LOG_DEBUG)
	} else {
		rns.SetLogLevel(-1)
	}

	id, err := loadOrCreateIdentity(identityPath)
	if err != nil {
		fatalf("identity failed: %v", err)
	}

	if listenMode {
		if err := runListener(id, waitSeconds, keepaliveSeconds); err != nil {
			fatalf("listener failed: %v", err)
		}
		return
	}

	if destinationHex == "" {
		fatalf("destination is required in client mode")
	}
	if err := runClient(id, destinationHex, identify, teardown, expectClose, holdSeconds, waitSeconds, keepaliveSeconds); err != nil {
		fatalf("client failed: %v", err)
	}
}

func runListener(id *rns.Identity, waitSeconds, keepaliveSeconds float64) error {
	dest, err := rns.NewDestination(id, rns.DestinationIN, rns.DestinationSINGLE, appName, aspect)
	if err != nil {
		return err
	}

	closedCh := make(chan struct{}, 1)
	dest.SetLinkEstablishedCallback(func(l *rns.Link) {
		applyKeepalive(l, keepaliveSeconds)
		fmt.Printf("EVENT established %s\n", hex.EncodeToString(l.LinkID))
		if mtu := l.GetMTU(); mtu != nil {
			if mdu := l.GetMDU(); mdu != nil {
				fmt.Printf("EVENT mtu=%d mdu=%d\n", *mtu, *mdu)
			} else {
				fmt.Printf("EVENT mtu=%d\n", *mtu)
			}
		}
		l.SetRemoteIdentifiedCallback(func(_ *rns.Link, rid *rns.Identity) {
			if rid != nil {
				fmt.Printf("EVENT identified %s\n", hex.EncodeToString(rid.Hash))
			}
		})
		l.SetLinkClosedCallback(func(cl *rns.Link) {
			fmt.Printf("EVENT closed reason=%d initiator=%v\n", cl.TeardownReason, cl.Initiator)
			select {
			case closedCh <- struct{}{}:
			default:
			}
		})
	})

	fmt.Printf("LISTEN_HASH %s\n", hex.EncodeToString(dest.Hash))
	time.Sleep(1 * time.Second)

	deadline := time.Now().Add(durationSeconds(waitSeconds))
	for {
		dest.Announce(nil, false, nil, nil, true)
		if time.Now().After(deadline) {
			fmt.Println("EVENT timeout")
			return nil
		}
		select {
		case <-closedCh:
			return nil
		case <-time.After(3 * time.Second):
		}
	}
}

func runClient(id *rns.Identity, destinationHex string, identify, teardown, expectClose bool, holdSeconds, waitSeconds, keepaliveSeconds float64) error {
	destHash, err := hex.DecodeString(destinationHex)
	if err != nil {
		return err
	}

	if !awaitPath(destHash, durationSeconds(waitSeconds)) {
		return fmt.Errorf("path not found")
	}

	remoteID := rns.IdentityRecall(destHash)
	if remoteID == nil {
		return fmt.Errorf("could not recall remote identity")
	}
	remoteDest, err := rns.NewDestination(remoteID, rns.DestinationOUT, rns.DestinationSINGLE, appName, aspect)
	if err != nil {
		return err
	}

	closedCh := make(chan struct{}, 1)
	link, err := rns.NewOutgoingLink(remoteDest, rns.LinkModeDefault, nil, nil)
	if err != nil {
		return err
	}
	link.SetLinkClosedCallback(func(cl *rns.Link) {
		fmt.Printf("EVENT closed reason=%d initiator=%v\n", cl.TeardownReason, cl.Initiator)
		select {
		case closedCh <- struct{}{}:
		default:
		}
	})

	deadline := time.Now().Add(durationSeconds(waitSeconds))
	for time.Now().Before(deadline) {
		if link.Status == rns.LinkActive {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if link.Status != rns.LinkActive {
		return fmt.Errorf("link not active")
	}

	applyKeepalive(link, keepaliveSeconds)
	fmt.Printf("EVENT established %s\n", hex.EncodeToString(link.LinkID))
	if mtu := link.GetMTU(); mtu != nil {
		if mdu := link.GetMDU(); mdu != nil {
			fmt.Printf("EVENT mtu=%d mdu=%d\n", *mtu, *mdu)
		} else {
			fmt.Printf("EVENT mtu=%d\n", *mtu)
		}
	}

	if identify {
		link.Identify(id)
		fmt.Println("EVENT identify_sent")
		time.Sleep(200 * time.Millisecond)
	}

	if holdSeconds > 0 {
		time.Sleep(durationSeconds(holdSeconds))
		if expectClose {
			if link.Status == rns.LinkClosed {
				fmt.Println("EVENT stale_closed")
			} else {
				return fmt.Errorf("link did not close during hold, status=%d", link.Status)
			}
		} else if link.Status == rns.LinkActive {
			fmt.Println("EVENT still_active")
		} else {
			return fmt.Errorf("link not active after hold, status=%d", link.Status)
		}
	}

	if teardown {
		link.Teardown()
		deadline = time.Now().Add(durationSeconds(waitSeconds))
		for time.Now().Before(deadline) {
			if link.Status == rns.LinkClosed {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if link.Status != rns.LinkClosed {
			return fmt.Errorf("link did not close")
		}
		select {
		case <-closedCh:
		default:
		}
	}

	return nil
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

func applyKeepalive(link *rns.Link, keepaliveSeconds float64) {
	if link == nil || keepaliveSeconds <= 0 {
		return
	}
	keep := durationSeconds(keepaliveSeconds)
	link.Keepalive = keep
	link.StaleTime = keep * 2
}

func loadOrCreateIdentity(path string) (*rns.Identity, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".reticulum", "storage", "identities", appName)
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

func durationSeconds(sec float64) time.Duration {
	return time.Duration(sec * float64(time.Second))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
