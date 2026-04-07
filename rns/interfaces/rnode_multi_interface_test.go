package interfaces

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestRNodeMulti_parseTruthy(t *testing.T) {
	t.Parallel()

	if parseTruthy("yes", false) != true {
		t.Fatalf("expected yes=true")
	}
	if parseTruthy("0", true) != false {
		t.Fatalf("expected 0=false")
	}
	if parseTruthy("nope", true) != true {
		t.Fatalf("expected default")
	}
}

func TestRNodeMulti_parseFloatSimple(t *testing.T) {
	t.Parallel()

	if v, ok := parseFloatSimple("1.25"); !ok || v != 1.25 {
		t.Fatalf("unexpected parseFloatSimple result: %v %v", v, ok)
	}
	if _, ok := parseFloatSimple("nope"); ok {
		t.Fatalf("expected parse failure")
	}
}

func TestRNodeMulti_parseRNodeMultiConfig_Minimal(t *testing.T) {
	t.Parallel()

	cfg, err := parseRNodeMultiConfig("rm", map[string]string{
		"port": "/dev/ttyUSB0",
		// Minimal subinterface section encoded by reticulum.go: "sub.<name>.<key>"
		"sub.a.vport":           "0",
		"sub.a.frequency":       "868000000",
		"sub.a.bandwidth":       "125000",
		"sub.a.txpower":         "10",
		"sub.a.spreadingfactor": "7",
		"sub.a.codingrate":      "5",
	})
	if err != nil {
		t.Fatalf("parseRNodeMultiConfig: %v", err)
	}
	if cfg.Port != "/dev/ttyUSB0" {
		t.Fatalf("unexpected port %q", cfg.Port)
	}
	if len(cfg.Subinterfaces) != 1 {
		t.Fatalf("expected 1 subinterface, got %d", len(cfg.Subinterfaces))
	}
}

func TestRNodeMulti_SubValidation_FailsOnMissingReports(t *testing.T) {
	t.Parallel()

	s := &RNodeSubDriver{
		frequency: 868000000,
		bandwidth: 125000,
		txpower:   10,
		sf:        7,
	}
	s.desiredState.Store(uint32(RADIO_STATE_ON))

	if err := s.validateReported(); err == nil {
		t.Fatalf("expected error when reports are missing")
	}
}

func TestRNodeMulti_SubValidation_PassesWhenReportsMatch(t *testing.T) {
	t.Parallel()

	s := &RNodeSubDriver{
		frequency: 868000000,
		bandwidth: 125000,
		txpower:   10,
		sf:        7,
	}
	s.desiredState.Store(uint32(RADIO_STATE_ON))

	s.rFrequency.Store(868000000)
	s.rBandwidth.Store(125000)
	s.rTXPower.Store(10)
	s.rSF.Store(7)
	s.rState.Store(uint32(RADIO_STATE_ON))

	s.repFreq.Store(true)
	s.repBW.Store(true)
	s.repTXP.Store(true)
	s.repSF.Store(true)
	s.repState.Store(true)

	if err := s.validateReported(); err != nil {
		t.Fatalf("unexpected validateReported error: %v", err)
	}
}

func TestRNodeMulti_makeSelDataFrame_EscapesAndReportsEscapedLen(t *testing.T) {
	t.Parallel()

	payload := []byte{0x01, FEND, 0x02, FESC, 0x03}
	frame, escLen := makeSelDataFrame(2, payload)
	escaped := kissEscape(payload)
	if escLen != len(escaped) {
		t.Fatalf("expected escaped len %d got %d", len(escaped), escLen)
	}
	want := append([]byte{FEND, cmdSelInt, 2, FEND, FEND, CMD_DATA}, escaped...)
	want = append(want, FEND)
	if !bytes.Equal(frame, want) {
		t.Fatalf("unexpected frame:\nwant %x\ngot  %x", want, frame)
	}
}

func TestRNodeMulti_Beacon_FirstTxGating(t *testing.T) {
	t.Parallel()

	d := &RNodeMultiDriver{}
	d.cfg.IDCallsign = []byte("CALL")
	d.cfg.IDInterval = 5 * time.Second

	if d.shouldSendBeacon(time.Now()) {
		t.Fatalf("should not send beacon without first tx")
	}

	d.firstTxAt.Store(time.Now().Add(-6 * time.Second).UnixNano())
	if !d.shouldSendBeacon(time.Now()) {
		t.Fatalf("expected shouldSendBeacon after interval")
	}
}

func TestRNodeMultiInterface_StringMatchesPython(t *testing.T) {
	t.Parallel()

	iface := &Interface{
		Name: "rm",
		Type: "RNodeMultiInterface",
	}

	if got := iface.String(); got != "RNodeMultiInterface[rm]" {
		t.Fatalf("unexpected string form %q", got)
	}
}

func TestNewRNodeMultiInterface_InitialOpenFailureStartsReconnectLoop(t *testing.T) {
	prevStart := rnodeMultiStartDriver
	prevReconnect := rnodeMultiStartReconnectLoop
	t.Cleanup(func() {
		rnodeMultiStartDriver = prevStart
		rnodeMultiStartReconnectLoop = prevReconnect
	})

	calledReconnect := make(chan struct{}, 1)
	rnodeMultiStartDriver = func(_ *RNodeMultiDriver) error {
		return errors.New("open failed")
	}
	rnodeMultiStartReconnectLoop = func(_ *RNodeMultiDriver) {
		select {
		case calledReconnect <- struct{}{}:
		default:
		}
	}

	iface, err := NewRNodeMultiInterface("rm", map[string]string{
		"port":                    "/dev/ttyUSB0",
		"sub.a.vport":             "0",
		"sub.a.frequency":         "868000000",
		"sub.a.bandwidth":         "125000",
		"sub.a.txpower":           "10",
		"sub.a.spreadingfactor":   "7",
		"sub.a.codingrate":        "5",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iface == nil || iface.rnodeMulti == nil {
		t.Fatalf("expected retained multi driver")
	}

	select {
	case <-calledReconnect:
	default:
		t.Fatalf("expected reconnect loop to be started")
	}
}
