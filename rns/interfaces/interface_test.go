package interfaces

import (
	"bytes"
	"crypto/sha256"
	"io"
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

func TestInterface_ProcessAnnounceQueue_DefaultAnnounceCapProvider(t *testing.T) {
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
		Name:        "local0",
		Type:        "LocalInterface",
		Online:      true,
		Bitrate:     62500,
		AnnounceCap: 1.0,
	}
	iface.setLocalConn(serverSide)

	iface.ICMu.Lock()
	iface.AnnounceQueue = append(iface.AnnounceQueue, AnnounceQueueEntry{
		Time: time.Now(),
		Hops: 1,
		Raw:  []byte("hello-announce"),
	})
	iface.AnnounceRunning = true
	iface.ICMu.Unlock()

	done := make(chan struct{})
	go func() {
		iface.ProcessAnnounceQueue()
		close(done)
	}()

	_ = clientSide.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 256)
	n, err := clientSide.Read(buf)
	if err != nil || n == 0 {
		t.Fatalf("expected framed bytes written, got n=%d err=%v", n, err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ProcessAnnounceQueue did not finish after announce send")
	}
	if iface.HasQueuedAnnounces() {
		t.Fatal("announce unexpectedly remained queued")
	}
}

func TestInterface_ReadLocalFramesLoop_ProcessesDataReturnedWithEOF(t *testing.T) {
	prevInbound := InboundHandler
	prevHeader := HeaderMinSize
	t.Cleanup(func() {
		InboundHandler = prevInbound
		HeaderMinSize = prevHeader
	})

	payload := []byte("eof-frame")
	framed := append([]byte{hdlcFlag}, hdlcEscape(payload)...)
	framed = append(framed, hdlcFlag)

	got := make(chan []byte, 1)
	InboundHandler = func(raw []byte, _ *Interface) {
		got <- append([]byte(nil), raw...)
	}
	HeaderMinSize = 0

	iface := &Interface{Name: "local0", Type: "LocalInterface"}
	iface.setLocalConn(&eofDataConn{data: framed})

	iface.readLocalFramesLoop()

	select {
	case raw := <-got:
		if !bytes.Equal(raw, payload) {
			t.Fatalf("inbound payload=%x, want %x", raw, payload)
		}
	default:
		t.Fatal("expected frame to be delivered before EOF")
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

type eofDataConn struct {
	data []byte
	read bool
}

func (c *eofDataConn) Read(p []byte) (int, error) {
	if c.read {
		return 0, io.EOF
	}
	c.read = true
	n := copy(p, c.data)
	return n, io.EOF
}

func (c *eofDataConn) Write(p []byte) (int, error)      { return len(p), nil }
func (c *eofDataConn) Close() error                     { return nil }
func (c *eofDataConn) LocalAddr() net.Addr              { return eofAddr("local") }
func (c *eofDataConn) RemoteAddr() net.Addr             { return eofAddr("remote") }
func (c *eofDataConn) SetDeadline(time.Time) error      { return nil }
func (c *eofDataConn) SetReadDeadline(time.Time) error  { return nil }
func (c *eofDataConn) SetWriteDeadline(time.Time) error { return nil }

type eofAddr string

func (a eofAddr) Network() string { return "test" }
func (a eofAddr) String() string  { return string(a) }
