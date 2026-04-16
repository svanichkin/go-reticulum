package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

type observeHandler struct {
	aspect string

	mu      sync.Mutex
	count   int
	firstAt time.Time
	packets map[string]struct{}
	dupes   int
}

func (h *observeHandler) AspectFilter() string { return h.aspect }

func (h *observeHandler) ReceivedAnnounce(destinationHash []byte, _ *rns.Identity, _ []byte) {
	h.ReceivedAnnounceWithPacketHash(destinationHash, nil, nil, nil)
}

func (h *observeHandler) ReceivedAnnounceWithPacketHash(destinationHash []byte, _ *rns.Identity, _ []byte, announcePacketHash []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.count++
	if h.firstAt.IsZero() {
		h.firstAt = time.Now()
	}
	if len(announcePacketHash) > 0 {
		if h.packets == nil {
			h.packets = map[string]struct{}{}
		}
		packetHex := rns.PrettyHash(announcePacketHash)
		if _, ok := h.packets[packetHex]; ok {
			h.dupes++
		} else {
			h.packets[packetHex] = struct{}{}
		}
		fmt.Printf("OBSERVED count=%d destination=%s packet=%s\n", h.count, rns.PrettyHash(destinationHash), packetHex)
	} else {
		fmt.Printf("OBSERVED count=%d destination=%s\n", h.count, rns.PrettyHash(destinationHash))
	}
}

func (h *observeHandler) snapshot() (int, time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count, h.firstAt
}

func main() {
	var (
		configDir string
		aspect    string
		timeout   time.Duration
		settle    time.Duration
		logLevel  int
	)

	flag.StringVar(&configDir, "config", "", "Reticulum config dir")
	flag.StringVar(&aspect, "aspect", "", "announce aspect filter")
	flag.DurationVar(&timeout, "timeout", 12*time.Second, "overall observer timeout")
	flag.DurationVar(&settle, "settle", 3*time.Second, "extra wait after first observe")
	flag.IntVar(&logLevel, "loglevel", 1, "Reticulum log level")
	flag.Parse()

	if configDir == "" || aspect == "" {
		fmt.Fprintln(os.Stderr, "usage: observe_announces_go.go -config DIR -aspect app.aspect")
		os.Exit(2)
	}

	if _, err := rns.NewReticulum(&configDir, &logLevel, nil, nil, false, nil); err != nil {
		fmt.Fprintf(os.Stderr, "NewReticulum failed: %v\n", err)
		os.Exit(1)
	}

	handler := &observeHandler{aspect: aspect}
	rns.RegisterAnnounceHandler(handler)
	defer rns.DeregisterAnnounceHandler(handler)

	fmt.Println("READY")

	deadline := time.Now().Add(timeout)
	for {
		count, firstAt := handler.snapshot()
		now := time.Now()

		if count > 0 && !firstAt.IsZero() && now.After(firstAt.Add(settle)) {
			break
		}
		if now.After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	count, _ := handler.snapshot()
	fmt.Printf("COUNT=%d\n", count)
	handler.mu.Lock()
	fmt.Printf("PACKETS=%d\n", len(handler.packets))
	fmt.Printf("DUPLICATES=%d\n", handler.dupes)
	handler.mu.Unlock()
}
