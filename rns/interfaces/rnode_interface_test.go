package interfaces

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

func TestRNode_KISSEscape(t *testing.T) {
	t.Parallel()

	in := []byte{0x00, KISS_FEND, 0x01, KISS_FESC, 0x02}
	out := kissEscape(in)
	if bytes.Contains(out, []byte{KISS_FEND}) {
		t.Fatalf("unexpected raw FEND in output: %x", out)
	}
}

func TestRNode_BatteryStateString(t *testing.T) {
	t.Parallel()

	r := &RNodeInterface{}
	r.batteryState = 0
	_ = r.BatteryStateString()
}

type scriptedTransport struct {
	mu      sync.Mutex
	readBuf []byte
	writes  [][]byte
	open    bool
}

func (t *scriptedTransport) Open(context.Context) error { t.mu.Lock(); t.open = true; t.mu.Unlock(); return nil }
func (t *scriptedTransport) Close() error              { t.mu.Lock(); t.open = false; t.mu.Unlock(); return nil }
func (t *scriptedTransport) IsOpen() bool              { t.mu.Lock(); defer t.mu.Unlock(); return t.open }
func (t *scriptedTransport) String() string            { return "test://" }

func (t *scriptedTransport) Read(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.open {
		return 0, io.EOF
	}
	if len(t.readBuf) == 0 {
		return 0, nil
	}
	p[0] = t.readBuf[0]
	t.readBuf = t.readBuf[1:]
	return 1, nil
}

func (t *scriptedTransport) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.open {
		return 0, errors.New("closed")
	}
	cp := append([]byte(nil), p...)
	t.writes = append(t.writes, cp)
	return len(p), nil
}

func (t *scriptedTransport) Feed(b []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.readBuf = append(t.readBuf, b...)
}

type rnodeTestOwner struct {
	mu   sync.Mutex
	got  [][]byte
	done chan struct{}
}

func (o *rnodeTestOwner) Inbound(b []byte, _ *RNodeInterface) {
	o.mu.Lock()
	o.got = append(o.got, append([]byte(nil), b...))
	if o.done != nil {
		select {
		case <-o.done:
		default:
			close(o.done)
		}
	}
	o.mu.Unlock()
}

func TestRNode_SendData_FramesEscapes(t *testing.T) {
	t.Parallel()

	tr := &scriptedTransport{open: true}
	r := NewRNodeInterface(&rnodeTestOwner{}, nil, "r0", tr)
	r.online.Store(true)
	r.ready.Store(true)

	payload := []byte{0x01, FEND, 0x02, FESC, 0x03}
	if err := r.SendData(payload); err != nil {
		t.Fatalf("SendData: %v", err)
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()
	if len(tr.writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(tr.writes))
	}
	want := r.makeKISSFrame(CMD_DATA, payload)
	if !bytes.Equal(tr.writes[0], want) {
		t.Fatalf("unexpected frame:\nwant %x\ngot  %x", want, tr.writes[0])
	}
}

func TestRNode_ReadLoop_ParsesCmdsAndData(t *testing.T) {
	tr := &scriptedTransport{open: true}
	owner := &rnodeTestOwner{done: make(chan struct{})}
	r := NewRNodeInterface(owner, nil, "r0", tr)
	r.ReadTimeout = 500 * time.Millisecond

	go r.readLoop()
	t.Cleanup(func() { _ = r.Stop() })

	// Send frequency report.
	freq := uint32(915000000)
	fb := make([]byte, 4)
	binary.BigEndian.PutUint32(fb, freq)
	tr.Feed(r.makeKISSFrame(CMD_FREQUENCY, fb))

	// Send a data frame.
	payload := []byte("hello\xC0\xDBworld")
	tr.Feed(r.makeKISSFrame(CMD_DATA, payload))

	select {
	case <-owner.done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for inbound")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && r.rFreq != freq {
		time.Sleep(10 * time.Millisecond)
	}
	if r.rFreq != freq {
		t.Fatalf("expected rFreq %d got %d", freq, r.rFreq)
	}
	if r.repStatRSSI.Load() || r.repStatSNR.Load() {
		t.Fatalf("expected RSSI/SNR stat flags to be cleared after data receive")
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if len(owner.got) == 0 || !bytes.Equal(owner.got[0], payload) {
		t.Fatalf("unexpected inbound payload: %q", string(owner.got[0]))
	}
}

func TestRNode_FlowControl_READY_FlushesQueue(t *testing.T) {
	tr := &scriptedTransport{open: true}
	r := NewRNodeInterface(&rnodeTestOwner{}, nil, "r0", tr)
	r.online.Store(true)
	r.FlowCtrl = true
	r.ready.Store(false)

	_ = r.SendData([]byte("a"))

	// Now feed a READY command; it should flush the queued payload.
	go r.readLoop()
	t.Cleanup(func() { _ = r.Stop() })
	tr.Feed([]byte{FEND, CMD_READY, 0x00, FEND})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tr.mu.Lock()
		n := len(tr.writes)
		tr.mu.Unlock()
		if n > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected queued frame to be flushed after READY")
}

func TestRNode_ConfigureDevice_StartsReadLoopAfterSleep(t *testing.T) {
	prevStartReadLoop := rnodeStartReadLoop
	t.Cleanup(func() { rnodeStartReadLoop = prevStartReadLoop })

	tr := &scriptedTransport{open: true}
	r := NewRNodeInterface(&rnodeTestOwner{}, nil, "r0", tr)
	r.Frequency = 915000000
	r.Bandwidth = 125000
	r.TXPower = 2
	r.SF = 7
	r.CR = 5

	started := make(chan struct{}, 1)
	rnodeStartReadLoop = func(_ *RNodeInterface) {
		select {
		case started <- struct{}{}:
		default:
		}
	}

	done := make(chan struct{})
	go func() {
		_ = r.ConfigureDevice()
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2500 * time.Millisecond):
		t.Fatalf("expected read loop to start from ConfigureDevice")
	}

	_ = r.Stop()
	<-done
}
