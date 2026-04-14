package vendor

import (
	"encoding/hex"
	"testing"
)

func TestPickleDumps_InterfaceStatsRequest(t *testing.T) {
	// This is the exact request Go sends: {"get": "interface_stats"}
	data := map[string]any{"get": "interface_stats"}
	b, err := PickleDumps(data)
	if err != nil {
		t.Fatalf("PickleDumps: %v", err)
	}
	t.Logf("Encoded: %s", hex.EncodeToString(b))

	// Verify Python can decode it
	result, err := PickleLoads(b)
	if err != nil {
		t.Fatalf("PickleLoads: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if m["get"] != "interface_stats" {
		t.Fatalf("expected 'interface_stats', got %v", m["get"])
	}
}

func TestPickleLoads_PythonForkingPickler(t *testing.T) {
	// This is the exact bytes Python's ForkingPickler.dumps({"get": "interface_stats"}) produces
	// (protocol 4, from our earlier test)
	pyHex := "8004951c000000000000007d948c03676574948c0f696e746572666163655f737461747394732e"
	pyBytes, err := hex.DecodeString(pyHex)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}

	result, err := PickleLoads(pyBytes)
	if err != nil {
		t.Fatalf("PickleLoads: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if m["get"] != "interface_stats" {
		t.Fatalf("expected 'interface_stats', got %v", m["get"])
	}
}

func TestPickleRoundTrip_InterfaceStatsResponse(t *testing.T) {
	// Simulate a response like Python's get_interface_stats() returns
	resp := map[string]any{
		"interfaces": []any{
			map[string]any{
				"name":    "Shared Instance[default]",
				"status":  true,
				"rxb":     0,
				"txb":     0,
				"bitrate": 1000000000,
				"clients": 1,
			},
		},
		"rxb":              uint64(0),
		"txb":              uint64(0),
		"rxs":              0.0,
		"txs":              0.0,
		"transport_id":     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		"transport_uptime": 123.456,
		"probe_responder":  nil,
	}

	b, err := PickleDumps(resp)
	if err != nil {
		t.Fatalf("PickleDumps: %v", err)
	}

	result, err := PickleLoads(b)
	if err != nil {
		t.Fatalf("PickleLoads: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	ifaces, ok := m["interfaces"].([]any)
	if !ok {
		t.Fatalf("expected []any for interfaces, got %T", m["interfaces"])
	}
	if len(ifaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(ifaces))
	}

	iface0 := ifaces[0].(map[string]any)
	if iface0["name"] != "Shared Instance[default]" {
		t.Fatalf("wrong name: %v", iface0["name"])
	}
	if iface0["status"] != true {
		t.Fatalf("wrong status: %v", iface0["status"])
	}
	if iface0["bitrate"] != 1000000000 {
		t.Fatalf("wrong bitrate: %v", iface0["bitrate"])
	}

	// Check transport_id bytes
	tid, ok := m["transport_id"].([]byte)
	if !ok {
		t.Fatalf("expected []byte for transport_id, got %T", m["transport_id"])
	}
	if len(tid) != 16 {
		t.Fatalf("expected 16 bytes for transport_id, got %d", len(tid))
	}
}

func TestPickleAssign_Types(t *testing.T) {
	// Test assigning decoded pickle values to typed Go variables
	var s string
	if err := PickleAssign("hello", &s); err != nil {
		t.Fatalf("string assign: %v", err)
	}
	if s != "hello" {
		t.Fatalf("expected 'hello', got %q", s)
	}

	var n int
	if err := PickleAssign(42, &n); err != nil {
		t.Fatalf("int assign: %v", err)
	}
	if n != 42 {
		t.Fatalf("expected 42, got %d", n)
	}

	var f float64
	if err := PickleAssign(3.14, &f); err != nil {
		t.Fatalf("float64 assign: %v", err)
	}
	if f != 3.14 {
		t.Fatalf("expected 3.14, got %f", f)
	}

	var b bool
	if err := PickleAssign(true, &b); err != nil {
		t.Fatalf("bool assign: %v", err)
	}
	if !b {
		t.Fatalf("expected true")
	}

	var m map[string]any
	if err := PickleAssign(map[string]any{"key": "val"}, &m); err != nil {
		t.Fatalf("map assign: %v", err)
	}
	if m["key"] != "val" {
		t.Fatalf("expected 'val', got %v", m["key"])
	}
}