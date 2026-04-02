package rns

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInterfaceDiscovery_ListDiscoveredInterfaces_FiltersSourcesAndInvalidReachable(t *testing.T) {
	prevSources := InterfaceDiscoverySources()
	interfaceDiscoverySources = [][]byte{{0xAA, 0xBB}}
	t.Cleanup(func() {
		interfaceDiscoverySources = prevSources
	})

	discovery, err := newInterfaceDiscoveryWithStorage(t.TempDir(), 14, nil, false)
	if err != nil {
		t.Fatalf("newInterfaceDiscoveryWithStorage: %v", err)
	}

	discovery.interfaceDiscovered(map[string]any{
		"discovery_hash": []byte{0x01},
		"name":           "valid",
		"type":           "BackboneInterface",
		"transport":      true,
		"reachable_on":   "reticulum.example",
		"port":           4242,
		"value":          20,
		"network_id":     hex.EncodeToString([]byte{0xAA, 0xBB}),
	})
	discovery.interfaceDiscovered(map[string]any{
		"discovery_hash": []byte{0x02},
		"name":           "wrong-source",
		"type":           "BackboneInterface",
		"transport":      true,
		"reachable_on":   "reticulum.example",
		"port":           4243,
		"value":          20,
		"network_id":     hex.EncodeToString([]byte{0xCC, 0xDD}),
	})
	discovery.interfaceDiscovered(map[string]any{
		"discovery_hash": []byte{0x03},
		"name":           "bad-reachable",
		"type":           "BackboneInterface",
		"transport":      true,
		"reachable_on":   "bad host!",
		"port":           4244,
		"value":          20,
		"network_id":     hex.EncodeToString([]byte{0xAA, 0xBB}),
	})

	list := discovery.ListDiscoveredInterfaces(false, false)
	if len(list) != 1 {
		t.Fatalf("discovered interfaces=%d, want 1", len(list))
	}
	if got, _ := list[0]["name"].(string); got != "valid" {
		t.Fatalf("name=%q, want valid", got)
	}

	files, err := os.ReadDir(filepath.Join(discovery.storagePath))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("persisted files=%d, want 1 after filtering", len(files))
	}
}

func TestInterfaceDiscovery_Autoconnect_AddsInterfaceAndSkipsDuplicate(t *testing.T) {
	prevOwner := Owner
	prevInterfaces := Interfaces
	prevFactory := discoveryAutoconnectInterfaceFactory
	prevAutoconnect := autoconnectDiscoveredInterfaces

	Owner = &Reticulum{}
	Interfaces = nil
	autoconnectDiscoveredInterfaces = 1

	var builds int
	discoveryAutoconnectInterfaceFactory = func(info map[string]any) (*Interface, error) {
		builds++
		return &Interface{
			Name:              info["name"].(string),
			Type:              "BackboneClientInterface",
			DriverImplemented: true,
			Online:            true,
		}, nil
	}
	discovery := &InterfaceDiscovery{
		monitorEvery: time.Hour,
		detachAfter:  discoveryDetachThreshold,
	}
	t.Cleanup(func() {
		Owner = prevOwner
		Interfaces = prevInterfaces
		discoveryAutoconnectInterfaceFactory = prevFactory
		autoconnectDiscoveredInterfaces = prevAutoconnect
		discovery.mu.Lock()
		discovery.monitoring = false
		discovery.mu.Unlock()
	})

	info := map[string]any{
		"name":         "public-bb",
		"type":         "BackboneInterface",
		"reachable_on": "198.51.100.10",
		"port":         4242,
		"network_id":   "abcd",
	}

	discovery.autoconnect(info)
	discovery.autoconnect(info)

	if builds != 1 {
		t.Fatalf("factory calls=%d, want 1", builds)
	}
	if len(Interfaces) != 1 {
		t.Fatalf("interfaces=%d, want 1", len(Interfaces))
	}
	ifc := Interfaces[0]
	if ifc == nil {
		t.Fatalf("expected autoconnected interface")
	}
	if !ifc.OUT {
		t.Fatalf("expected OUT=true on autoconnected interface")
	}
	if ifc.Bitrate != discoveryAutoconnectBitrate {
		t.Fatalf("bitrate=%d, want %d", ifc.Bitrate, discoveryAutoconnectBitrate)
	}
	if len(ifc.AutoconnectHash) == 0 {
		t.Fatalf("expected autoconnect hash to be set")
	}
	if ifc.AutoconnectSource != "abcd" {
		t.Fatalf("autoconnect source=%q, want abcd", ifc.AutoconnectSource)
	}
	discovery.mu.Lock()
	monitored := len(discovery.monitored)
	discovery.monitoring = false
	discovery.mu.Unlock()
	if monitored != 1 {
		t.Fatalf("monitored interfaces=%d, want 1", monitored)
	}
}

