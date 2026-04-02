package rns

import (
	"encoding/binary"
	"sync"
	"testing"
	"time"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

type testPacket struct {
	mu sync.Mutex

	state    MessageState
	raw      []byte
	packetID uint64

	tries int

	timeout   float64
	timeoutID uint64
	instances int

	timeoutCb   func(any)
	deliveredCb func(any)
}

var (
	testPktIDMu sync.Mutex
	testPktID   uint64
)

func nextTestPacketID() uint64 {
	testPktIDMu.Lock()
	defer testPktIDMu.Unlock()
	testPktID++
	return testPktID
}

func (p *testPacket) setTimeout(cb func(any), timeout *float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if timeout != nil {
		p.timeout = *timeout
	}
	p.timeoutCb = cb
}

func (p *testPacket) clearTimeoutLocked() {
	p.timeoutID = 0
}

func (p *testPacket) send() {
	p.mu.Lock()
	p.tries++
	p.state = MSGSTATE_SENT
	p.instances++
	p.timeoutID = nextTestPacketID()
	timeoutID := p.timeoutID
	timeout := p.timeout
	p.mu.Unlock()

	go func() {
		defer func() {
			p.mu.Lock()
			p.instances--
			p.mu.Unlock()
		}()
		time.Sleep(time.Duration(timeout * float64(time.Second)))
		p.mu.Lock()
		if p.timeoutID == timeoutID && p.timeoutID != 0 {
			p.timeoutID = 0
			p.state = MSGSTATE_FAILED
			cb := p.timeoutCb
			p.mu.Unlock()
			if cb != nil {
				cb(p)
			}
			return
		}
		p.mu.Unlock()
	}()
}

func (p *testPacket) delivered() {
	p.mu.Lock()
	p.state = MSGSTATE_DELIVERED
	p.clearTimeoutLocked()
	cb := p.deliveredCb
	p.mu.Unlock()
	if cb != nil {
		cb(p)
	}
}

type channelOutletTest struct {
	mu sync.Mutex

	linkID uint64
	mdu    int
	rtt    float64
	usable bool

	defaultTimeout float64

	packets []*testPacket

	timeoutCallbacks int
}

func (o *channelOutletTest) Send(raw []byte) any {
	o.mu.Lock()
	defer o.mu.Unlock()
	p := &testPacket{
		state:    MSGSTATE_NEW,
		raw:      append([]byte(nil), raw...),
		packetID: nextTestPacketID(),
		timeout:  o.defaultTimeout,
	}
	p.send()
	o.packets = append(o.packets, p)
	return p
}

func (o *channelOutletTest) Resend(packet any) any {
	p := packet.(*testPacket)
	p.send()
	return p
}

func (o *channelOutletTest) Mdu() int      { return o.mdu }
func (o *channelOutletTest) Rtt() float64  { return o.rtt }
func (o *channelOutletTest) IsUsable() bool { return o.usable }
func (o *channelOutletTest) String() string { return "test-outlet" }

func (o *channelOutletTest) GetPacketState(packet any) MessageState {
	p := packet.(*testPacket)
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (o *channelOutletTest) TimedOut() {
	o.mu.Lock()
	o.timeoutCallbacks++
	o.mu.Unlock()
}

func (o *channelOutletTest) SetPacketTimeoutCallback(packet any, cb func(any), timeout *float64) {
	p := packet.(*testPacket)
	p.setTimeout(cb, timeout)
}

func (o *channelOutletTest) SetPacketDeliveredCallback(packet any, cb func(any)) {
	p := packet.(*testPacket)
	p.mu.Lock()
	p.deliveredCb = cb
	p.mu.Unlock()
}

func (o *channelOutletTest) GetPacketID(packet any) any {
	return packet.(*testPacket).packetID
}

type messageTest struct {
	ID   string
	Data string
}

func (m *messageTest) MsgType() uint16 { return 0xABCD }
func (m *messageTest) Pack() ([]byte, error) {
	return umsgpack.Packb([]any{m.ID, m.Data})
}
func (m *messageTest) Unpack(raw []byte) error {
	var v []any
	if err := umsgpack.Unpackb(raw, &v); err != nil {
		return err
	}
	if len(v) >= 2 {
		if s, ok := v[0].(string); ok {
			m.ID = s
		}
		if s, ok := v[1].(string); ok {
			m.Data = s
		}
	}
	return nil
}

type zeroMsgTypeMessage struct {
	Data string
}

func (m *zeroMsgTypeMessage) MsgType() uint16 { return 0 }
func (m *zeroMsgTypeMessage) Pack() ([]byte, error) { return []byte(m.Data), nil }
func (m *zeroMsgTypeMessage) Unpack(raw []byte) error {
	m.Data = string(raw)
	return nil
}

func TestChannel_SendOneRetry(t *testing.T) {
	maybeParallel(t)
	rtt := 0.01
	outlet := &channelOutletTest{mdu: 500, rtt: rtt, usable: true, linkID: nextTestPacketID()}
	ch := NewChannel(outlet)
	defer ch.Close()
	outlet.defaultTimeout = ch.getPacketTimeoutTime(1)

	msg := &messageTest{ID: "id", Data: "test"}
	if len(outlet.packets) != 0 {
		t.Fatalf("expected no packets")
	}

	env, err := ch.TrySend(msg)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if env == nil || env.raw == nil {
		t.Fatalf("expected envelope with raw")
	}
	if len(outlet.packets) != 1 {
		t.Fatalf("expected 1 packet, got %d", len(outlet.packets))
	}
	if env.packet == nil {
		t.Fatalf("expected env.packet")
	}
	if !env.tracked {
		t.Fatalf("expected envelope tracked")
	}

	p := outlet.packets[0]
	if env.packet != p {
		t.Fatalf("expected envelope to reference the packet")
	}
	if env.tries != 1 || p.tries != 1 {
		t.Fatalf("expected tries=1")
	}
	if p.instances != 1 {
		t.Fatalf("expected instances=1")
	}
	if p.state != MSGSTATE_SENT {
		t.Fatalf("expected packet state sent")
	}

	time.Sleep(time.Duration(ch.getPacketTimeoutTime(1) * 1.1 * float64(time.Second)))
	if env.tries != 2 || p.tries != 2 {
		t.Fatalf("expected tries=2 after timeout")
	}

	time.Sleep(time.Duration(ch.getPacketTimeoutTime(2) * 1.1 * float64(time.Second)))
	if env.tries != 3 || p.tries != 3 {
		t.Fatalf("expected tries=3 after 2nd timeout")
	}
	if p.state != MSGSTATE_SENT {
		t.Fatalf("expected packet state sent")
	}

	p.delivered()
	if p.state != MSGSTATE_DELIVERED {
		t.Fatalf("expected delivered")
	}

	time.Sleep(time.Duration(ch.getPacketTimeoutTime(3) * 1.1 * float64(time.Second)))
	if env.tracked {
		t.Fatalf("expected envelope untracked after delivery")
	}
	p.mu.Lock()
	instances := p.instances
	p.mu.Unlock()
	if instances != 0 {
		t.Fatalf("expected instances=0, got %d", instances)
	}
}

type packetIDOnly struct {
	id uint64
}

type channelOutletIDOnly struct {
	mdu int
	rtt float64

	lastDeliveredCb func(any)
}

func (o *channelOutletIDOnly) Send(_ []byte) any { return &packetIDOnly{id: 1} }
func (o *channelOutletIDOnly) Resend(packet any) any {
	return packet
}
func (o *channelOutletIDOnly) Mdu() int       { return o.mdu }
func (o *channelOutletIDOnly) Rtt() float64   { return o.rtt }
func (o *channelOutletIDOnly) IsUsable() bool { return true }
func (o *channelOutletIDOnly) TimedOut()      {}
func (o *channelOutletIDOnly) String() string { return "id-only" }
func (o *channelOutletIDOnly) GetPacketState(_ any) MessageState {
	return MSGSTATE_SENT
}
func (o *channelOutletIDOnly) SetPacketTimeoutCallback(_ any, _ func(any), _ *float64) {}
func (o *channelOutletIDOnly) SetPacketDeliveredCallback(_ any, cb func(any)) {
	o.lastDeliveredCb = cb
}
func (o *channelOutletIDOnly) GetPacketID(packet any) any {
	return packet.(*packetIDOnly).id
}

func TestChannel_PacketKey_UsesPacketIDNotPointer(t *testing.T) {
	t.Parallel()

	outlet := &channelOutletIDOnly{mdu: 500, rtt: 0.01}
	ch := NewChannel(outlet)
	defer ch.Close()

	if err := ch.TryRegisterMessageType(&messageTest{}); err != nil {
		t.Fatalf("RegisterMessageType: %v", err)
	}
	env, err := ch.TrySend(&messageTest{ID: "id", Data: "data"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if env == nil || outlet.lastDeliveredCb == nil {
		t.Fatalf("missing envelope/callback")
	}

	// Simulate a callback arriving with a different packet instance but the same packet id.
	outlet.lastDeliveredCb(&packetIDOnly{id: 1})

	ch.lock.RLock()
	defer ch.lock.RUnlock()
	if len(ch.txRing) != 0 {
		t.Fatalf("expected txRing cleared after delivery")
	}
}

func TestChannel_SendTimeout(t *testing.T) {
	maybeParallel(t)
	rtt := 0.01
	outlet := &channelOutletTest{mdu: 500, rtt: rtt, usable: true, linkID: nextTestPacketID()}
	ch := NewChannel(outlet)
	defer ch.Close()
	outlet.defaultTimeout = ch.getPacketTimeoutTime(1)

	msg := &messageTest{ID: "id", Data: "test"}
	env, err := ch.TrySend(msg)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if env == nil || env.packet == nil {
		t.Fatalf("expected envelope packet")
	}
	if len(outlet.packets) != 1 {
		t.Fatalf("expected 1 packet")
	}

	// Allow retries to exceed maxTries (default 5).
	time.Sleep(time.Duration(ch.getPacketTimeoutTime(1) * float64(time.Second)))
	time.Sleep(time.Duration(ch.getPacketTimeoutTime(2) * float64(time.Second)))
	time.Sleep(time.Duration(ch.getPacketTimeoutTime(3) * float64(time.Second)))
	time.Sleep(time.Duration(ch.getPacketTimeoutTime(4) * float64(time.Second)))
	time.Sleep(time.Duration(ch.getPacketTimeoutTime(5) * 1.1 * float64(time.Second)))

	if env.tracked {
		t.Fatalf("expected envelope untracked after timeout")
	}
	if outlet.timeoutCallbacks == 0 {
		t.Fatalf("expected at least one outlet timeout callback")
	}
}

func TestChannel_RegisterMessageType_AllowsZeroMsgType(t *testing.T) {
	t.Parallel()

	outlet := &channelOutletTest{mdu: 500, rtt: 0.01, usable: true, linkID: nextTestPacketID()}
	ch := NewChannel(outlet)
	defer ch.Close()

	if err := ch.TryRegisterMessageType(func() MessageBase { return &zeroMsgTypeMessage{} }); err != nil {
		t.Fatalf("TryRegisterMessageType: %v", err)
	}
	env, err := ch.TrySend(&zeroMsgTypeMessage{Data: "zero"})
	if err != nil {
		t.Fatalf("TrySend: %v", err)
	}
	if env == nil || len(env.raw) < 2 {
		t.Fatalf("expected packed envelope")
	}
	if got := binary.BigEndian.Uint16(env.raw[:2]); got != 0 {
		t.Fatalf("msgtype=%d, want 0", got)
	}
}

func TestChannel_Send_PanicsWithChannelException(t *testing.T) {
	t.Parallel()

	outlet := &channelOutletTest{mdu: 500, rtt: 0.01, usable: false, linkID: nextTestPacketID()}
	ch := NewChannel(outlet)
	defer ch.Close()

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected panic")
		}
		cex, ok := rec.(*ChannelException)
		if !ok {
			t.Fatalf("panic type=%T, want *ChannelException", rec)
		}
		if cex.Type != ME_LINK_NOT_READY {
			t.Fatalf("panic type code=%v, want %v", cex.Type, ME_LINK_NOT_READY)
		}
	}()

	_ = ch.Send(&messageTest{ID: "x", Data: "y"})
}
