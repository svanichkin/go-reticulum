package rns

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	configobj "github.com/svanichkin/configobj"
	ifaces "github.com/svanichkin/go-reticulum/rns/interfaces"
)

func TestReticulumBringUpSystemInterfaces_EnabledGating(t *testing.T) {
	// Modifies global interface registry; keep serial.

	dir, err := os.MkdirTemp("", "rns_reticulum_ifcfg_*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	configPath := filepath.Join(dir, "config")
	cfgText := strings.Join([]string{
		"[reticulum]",
		"share_instance = no",
		"",
		"[interfaces]",
		"  [[NoEnabledKey]]",
		"    type = UDPInterface",
		"  [[ExplicitNo]]",
		"    type = UDPInterface",
		"    enabled = no",
		"  [[ExplicitYes]]",
		"    type = UDPInterface",
		"    enabled = yes",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(cfgText), 0o644); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}

	cfg, err := configobj.Load(configPath)
	if err != nil {
		t.Fatalf("Load(config): %v", err)
	}

	prev := Interfaces
	t.Cleanup(func() { Interfaces = prev })
	Interfaces = nil

	r := &Reticulum{
		Config:        cfg,
		ConfigPath:    configPath,
		StoragePath:   filepath.Join(dir, "storage"),
		CachePath:     filepath.Join(dir, "storage/cache"),
		ResourcePath:  filepath.Join(dir, "storage/resources"),
		IdentityPath:  filepath.Join(dir, "storage/identities"),
		InterfacePath: filepath.Join(dir, "interfaces"),
	}

	if err := r.bringUpSystemInterfaces(); err != nil {
		t.Fatalf("bringUpSystemInterfaces: %v", err)
	}
	if len(Interfaces) != 1 {
		t.Fatalf("expected only explicitly enabled interface to be brought up, got %d", len(Interfaces))
	}
	if Interfaces[0] == nil || Interfaces[0].Name != "ExplicitYes" {
		t.Fatalf("unexpected interface: %#v", Interfaces[0])
	}
}

func TestReticulumBringUpSystemInterfaces_TCPInterfaceAlias_Client(t *testing.T) {
	// Modifies global interface registry; keep serial.

	dir, err := os.MkdirTemp("", "rns_reticulum_tcpif_*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	configPath := filepath.Join(dir, "config")
	cfgText := strings.Join([]string{
		"[reticulum]",
		"share_instance = no",
		"",
		"[interfaces]",
		"  [[Uplink]]",
		"    type = TCPInterface",
		"    enabled = yes",
		"    target_host = 127.0.0.1",
		"    target_port = 1",
		"    connect_timeout = 0.01",
		"    reconnect_wait = 0.01",
		"    max_reconnect_tries = 0",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(cfgText), 0o644); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}

	cfg, err := configobj.Load(configPath)
	if err != nil {
		t.Fatalf("Load(config): %v", err)
	}

	prev := Interfaces
	t.Cleanup(func() { Interfaces = prev })
	Interfaces = nil

	r := &Reticulum{
		Config:        cfg,
		ConfigPath:    configPath,
		StoragePath:   filepath.Join(dir, "storage"),
		CachePath:     filepath.Join(dir, "storage/cache"),
		ResourcePath:  filepath.Join(dir, "storage/resources"),
		IdentityPath:  filepath.Join(dir, "storage/identities"),
		InterfacePath: filepath.Join(dir, "interfaces"),
	}

	if err := r.bringUpSystemInterfaces(); err != nil {
		t.Fatalf("bringUpSystemInterfaces: %v", err)
	}
	if len(Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(Interfaces))
	}
	if Interfaces[0] == nil || Interfaces[0].Type != "TCPClientInterface" {
		t.Fatalf("expected TCPClientInterface, got %#v", Interfaces[0])
	}
}

func TestReticulumBringUpSystemInterfaces_OutgoingAndAnnounceCapAndIFACDefaults(t *testing.T) {
	// Modifies global interface registry; keep serial.

	dir, err := os.MkdirTemp("", "rns_reticulum_ifparity_*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	configPath := filepath.Join(dir, "config")
	cfgText := strings.Join([]string{
		"[reticulum]",
		"share_instance = no",
		"",
		"[interfaces]",
		"  [[Iface]]",
		"    type = UDPInterface",
		"    enabled = yes",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"    announce_cap = 50",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(cfgText), 0o644); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}

	cfg, err := configobj.Load(configPath)
	if err != nil {
		t.Fatalf("Load(config): %v", err)
	}

	prev := Interfaces
	t.Cleanup(func() { Interfaces = prev })
	Interfaces = nil

	r := &Reticulum{
		Config:        cfg,
		ConfigPath:    configPath,
		StoragePath:   filepath.Join(dir, "storage"),
		CachePath:     filepath.Join(dir, "storage/cache"),
		ResourcePath:  filepath.Join(dir, "storage/resources"),
		IdentityPath:  filepath.Join(dir, "storage/identities"),
		InterfacePath: filepath.Join(dir, "interfaces"),
	}

	if err := r.bringUpSystemInterfaces(); err != nil {
		t.Fatalf("bringUpSystemInterfaces: %v", err)
	}
	if len(Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(Interfaces))
	}
	ifc := Interfaces[0]
	if ifc == nil {
		t.Fatalf("nil interface")
	}
	if !ifc.OUT {
		t.Fatalf("expected outgoing enabled by default")
	}
	if ifc.AnnounceCap != 0.5 {
		t.Fatalf("expected AnnounceCap=0.5 got %v (%#v)", ifc.AnnounceCap, ifc)
	}
	if ifc.IFACSize != 16 {
		t.Fatalf("expected IFACSize default 16 for UDPInterface got %d", ifc.IFACSize)
	}
}

func TestReticulumBringUpSystemInterfaces_BackboneInterfaceAlias(t *testing.T) {
	// Modifies global interface registry; keep serial.
	if runtime.GOOS != "linux" && runtime.GOOS != "android" {
		t.Skip("Backbone* interfaces are only supported on Linux-based operating systems")
	}

	dir, err := os.MkdirTemp("", "rns_reticulum_bbif_*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	configPath := filepath.Join(dir, "config")
	cfgText := strings.Join([]string{
		"[reticulum]",
		"share_instance = no",
		"",
		"[interfaces]",
		"  [[BB]]",
		"    type = BackboneInterface",
		"    enabled = yes",
		"    target_host = 127.0.0.1",
		"    target_port = 4242",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(cfgText), 0o644); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}

	cfg, err := configobj.Load(configPath)
	if err != nil {
		t.Fatalf("Load(config): %v", err)
	}

	prev := Interfaces
	t.Cleanup(func() { Interfaces = prev })
	Interfaces = nil

	r := &Reticulum{
		Config:        cfg,
		ConfigPath:    configPath,
		StoragePath:   filepath.Join(dir, "storage"),
		CachePath:     filepath.Join(dir, "storage/cache"),
		ResourcePath:  filepath.Join(dir, "storage/resources"),
		IdentityPath:  filepath.Join(dir, "storage/identities"),
		InterfacePath: filepath.Join(dir, "interfaces"),
	}

	if err := r.bringUpSystemInterfaces(); err != nil {
		t.Fatalf("bringUpSystemInterfaces: %v", err)
	}
	if len(Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(Interfaces))
	}
	if Interfaces[0] == nil || Interfaces[0].Type != "BackboneClientInterface" {
		t.Fatalf("expected BackboneClientInterface, got %#v", Interfaces[0])
	}
}

func TestReticulum_TCPServerAcceptedPeer_InheritsIngressControl(t *testing.T) {
	prevInterfaces := Interfaces
	Interfaces = nil
	t.Cleanup(func() { Interfaces = prevInterfaces })

	parent := &Interface{
		Name:                "tcp-server",
		Type:                "TCPServerInterface",
		IN:                  true,
		OUT:                 false,
		IngressControl:      false,
		ICMaxHeldAnnounces:  intPtr(123),
		AnnounceRateTarget:  intPtr(7),
		AnnounceRateGrace:   intPtr(8),
		AnnounceRatePenalty: intPtr(9),
		AutoconfigureMTU:    true,
		FixedMTU:            false,
		Bitrate:             1000,
		HWMTU:               4096,
		DriverImplemented:   true,
		Online:              true,
	}

	server := &ifaces.TCPServerInterface{}
	server.OnNewClient = func(ci *ifaces.TCPClientInterface) {
		if ci == nil {
			return
		}
		peer := &Interface{
			Name:              ci.String(),
			Type:              "TCPClientInterface",
			Parent:            parent,
			DriverImplemented: true,
			Online:            true,
			Bitrate:           parent.Bitrate,
			HWMTU:             parent.HWMTU,
			AutoconfigureMTU:  parent.AutoconfigureMTU,
			FixedMTU:          parent.FixedMTU,
		}
		inheritInterfaceConfig(peer, parent)
		peer.SetTCPClient(ci)
		ci.Owner = tcpOwnerAdapter{ifc: peer}
		interfacesMu.Lock()
		Interfaces = append(Interfaces, peer)
		interfacesMu.Unlock()
	}

	ci := ifaces.NewTCPClientFromAccepted(nil, nil, "client0", nil, false, false)
	server.OnNewClient(ci)

	if len(Interfaces) != 1 {
		t.Fatalf("expected 1 accepted peer interface, got %d", len(Interfaces))
	}
	peer := Interfaces[0]
	if peer == nil {
		t.Fatalf("nil peer")
	}
	if peer.IngressControl != parent.IngressControl {
		t.Fatalf("peer IngressControl=%v, want %v", peer.IngressControl, parent.IngressControl)
	}
	if peer.OUT != parent.OUT || peer.IN != parent.IN {
		t.Fatalf("peer IN/OUT=(%v,%v), want (%v,%v)", peer.IN, peer.OUT, parent.IN, parent.OUT)
	}
	if peer.ICMaxHeldAnnounces == nil || *peer.ICMaxHeldAnnounces != 123 {
		t.Fatalf("peer ICMaxHeldAnnounces=%v, want 123", peer.ICMaxHeldAnnounces)
	}
	if peer.AnnounceRateTarget == nil || *peer.AnnounceRateTarget != 7 {
		t.Fatalf("peer AnnounceRateTarget=%v, want 7", peer.AnnounceRateTarget)
	}
}

func intPtr(v int) *int { return &v }
