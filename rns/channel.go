package rns

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sync"
	"time"
)

// ====== system message types ================================================

type SystemMessageTypes uint16

const (
	SMT_STREAM_DATA SystemMessageTypes = 0xff00
)

// ====== base transport =======================================================

type ChannelOutletBase interface {
	Send(raw []byte) any
	Resend(packet any) any

	Mdu() int
	Rtt() float64
	IsUsable() bool

	GetPacketState(packet any) MessageState
	TimedOut()

	String() string

	SetPacketTimeoutCallback(packet any, cb func(any), timeout *float64)
	SetPacketDeliveredCallback(packet any, cb func(any))
	GetPacketID(packet any) any
}

// ====== errors ===============================================================

type CEType int

const (
	ME_NO_MSG_TYPE      CEType = 0
	ME_INVALID_MSG_TYPE CEType = 1
	ME_NOT_REGISTERED   CEType = 2
	ME_LINK_NOT_READY   CEType = 3
	ME_ALREADY_SENT     CEType = 4
	ME_TOO_BIG          CEType = 5
)

type ChannelException struct {
	Type CEType
	Msg  string
}

func (e *ChannelException) Error() string {
	return e.Msg
}

// ====== message state ========================================================

type MessageState int

const (
	MSGSTATE_NEW       MessageState = 0
	MSGSTATE_SENT      MessageState = 1
	MSGSTATE_DELIVERED MessageState = 2
	MSGSTATE_FAILED    MessageState = 3
)

// ====== base message interface ==============================================

type MessageBase interface {
	Pack() ([]byte, error)
	Unpack(raw []byte) error
}

type messageTyper interface {
	MsgType() uint16
}

// ====== message callback =====================================================

type MessageCallbackType func(MessageBase) bool

// ====== Envelope =============================================================

type Envelope struct {
	ts       float64
	id       uintptr
	message  MessageBase
	raw      []byte
	msgType  uint16
	msgSet   bool
	packet   any
	sequence uint16
	outlet   ChannelOutletBase
	tries    int
	unpacked bool
	packed   bool
	tracked  bool
	timeout  float64
}

func (e *Envelope) Unpack(factories map[uint16]func() MessageBase) (MessageBase, error) {
	if len(e.raw) < 6 {
		return nil, errors.New("envelope raw too short")
	}
	msgType := binary.BigEndian.Uint16(e.raw[0:2])
	e.sequence = binary.BigEndian.Uint16(e.raw[2:4])
	// Python stores a length field, but proceeds with the remaining bytes as payload.
	// Keep parity and let message unpackers decide what to do with payload size.
	_ = binary.BigEndian.Uint16(e.raw[4:6])
	raw := e.raw[6:]

	ctor, ok := factories[msgType]
	if !ok {
		return nil, &ChannelException{
			Type: ME_NOT_REGISTERED,
			Msg:  "unable to find constructor for Channel MSGTYPE",
		}
	}
	msg := ctor()
	if err := msg.Unpack(raw); err != nil {
		return nil, err
	}
	e.unpacked = true
	e.message = msg
	return msg, nil
}

func (e *Envelope) Pack() ([]byte, error) {
	if e.message == nil {
		return nil, errors.New("envelope has no message")
	}
	mt := e.msgType
	if !e.msgSet {
		if typer, ok := e.message.(messageTyper); ok {
			mt = typer.MsgType()
			e.msgType = mt
			e.msgSet = true
		} else {
			return nil, errors.New("message type not registered")
		}
	}
	data, err := e.message.Pack()
	if err != nil {
		return nil, err
	}
	if len(data) > 0xffff {
		return nil, &ChannelException{
			Type: ME_TOO_BIG,
			Msg:  "packed message too big (> 65535 bytes)",
		}
	}
	buf := make([]byte, 6+len(data))
	binary.BigEndian.PutUint16(buf[0:2], mt)
	binary.BigEndian.PutUint16(buf[2:4], e.sequence)
	binary.BigEndian.PutUint16(buf[4:6], uint16(len(data)))
	copy(buf[6:], data)
	e.raw = buf
	e.packed = true
	return buf, nil
}

// ====== Channel ==============================================================