func TestInterfaceDiscovery_MonitorOnce_DetachesExpiredAutoconnect(t *testing.T) {
	prevInterfaces := Interfaces
	Interfaces = nil
	t.Cleanup(func() {
		Interfaces = prevInterfaces
	})

	ifc := &Interface{
		Name:            "auto0",
		Type:            "BackboneClientInterface",
		AutoconnectHash: []byte{0x01},
		Online:          false,
		AutoconnectDown: time.Now().Add(-13 * time.Second),
	}
	Interfaces = []*Interface{ifc}

	discovery := &InterfaceDiscovery{
		monitored:    []*Interface{ifc},
		monitoring:   true,
		detachAfter:  12 * time.Second,
		monitorEvery: 5 * time.Second,
	}

	running := discovery.monitorOnce(time.Now())
	if running {
		t.Fatalf("monitorOnce() = true, want false after detach")
	}
	if len(Interfaces) != 0 {
		t.Fatalf("interfaces=%d, want 0", len(Interfaces))
	}
	discovery.mu.Lock()
	defer discovery.mu.Unlock()
	if len(discovery.monitored) != 0 {
		t.Fatalf("monitored interfaces=%d, want 0", len(discovery.monitored))
	}
}

func TestInterfaceDiscovery_MonitorOnce_TearsDownBootstrapOnlyAtTarget(t *testing.T) {
	prevOwner := Owner
	prevInterfaces := Interfaces
	prevAutoconnect := autoconnectDiscoveredInterfaces

	Owner = &Reticulum{}
	Interfaces = nil
	autoconnectDiscoveredInterfaces = 1
	t.Cleanup(func() {
		Owner = prevOwner
		Interfaces = prevInterfaces
		autoconnectDiscoveredInterfaces = prevAutoconnect
	})

	autoIfc := &Interface{
		Name:            "auto0",
		Type:            "BackboneClientInterface",
		AutoconnectHash: []byte{0x01},
		Online:          true,
	}
	bootstrapIfc := &Interface{
		Name:          "bootstrap0",
		Type:          "WeaveInterface",
		BootstrapOnly: true,
		Online:        true,
	}
	Interfaces = []*Interface{autoIfc, bootstrapIfc}

	discovery := &InterfaceDiscovery{
		monitored:    []*Interface{autoIfc},
		monitoring:   true,
		detachAfter:  12 * time.Second,
		monitorEvery: 5 * time.Second,
	}

	running := discovery.monitorOnce(time.Now())
	if !running {
		t.Fatal("monitorOnce() = false, want true while autoconnected interface remains monitored")
	}
	if len(Interfaces) != 1 || Interfaces[0] != autoIfc {
		t.Fatalf("interfaces after bootstrap teardown=%#v, want only autoconnected interface", Interfaces)
	}
}

func TestInterfaceDiscovery_MonitorOnce_ReenablesBootstrapOnlyWhenNoAutoConnectedRemain(t *testing.T) {
	prevOwner := Owner
	prevInterfaces := Interfaces

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

	Owner = &Reticulum{
		ConfigPath: cfgPath,
		BootstrapConfigs: []interfaceConfigEntry{{
			Name: "BootstrapWeave",
			KV: map[string]string{
				"enabled":        "True",
				"type":           "WeaveInterface",
				"port":           "fake",
				"bootstrap_only": "True",
			},
		}},
	}
	Interfaces = nil
	t.Cleanup(func() {
		for _, ifc := range Interfaces {
			if ifc != nil {
				ifc.Detach()
			}
		}
		Owner = prevOwner
		Interfaces = prevInterfaces
	})

	discovery := &InterfaceDiscovery{
		monitoring:   true,
		detachAfter:  12 * time.Second,
		monitorEvery: 5 * time.Second,
	}

	running := discovery.monitorOnce(time.Now())
	if running {
		t.Fatal("monitorOnce() = true, want false with no monitored autoconnects")
	}
	if len(Interfaces) != 1 {
		t.Fatalf("interfaces=%d, want 1 bootstrap interface re-enabled", len(Interfaces))
	}
	if !Interfaces[0].BootstrapOnly {
		t.Fatal("expected re-enabled interface BootstrapOnly=true")
	}
	if Interfaces[0].Name != "BootstrapWeave" {
		t.Fatalf("interface name=%q, want BootstrapWeave", Interfaces[0].Name)
	}
}
