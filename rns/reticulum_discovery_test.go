package rns

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	configobj "github.com/svanichkin/configobj"
)

func TestReticulumApplyConfig_DiscoverySettings(t *testing.T) {
	prevNetworkIdentity := NetworkIdentity
	prevDiscoverInterfacesMode := discoverInterfacesMode
	prevRequiredDiscoveryValue := requiredDiscoveryValue
	prevTransportDiscoveryRequiredValue := discoveryRequiredValue
	prevInterfaceDiscoverySources := InterfaceDiscoverySources()
	prevAutoconnectDiscoveredInterfaces := autoconnectDiscoveredInterfaces

	NetworkIdentity = nil
	discoverInterfacesMode = false
	requiredDiscoveryValue = nil
	discoveryRequiredValue = DefaultDiscoveryRequiredValue
	interfaceDiscoverySources = nil
	autoconnectDiscoveredInterfaces = 0

	t.Cleanup(func() {
		NetworkIdentity = prevNetworkIdentity
		discoverInterfacesMode = prevDiscoverInterfacesMode
		requiredDiscoveryValue = prevRequiredDiscoveryValue
		discoveryRequiredValue = prevTransportDiscoveryRequiredValue
		interfaceDiscoverySources = prevInterfaceDiscoverySources
		autoconnectDiscoveredInterfaces = prevAutoconnectDiscoveredInterfaces
	})

	dir := t.TempDir()
	identityPath := filepath.Join(dir, "network_identity")
	sourceHash := make([]byte, truncatedHashBytes)
	for i := range sourceHash {
		sourceHash[i] = byte(i + 1)
	}

	cfg, err := configobj.LoadReader(strings.NewReader(strings.Join([]string{
		"[reticulum]",
		"network_identity = " + identityPath,
		"discover_interfaces = yes",
		"required_discovery_value = 19",
		"interface_discovery_sources = " + hex.EncodeToString(sourceHash),
		"autoconnect_discovered_interfaces = 2",
	}, "\n")))
	if err != nil {
		t.Fatalf("LoadReader: %v", err)
	}

	r := &Reticulum{
		Config:  cfg,
		UserDir: dir,
	}
	if err := r.applyConfig(); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	if !HasNetworkIdentity() {
		t.Fatalf("expected network identity to be configured")
	}
	if _, err := os.Stat(identityPath); err != nil {
		t.Fatalf("expected network identity file to be created: %v", err)
	}
	if !discoverInterfacesMode {
		t.Fatalf("expected discover_interfaces to be enabled")
	}
	if RequiredDiscoveryValue() == nil || *RequiredDiscoveryValue() != 19 {
		t.Fatalf("required discovery value=%v, want 19", RequiredDiscoveryValue())
	}
	if discoveryRequiredValue != 19 {
		t.Fatalf("transport discoveryRequiredValue=%d, want 19", discoveryRequiredValue)
	}
	sources := InterfaceDiscoverySources()
	if len(sources) != 1 || !bytes.Equal(sources[0], sourceHash) {
		t.Fatalf("interface discovery sources=%x, want %x", sources, sourceHash)
	}
	if !ShouldAutoconnectDiscoveredInterfaces() {
		t.Fatalf("expected autoconnect discovered interfaces to be enabled")
	}
	if MaxAutoconnectedInterfaces() != 2 {
		t.Fatalf("max autoconnected interfaces=%d, want 2", MaxAutoconnectedInterfaces())
	}
}

func TestRequiredDiscoveryValue_DefaultsNilUntilConfigured(t *testing.T) {
	prevRequiredDiscoveryValue := requiredDiscoveryValue
	prevTransportDiscoveryRequiredValue := discoveryRequiredValue
	requiredDiscoveryValue = nil
	discoveryRequiredValue = DefaultDiscoveryRequiredValue
	t.Cleanup(func() {
		requiredDiscoveryValue = prevRequiredDiscoveryValue
		discoveryRequiredValue = prevTransportDiscoveryRequiredValue
	})

	if got := RequiredDiscoveryValue(); got != nil {
		t.Fatalf("RequiredDiscoveryValue()=%v, want nil", got)
	}
}