type Channel struct {
	outlet ChannelOutletBase

	lock     sync.RWMutex
	nameOnce sync.Once

	txRing []*Envelope
	rxRing []*Envelope

	messageCallbacks []MessageCallbackType
	nextSequence     uint16
	nextRxSequence   uint16

	messageFactories map[uint16]func() MessageBase
	messageTypes     map[reflect.Type]uint16

	maxTries          int
	fastRateRounds    int
	mediumRateRounds  int
	window            int
	windowMax         int
	windowMin         int
	windowFlexibility int

	packetIndex   map[string]*Envelope
	messageStates map[uint16]MessageState
	closed        bool
	nameCache     string
}

func channelLog(c *Channel, level int, format string, args ...any) {
	Log(fmt.Sprintf("%s: %s", c.String(), fmt.Sprintf(format, args...)), level)
}

// window and sequence constants

const (
	Window             = 2
	WindowMin          = 2
	WindowMinLimitSlow = 2
	WindowMinLimitMed  = 5
	WindowMinLimitFast = 16
	WindowMaxSlow      = 5
	WindowMaxMed       = 12
	WindowMaxFast      = 48
	WindowMaxGlobal    = WindowMaxFast
	FastRateThreshold  = 10
	RTT_FAST           = 0.18
	RTT_MEDIUM         = 0.75
	RTT_SLOW           = 1.45
	WindowFlexibility  = 4
	SeqMax             = 0xFFFF
	SeqModulus         = SeqMax + 1
)

// NewChannel mirrors __init__.

func NewChannel(outlet ChannelOutletBase) *Channel {
	c := &Channel{
		outlet:           outlet,
		txRing:           make([]*Envelope, 0),
		rxRing:           make([]*Envelope, 0),
		messageCallbacks: make([]MessageCallbackType, 0),
		messageFactories: make(map[uint16]func() MessageBase),
		messageTypes:     make(map[reflect.Type]uint16),
		maxTries:         5,
		packetIndex:      make(map[string]*Envelope),
		messageStates:    make(map[uint16]MessageState),
	}
	if outlet.Rtt() > RTT_SLOW {
		c.window = 1
		c.windowMax = 1
		c.windowMin = 1
		c.windowFlexibility = 1
	} else {
		c.window = Window
		c.windowMax = WindowMaxSlow
		c.windowMin = WindowMin
		c.windowFlexibility = WindowFlexibility
	}
	return c
}

// Enter mirrors Python Channel.__enter__() and returns the channel itself.
func (c *Channel) Enter() *Channel {
	return c
}

// Exit mirrors Python Channel.__exit__(): it clears channel state and returns
// false so callers do not suppress errors.
func (c *Channel) Exit() bool {
	c.shutdown()
	return false
}

// String returns a channel identifier (mirrors Python __str__).
func (c *Channel) String() string {
	c.nameOnce.Do(func() {
		if c.outlet != nil {
			c.nameCache = fmt.Sprintf("Channel(%s)", c.outlet.String())
		} else {
			c.nameCache = fmt.Sprintf("Channel(%p)", c)
		}
	})
	if c.nameCache == "" {
		return "Channel(<nil>)"
	}
	return c.nameCache
}

func channelClose(c *Channel) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	channelShutdownLocked(c)
}

// shutdownLocked mirrors Python Channel._shutdown(): clears handlers and rings,
// but does not permanently "close" the Channel object.
func channelShutdownLocked(c *Channel) {
	c.messageCallbacks = nil
	c.clearRings()
}

func (c *Channel) shutdown() {
	c.lock.Lock()
	defer c.lock.Unlock()
	channelShutdownLocked(c)
}

// RegisterMessageType is the public registration method.

func (c *Channel) RegisterMessageType(msg any) error {
	return c._register_message_type(msg, false)
}

// _register_message_type is internal.

func (c *Channel) _register_message_type(msg any, isSystemType bool) error {
	return channelRegisterMessageTypeLocked(c, msg, isSystemType, nil)
}

