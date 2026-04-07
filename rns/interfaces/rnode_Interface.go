package interfaces

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	serial "go.bug.st/serial"
)

type Logger interface {
	Debugf(fmt string, args ...any)
	Infof(fmt string, args ...any)
	Warnf(fmt string, args ...any)
	Errorf(fmt string, args ...any)
}

type Owner interface {
	Inbound(data []byte, iface *RNodeInterface)
}

const (
	FEND  = 0xC0
	FESC  = 0xDB
	TFEND = 0xDC
	TFESC = 0xDD

	CMD_UNKNOWN     = 0xFE
	CMD_DATA        = 0x00
	CMD_FREQUENCY   = 0x01
	CMD_BANDWIDTH   = 0x02
	CMD_TXPOWER     = 0x03
	CMD_SF          = 0x04
	CMD_CR          = 0x05
	CMD_RADIO_STATE = 0x06
	CMD_RADIO_LOCK  = 0x07
	CMD_DETECT      = 0x08
	CMD_LEAVE       = 0x0A
	CMD_ST_ALOCK    = 0x0B
	CMD_LT_ALOCK    = 0x0C
	CMD_READY       = 0x0F

	CMD_STAT_RX     = 0x21
	CMD_STAT_TX     = 0x22
	CMD_STAT_RSSI   = 0x23
	CMD_STAT_SNR    = 0x24
	CMD_STAT_CHTM   = 0x25
	CMD_STAT_PHYPRM = 0x26
	CMD_STAT_BAT    = 0x27
	CMD_STAT_CSMA   = 0x28
	CMD_STAT_TEMP   = 0x29
	CMD_BLINK       = 0x30

	CMD_RANDOM     = 0x40
	CMD_FB_EXT     = 0x41
	CMD_FB_READ    = 0x42
	CMD_FB_WRITE   = 0x43
	CMD_DISP_READ  = 0x66
	CMD_BT_CTRL    = 0x46
	CMD_PLATFORM   = 0x48
	CMD_MCU        = 0x49
	CMD_FW_VERSION = 0x50
	CMD_ROM_READ   = 0x51
	CMD_RESET      = 0x55
	CMD_ERROR      = 0x90

	DETECT_REQ  = 0x73
	DETECT_RESP = 0x46

	RADIO_STATE_OFF = 0x00
	RADIO_STATE_ON  = 0x01
	RADIO_STATE_ASK = 0xFF
)

const (
	FREQ_MIN                uint32 = 137000000
	FREQ_MAX                uint32 = 3000000000
	RSSI_OFFSET                    = 157
	REQUIRED_FW_VER_MAJ            = 1
	REQUIRED_FW_VER_MIN            = 52
	BLE_DETECT_TIMEOUT             = 5 * time.Second
	TCP_DETECT_TIMEOUT             = 5 * time.Second
	DETECTION_POLL_INTERVAL        = 100 * time.Millisecond
	MAX_CHUNK_LEN                  = 1024 * 32 // Python MAX_CHUNK = 32768
	CALLSIGN_MAX_LEN               = 32

	QSNRMinBase = -9.0
	QSNRMax     = 6.0
	QSNRStep    = 2.0

	DisplayReadIntervalDefault = 1.0

	TCPActivityTimeout   = 6.0
	TCPActivityKeepalive = TCPActivityTimeout - 2.5
)

var rnodeStartReadLoop func(r *RNodeInterface)

func init() {
	rnodeStartReadLoop = func(r *RNodeInterface) { go r.readLoop() }
}

const (
	ErrorInitRadio     = 0x01
	ErrorTXFailed      = 0x02
	ErrorEEPROMLocked  = 0x03
	ErrorQueueFull     = 0x04
	ErrorMemoryLow     = 0x05
	ErrorModemTimeout  = 0x06
	PlatformAVR        = 0x90
	PlatformESP32      = 0x80
	PlatformNRF52      = 0x70
	BatteryUnknown     = 0x00
	BatteryDischarging = 0x01
	BatteryCharging    = 0x02
	BatteryCharged     = 0x03
)

func kissEscape(in []byte) []byte {
	// Like Python: 0xDB->0xDB 0xDD, 0xC0->0xDB 0xDC
	out := make([]byte, 0, len(in)+8)
	for _, b := range in {
		switch b {
		case FESC:
			out = append(out, FESC, TFESC)
		case FEND:
			out = append(out, FESC, TFEND)
		default:
			out = append(out, b)
		}
	}
	return out
}

func kissUnescapeStream(dst *bytes.Buffer, b byte, esc *bool) {
	if b == FESC {
		*esc = true
		return
	}
	if *esc {
		switch b {
		case TFEND:
			dst.WriteByte(FEND)
		case TFESC:
			dst.WriteByte(FESC)
		default:
			dst.WriteByte(b) // "as-is", just in case
		}
		*esc = false
		return
	}
	dst.WriteByte(b)
}

type Transport interface {
	Open(ctx context.Context) error
	Close() error
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	IsOpen() bool
	String() string
}

// ---------- Serial transport ----------
type SerialTransport struct {
	PortName string
	Baud     int
	mu       sync.Mutex
	p        serial.Port
}

func (s *SerialTransport) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.p != nil {
		return nil
	}
	mode := &serial.Mode{BaudRate: s.Baud}
	p, err := serial.Open(s.PortName, mode)
	if err != nil {
		return err
	}
	s.p = p
	return nil
}
func (s *SerialTransport) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.p == nil {
		return nil
	}
	err := s.p.Close()
	s.p = nil
	return err
}
func (s *SerialTransport) Read(p []byte) (int, error) {
	s.mu.Lock()
	pp := s.p
	s.mu.Unlock()
	if pp == nil {
		return 0, io.EOF
	}
	return pp.Read(p)
}
func (s *SerialTransport) Write(p []byte) (int, error) {
	s.mu.Lock()
	pp := s.p
	s.mu.Unlock()
	if pp == nil {
		return 0, io.EOF
	}
	return pp.Write(p)
}
func (s *SerialTransport) IsOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.p != nil
}
func (s *SerialTransport) String() string { return "serial://" + s.PortName }

