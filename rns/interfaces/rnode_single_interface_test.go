package interfaces

import (
	"context"
	"errors"
	"testing"
)

func TestRNodeSingleInterface_StringMatchesPython(t *testing.T) {
	t.Parallel()

	iface := &Interface{
		Name: "r0",
		Type: "RNodeInterface",
	}

	if got := iface.String(); got != "RNodeInterface[r0]" {
		t.Fatalf("unexpected string form %q", got)
	}
}

func TestNewRNodeInterfaceFromConfig_InitialOpenFailureStartsReconnectLoop(t *testing.T) {
	prevFactory := rnodeTransportFromPort
	prevStart := rnodeStartInterface
	prevReconnect := rnodeStartReconnectLoop
	t.Cleanup(func() {
		rnodeTransportFromPort = prevFactory
		rnodeStartInterface = prevStart
		rnodeStartReconnectLoop = prevReconnect
	})

	rn := NewRNodeInterface(&rnodeTestOwner{}, nil, "r0", &scriptedTransport{})
	calledReconnect := make(chan struct{}, 1)

	rnodeTransportFromPort = func(owner Owner, log Logger, name string, port string) (*RNodeInterface, error) {
		_ = owner
		_ = log
		_ = name
		_ = port
		return rn, nil
	}
	rnodeStartInterface = func(_ *RNodeInterface, _ context.Context) error {
		return errors.New("open failed")
	}
	rnodeStartReconnectLoop = func(_ *RNodeInterface) {
		select {
		case calledReconnect <- struct{}{}:
		default:
		}
	}

	iface, err := NewRNodeInterfaceFromConfig("r0", map[string]string{
		"port":            "/dev/ttyUSB0",
		"frequency":       "915000000",
		"bandwidth":       "125000",
		"txpower":         "2",
		"spreadingfactor": "7",
		"codingrate":      "5",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iface == nil || iface.rnodeSingle != rn {
		t.Fatalf("expected interface with retained rnode driver")
	}

	select {
	case <-calledReconnect:
	default:
		t.Fatalf("expected reconnect loop to be started")
	}
}
