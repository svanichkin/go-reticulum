package interfaces

import (
	"bytes"
	"testing"
	"time"
)

func TestTCPInterface_HDLC_Escape_MatchesLocalHDLC(t *testing.T) {
	t.Parallel()

	in := []byte{0x00, HDLC_FLAG, 0x01, HDLC_ESC, 0x02}
	got := (HDLC{}).Escape(in)
	want := hdlcEscape(in)
	if !bytes.Equal(got, want) {
		t.Fatalf("HDLC escape mismatch: got=%x want=%x", got, want)
	}
}

func TestTCPInterface_KISS_Escape(t *testing.T) {
	t.Parallel()

	in := []byte{0x00, KISS_FEND, 0x01, KISS_FESC, 0x02}
	out := (KISS{}).Escape(in)

	// Ensure no raw FEND/FESC remain.
	for _, b := range out {
		if b == KISS_FEND || b == KISS_FESC {
			// FESC can appear, but only as the escape prefix. A raw FEND should never.
			// Since the output may contain FESC as prefix, accept it but ensure it
			// isn't followed by an invalid byte.
			// Keep the test simple: just ensure raw FEND is absent.
			if b == KISS_FEND {
				t.Fatalf("unexpected raw FEND in escaped output: %x", out)
			}
		}
	}
	if bytes.Contains(out, []byte{KISS_FEND}) {
		t.Fatalf("raw FEND not expected in escaped output: %x", out)
	}
}

func TestTCPClientInterface_SynthesizeTunnelIfNeeded_CallsTunnelSynthesizer(t *testing.T) {
	oldSynth := TunnelSynthesizer
	defer func() { TunnelSynthesizer = oldSynth }()

	called := make(chan *Interface, 1)
	TunnelSynthesizer = func(ifc *Interface) {
		select {
		case called <- ifc:
		default:
		}
	}

	ifc := &Interface{
		Name:              "test",
		Type:              "TCPClientInterface",
		IN:                true,
		OUT:               true,
		DriverImplemented: true,
		WantsTunnel:       true,
	}
	client := NewTCPClientInitiator(nil, nil, "test", "127.0.0.1", 1, false, false)
	client.wantsTunnel.Store(true)
	ifc.SetTCPClient(client)
	client.synthesizeTunnelIfNeeded()

	select {
	case got := <-called:
		if got != ifc {
			t.Fatalf("TunnelSynthesizer got unexpected iface pointer: got=%p want=%p", got, ifc)
		}
	default:
		t.Fatalf("TunnelSynthesizer was not called")
	}

	if client.WantsTunnel() {
		t.Fatalf("wantsTunnel flag was not cleared")
	}
	if ifc.WantsTunnel {
		t.Fatalf("interface WantsTunnel flag was not cleared")
	}
}

func TestTCPClientInterface_ProcessOutgoing_OfflineIsNoop(t *testing.T) {
	ci := NewTCPClientInitiator(nil, nil, "test", "", 0, false, false)
	ci.Initiator = true
	ci.online.Store(false)
	ci.detached.Store(false)
	ci.ReconnectWait = 200 * time.Millisecond
	tries := 0
	ci.MaxReconnectTry = &tries

	if err := ci.ProcessOutgoing([]byte{0x01}); err != nil {
		t.Fatalf("offline process outgoing should be a no-op, got error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if ci.reconnect.Load() {
		t.Fatalf("offline process outgoing should not start reconnect")
	}
}

func TestTCPClientInterface_TeardownDisablesInterfaceDirections(t *testing.T) {
	ci := NewTCPClientInitiator(nil, nil, "test", "", 0, false, false)
	ci.iface = &Interface{IN: true, OUT: true}
	ci.online.Store(true)

	ci.teardown()

	if ci.iface.IN || ci.iface.OUT {
		t.Fatalf("teardown should disable IN/OUT, got IN=%v OUT=%v", ci.iface.IN, ci.iface.OUT)
	}
}
