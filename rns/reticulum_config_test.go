package rns

import (
	"encoding/hex"
	"strings"
	"testing"

	configobj "github.com/svanichkin/configobj"
)

func TestReticulumApplyConfig_MTU(t *testing.T) {
	maybeParallel(t)

	prevMTU := MTU
	t.Cleanup(func() { _ = SetMTU(prevMTU) })

	cfg, err := configobj.LoadReader(strings.NewReader(strings.Join([]string{
		"[reticulum]",
		"mtu = 700",
	}, "\n")))
	if err != nil {
		t.Fatalf("LoadReader: %v", err)
	}

	r := &Reticulum{Config: cfg}
	if err := r.applyConfig(); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if MTU != 700 {
		t.Fatalf("expected MTU=700 got %d", MTU)
	}
	if MDU <= 0 {
		t.Fatalf("expected derived MDU > 0")
	}
}

func TestReticulumApplyConfig_RPCKey_InvalidHex(t *testing.T) {
	maybeParallel(t)

	cfg, err := configobj.LoadReader(strings.NewReader(strings.Join([]string{
		"[reticulum]",
		"rpc_key = this-is-not-hex",
	}, "\n")))
	if err != nil {
		t.Fatalf("LoadReader: %v", err)
	}

	r := &Reticulum{Config: cfg, RPCKey: []byte("preexisting")}
	if err := r.applyConfig(); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if r.RPCKey != nil {
		t.Fatalf("expected RPCKey to be nil on invalid hex")
	}
}

func TestReticulumApplyConfig_LinkMTUDiscovery(t *testing.T) {
	maybeParallel(t)

	prev := linkMTUDiscovery
	t.Cleanup(func() { linkMTUDiscovery = prev })
	linkMTUDiscovery = false

	cfg, err := configobj.LoadReader(strings.NewReader(strings.Join([]string{
		"[reticulum]",
		"link_mtu_discovery = yes",
	}, "\n")))
	if err != nil {
		t.Fatalf("LoadReader: %v", err)
	}

	r := &Reticulum{Config: cfg}
	if err := r.applyConfig(); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if !LinkMTUDiscovery() {
		t.Fatalf("expected link MTU discovery enabled")
	}
}

func TestReticulumApplyConfig_PanicOnInterfaceError(t *testing.T) {
	maybeParallel(t)

	cfg, err := configobj.LoadReader(strings.NewReader(strings.Join([]string{
		"[reticulum]",
		"panic_on_interface_error = yes",
	}, "\n")))
	if err != nil {
		t.Fatalf("LoadReader: %v", err)
	}

	r := &Reticulum{Config: cfg}
	if err := r.applyConfig(); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if !r.PanicOnInterfaceError {
		t.Fatalf("expected PanicOnInterfaceError=true")
	}
}

func TestReticulumApplyConfig_RespondToProbes(t *testing.T) {
	maybeParallel(t)

	prev := allowProbes
	t.Cleanup(func() { allowProbes = prev })
	allowProbes = false

	cfg, err := configobj.LoadReader(strings.NewReader(strings.Join([]string{
		"[reticulum]",
		"respond_to_probes = yes",
	}, "\n")))
	if err != nil {
		t.Fatalf("LoadReader: %v", err)
	}

	r := &Reticulum{Config: cfg}
	if err := r.applyConfig(); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if !allowProbes {
		t.Fatalf("expected allowProbes=true")
	}
}

func TestReticulumApplyConfig_RemoteManagementAllowed_LengthValidation(t *testing.T) {
	maybeParallel(t)

	cfg, err := configobj.LoadReader(strings.NewReader(strings.Join([]string{
		"[reticulum]",
		"remote_management_allowed = 0123",
	}, "\n")))
	if err != nil {
		t.Fatalf("LoadReader: %v", err)
	}

	r := &Reticulum{Config: cfg}
	if err := r.applyConfig(); err == nil {
		t.Fatalf("expected length validation error")
	}
}

func TestReticulumApplyConfig_RemoteManagementAllowed_AddsHash(t *testing.T) {
	maybeParallel(t)

	// Provide a correctly-sized identity hash.
	hexLen := (TRUNCATED_HASHLENGTH / 8) * 2
	raw := make([]byte, TRUNCATED_HASHLENGTH/8)
	for i := range raw {
		raw[i] = byte(i)
	}
	hashHex := hex.EncodeToString(raw)
	if len(hashHex) != hexLen {
		t.Fatalf("expected hex length %d got %d", hexLen, len(hashHex))
	}

	cfg, err := configobj.LoadReader(strings.NewReader(strings.Join([]string{
		"[reticulum]",
		"remote_management_allowed = " + hashHex,
	}, "\n")))
	if err != nil {
		t.Fatalf("LoadReader: %v", err)
	}

	r := &Reticulum{Config: cfg}
	if err := r.applyConfig(); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	remoteManagementAllowedMu.RLock()
	_, ok := remoteManagementAllowed[string(raw)]
	remoteManagementAllowedMu.RUnlock()
	if !ok {
		t.Fatalf("expected hash to be in remote management ACL")
	}
}

func TestReticulumApplyConfig_BlackholeSettings(t *testing.T) {
	prevPublish := publishBlackholeEnabled
	prevSources := BlackholeSources()
	t.Cleanup(func() {
		publishBlackholeEnabled = prevPublish
		blackholeSources = prevSources
	})

	source := make([]byte, TRUNCATED_HASHLENGTH/8)
	for i := range source {
		source[i] = byte(0xA0 + i)
	}

	cfg, err := configobj.LoadReader(strings.NewReader(strings.Join([]string{
		"[reticulum]",
		"publish_blackhole = yes",
		"blackhole_sources = " + hex.EncodeToString(source),
	}, "\n")))
	if err != nil {
		t.Fatalf("LoadReader: %v", err)
	}

	r := &Reticulum{Config: cfg}
	if err := r.applyConfig(); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if !PublishBlackholeEnabled() {
		t.Fatalf("expected publish_blackhole enabled")
	}
	sources := BlackholeSources()
	if len(sources) != 1 || hex.EncodeToString(sources[0]) != hex.EncodeToString(source) {
		t.Fatalf("blackhole sources=%x, want %x", sources, source)
	}
}