func (s *SerialTransport) SetReadTimeout(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.p == nil {
		return
	}
	_ = s.p.SetReadTimeout(d)
}

// ---------- TCP transport ----------
type TCPTransport struct {
	HostPort string
	DialTO   time.Duration

	mu   sync.Mutex
	conn net.Conn

	connected      atomic.Bool
	mustDisconnect atomic.Bool

	rxMu      sync.Mutex
	rxQueue   []byte
	txMu      sync.Mutex
	txQueue   []byte
	lastWrite atomic.Int64 // unixnano
}

func (t *TCPTransport) Open(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn != nil {
		return nil
	}
	d := net.Dialer{Timeout: t.DialTO}
	c, err := d.DialContext(ctx, "tcp", t.HostPort)
	if err != nil {
		return err
	}
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
		_ = tc.SetKeepAlive(true)
		_ = tc.SetKeepAlivePeriod(5 * time.Second)
		_ = setTCPPlatformOptions(tc)
	}
	t.conn = c
	t.mustDisconnect.Store(false)
	t.connected.Store(true)

	go t.readLoop()
	return nil
}
func (t *TCPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil
	}
	t.mustDisconnect.Store(true)
	err := t.conn.Close()
	t.conn = nil
	t.connected.Store(false)
	return err
}
func (t *TCPTransport) Read(p []byte) (int, error) {
	t.rxMu.Lock()
	defer t.rxMu.Unlock()
	if len(t.rxQueue) == 0 {
		if !t.connected.Load() {
			return 0, io.EOF
		}
		return 0, nil
	}
	n := len(p)
	if n > len(t.rxQueue) {
		n = len(t.rxQueue)
	}
	copy(p[:n], t.rxQueue[:n])
	t.rxQueue = t.rxQueue[n:]
	return n, nil
}
func (t *TCPTransport) Write(p []byte) (int, error) {
	// Mirror Python semantics: accept writes even if disconnected by queueing.
	if len(p) == 0 {
		return 0, nil
	}

	t.mu.Lock()
	c := t.conn
	t.mu.Unlock()

	if c == nil || !t.connected.Load() {
		t.txMu.Lock()
		t.txQueue = append(t.txQueue, p...)
		t.txMu.Unlock()
		return len(p), nil
	}

	// Flush pending
	t.txMu.Lock()
	pending := t.txQueue
	t.txQueue = nil
	t.txMu.Unlock()
	if len(pending) > 0 {
		if _, err := c.Write(pending); err != nil {
			// re-queue on failure
			t.txMu.Lock()
			t.txQueue = append(pending, t.txQueue...)
			t.txMu.Unlock()
		}
	}

	_, err := c.Write(p)
	if err == nil {
		t.lastWrite.Store(time.Now().UnixNano())
	}
	return len(p), err
}
func (t *TCPTransport) IsOpen() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn != nil
}
func (t *TCPTransport) String() string { return "tcp://" + t.HostPort }

func (t *TCPTransport) SetReadDeadline(deadline time.Time) {
	t.mu.Lock()
	c := t.conn
	t.mu.Unlock()
	if c == nil {
		return
	}
	_ = c.SetReadDeadline(deadline)
}

func (t *TCPTransport) readLoop() {
	buf := make([]byte, 4096)
	for {
		if t.mustDisconnect.Load() {
			return
		}
		t.mu.Lock()
		c := t.conn
		t.mu.Unlock()
		if c == nil {
			return
		}
		n, err := c.Read(buf)
		if err != nil {
			t.connected.Store(false)
			return
		}
		if n > 0 {
			t.rxMu.Lock()
			t.rxQueue = append(t.rxQueue, buf[:n]...)
			t.rxMu.Unlock()
		}
	}
}

// ---------- RNode core ----------
type RNodeInterface struct {
	Owner Owner
	Log   Logger
	Name  string

	// config
	Frequency uint32
	Bandwidth uint32
	TXPower   byte
	SF        byte
	CR        byte
	FlowCtrl  bool

	HWMTU          int
	ReadTimeout    time.Duration
	ReconnectDelay time.Duration

	tr Transport

	online            atomic.Bool
	ready             atomic.Bool
	detected          atomic.Bool
	fwMaj             atomic.Uint32
	fwMin             atomic.Uint32
	radioState        atomic.Uint32 // reported state (CMD_RADIO_STATE)
	desiredRadioState atomic.Uint32 // configured/desired state

	// reported params
	rFreq    uint32
	rBW      uint32
	rTXP     byte
	rSF      byte
	rCR      byte
	repFreq  atomic.Bool
	repBW    atomic.Bool
	repTXP   atomic.Bool
	repSF    atomic.Bool
	repCR    atomic.Bool
	repState atomic.Bool

	// stats (partial)
	rssi        int32
	snr         float32
	qSNR        float32
	bitrate     float64
	repStatRSSI atomic.Bool
	repStatSNR  atomic.Bool

	rStatRX uint32
	rStatTX uint32

	rSTAirtimeLimit float32
	rLTAirtimeLimit float32

	rAirtimeShort     float32
	rAirtimeLong      float32
	rChannelLoadShort float32
	rChannelLoadLong  float32
	rCurrentRSSI      int32
	rNoiseFloor       int32
	rInterference     *int32
	rInterferenceAt   time.Time
	rSymbolTimeMS     float32
	rSymbolRate       uint32
	rPreambleSymbols  uint32
	rPreambleTimeMS   uint32
	rCSMASlotTimeMS   uint32
	rCSMADIFSMS       uint32
	rCSMACWBand       byte
	rCSMACWMin        byte
	rCSMACWMax        byte
	batteryState      byte
	batteryPercent    byte
	temperatureC      *int32
	randomByte        byte
	platform          byte
	mcu               byte

	shouldReadDisplay   atomic.Bool
	readDisplayInterval atomic.Value // float64
	framebuffer         []byte
	framebufferReadAt   time.Time
	framebufferLatency  time.Duration
	displayBuf          []byte
	displayReadAt       time.Time
	displayLatency      time.Duration

	idInterval time.Duration
	idCallsign []byte
	firstTxAt  atomic.Int64 // unixnano

	lastWriteAt atomic.Int64 // unixnano

	displayCapable atomic.Bool

	rxb atomic.Uint64
	txb atomic.Uint64

	detached   atomic.Bool
	hwErrorsMu sync.Mutex
	hwErrors   []HardwareError

	// queues
	qMu   sync.Mutex
	queue [][]byte

	writeMu  sync.Mutex
	stopCh   chan struct{}
	validCfg atomic.Bool

	ShortAirtimeLimit float64
	LongAirtimeLimit  float64
}