func channelRegisterMessageTypeLocked(c *Channel, msg any, isSystemType bool, explicitType *uint16) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if msg == nil {
		return &ChannelException{Type: ME_INVALID_MSG_TYPE, Msg: "nil message_class"}
	}

	factory, sample, err := channelFactoryFrom(msg)
	if err != nil {
		return err
	}

	mt, ok := messageTypeFrom(sample)
	if explicitType != nil {
		mt = *explicitType
		ok = true
	}
	if !ok {
		return &ChannelException{
			Type: ME_INVALID_MSG_TYPE,
			Msg:  "message type not provided",
		}
	}
	if mt >= 0xf000 && !isSystemType {
		return &ChannelException{
			Type: ME_INVALID_MSG_TYPE,
			Msg:  "system-reserved message type",
		}
	}

	c.messageFactories[mt] = factory
	c.messageTypes[reflect.TypeOf(sample)] = mt
	return nil
}

func messageTypeFrom(msg any) (uint16, bool) {
	typer, ok := msg.(messageTyper)
	if !ok {
		return 0, false
	}
	return typer.MsgType(), true
}

func channelFactoryFrom(msg any) (func() MessageBase, MessageBase, error) {
	msgIface := reflect.TypeOf((*MessageBase)(nil)).Elem()

	switch v := msg.(type) {
	case MessageBase:
		t := reflect.TypeOf(v)
		if t.Kind() != reflect.Ptr {
			return nil, nil, &ChannelException{Type: ME_INVALID_MSG_TYPE, Msg: "message must be pointer type"}
		}
		elem := t.Elem()
		ptrType := reflect.PointerTo(elem)
		if !ptrType.Implements(msgIface) {
			return nil, nil, &ChannelException{Type: ME_INVALID_MSG_TYPE, Msg: "message does not implement MessageBase"}
		}
		factory := func() MessageBase {
			return reflect.New(elem).Interface().(MessageBase)
		}
		sample, err := channelFactorySample(factory)
		return factory, sample, err
	default:
		rv := reflect.ValueOf(msg)
		rt := rv.Type()
		if rt.Kind() != reflect.Func || rt.NumIn() != 0 || rt.NumOut() != 1 {
			return nil, nil, &ChannelException{Type: ME_INVALID_MSG_TYPE, Msg: "message_class must be MessageBase or zero-arg constructor"}
		}
		if !rt.Out(0).Implements(msgIface) {
			return nil, nil, &ChannelException{Type: ME_INVALID_MSG_TYPE, Msg: "message constructor does not return MessageBase"}
		}
		factory := func() MessageBase {
			out := rv.Call(nil)
			if len(out) != 1 || !out[0].IsValid() || out[0].IsNil() {
				panic("message constructor returned nil")
			}
			return out[0].Interface().(MessageBase)
		}
		sample, err := channelFactorySample(factory)
		return factory, sample, err
	}
}

func channelFactorySample(factory func() MessageBase) (sample MessageBase, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &ChannelException{
				Type: ME_INVALID_MSG_TYPE,
				Msg:  fmt.Sprintf("message constructor panicked: %v", r),
			}
		}
	}()
	sample = factory()
	if sample == nil {
		return nil, &ChannelException{Type: ME_INVALID_MSG_TYPE, Msg: "message constructor returned nil"}
	}
	return sample, nil
}

// AddMessageHandler

func (c *Channel) AddMessageHandler(cb MessageCallbackType) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, existing := range c.messageCallbacks {
		if reflect.ValueOf(existing).Pointer() == reflect.ValueOf(cb).Pointer() {
			return
		}
	}
	c.messageCallbacks = append(c.messageCallbacks, cb)
}

// RemoveMessageHandler

func (c *Channel) RemoveMessageHandler(cb MessageCallbackType) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for i, existing := range c.messageCallbacks {
		if reflect.ValueOf(existing).Pointer() == reflect.ValueOf(cb).Pointer() {
			c.messageCallbacks = append(c.messageCallbacks[:i], c.messageCallbacks[i+1:]...)
			break
		}
	}
}

func (c *Channel) clearRings() {
	for _, env := range c.txRing {
		if env.packet != nil {
			c.outlet.SetPacketTimeoutCallback(env.packet, nil, nil)
			c.outlet.SetPacketDeliveredCallback(env.packet, nil)
		}
	}
	c.txRing = nil
	c.rxRing = nil
	c.packetIndex = make(map[string]*Envelope)
	c.messageStates = make(map[uint16]MessageState)
}

