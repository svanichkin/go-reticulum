package interfaces

import (
	"errors"
	"net"
	"sync"
	"testing"
)

type trackingListener struct {
	mu     sync.Mutex
	closed bool
}

func (l *trackingListener) Accept() (net.Conn, error) { return nil, errors.New("closed") }

func (l *trackingListener) Close() error {
	l.mu.Lock()
	l.closed = true
	l.mu.Unlock()
	return nil
}

func (l *trackingListener) Addr() net.Addr { return trackingAddr("tracking") }

func (l *trackingListener) Closed() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.closed
}

type trackingAddr string

func (a trackingAddr) Network() string { return string(a) }
func (a trackingAddr) String() string  { return string(a) }

func TestBackboneDeregisterListenersClosesRegisteredListeners(t *testing.T) {
	backboneDriversMu.Lock()
	prevDrivers := backboneDrivers
	backboneDrivers = make(map[*BackboneInterfaceDriver]struct{})
	backboneDriversMu.Unlock()
	t.Cleanup(func() {
		backboneDriversMu.Lock()
		backboneDrivers = prevDrivers
		backboneDriversMu.Unlock()
	})

	ln := &trackingListener{}
	driver := &BackboneInterfaceDriver{
		stopCh: make(chan struct{}),
		lns:    []net.Listener{ln},
	}
	registerBackboneDriver(driver)

	DeregisterListeners()

	if !ln.Closed() {
		t.Fatal("expected listener to be closed")
	}

	select {
	case <-driver.stopCh:
	default:
		t.Fatal("expected driver stop channel to be closed")
	}

	backboneDriversMu.Lock()
	remaining := len(backboneDrivers)
	backboneDriversMu.Unlock()
	if remaining != 0 {
		t.Fatalf("expected backbone driver registry to be empty, got %d", remaining)
	}
}