type HardwareError struct {
	Code        byte
	Description string
	At          time.Time
}

func NewRNodeInterface(owner Owner, log Logger, name string, tr Transport) *RNodeInterface {
	return &RNodeInterface{
		Owner:          owner,
		Log:            log,
		Name:           name,
		tr:             tr,
		HWMTU:          508,
		ReadTimeout:    100 * time.Millisecond,
		ReconnectDelay: 5 * time.Second,
		stopCh:         make(chan struct{}),
	}
}

func (r *RNodeInterface) String() string { return fmt.Sprintf("RNodeInterface[%s]", r.Name) }

func NewRNodeInterfaceFromPort(owner Owner, log Logger, name string, port string) (*RNodeInterface, error) {
	tr, err := transportFromPort(port)
	if err != nil {
		return nil, err
	}
	return NewRNodeInterface(owner, log, name, tr), nil
}

func transportFromPort(port string) (Transport, error) {
	if port == "" {
		return nil, errors.New("no port specified for RNode interface")
	}
	lower := strings.ToLower(port)
	switch {
	case strings.HasPrefix(lower, "tcp://"):
		host := strings.TrimPrefix(port, "tcp://")
		if host == "" {
			return nil, errors.New("tcp:// missing host")
		}
		// Python uses default 7633 if no port specified.
		if !strings.Contains(host, ":") {
			host = host + ":7633"
		}
		return &TCPTransport{HostPort: host, DialTO: 5 * time.Second}, nil
	case strings.HasPrefix(lower, "ble://"):
		target := strings.TrimPrefix(port, "ble://")
		return newBLETransport(target)
	default:
		return &SerialTransport{PortName: port, Baud: 115200}, nil
	}
}

func (r *RNodeInterface) Start(ctx context.Context) error {
	if !r.ValidateConfig() {
		return errors.New("invalid configuration")
	}
	if err := r.tr.Open(ctx); err != nil {
		return err
	}
	r.online.Store(false)
	r.ready.Store(false)

	r.readDisplayInterval.Store(DisplayReadIntervalDefault)

	// Ensure the transport doesn't block the read loop forever.
	if st, ok := r.tr.(*SerialTransport); ok {
		st.SetReadTimeout(80 * time.Millisecond)
	}
	_ = r.ConfigureDevice()
	return nil
}

func (r *RNodeInterface) Stop() error {
	select {
	case <-r.stopCh:
	default:
		close(r.stopCh)
	}
	r.online.Store(false)
	return r.tr.Close()
}

func (r *RNodeInterface) ResetRadioState() {
	r.rFreq = 0
	r.rBW = 0
	r.rTXP = 0
	r.rSF = 0
	r.rCR = 0
	r.radioState.Store(0)
	r.desiredRadioState.Store(0)
	r.repFreq.Store(false)
	r.repBW.Store(false)
	r.repTXP.Store(false)
	r.repSF.Store(false)
	r.repCR.Store(false)
	r.repState.Store(false)
	r.repStatRSSI.Store(false)
	r.repStatSNR.Store(false)
	r.detected.Store(false)
	r.platform = 0
	r.mcu = 0
	r.randomByte = 0
	r.displayCapable.Store(false)
}

func (r *RNodeInterface) ConfigureDevice() error {
	// Python: reset_radio_state(); sleep(2.0); start readLoop; detect; wait; initRadio; validateRadioState; online=True
	r.ResetRadioState()
	time.Sleep(2 * time.Second)
	rnodeStartReadLoop(r)

	if err := r.Detect(); err != nil {
		return err
	}

	detectTimeout := 200 * time.Millisecond
	switch {
	case strings.HasPrefix(r.tr.String(), "ble://"):
		detectTimeout = BLE_DETECT_TIMEOUT
	case strings.HasPrefix(r.tr.String(), "tcp://"):
		detectTimeout = TCP_DETECT_TIMEOUT
	default:
		detectTimeout = 200 * time.Millisecond
	}

	if !r.waitForDetection(detectTimeout) {
		if r.Log != nil {
			r.Log.Errorf("%s could not detect device", r.String())
		}
		_ = r.tr.Close()
		return errors.New("device not detected")
	}

	if r.platform == PlatformESP32 || r.platform == PlatformNRF52 {
		// Python sets display capability on these.
		r.displayCapable.Store(true)
	}

	if r.Log != nil {
		r.Log.Infof("%s configuring RNode interface...", r.String())
	}
	if err := r.ConfigureRadio(); err != nil {
		_ = r.tr.Close()
		return err
	}

	if !r.validateRadioState() {
		_ = r.tr.Close()
		return errors.New("radio state validation failed")
	}

	r.ready.Store(true)
	r.online.Store(true)
	return nil
}

