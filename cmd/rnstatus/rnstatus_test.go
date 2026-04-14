package main

import (
	"strings"
	"testing"
	"time"
)

func TestSpeedStr(t *testing.T) {
	if got := speedStr(999, "bps"); got != "999.00 bps" {
		t.Fatalf("got %q", got)
	}
	if got := speedStr(1000, "bps"); got != "1.00 kbps" {
		t.Fatalf("got %q", got)
	}
	if got := speedStr(8000, "Bps"); got != "1.00 KBps" {
		t.Fatalf("got %q", got)
	}
}

func TestNumField_UnsignedCounters(t *testing.T) {
	m := map[string]any{
		"u64": uint64(123),
		"u32": uint32(456),
		"u":   uint(789),
	}
	if got, ok := numField(m, "u64"); !ok || got != 123 {
		t.Fatalf("u64: got=%d ok=%v", got, ok)
	}
	if got, ok := numField(m, "u32"); !ok || got != 456 {
		t.Fatalf("u32: got=%d ok=%v", got, ok)
	}
	if got, ok := numField(m, "u"); !ok || got != 789 {
		t.Fatalf("u: got=%d ok=%v", got, ok)
	}
}

func TestByteField_BytesLikeValues(t *testing.T) {
	type namedBytes []byte

	m := map[string]any{
		"raw":   []byte{0x01, 0x02, 0x03},
		"named": namedBytes{0x04, 0x05, 0x06},
		"text":  "abc",
	}

	if got, ok := byteField(m, "raw"); !ok || string(got) != "\x01\x02\x03" {
		t.Fatalf("raw: got=%v ok=%v", got, ok)
	}
	if got, ok := byteField(m, "named"); !ok || string(got) != "\x04\x05\x06" {
		t.Fatalf("named: got=%v ok=%v", got, ok)
	}
	if got, ok := byteField(m, "text"); !ok || string(got) != "abc" {
		t.Fatalf("text: got=%q ok=%v", got, ok)
	}
}

func TestSortInterfaces_MissingKeys(t *testing.T) {
	ifs := []map[string]any{
		{"name": "a", "bitrate": 200},
		{"name": "b"},                 // missing bitrate
		{"name": "c", "bitrate": nil}, // nil bitrate
		{"name": "d", "bitrate": 100},
	}
	// Ascending by bitrate should put 0-valued entries first, stable among equals.
	sortInterfaces(ifs, "rate", false)
	if ifs[0]["name"] != "b" || ifs[1]["name"] != "c" || ifs[2]["name"] != "d" || ifs[3]["name"] != "a" {
		t.Fatalf("unexpected order: %v", []any{ifs[0]["name"], ifs[1]["name"], ifs[2]["name"], ifs[3]["name"]})
	}
}

func TestRenderDiscoveredInterfaces_JSONNormalisesBytes(t *testing.T) {
	ifs := []map[string]any{
		{
			"name":  "public-tcp",
			"stamp": []byte{0xaa, 0xbb},
		},
	}
	out, err := renderDiscoveredInterfaces(ifs, nil, true, false)
	if err != nil {
		t.Fatalf("renderDiscoveredInterfaces: %v", err)
	}
	if !strings.Contains(out, `"stamp":"aabb"`) {
		t.Fatalf("expected hex-normalised stamp in json output, got %q", out)
	}
}

func TestRenderDiscoveredInterfaces_CompactFiltersAndPrintsPortlessRow(t *testing.T) {
	now := time.Now()
	ifs := []map[string]any{
		{
			"name":       "public-tcp-listener",
			"type":       "TCPServerInterface",
			"status":     "available",
			"last_heard": now.Unix(),
			"value":      23,
		},
		{
			"name":       "backbone",
			"type":       "BackboneInterface",
			"status":     "unknown",
			"last_heard": now.Add(-2 * time.Hour).Unix(),
			"value":      7,
		},
	}
	filter := "tcp"
	out, err := renderDiscoveredInterfaces(ifs, &filter, false, false)
	if err != nil {
		t.Fatalf("renderDiscoveredInterfaces: %v", err)
	}
	if !strings.Contains(out, "public-tcp-listener") {
		t.Fatalf("expected filtered row in output, got %q", out)
	}
	if strings.Contains(out, "backbone") {
		t.Fatalf("unexpected unfiltered row in output, got %q", out)
	}
	if !strings.Contains(out, "✓ Available") {
		t.Fatalf("expected compact status label, got %q", out)
	}
}

func TestRenderDiscoveredInterfaces_DetailedUsesPortFieldAndConfigEntry(t *testing.T) {
	now := time.Now()
	ifs := []map[string]any{
		{
			"name":         "public-tcp",
			"type":         "TCPServerInterface",
			"status":       "available",
			"transport":    true,
			"hops":         2,
			"discovered":   now.Add(-5 * time.Minute).Unix(),
			"last_heard":   now.Add(-90 * time.Second).Unix(),
			"reachable_on": "reticulum.example",
			"port":         4242,
			"value":        23,
			"config_entry": "[[TCP public-tcp]]\nport = 4242",
			"transport_id": "abcd",
			"network_id":   "dcba",
		},
	}
	out, err := renderDiscoveredInterfaces(ifs, nil, false, true)
	if err != nil {
		t.Fatalf("renderDiscoveredInterfaces: %v", err)
	}
	if !strings.Contains(out, "Transport ID : abcd") {
		t.Fatalf("expected transport id in output, got %q", out)
	}
	if !strings.Contains(out, "Network   ID : dcba") {
		t.Fatalf("expected network id in output, got %q", out)
	}
	if !strings.Contains(out, "Port         : 4242") {
		t.Fatalf("expected port line in output, got %q", out)
	}
	if !strings.Contains(out, "Configuration Entry:") {
		t.Fatalf("expected configuration entry in output, got %q", out)
	}
	if !strings.Contains(out, "[[TCP public-tcp]]") {
		t.Fatalf("expected config entry lines in output, got %q", out)
	}
}