func TestBringUpSystemInterfaces_AppliesDiscoveryConfig(t *testing.T) {
	prevInterfaces := Interfaces
	prevDiscoveryEnabled := discoveryEnabled

	Interfaces = nil
	discoveryEnabled = false
	t.Cleanup(func() {
		for _, ifc := range Interfaces {
			if ifc != nil {
				ifc.Detach()
			}
		}
		Interfaces = prevInterfaces
		discoveryEnabled = prevDiscoveryEnabled
	})

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config")
	cfg := strings.Join([]string{
		"[interfaces]",
		"  [[DiscoveryWeave]]",
		"    enabled = True",
		"    type = WeaveInterface",
		"    port = fake",
		"    discoverable = True",
		"    announce_interval = 3",
		"    discovery_stamp_value = 17",
		"    discovery_name = Public Weave",
		"    discovery_encrypt = True",
		"    reachable_on = discovery.example",
		"    publish_ifac = True",
		"    latitude = 1.5",
		"    longitude = 2.5",
		"    height = 3.5",
		"    discovery_frequency = 868000000",
		"    discovery_bandwidth = 125000",
		"    discovery_channel = 4",
		"    discovery_modulation = lora",
	}, "\n")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}
	parsed, err := configobj.Load(cfgPath)
	if err != nil {
		t.Fatalf("configobj.Load: %v", err)
	}

	r := &Reticulum{
		Config:                parsed,
		ConfigPath:            cfgPath,
		PanicOnInterfaceError: false,
	}
	if err := r.synthesizeConfiguredInterfaces(); err != nil {
		t.Fatalf("synthesizeConfiguredInterfaces: %v", err)
	}
	if len(Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(Interfaces))
	}

	ifc := Interfaces[0]
	if ifc == nil {
		t.Fatalf("expected non-nil interface")
	}
	if !discoveryEnabled {
		t.Fatalf("expected discoveryEnabled to be raised by discoverable interface")
	}
	if !ifc.Discoverable {
		t.Fatalf("expected interface to be discoverable")
	}
	if ifc.Mode != InterfaceModeGateway {
		t.Fatalf("mode=%d, want gateway %d", ifc.Mode, InterfaceModeGateway)
	}
	if ifc.DiscoveryAnnounceInterval != 5*time.Minute {
		t.Fatalf("announce interval=%v, want 5m floor", ifc.DiscoveryAnnounceInterval)
	}
	if ifc.DiscoveryStampValue == nil || *ifc.DiscoveryStampValue != 17 {
		t.Fatalf("stamp value=%v, want 17", ifc.DiscoveryStampValue)
	}
	if ifc.DiscoveryName != "Public Weave" {
		t.Fatalf("discovery name=%q, want Public Weave", ifc.DiscoveryName)
	}
	if !ifc.DiscoveryEncrypt {
		t.Fatalf("expected discovery encryption enabled")
	}
	if ifc.DiscoveryReachableOn != "discovery.example" {
		t.Fatalf("reachable_on=%q, want discovery.example", ifc.DiscoveryReachableOn)
	}
	if !ifc.DiscoveryPublishIFAC {
		t.Fatalf("expected publish_ifac enabled")
	}
	if ifc.DiscoveryLatitude == nil || *ifc.DiscoveryLatitude != 1.5 {
		t.Fatalf("latitude=%v, want 1.5", ifc.DiscoveryLatitude)
	}
	if ifc.DiscoveryLongitude == nil || *ifc.DiscoveryLongitude != 2.5 {
		t.Fatalf("longitude=%v, want 2.5", ifc.DiscoveryLongitude)
	}
	if ifc.DiscoveryHeight == nil || *ifc.DiscoveryHeight != 3.5 {
		t.Fatalf("height=%v, want 3.5", ifc.DiscoveryHeight)
	}
	if ifc.DiscoveryFrequency == nil || *ifc.DiscoveryFrequency != 868000000 {
		t.Fatalf("frequency=%v, want 868000000", ifc.DiscoveryFrequency)
	}
	if ifc.DiscoveryBandwidth == nil || *ifc.DiscoveryBandwidth != 125000 {
		t.Fatalf("bandwidth=%v, want 125000", ifc.DiscoveryBandwidth)
	}
	if ifc.DiscoveryChannel == nil || *ifc.DiscoveryChannel != 4 {
		t.Fatalf("channel=%v, want 4", ifc.DiscoveryChannel)
	}
	if ifc.DiscoveryModulation != "lora" {
		t.Fatalf("modulation=%q, want lora", ifc.DiscoveryModulation)
	}
}