func channelPacketKey(c *Channel, packet any) string {
	if packet == nil {
		return ""
	}
	id := c.outlet.GetPacketID(packet)
	if id == nil {
		return fmt.Sprintf("%p", packet)
	}
	// Python parity: packets are tracked by the outlet-provided packet id.
	return fmt.Sprintf("%v", id)
}

func channelTrackPacketLocked(c *Channel, env *Envelope) {
	if key := channelPacketKey(c, env.packet); key != "" {
		c.packetIndex[key] = env
	}
}

func channelUntrackPacketLocked(c *Channel, packet any) {
	if key := channelPacketKey(c, packet); key != "" {
		delete(c.packetIndex, key)
	}
}

func channelSetMessageStateLocked(c *Channel, seq uint16, state MessageState) {
	c.messageStates[seq] = state
}

func channelUpdatePacketTimeout(outlet ChannelOutletBase, packet any, timeout float64) {
	switch outlet.(type) {
	case *LinkChannelOutlet:
		pkt, ok := packet.(*Packet)
		if !ok || pkt == nil || pkt.Receipt == nil {
			return
		}
		pkt.Receipt.SetTimeout(timeout)
	default:
		if timeoutSetter, ok := outlet.(interface {
			SetPacketTimeout(packet any, timeout float64)
		}); ok {
			timeoutSetter.SetPacketTimeout(packet, timeout)
		}
	}
}

// insert envelope into the ring by sequence

func (c *Channel) emplaceEnvelope(env *Envelope, ring *[]*Envelope) bool {
	i := 0
	for _, existing := range *ring {
		if env.sequence == existing.sequence {
			channelLog(c, LOG_EXTREME, "duplicate envelope with sequence %d", env.sequence)
			return false
		}
		if env.sequence < existing.sequence &&
			!(int(c.nextRxSequence-env.sequence) > (SeqMax / 2)) {
			*ring = append((*ring)[:i], append([]*Envelope{env}, (*ring)[i:]...)...)
			env.tracked = true
			return true
		}
		i++
	}
	env.tracked = true
	*ring = append(*ring, env)
	return true
}

func (c *Channel) runCallbacks(msg MessageBase) {
	c.lock.RLock()
	cbs := append([]MessageCallbackType{}, c.messageCallbacks...)
	c.lock.RUnlock()

	for _, cb := range cbs {
		stop := false
		func(cb MessageCallbackType) {
			defer func() {
				if r := recover(); r != nil {
					channelLog(c, LOG_ERROR, "Channel %s experienced an error while running a message callback. The contained exception was: %v", c, r)
				}
			}()
			if cb(msg) {
				stop = true
			}
		}(cb)
		if stop {
			break
		}
	}
}

// Receive mirrors _receive(self, raw).