func (r *RNodeInterface) ValidateConfig() bool {
	valid := true
	if r.Frequency < FREQ_MIN || r.Frequency > FREQ_MAX {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s configured frequency %d outside valid range [%d,%d]", r.String(), r.Frequency, FREQ_MIN, FREQ_MAX)
		}
	}
	if r.TXPower > 37 {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s configured txpower %d out of range", r.String(), r.TXPower)
		}
	}
	if r.Bandwidth < 7800 || r.Bandwidth > 1625000 {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s configured bandwidth %d out of range", r.String(), r.Bandwidth)
		}
	}
	if r.SF < 5 || r.SF > 12 {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s configured spreading factor %d out of range", r.String(), r.SF)
		}
	}
	if r.CR < 5 || r.CR > 8 {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s configured coding rate %d out of range", r.String(), r.CR)
		}
	}
	if r.ShortAirtimeLimit < 0.0 || r.ShortAirtimeLimit > 100.0 {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s configured short-term airtime limit %.2f out of range", r.String(), r.ShortAirtimeLimit)
		}
	}
	if r.LongAirtimeLimit < 0.0 || r.LongAirtimeLimit > 100.0 {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s configured long-term airtime limit %.2f out of range", r.String(), r.LongAirtimeLimit)
		}
	}
	r.validCfg.Store(valid)
	return valid
}

func (r *RNodeInterface) ConfigureRadio() error {
	if err := r.SetFrequency(r.Frequency); err != nil {
		return err
	}
	if err := r.SetBandwidth(r.Bandwidth); err != nil {
		return err
	}
	if err := r.SetTXPower(r.TXPower); err != nil {
		return err
	}
	if err := r.SetSF(r.SF); err != nil {
		return err
	}
	if err := r.SetCR(r.CR); err != nil {
		return err
	}
	if err := r.SetSTALock(); err != nil {
		return err
	}
	if err := r.SetLTALock(); err != nil {
		return err
	}
	if err := r.SetRadioState(RADIO_STATE_ON); err != nil {
		return err
	}
	if err := r.ValidateFirmware(); err != nil {
		return err
	}
	r.ready.Store(true)
	return nil
}

func (r *RNodeInterface) SetSTALock() error {
	return r.setAirtimeLock(CMD_ST_ALOCK, r.ShortAirtimeLimit)
}

func (r *RNodeInterface) SetLTALock() error {
	return r.setAirtimeLock(CMD_LT_ALOCK, r.LongAirtimeLimit)
}

func (r *RNodeInterface) setAirtimeLock(cmd byte, pct float64) error {
	if pct <= 0 {
		return nil
	}
	at := int(math.Round(pct * 100.0))
	if at < 0 {
		at = 0
	}
	if at > 0xFFFF {
		at = 0xFFFF
	}
	data := []byte{byte(at >> 8), byte(at)}
	return r.sendCmd(cmd, data)
}

// -------- outgoing ----------
func (r *RNodeInterface) SendData(payload []byte) error {
	if !r.online.Load() {
		return errors.New("offline")
	}

	if r.FlowCtrl && !r.ready.Load() {
		r.enqueue(payload)
		return nil
	}

	if r.FlowCtrl {
		r.ready.Store(false)
	}

	// Track first TX and ID logic as in Python.
	if len(r.idCallsign) > 0 && bytes.Equal(payload, r.idCallsign) {
		r.firstTxAt.Store(0)
	} else if r.firstTxAt.Load() == 0 {
		r.firstTxAt.Store(time.Now().UnixNano())
	}

	frame := r.makeKISSFrame(CMD_DATA, payload)

	r.writeMu.Lock()
	_, err := r.tr.Write(frame)
	r.writeMu.Unlock()
	r.lastWriteAt.Store(time.Now().UnixNano())
	r.txb.Add(uint64(len(payload)))

	return err
}

func (r *RNodeInterface) enqueue(p []byte) {
	r.qMu.Lock()
	defer r.qMu.Unlock()
	cp := append([]byte(nil), p...)
	r.queue = append(r.queue, cp)
}

func (r *RNodeInterface) flushOne() {
	r.qMu.Lock()
	if len(r.queue) == 0 {
		r.qMu.Unlock()
		r.ready.Store(true)
		return
	}
	p := r.queue[0]
	r.queue = r.queue[1:]
	r.qMu.Unlock()

	r.ready.Store(true)
	_ = r.SendData(p)
}

func (r *RNodeInterface) makeKISSFrame(cmd byte, data []byte) []byte {
	esc := kissEscape(data)
	out := make([]byte, 0, 2+len(esc)+2)
	out = append(out, FEND)
	out = append(out, cmd)
	out = append(out, esc...)
	out = append(out, FEND)
	return out
}

// -------- commands ----------
func (r *RNodeInterface) Detect() error {
	// Like Python: FEND, CMD_DETECT, DETECT_REQ, FEND, CMD_FW_VERSION, 0x00, FEND, CMD_PLATFORM, 0x00, FEND, CMD_MCU, 0x00, FEND
	buf := []byte{FEND, CMD_DETECT, DETECT_REQ, FEND, CMD_FW_VERSION, 0x00, FEND, CMD_PLATFORM, 0x00, FEND, CMD_MCU, 0x00, FEND}
	r.detected.Store(false)
	r.writeMu.Lock()
	_, err := r.tr.Write(buf)
	r.writeMu.Unlock()
	r.lastWriteAt.Store(time.Now().UnixNano())
	return err
}

func (r *RNodeInterface) Leave() error {
	return r.sendCmd(CMD_LEAVE, []byte{0xFF})
}

