package interfaces

import (
	"net"
	"os"
	"testing"
	"time"

	"errors"
	"sync/atomic"
)

func TestBackbone_parseBackboneConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := parseBackboneConfig("bb", map[string]string{
		"listen_ip":   "127.0.0.1",
		"listen_port": "1234",
	})
	if err != nil {
		t.Fatalf("parseBackboneConfig: %v", err)
	}
	if cfg.Listen != "127.0.0.1" || cfg.Port != 1234 {
		t.Fatalf("unexpected cfg: %#v", cfg)
	}
}

func TestBackbone_parseBackboneClientConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := parseBackboneClientConfig("bb", map[string]string{
		"target_host": "example.com",
		"target_port": "1234",
	})
	if err != nil {
		t.Fatalf("parseBackboneClientConfig: %v", err)
	}
	if cfg.TargetHost != "example.com" || cfg.TargetPort != 1234 {
		t.Fatalf("unexpected cfg: %#v", cfg)
	}
	if cfg.ConnectTimeout != 5*time.Second {
		t.Fatalf("unexpected connect timeout %v", cfg.ConnectTimeout)
	}
	if cfg.ReconnectWait != 5*time.Second {
		t.Fatalf("unexpected reconnect wait %v", cfg.ReconnectWait)
	}
}

func TestBackbone_stripIPv6Brackets(t *testing.T) {
	t.Parallel()

	if got := stripIPv6Brackets("[::1]"); got != "::1" {
		t.Fatalf("unexpected host: %q", got)
	}
	if got := stripIPv6Brackets("::1"); got != "::1" {
		t.Fatalf("unexpected host: %q", got)
	}
	if got := stripIPv6Brackets(" [fe80::1%eth0] "); got != "fe80::1%eth0" {
		t.Fatalf("unexpected host: %q", got)
	}
}

func TestBackbone_selectDialTarget_IPv6Bracketed(t *testing.T) {
	t.Parallel()

	addr, network, err := selectDialTarget("[::1]", 4242, false)
	if err != nil {
		t.Fatalf("selectDialTarget: %v", err)
	}
	if network != "tcp6" {
		t.Fatalf("unexpected network: %q", network)
	}
	if addr != "[::1]:4242" {
		t.Fatalf("unexpected addr: %q", addr)
	}
}

func TestBackbone_selectListenTarget_IPv6Bracketed(t *testing.T) {
	t.Parallel()

	addr, network, err := selectListenTarget("[::1]", 4242, false)
	if err != nil {
		t.Fatalf("selectListenTarget: %v", err)
	}
	if network != "tcp6" {
		t.Fatalf("unexpected network: %q", network)
	}
	if addr != "[::1]:4242" {
		t.Fatalf("unexpected addr: %q", addr)
	}
}