func (c *Channel) Receive(raw []byte) {
	defer func() {
		if r := recover(); r != nil {
			channelLog(c, LOG_ERROR, "An error ocurred while receiving data on %s. The contained exception was: %v", c, r)
		}
	}()

	env := &Envelope{
		ts:     float64(time.Now().UnixNano()) / 1e9,
		id:     reflect.ValueOf(&struct{}{}).Pointer(),
		raw:    raw,
		outlet: c.outlet,
	}

	c.lock.Lock()
	if _, err := env.Unpack(c.messageFactories); err != nil {
		c.lock.Unlock()
		channelLog(c, LOG_ERROR, "error unpacking envelope: %v", err)
		return
	}

	if env.sequence < c.nextRxSequence {
		windowOverflow := uint16(int(c.nextRxSequence+WindowMaxGlobal) % SeqModulus)
		if windowOverflow < c.nextRxSequence {
			if env.sequence > windowOverflow {
				c.lock.Unlock()
				channelLog(c, LOG_EXTREME, "Invalid packet sequence (%d) received on channel %s", env.sequence, c)
				return
			}
		} else {
			c.lock.Unlock()
			channelLog(c, LOG_EXTREME, "Invalid packet sequence (%d) received on channel %s", env.sequence, c)
			return
		}
	}

	isNew := c.emplaceEnvelope(env, &c.rxRing)
	if !isNew {
		c.lock.Unlock()
		channelLog(c, LOG_EXTREME, "Duplicate message received on channel %s", c)
		return
	}

	contiguous := make([]*Envelope, 0)
	for {
		found := false
		for _, e := range c.rxRing {
			if e.sequence == c.nextRxSequence {
				contiguous = append(contiguous, e)
				c.nextRxSequence = uint16((int(c.nextRxSequence) + 1) % SeqModulus)
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	messages := make([]MessageBase, 0, len(contiguous))
	for _, e := range contiguous {
		var (
			msg MessageBase
			err error
		)
		if !e.unpacked {
			msg, err = e.Unpack(c.messageFactories)
			if err != nil {
				channelLog(c, LOG_ERROR, "error unpacking queued envelope: %v", err)
				continue
			}
		} else {
			msg = e.message
		}

		for i, re := range c.rxRing {
			if re == e {
				c.rxRing = append(c.rxRing[:i], c.rxRing[i+1:]...)
				break
			}
		}
		messages = append(messages, msg)
	}
	c.lock.Unlock()

	for _, m := range messages {
		if m != nil {
			c.runCallbacks(m)
		}
	}
}

// IsReadyToSend mirrors is_ready_to_send().

func (c *Channel) IsReadyToSend() bool {
	if !c.outlet.IsUsable() {
		return false
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	return channelIsReadyToSendLocked(c)
}

func channelIsReadyToSendLocked(c *Channel) bool {
	outstanding := 0
	for _, env := range c.txRing {
		if env.outlet == c.outlet {
			state := c.outlet.GetPacketState(env.packet)
			if env.packet == nil || state != MSGSTATE_DELIVERED {
				outstanding++
			}
		}
	}
	return outstanding < c.window
}

// ====== timeouts/delivery ====================================================

func (c *Channel) packetTxOp(packet any, op func(*Envelope) (bool, *MessageState)) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return
	}

	key := channelPacketKey(c, packet)
	env := c.packetIndex[key]
	if env == nil {
		channelLog(c, LOG_EXTREME, "Spurious message received on %s", c)
		return
	}

	remove, newState := op(env)
	if !remove {
		return
	}

	channelUntrackPacketLocked(c, packet)
	env.tracked = false

	found := false
	for i, e := range c.txRing {
		if e == env {
			c.txRing = append(c.txRing[:i], c.txRing[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		channelLog(c, LOG_EXTREME, "Envelope not found in TX ring for %s", c)
		return
	}

	if newState != nil {
		channelSetMessageStateLocked(c, env.sequence, *newState)
	}
	delete(c.messageStates, env.sequence)

	if c.window < c.windowMax {
		c.window++
	}

	rtt := c.outlet.Rtt()
	if rtt != 0 {
		if rtt > RTT_FAST {
			c.fastRateRounds = 0
			if rtt > RTT_MEDIUM {
				c.mediumRateRounds = 0
			} else {
				c.mediumRateRounds++
				if c.windowMax < WindowMaxMed && c.mediumRateRounds == FastRateThreshold {
					c.windowMax = WindowMaxMed
					c.windowMin = WindowMinLimitMed
				}
			}
		} else {
			c.fastRateRounds++
			if c.windowMax < WindowMaxFast && c.fastRateRounds == FastRateThreshold {
				c.windowMax = WindowMaxFast
				c.windowMin = WindowMinLimitFast
			}
		}
	}
}

func (c *Channel) packetDelivered(packet any) {
	c.packetTxOp(packet, func(env *Envelope) (bool, *MessageState) {
		state := MSGSTATE_DELIVERED
		return true, &state
	})
}

func (c *Channel) getPacketTimeoutTime(tries int) float64 {
	rtt := c.outlet.Rtt()
	base := rtt * 2.5
	if base < 0.025 {
		base = 0.025
	}
	return math.Pow(1.5, float64(tries-1)) * base * (float64(len(c.txRing)) + 1.5)
}

func (c *Channel) updatePacketTimeouts() {
	// Python parity: Channel._update_packet_timeouts() only increases receipt.timeout
	// and must not replace timeout callbacks.
	for _, env := range c.txRing {
		to := c.getPacketTimeoutTime(env.tries)
		if env.packet != nil && (env.timeout == 0 || to > env.timeout) {
			env.timeout = to
			channelUpdatePacketTimeout(c.outlet, env.packet, to)
		}
	}
}

func (c *Channel) packetTimeout(packet any) {
	if c.outlet.GetPacketState(packet) == MSGSTATE_DELIVERED {
		return
	}

	var tearDown bool
	retryEnv := func(env *Envelope) (bool, *MessageState) {
		if env.tries >= c.maxTries {
			tearDown = true
			state := MSGSTATE_FAILED
			return true, &state
		}

		env.tries++
		// Python order: resend first, then update callbacks/timeouts.
		c.outlet.Resend(env.packet)
		c.outlet.SetPacketDeliveredCallback(env.packet, c.packetDelivered)
		timeout := c.getPacketTimeoutTime(env.tries)
		env.timeout = timeout
		c.outlet.SetPacketTimeoutCallback(env.packet, c.packetTimeout, &timeout)
		c.updatePacketTimeouts()

		if c.window > c.windowMin {
			c.window--
			if c.windowMax > (c.windowMin + c.windowFlexibility) {
				c.windowMax--
			}
		}
		return false, nil
	}

	c.packetTxOp(packet, retryEnv)

	if tearDown {
		channelLog(c, LOG_ERROR, "Retry count exceeded on %s, tearing down Link.", c)
		// Python parity: Channel._shutdown() is called, but the object is not "closed"
		// (it can still exist; handlers are just cleared).
		c.shutdown()
		c.outlet.TimedOut()
	}
}

// ====== Send =================================================================

func (c *Channel) Send(message MessageBase) (*Envelope, error) {
	c.lock.Lock()
	if c.closed {
		c.lock.Unlock()
		return nil, &ChannelException{Type: ME_LINK_NOT_READY, Msg: "channel closed"}
	}
	if !c.outlet.IsUsable() || !channelIsReadyToSendLocked(c) {
		c.lock.Unlock()
		return nil, &ChannelException{Type: ME_LINK_NOT_READY, Msg: "link is not ready"}
	}

	seq := c.nextSequence
	c.nextSequence = uint16((int(c.nextSequence) + 1) % SeqModulus)
	msgType, ok := c.messageTypes[reflect.TypeOf(message)]
	if !ok {
		if mt, ok := messageTypeFrom(message); ok {
			msgType = mt
		} else {
			c.lock.Unlock()
			return nil, &ChannelException{Type: ME_INVALID_MSG_TYPE, Msg: "message type not registered"}
		}
	}
	env := &Envelope{
		ts:       float64(time.Now().UnixNano()) / 1e9,
		id:       reflect.ValueOf(&struct{}{}).Pointer(),
		message:  message,
		msgType:  msgType,
		msgSet:   true,
		sequence: seq,
		outlet:   c.outlet,
	}
	c.emplaceEnvelope(env, &c.txRing)
	channelSetMessageStateLocked(c, seq, MSGSTATE_NEW)
	c.lock.Unlock()

	if _, err := env.Pack(); err != nil {
		return nil, err
	}
	if len(env.raw) > c.outlet.Mdu() {
		channelLog(c, LOG_WARNING, "packed message exceeds outlet MDU (%d > %d)", len(env.raw), c.outlet.Mdu())
		return nil, &ChannelException{
			Type: ME_TOO_BIG,
			Msg:  "packed message too big for packet",
		}
	}

	env.packet = c.outlet.Send(env.raw)
	c.lock.Lock()
	channelTrackPacketLocked(c, env)
	channelSetMessageStateLocked(c, env.sequence, MSGSTATE_SENT)
	c.lock.Unlock()
	env.tries++
	to := c.getPacketTimeoutTime(env.tries)
	env.timeout = to
	c.outlet.SetPacketDeliveredCallback(env.packet, c.packetDelivered)
	c.outlet.SetPacketTimeoutCallback(env.packet, c.packetTimeout, &to)
	c.updatePacketTimeouts()

	return env, nil
}

// Mdu mirrors @property mdu.

func (c *Channel) Mdu() int {
	mdu := c.outlet.Mdu() - 6
	if mdu > 0xFFFF {
		mdu = 0xFFFF
	}
	return mdu
}

// ====== LinkChannelOutlet ====================================================

type LinkChannelOutlet struct {
	link *Link
}

func NewLinkChannelOutlet(link *Link) *LinkChannelOutlet {
	return &LinkChannelOutlet{link: link}
}

func (o *LinkChannelOutlet) Send(raw []byte) any {
	if o.link == nil {
		return nil
	}

	// Python parity: create the packet, but only send if link is ACTIVE.
	o.link.mu.Lock()
	status := o.link.Status
	o.link.mu.Unlock()

	now := time.Now()
	o.link.mu.Lock()
	o.link.lastOutbound = now
	o.link.lastData = now
	o.link.mu.Unlock()
	packet := NewPacket(
		o.link,
		raw,
		PacketTypeData,
		PacketCtxChannel,
		Broadcast,
		HeaderType1,
		nil,
		nil,
		true,
		FlagUnset,
	)

	if status == LinkActive {
		if receipt := packet.Send(); receipt != nil {
			packet.Receipt = receipt
		} else if packet.CreateReceipt && packet.Receipt == nil {
			packet.Receipt = NewPacketReceipt(packet)
		}
	}

	return packet
}

func (o *LinkChannelOutlet) Resend(packet any) any {
	pkt, ok := packet.(*Packet)
	if !ok || pkt == nil {
		return nil
	}
	if pkt.Receipt == nil && pkt.CreateReceipt {
		pkt.Receipt = NewPacketReceipt(pkt)
	}
	receipt := pkt.Resend()
	if receipt == nil && pkt.CreateReceipt {
		Log("Failed to resend packet", LOG_ERROR)
	}
	return pkt
}

func (o *LinkChannelOutlet) Mdu() int {
	if o.link == nil {
		return 0
	}
	return o.link.MDU
}

func (o *LinkChannelOutlet) Rtt() float64 {
	if o.link == nil {
		return 0
	}
	return o.link.RTT.Seconds()
}

func (o *LinkChannelOutlet) IsUsable() bool {
	return true
}

func (o *LinkChannelOutlet) GetPacketState(packet any) MessageState {
	pkt, ok := packet.(*Packet)
	if !ok || pkt == nil || pkt.Receipt == nil {
		return MSGSTATE_FAILED
	}
	switch pkt.Receipt.Status {
	case PacketReceiptSENT:
		return MSGSTATE_SENT
	case PacketReceiptDELIVERED:
		return MSGSTATE_DELIVERED
	case PacketReceiptFAILED:
		return MSGSTATE_FAILED
	default:
		return MSGSTATE_FAILED
	}
}

func (o *LinkChannelOutlet) TimedOut() {
	if o.link != nil {
		o.link.Teardown()
	}
}

func (o *LinkChannelOutlet) String() string {
	if o.link == nil {
		return "<LinkChannelOutlet nil>"
	}
	return fmt.Sprintf("<LinkChannelOutlet %s>", o.link.String())
}

func (o *LinkChannelOutlet) SetPacketTimeoutCallback(packet any, cb func(any), timeout *float64) {
	pkt, ok := packet.(*Packet)
	if !ok || pkt == nil || pkt.Receipt == nil {
		return
	}
	if timeout != nil {
		pkt.Receipt.SetTimeout(*timeout)
	}
	if cb == nil {
		pkt.Receipt.SetTimeoutCallback(nil)
		return
	}
	pkt.Receipt.SetTimeoutCallback(func(*PacketReceipt) {
		cb(pkt)
	})
}

func (o *LinkChannelOutlet) SetPacketDeliveredCallback(packet any, cb func(any)) {
	pkt, ok := packet.(*Packet)
	if !ok || pkt == nil || pkt.Receipt == nil {
		return
	}
	if cb == nil {
		pkt.Receipt.SetDeliveryCallback(nil)
		return
	}
	pkt.Receipt.SetDeliveryCallback(func(*PacketReceipt) {
		cb(pkt)
	})
}

func (o *LinkChannelOutlet) GetPacketID(packet any) any {
	pkt, ok := packet.(*Packet)
	if !ok || pkt == nil {
		return nil
	}
	return pkt.GetHash()
}