func (r *RNodeInterface) EnableExternalFramebuffer() error {
	return r.sendCmd(CMD_FB_EXT, []byte{0x01})
}

func (r *RNodeInterface) DisableExternalFramebuffer() error {
	return r.sendCmd(CMD_FB_EXT, []byte{0x00})
}

func (r *RNodeInterface) HardReset() error {
	// Python: [FEND,CMD_RESET,0xF8,FEND], then sleep 2.25s
	if err := r.sendCmd(CMD_RESET, []byte{0xF8}); err != nil {
		return err
	}
	time.Sleep(2250 * time.Millisecond)
	return nil
}

const (
	fbPixelWidth    = 64
	fbBytesPerLine  = fbPixelWidth / 8
	framebufferSize = 512
	displaySize     = 1024
)

func (r *RNodeInterface) WriteFramebuffer(line byte, lineData []byte) error {
	if len(lineData) != fbBytesPerLine {
		return fmt.Errorf("line data must be %d bytes", fbBytesPerLine)
	}
	data := append([]byte{line}, lineData...)
	return r.sendCmd(CMD_FB_WRITE, data)
}

func (r *RNodeInterface) DisplayImage(imageData []byte) error {
	if len(imageData)%fbBytesPerLine != 0 {
		return errors.New("imageData length must be a multiple of framebuffer line width")
	}
	lines := len(imageData) / fbBytesPerLine
	for i := 0; i < lines; i++ {
		start := i * fbBytesPerLine
		end := start + fbBytesPerLine
		if err := r.WriteFramebuffer(byte(i), imageData[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (r *RNodeInterface) SetFrequency(freq uint32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, freq)
	return r.sendCmd(CMD_FREQUENCY, b)
}
func (r *RNodeInterface) SetBandwidth(bw uint32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, bw)
	return r.sendCmd(CMD_BANDWIDTH, b)
}
func (r *RNodeInterface) SetTXPower(tx byte) error { return r.sendCmd(CMD_TXPOWER, []byte{tx}) }
func (r *RNodeInterface) SetSF(sf byte) error      { return r.sendCmd(CMD_SF, []byte{sf}) }
func (r *RNodeInterface) SetCR(cr byte) error      { return r.sendCmd(CMD_CR, []byte{cr}) }
func (r *RNodeInterface) SetRadioState(state byte) error {
	r.desiredRadioState.Store(uint32(state))
	return r.sendCmd(CMD_RADIO_STATE, []byte{state})
}

func (r *RNodeInterface) sendCmd(cmd byte, data []byte) error {
	frame := r.makeKISSFrame(cmd, data)
	r.writeMu.Lock()
	_, err := r.tr.Write(frame)
	r.writeMu.Unlock()
	r.lastWriteAt.Store(time.Now().UnixNano())
	return err
}

// -------- read loop / parser ----------
func (r *RNodeInterface) readLoop() {
	defer func() {
		r.online.Store(false)
		if r.Log != nil {
			r.Log.Errorf("%s readLoop ended", r.String())
		}
		go r.reconnectLoop()
	}()

	inFrame := false
	escape := false
	cmd := byte(CMD_UNKNOWN)

	var dataBuf bytes.Buffer
	var cmdBuf bytes.Buffer

	lastByteAt := time.Now()

	byteCh := make(chan byte, 256)
	errCh := make(chan error, 1)

	go func() {
		defer close(byteCh)
		buf := make([]byte, 1)
		for {
			select {
			case <-r.stopCh:
				return
			default:
			}

			// For TCP, periodically nudge Read to unblock.
			if tt, ok := r.tr.(*TCPTransport); ok {
				tt.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
			}

			n, err := r.tr.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			if n <= 0 {
				continue
			}
			byteCh <- buf[0]
		}
	}()

	tick := time.NewTicker(80 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case err := <-errCh:
			_ = err
			return
		case <-tick.C:
			// Read timeout handling like Python: reset partial frames if we stalled.
			if time.Since(lastByteAt) > r.ReadTimeout && (dataBuf.Len() > 0 || cmdBuf.Len() > 0) {
				if r.Log != nil {
					r.Log.Warnf("%s device read timeout in cmd 0x%02X after %s", r.String(), cmd, r.ReadTimeout)
				}
				inFrame = false
				cmd = CMD_UNKNOWN
				escape = false
				dataBuf.Reset()
				cmdBuf.Reset()
			}

			// ID beaconing like Python.
			if r.idInterval > 0 && len(r.idCallsign) > 0 {
				first := r.firstTxAt.Load()
				if first != 0 && time.Since(time.Unix(0, first)) > r.idInterval {
					_ = r.SendData(r.idCallsign)
				}
			}

			// TCP keepalive: re-detect periodically if idle.
			if _, ok := r.tr.(*TCPTransport); ok {
				lastW := r.lastWriteAt.Load()
				if lastW != 0 && time.Since(time.Unix(0, lastW)).Seconds() > TCPActivityKeepalive {
					_ = r.Detect()
				}
			}

			// Display polling if enabled.
			if r.shouldReadDisplay.Load() {
				if intervalAny := r.readDisplayInterval.Load(); intervalAny != nil {
					interval, _ := intervalAny.(float64)
					if interval <= 0 {
						interval = DisplayReadIntervalDefault
					}
					// best-effort periodic read
					if r.displayReadAt.IsZero() || time.Since(r.displayReadAt).Seconds() >= interval {
						_ = r.ReadDisplay()
					}
				}
			}
		case b := <-byteCh:
			lastByteAt = time.Now()

			// end of DATA frame
			if inFrame && b == FEND && cmd == CMD_DATA {
				inFrame = false
				r.onData(dataBuf.Bytes())
				dataBuf.Reset()
				cmdBuf.Reset()
				cmd = CMD_UNKNOWN
				escape = false
				continue
			}

			// new frame start
			if b == FEND {
				inFrame = true
				cmd = CMD_UNKNOWN
				dataBuf.Reset()
				cmdBuf.Reset()
				escape = false
				continue
			}

			if !inFrame {
				continue
			}
			if dataBuf.Len() >= r.HWMTU {
				continue
			}

			// first byte after FEND is command
			if dataBuf.Len() == 0 && cmd == CMD_UNKNOWN {
				cmd = b
				continue
			}

			// decode bytes into appropriate buffer
			switch cmd {
			case CMD_DATA:
				kissUnescapeStream(&dataBuf, b, &escape)
			default:
				// most commands accumulate in cmdBuf (with unescape)
				kissUnescapeStream(&cmdBuf, b, &escape)
				r.tryConsumeCmd(cmd, &cmdBuf)
			}
		}
	}
}

func (r *RNodeInterface) onData(p []byte) {
	if len(p) == 0 {
		return
	}
	// Python increments RX bytes.
	r.rxb.Add(uint64(len(p)))
	if r.Owner != nil {
		cp := append([]byte(nil), p...)
		r.Owner.Inbound(cp, r)
	}
	// Like Python: resets RSSI/SNR after reception
	r.rssi = 0
	r.snr = 0
	r.repStatRSSI.Store(false)
	r.repStatSNR.Store(false)
}

func (r *RNodeInterface) tryConsumeCmd(cmd byte, buf *bytes.Buffer) {
	// In Python, commands "fire" once cmdBuf reaches the expected length.
	switch cmd {
	case CMD_FREQUENCY:
		if buf.Len() == 4 {
			r.rFreq = binary.BigEndian.Uint32(buf.Bytes())
			r.repFreq.Store(true)
			buf.Reset()
			r.updateBitrate()
		}
	case CMD_BANDWIDTH:
		if buf.Len() == 4 {
			r.rBW = binary.BigEndian.Uint32(buf.Bytes())
			r.repBW.Store(true)
			buf.Reset()
			r.updateBitrate()
		}
	case CMD_TXPOWER:
		if buf.Len() == 1 {
			r.rTXP = buf.Bytes()[0]
			r.repTXP.Store(true)
			buf.Reset()
		}
	case CMD_SF:
		if buf.Len() == 1 {
			r.rSF = buf.Bytes()[0]
			r.repSF.Store(true)
			buf.Reset()
			r.updateBitrate()
		}
	case CMD_CR:
		if buf.Len() == 1 {
			r.rCR = buf.Bytes()[0]
			r.repCR.Store(true)
			buf.Reset()
			r.updateBitrate()
		}
	case CMD_RADIO_STATE:
		if buf.Len() == 1 {
			r.radioState.Store(uint32(buf.Bytes()[0]))
			r.repState.Store(true)
			buf.Reset()
		}
	case CMD_FW_VERSION:
		if buf.Len() == 2 {
			r.fwMaj.Store(uint32(buf.Bytes()[0]))
			r.fwMin.Store(uint32(buf.Bytes()[1]))
			buf.Reset()
		}
	case CMD_STAT_RX:
		if buf.Len() == 4 {
			r.rStatRX = binary.BigEndian.Uint32(buf.Bytes())
			buf.Reset()
		}
	case CMD_STAT_TX:
		if buf.Len() == 4 {
			r.rStatTX = binary.BigEndian.Uint32(buf.Bytes())
			buf.Reset()
		}
	case CMD_STAT_RSSI:
		if buf.Len() == 1 {
			// python: byte - RSSI_OFFSET(157)
			r.rssi = int32(int(buf.Bytes()[0]) - 157)
			r.repStatRSSI.Store(true)
			buf.Reset()
		}
	case CMD_STAT_SNR:
		if buf.Len() == 1 {
			sb := int8(buf.Bytes()[0])
			r.snr = float32(sb) * 0.25
			r.repStatSNR.Store(true)
			// Compute quality metric like Python.
			sfs := float32(r.rSF) - 7
			qMin := float32(QSNRMinBase) - sfs*float32(QSNRStep)
			qMax := float32(QSNRMax)
			span := qMax - qMin
			if span > 0 {
				quality := ((r.snr - qMin) / span) * 100
				if quality > 100 {
					quality = 100
				}
				if quality < 0 {
					quality = 0
				}
				r.qSNR = quality
			}
			buf.Reset()
		}
	case CMD_ST_ALOCK:
		if buf.Len() == 2 {
			at := binary.BigEndian.Uint16(buf.Bytes())
			r.rSTAirtimeLimit = float32(at) / 100.0
			buf.Reset()
		}
	case CMD_LT_ALOCK:
		if buf.Len() == 2 {
			at := binary.BigEndian.Uint16(buf.Bytes())
			r.rLTAirtimeLimit = float32(at) / 100.0
			buf.Reset()
		}
	case CMD_STAT_CHTM:
		if buf.Len() == 11 {
			b := buf.Bytes()
			ats := binary.BigEndian.Uint16(b[0:2])
			atl := binary.BigEndian.Uint16(b[2:4])
			cus := binary.BigEndian.Uint16(b[4:6])
			cul := binary.BigEndian.Uint16(b[6:8])
			crs := b[8]
			nfl := b[9]
			ntf := b[10]

			r.rAirtimeShort = float32(ats) / 100.0
			r.rAirtimeLong = float32(atl) / 100.0
			r.rChannelLoadShort = float32(cus) / 100.0
			r.rChannelLoadLong = float32(cul) / 100.0
			r.rCurrentRSSI = int32(int(crs) - RSSI_OFFSET)
			r.rNoiseFloor = int32(int(nfl) - RSSI_OFFSET)
			if ntf == 0xFF {
				r.rInterference = nil
			} else {
				v := int32(int(ntf) - RSSI_OFFSET)
				r.rInterference = &v
				r.rInterferenceAt = time.Now()
			}
			buf.Reset()
		}
	case CMD_STAT_PHYPRM:
		if buf.Len() == 12 {
			b := buf.Bytes()
			lst := float32(binary.BigEndian.Uint16(b[0:2])) / 1000.0
			lsr := binary.BigEndian.Uint16(b[2:4])
			prs := binary.BigEndian.Uint16(b[4:6])
			prt := binary.BigEndian.Uint16(b[6:8])
			cst := binary.BigEndian.Uint16(b[8:10])
			dft := binary.BigEndian.Uint16(b[10:12])

			r.rSymbolTimeMS = lst
			r.rSymbolRate = uint32(lsr)
			r.rPreambleSymbols = uint32(prs)
			r.rPreambleTimeMS = uint32(prt)
			r.rCSMASlotTimeMS = uint32(cst)
			r.rCSMADIFSMS = uint32(dft)
			buf.Reset()
		}
	case CMD_STAT_CSMA:
		if buf.Len() == 3 {
			b := buf.Bytes()
			r.rCSMACWBand = b[0]
			r.rCSMACWMin = b[1]
			r.rCSMACWMax = b[2]
			buf.Reset()
		}
	case CMD_STAT_BAT:
		if buf.Len() == 2 {
			b := buf.Bytes()
			st := b[0]
			pct := b[1]
			if pct > 100 {
				pct = 100
			}
			r.batteryState = st
			r.batteryPercent = pct
			buf.Reset()
		}
	case CMD_STAT_TEMP:
		if buf.Len() == 1 {
			temp := int32(int(buf.Bytes()[0]) - 120)
			if temp >= -30 && temp <= 90 {
				r.temperatureC = &temp
			} else {
				r.temperatureC = nil
			}
			buf.Reset()
		}
	case CMD_RANDOM:
		if buf.Len() == 1 {
			r.randomByte = buf.Bytes()[0]
			buf.Reset()
		}
	case CMD_PLATFORM:
		if buf.Len() == 1 {
			r.platform = buf.Bytes()[0]
			buf.Reset()
		}
	case CMD_MCU:
		if buf.Len() == 1 {
			r.mcu = buf.Bytes()[0]
			buf.Reset()
		}
	case CMD_READY:
		// Python: call process_queue() immediately
		r.flushOne()
		buf.Reset()
	case CMD_DETECT:
		if buf.Len() == 1 {
			r.detected.Store(buf.Bytes()[0] == DETECT_RESP)
			buf.Reset()
		}
	case CMD_ERROR:
		if buf.Len() == 1 {
			code := buf.Bytes()[0]
			buf.Reset()
			if r.Log != nil {
				r.Log.Errorf("%s hardware error code 0x%02X", r.String(), code)
			}
			// For critical errors, mark offline to trigger reconnect like Python raising IOErrors.
			switch code {
			case ErrorMemoryLow:
				r.appendHWError(code, "Memory exhausted on connected device")
			case ErrorModemTimeout:
				r.appendHWError(code, "Modem communication timed out on connected device")
			case ErrorInitRadio:
				r.online.Store(false)
				_ = r.tr.Close()
			case ErrorTXFailed:
				r.online.Store(false)
				_ = r.tr.Close()
			default:
				r.online.Store(false)
				_ = r.tr.Close()
			}
		}
	case CMD_RESET:
		if buf.Len() == 1 {
			code := buf.Bytes()[0]
			buf.Reset()
			if code == 0xF8 && r.platform == PlatformESP32 && r.online.Load() {
				r.online.Store(false)
				_ = r.tr.Close()
			}
		}
	case CMD_FB_READ:
		if buf.Len() == framebufferSize {
			r.framebufferLatency = time.Since(r.framebufferReadAt)
			r.framebuffer = append([]byte(nil), buf.Bytes()...)
			buf.Reset()
		}
	case CMD_DISP_READ:
		if buf.Len() == displaySize {
			r.displayLatency = time.Since(r.displayReadAt)
			r.displayBuf = append([]byte(nil), buf.Bytes()...)
			buf.Reset()
		}
	// The rest (STAT_CHTM/PHYPRM/CSMA/BAT/TEMP/FB/DISP) is added similarly:
	// just "if len==N -> parse -> buf.Reset()"
	default:
		// nothing
	}
}

func (r *RNodeInterface) BitrateEstimate() float64 {
	// python: bitrate = sf * ((4/cr) / (2^sf / (bw/1000))) * 1000
	sf := float64(r.rSF)
	cr := float64(r.rCR)
	bw := float64(r.rBW)
	if sf == 0 || cr == 0 || bw == 0 {
		return 0
	}
	return sf * ((4.0 / cr) / (math.Pow(2, sf) / (bw / 1000.0))) * 1000.0
}

func (r *RNodeInterface) updateBitrate() {
	newRate := r.BitrateEstimate()
	if newRate == 0 {
		return
	}
	r.bitrate = newRate
	if r.Log != nil {
		r.Log.Infof("%s on-air bitrate is %.2f kbps", r.String(), newRate/1000.0)
	}
}

func (r *RNodeInterface) SetID(interval time.Duration, callsign string) error {
	if interval <= 0 || callsign == "" {
		r.idInterval = 0
		r.idCallsign = nil
		return nil
	}
	b := []byte(callsign)
	if len(b) > CALLSIGN_MAX_LEN {
		return fmt.Errorf("id callsign exceeds max length %d bytes", CALLSIGN_MAX_LEN)
	}
	r.idInterval = interval
	r.idCallsign = append([]byte(nil), b...)
	return nil
}

func (r *RNodeInterface) EnableDisplayUpdates(interval time.Duration) {
	if interval > 0 {
		r.readDisplayInterval.Store(interval.Seconds())
	}
	r.shouldReadDisplay.Store(true)
}

func (r *RNodeInterface) DisableDisplayUpdates() {
	r.shouldReadDisplay.Store(false)
}

func (r *RNodeInterface) ReadDisplay() error {
	r.displayReadAt = time.Now()
	return r.sendCmd(CMD_DISP_READ, []byte{0x01})
}

func (r *RNodeInterface) ReadFramebuffer() error {
	r.framebufferReadAt = time.Now()
	return r.sendCmd(CMD_FB_READ, []byte{0x01})
}

func (r *RNodeInterface) validateRadioState() bool {
	// Python waits longer for TCP/BLE.
	switch {
	case strings.HasPrefix(r.tr.String(), "ble://"):
		time.Sleep(1 * time.Second)
	case strings.HasPrefix(r.tr.String(), "tcp://"):
		time.Sleep(1500 * time.Millisecond)
	default:
		time.Sleep(250 * time.Millisecond)
	}

	if bt, ok := r.tr.(interface{ DeviceDisappeared() bool }); ok && bt.DeviceDisappeared() {
		if r.Log != nil {
			r.Log.Errorf("%s device disappeared during radio state validation", r.String())
		}
		return false
	}

	valid := true
	if r.Frequency != 0 && !r.repFreq.Load() {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s frequency report missing", r.String())
		}
	} else if r.repFreq.Load() && r.Frequency != 0 && math.Abs(float64(r.Frequency)-float64(r.rFreq)) > 100 {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s frequency mismatch %d vs %d", r.String(), r.Frequency, r.rFreq)
		}
	}
	if r.Bandwidth != 0 && !r.repBW.Load() {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s bandwidth report missing", r.String())
		}
	} else if r.Bandwidth != 0 && r.repBW.Load() && r.Bandwidth != r.rBW {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s bandwidth mismatch %d vs %d", r.String(), r.Bandwidth, r.rBW)
		}
	}
	if !r.repTXP.Load() {
		// Python treats missing r_txpower as mismatch (None).
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s txpower report missing", r.String())
		}
	} else if r.TXPower != r.rTXP {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s txpower mismatch %d vs %d", r.String(), r.TXPower, r.rTXP)
		}
	}
	if r.SF != 0 && !r.repSF.Load() {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s sf report missing", r.String())
		}
	} else if r.SF != 0 && r.repSF.Load() && r.SF != r.rSF {
		valid = false
		if r.Log != nil {
			r.Log.Errorf("%s sf mismatch %d vs %d", r.String(), r.SF, r.rSF)
		}
	}
	if want := r.desiredRadioState.Load(); want != 0 {
		if !r.repState.Load() {
			valid = false
			if r.Log != nil {
				r.Log.Errorf("%s radio state report missing", r.String())
			}
		} else if got := r.radioState.Load(); got != want {
			valid = false
			if r.Log != nil {
				r.Log.Errorf("%s radio state mismatch %d vs %d", r.String(), want, got)
			}
		}
	}
	return valid
}

