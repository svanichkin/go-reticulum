package interfaces

import (
	"bytes"
	"crypto/sha256"
	"net"
	"testing"
	"time"
)

func TestHDLCEscapeUnescape_RoundTrip(t *testing.T) {
	t.Parallel()

	cases := [][]byte{
		nil,
		{},
		{0x01, 0x02, 0x03},
		{hdlcFlag},
		{hdlcEsc},
		{0x00, hdlcFlag, 0xFF, hdlcEsc, 0x7E},
	}

	for _, in := range cases {
		enc := hdlcEscape(in)
		out := hdlcUnescape(enc)
		if !bytes.Equal(out, in) {
			t.Fatalf("roundtrip mismatch: in=%x out=%x enc=%x", in, out, enc)
		}
	}
}

func TestInterface_ProcessAnnounceRaw_DefaultAnnounceCapProvider(t *testing.T) {
	// Do not run in parallel: overrides global hooks.

	prevCap := DefaultAnnounceCapProvider
	prevHeader := HeaderMinSize
	t.Cleanup(func() {
		DefaultAnnounceCapProvider = prevCap
		HeaderMinSize = prevHeader
	})

	DefaultAnnounceCapProvider = func() float64 { return 1.0 }
	HeaderMinSize = 0

	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()

	iface := &Interface{
		Name:      "local0",
		Type:      "LocalInterface",
		Online:    true,
		Bitrate:   62500,
		AnnounceCap: 0, // should default via provider
	}
	iface.setLocalConn(serverSide)

	raw := []byte("hello-announce")
	iface.ProcessAnnounceRaw(raw, 1)

	_ = clientSide.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 256)
	n, err := clientSide.Read(buf)
	if err != nil || n == 0 {
		t.Fatalf("expected framed bytes written, got n=%d err=%v", n, err)
	}
}

func TestInterface_IncomingAnnounceFrequency_RequiresMinimumSamples(t *testing.T) {
	t.Parallel()

	iface := &Interface{Name: "if0"}
	for range ICDequeMinSample {
		iface.ReceivedAnnounce()
	}

	if v := iface.IncomingAnnounceFrequency(); v != 0 {
		t.Fatalf("expected 0 frequency below/equal min sample threshold, got %f", v)
	}

	iface.ReceivedAnnounce()

	if v := iface.IncomingAnnounceFrequency(); v <= 0 {
		t.Fatalf("expected >0 frequency once threshold exceeded, got %f", v)
	}
}

func TestInterface_OutgoingAnnounceFrequency_UsesWindowSpan(t *testing.T) {
	t.Parallel()

	now := time.Now()
	iface := &Interface{
		Name: "if0",
		oaFreq: []time.Time{
			now.Add(-4 * time.Second),
			now.Add(-3 * time.Second),
			now.Add(-2 * time.Second),
			now.Add(-1 * time.Second),
		},
	}

	got := iface.OutgoingAnnounceFrequency()
	if got < 0.95 || got > 1.05 {
		t.Fatalf("expected outgoing frequency close to 1.0 Hz, got %f", got)
	}
}

func TestInterface_Hash_MatchesSHA256OfString(t *testing.T) {
	t.Parallel()

	iface := &Interface{Name: "if0"}
	want := sha256.Sum256([]byte("if0"))
	got := iface.Hash()
	if !bytes.Equal(got, want[:]) {
		t.Fatalf("unexpected hash: want %x got %x", want[:], got)
	}
}