func TestBringUpSystemInterfaces_TracksBootstrapOnlyConfig(t *testing.T) {
	prevInterfaces := Interfaces
	Interfaces = nil
	t.Cleanup(func() {
		for _, ifc := range Interfaces {
			if ifc != nil {
				ifc.Detach()
			}
		}
		Interfaces = prevInterfaces
	})

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config")
	cfg := strings.Join([]string{
		"[interfaces]",
		"  [[BootstrapWeave]]",
		"    enabled = True",
		"    type = WeaveInterface",
		"    port = fake",
		"    bootstrap_only = True",
	}, "\n")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}
	parsed, err := configobj.Load(cfgPath)
	if err != nil {
		t.Fatalf("configobj.Load: %v", err)
	}

	r := &Reticulum{
		Config:                parsed,
		ConfigPath:            cfgPath,
		PanicOnInterfaceError: false,
	}
	if err := r.synthesizeConfiguredInterfaces(); err != nil {
		t.Fatalf("synthesizeConfiguredInterfaces: %v", err)
	}
	if len(Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(Interfaces))
	}
	if !Interfaces[0].BootstrapOnly {
		t.Fatal("expected interface BootstrapOnly=true")
	}
	if len(r.BootstrapConfigs) != 1 {
		t.Fatalf("bootstrap configs=%d, want 1", len(r.BootstrapConfigs))
	}
	if got := r.BootstrapConfigs[0]["name"]; got != "BootstrapWeave" {
		t.Fatalf("bootstrap config name=%q, want BootstrapWeave", got)
	}
	if got := strings.ToLower(strings.TrimSpace(r.BootstrapConfigs[0]["bootstrap_only"])); got != "true" {
		t.Fatalf("bootstrap config bootstrap_only=%q, want true", got)
	}
}

func TestReticulumDiscoveredInterfaces(t *testing.T) {
	r := &Reticulum{StoragePath: t.TempDir()}
	discovery := &InterfaceDiscovery{
		requiredValue: 14,
		storagePath:   filepath.Join(r.StoragePath, "discovery", "interfaces"),
		monitorEvery:  discoveryMonitorInterval,
		detachAfter:   discoveryDetachThreshold,
	}
	if err := os.MkdirAll(discovery.storagePath, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	discovery.interfaceDiscovered(map[string]any{
		"discovery_hash": []byte{0x01, 0x02, 0x03},
		"name":           "public-tcp",
		"type":           "TCPServerInterface",
		"transport":      true,
		"reachable_on":   "reticulum.example",
		"port":           4242,
		"value":          23,
	})

	list := r.DiscoveredInterfaces()
	if len(list) != 1 {
		t.Fatalf("discovered interfaces=%d, want 1", len(list))
	}
	info := list[0]
	if got, _ := info["name"].(string); got != "public-tcp" {
		t.Fatalf("name=%q, want public-tcp", got)
	}
	if got, _ := info["reachable_on"].(string); got != "reticulum.example" {
		t.Fatalf("reachable_on=%q, want reticulum.example", got)
	}
	intOf := func(v any) int {
		switch x := v.(type) {
		case int:
			return x
		case int64:
			return int(x)
		case uint64:
			return int(x)
		case float64:
			return int(x)
		default:
			return 0
		}
	}
	if got := intOf(info["port"]); got != 4242 {
		t.Fatalf("port=%v, want 4242", info["port"])
	}
}