func (r *RNodeInterface) Detach() error {
	r.detached.Store(true)
	r.online.Store(false)

	// Best-effort: follow Python order.
	_ = r.DisableExternalFramebuffer()
	_ = r.SetRadioState(RADIO_STATE_OFF)
	_ = r.Leave()
	return r.Stop()
}

func (r *RNodeInterface) ValidateFirmware() error {
	maj := r.fwMaj.Load()
	min := r.fwMin.Load()
	if maj > REQUIRED_FW_VER_MAJ {
		return nil
	}
	if maj == REQUIRED_FW_VER_MAJ && min >= REQUIRED_FW_VER_MIN {
		return nil
	}
	err := fmt.Errorf("firmware version %d.%d below required %d.%d", maj, min, REQUIRED_FW_VER_MAJ, REQUIRED_FW_VER_MIN)
	if r.Log != nil {
		r.Log.Errorf(err.Error())
	}
	if PanicFunc != nil {
		PanicFunc()
	}
	return err
}

func (r *RNodeInterface) RXBytes() uint64 { return r.rxb.Load() }
func (r *RNodeInterface) TXBytes() uint64 { return r.txb.Load() }

func (r *RNodeInterface) BatteryState() byte   { return r.batteryState }
func (r *RNodeInterface) BatteryPercent() byte { return r.batteryPercent }

