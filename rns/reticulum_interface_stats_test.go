package rns

import (
	"bytes"
	"testing"
)

func TestGetInterfaceStats_UsesClientCountAndFullParentHash(t *testing.T) {
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevNetworkIdentity := NetworkIdentity
	prevInterfaces := Interfaces

	transportEnabled = true
	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	networkID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(network): %v", err)
	}
	TransportIdentity = transportID
	NetworkIdentity = networkID
	Interfaces = nil
	t.Cleanup(func() {
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		NetworkIdentity = prevNetworkIdentity
		Interfaces = prevInterfaces
	})

	parent := &Interface{Name: "ParentInterface", Type: "UDPInterface"}
	child := &Interface{Name: "ChildInterface", Type: "UDPInterface", Parent: parent, AutoconnectSource: "abcd"}

	shared := &Interface{Name: "Shared Instance[default]", Type: "LocalInterface"}
	shared.SetClientCountFunc(func() int { return 3 })

	Interfaces = []*Interface{child, shared}

	r := &Reticulum{}
	stats := r.GetInterfaceStats()

	raw, ok := stats["interfaces"]
	if !ok {
		t.Fatalf("missing interfaces list")
	}
	entries, ok := raw.([]map[string]any)
	if !ok {
		t.Fatalf("unexpected interfaces type %T", raw)
	}

	var (
		childEntry  map[string]any
		sharedEntry map[string]any
	)
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if name, _ := entry["name"].(string); name == "ChildInterface" {
			childEntry = entry
		}
		if name, _ := entry["name"].(string); name == "Shared Instance[default]" {
			sharedEntry = entry
		}
	}
	if childEntry == nil {
		t.Fatalf("missing child interface entry")
	}
	if sharedEntry == nil {
		t.Fatalf("missing shared instance entry")
	}

	gotClients, ok := sharedEntry["clients"].(int)
	if !ok {
		t.Fatalf("expected shared clients=int, got %T", sharedEntry["clients"])
	}
	if gotClients != 3 {
		t.Fatalf("expected shared clients=3, got %d", gotClients)
	}

	parentHash, ok := childEntry["parent_interface_hash"].([]byte)
	if !ok {
		t.Fatalf("expected parent_interface_hash=[]byte, got %T", childEntry["parent_interface_hash"])
	}
	if len(parentHash) != 32 {
		t.Fatalf("expected full 32-byte parent hash, got %d bytes", len(parentHash))
	}
	if want := parent.GetHash(); !bytes.Equal(parentHash, want) {
		t.Fatalf("unexpected parent_interface_hash")
	}

	if got, _ := childEntry["autoconnect_source"].(string); got != "abcd" {
		t.Fatalf("expected autoconnect_source=abcd, got %q", got)
	}

	networkHash, ok := stats["network_id"].([]byte)
	if !ok {
		t.Fatalf("expected network_id=[]byte, got %T", stats["network_id"])
	}
	if !bytes.Equal(networkHash, networkID.Hash) {
		t.Fatalf("unexpected network_id")
	}
}