func TestBackbone_selectListenTargets_Unix(t *testing.T) {
	t.Parallel()

	targets, err := selectListenTargets("unix:/tmp/reticulum.sock", 0, false)
	if err != nil {
		t.Fatalf("selectListenTargets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].network != "unix" || targets[0].addr != "/tmp/reticulum.sock" {
		t.Fatalf("unexpected target: %#v", targets[0])
	}
}

func TestBackbone_selectListenTargets_IPv6Bracketed(t *testing.T) {
	t.Parallel()

	targets, err := selectListenTargets("[::1]", 4242, false)
	if err != nil {
		t.Fatalf("selectListenTargets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].network != "tcp6" || targets[0].addr != "[::1]:4242" {
		t.Fatalf("unexpected target: %#v", targets[0])
	}
}

func TestBackbone_removeStaleUnixSocket_RefusesRegularFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := dir + "/sock"
	if err := os.WriteFile(path, []byte("nope"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := removeStaleUnixSocket(path); err == nil {
		t.Fatal("expected error for regular file")
	}
}

func TestBackbone_selectListenTargetsForDeviceAddrs_Order(t *testing.T) {
	t.Parallel()

	addrs := []net.Addr{
		&net.IPNet{IP: net.ParseIP("192.0.2.10"), Mask: net.CIDRMask(24, 32)},
		&net.IPNet{IP: net.ParseIP("2001:db8::1"), Mask: net.CIDRMask(64, 128)},
	}

	targets, err := selectListenTargetsForDeviceAddrs(addrs, "eth0", 4242, false)
	if err != nil {
		t.Fatalf("selectListenTargetsForDeviceAddrs: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
	if targets[0].network != "tcp4" || targets[1].network != "tcp6" {
		t.Fatalf("unexpected order: %#v", targets)
	}

	targets6, err := selectListenTargetsForDeviceAddrs(addrs, "eth0", 4242, true)
	if err != nil {
		t.Fatalf("selectListenTargetsForDeviceAddrs: %v", err)
	}
	if len(targets6) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets6))
	}
	if targets6[0].network != "tcp6" || targets6[1].network != "tcp4" {
		t.Fatalf("unexpected order prefer v6: %#v", targets6)
	}
}

func TestBackbone_backboneExceededMaxReconnect(t *testing.T) {
	t.Parallel()

	if backboneExceededMaxReconnect(1, 0) {
		t.Fatal("maxReconnect=0 must mean unlimited")
	}
	if backboneExceededMaxReconnect(0, 1) {
		t.Fatal("retries=0 should not reach max")
	}
	if backboneExceededMaxReconnect(1, 1) {
		t.Fatal("attempts=1 should not exceed max=1")
	}
	if !backboneExceededMaxReconnect(2, 1) {
		t.Fatal("attempts=2 should exceed max=1")
	}
}

func TestBackbone_selectListenTargetsForDeviceAddrs_LinkLocalAndPreferIPv6(t *testing.T) {
	t.Parallel()

	addrs := []net.Addr{
		&net.IPNet{IP: net.ParseIP("192.0.2.10"), Mask: net.CIDRMask(24, 32)},
		&net.IPNet{IP: net.ParseIP("fe80::1"), Mask: net.CIDRMask(64, 128)},
		&net.IPNet{IP: net.ParseIP("2001:db8::2"), Mask: net.CIDRMask(64, 128)},
	}

	targets, err := selectListenTargetsForDeviceAddrs(addrs, "eth0", 4242, true)
	if err != nil {
		t.Fatalf("selectListenTargetsForDeviceAddrs: %v", err)
	}
	if len(targets) < 2 {
		t.Fatalf("expected >=2 targets, got %d", len(targets))
	}
	if targets[0].network != "tcp6" {
		t.Fatalf("expected tcp6 first, got %#v", targets)
	}
	if targets[0].addr != "[2001:db8::2]:4242" {
		t.Fatalf("unexpected v6 global addr: %q", targets[0].addr)
	}
	if targets[1].network != "tcp6" || targets[1].addr != "[fe80::1%eth0]:4242" {
		t.Fatalf("unexpected v6 link-local addr: %#v", targets[1])
	}
}

func TestBackbone_pickPreferredIPAddr(t *testing.T) {
	t.Parallel()

	addrs := []net.IPAddr{
		{IP: net.ParseIP("192.0.2.10")},
		{IP: net.ParseIP("2001:db8::1")},
	}
	if got := pickPreferredIPAddr(addrs, false); got == nil || got.IP.To4() == nil {
		t.Fatalf("expected v4, got %#v", got)
	}
	if got := pickPreferredIPAddr(addrs, true); got == nil || got.IP.To4() != nil {
		t.Fatalf("expected v6, got %#v", got)
	}
}

func TestBackbone_splitFirstV4V6(t *testing.T) {
	t.Parallel()

	addrs := []net.IPAddr{
		{IP: net.ParseIP("2001:db8::1")},
		{IP: net.ParseIP("192.0.2.10")},
	}
	v4, v6 := splitFirstV4V6(addrs)
	if v4 == nil || v4.IP.To4() == nil {
		t.Fatalf("expected v4, got %#v", v4)
	}
	if v6 == nil || v6.IP.To4() != nil {
		t.Fatalf("expected v6, got %#v", v6)
	}
}

func TestBackbone_joinHostPort_ScopedLinkLocal(t *testing.T) {
	t.Parallel()

	ip := net.IPAddr{IP: net.ParseIP("fe80::1"), Zone: "eth0"}
	if got := net.JoinHostPort(ip.String(), "4242"); got != "[fe80::1%eth0]:4242" {
		t.Fatalf("unexpected addr: %q", got)
	}
}

func TestBackboneClient_ReconnectMax_TeardownDoesNotRemoveInterface(t *testing.T) {
	oldDial := backboneDial
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() {
		backboneDial = oldDial
		RemoveInterfaceHandler = oldRemove
	})

	backboneDial = func(*net.Dialer, string, string) (net.Conn, error) {
		return nil, errors.New("dial failed")
	}

	removed := make(chan *Interface, 1)
	RemoveInterfaceHandler = func(iface *Interface) { removed <- iface }

	iface := &Interface{Name: "bb-client", Type: "BackboneClientInterface", IN: true, OUT: false}
	driver := &BackboneClientDriver{
		iface: iface,
		cfg: backboneClientConfig{
			Name:           "bb-client",
			TargetHost:     "127.0.0.1",
			TargetPort:     4242,
			ConnectTimeout: 1 * time.Millisecond,
			ReconnectWait:  1 * time.Millisecond,
			MaxReconnect:   1,
			PreferIPv6:     false,
			I2PTunneled:    true,
		},
		stopCh: make(chan struct{}),
	}
	driver.neverConnected.Store(true)
	iface.backboneClient = driver

	done := make(chan struct{})
	go func() {
		defer close(done)
		driver.run()
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for driver to teardown")
	}

	if iface.IN || iface.OUT || iface.Online {
		t.Fatalf("expected iface disabled after teardown, got IN=%v OUT=%v Online=%v", iface.IN, iface.OUT, iface.Online)
	}
	select {
	case <-removed:
		t.Fatal("expected interface not to be removed from transport on max reconnect")
	default:
	}
}

func TestBackboneClient_Reconnect_CallsTunnelSynthesizer(t *testing.T) {
	oldDial := backboneDial
	oldSynth := TunnelSynthesizer
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() {
		backboneDial = oldDial
		TunnelSynthesizer = oldSynth
		RemoveInterfaceHandler = oldRemove
	})
	RemoveInterfaceHandler = func(*Interface) {}

	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()

	backboneDial = func(*net.Dialer, string, string) (net.Conn, error) {
		return clientSide, nil
	}

	var called atomic.Int32
	TunnelSynthesizer = func(*Interface) { called.Add(1) }

	iface := &Interface{Name: "bb-client", Type: "BackboneClientInterface", IN: true, OUT: false}
	driver := &BackboneClientDriver{
		iface: iface,
		cfg: backboneClientConfig{
			Name:           "bb-client",
			TargetHost:     "127.0.0.1",
			TargetPort:     4242,
			ConnectTimeout: 1 * time.Millisecond,
			ReconnectWait:  1 * time.Millisecond,
			MaxReconnect:   0,
		},
		stopCh: make(chan struct{}),
	}
	driver.neverConnected.Store(false)
	iface.backboneClient = driver

	done := make(chan struct{})
	go func() {
		defer close(done)
		driver.run()
	}()

	deadline := time.After(500 * time.Millisecond)
	for called.Load() == 0 {
		select {
		case <-deadline:
			close(driver.stopCh)
			<-done
			t.Fatal("timeout waiting for TunnelSynthesizer call")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	close(driver.stopCh)
	<-done
	if called.Load() != 1 {
		t.Fatalf("expected 1 TunnelSynthesizer call, got %d", called.Load())
	}
}

func TestBackboneInterface_StringMatchesPython(t *testing.T) {
	t.Parallel()

	server := &Interface{Name: "bb0", Type: "BackboneInterface"}
	server.backboneServer = &BackboneInterfaceDriver{
		iface: server,
		cfg: backboneServerConfig{
			Name:   "bb0",
			Listen: "127.0.0.1",
			Port:   4242,
		},
	}
	if got := server.String(); got != "BackboneInterface[bb0/127.0.0.1:4242]" {
		t.Fatalf("unexpected server string %q", got)
	}

	client := &Interface{Name: "bb1", Type: "BackboneClientInterface"}
	client.backboneClient = &BackboneClientDriver{
		iface: client,
		cfg: backboneClientConfig{
			Name:       "bb1",
			TargetHost: "::1",
			TargetPort: 4243,
		},
	}
	if got := client.String(); got != "BackboneInterface[bb1/[::1]:4243]" {
		t.Fatalf("unexpected client string %q", got)
	}
}