func (r *RNodeInterface) BatteryStateString() string {
	switch r.batteryState {
	case BatteryCharged:
		return "charged"
	case BatteryCharging:
		return "charging"
	case BatteryDischarging:
		return "discharging"
	default:
		return "unknown"
	}
}

// -------- reconnect ----------
func (r *RNodeInterface) reconnectLoop() {
	for {
		select {
		case <-r.stopCh:
			return
		default:
		}

		time.Sleep(r.ReconnectDelay)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := r.tr.Open(ctx)
		cancel()
		if err != nil {
			if r.Log != nil {
				r.Log.Warnf("%s reconnect failed: %v", r.String(), err)
			}
			continue
		}

		r.online.Store(true)
		r.ready.Store(false)
		_ = r.ConfigureDevice()

		if r.Log != nil {
			r.Log.Infof("%s reconnected via %s", r.String(), r.tr.String())
		}
		return
	}
}

func (r *RNodeInterface) waitForDetection(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if r.detected.Load() {
			return true
		}
		select {
		case <-r.stopCh:
			return false
		default:
		}
		time.Sleep(DETECTION_POLL_INTERVAL)
	}
	return r.detected.Load()
}

func (r *RNodeInterface) appendHWError(code byte, desc string) {
	r.hwErrorsMu.Lock()
	defer r.hwErrorsMu.Unlock()
	r.hwErrors = append(r.hwErrors, HardwareError{
		Code:        code,
		Description: desc,
		At:          time.Now(),
	})
}
