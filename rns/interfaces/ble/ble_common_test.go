package ble

import "testing"

func TestChunkByMaxWriteNoRsp(t *testing.T) {
	in := make([]byte, 55)
	for i := range in {
		in[i] = byte(i)
	}
	chunks := ChunkByMaxWriteNoRsp(in, 20)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 20 || len(chunks[1]) != 20 || len(chunks[2]) != 15 {
		t.Fatalf("unexpected chunk sizes: %d %d %d", len(chunks[0]), len(chunks[1]), len(chunks[2]))
	}
	if chunks[0][0] != 0 || chunks[2][14] != 54 {
		t.Fatalf("unexpected content boundaries")
	}
}

func TestParseTarget(t *testing.T) {
	if name, addr := ParseTarget(""); name != "" || addr != "" {
		t.Fatalf("expected empty, got name=%q addr=%q", name, addr)
	}
	if name, addr := ParseTarget("   "); name != "" || addr != "" {
		t.Fatalf("expected trimmed empty, got name=%q addr=%q", name, addr)
	}
	if name, addr := ParseTarget("name:RNode ABC"); name != "RNode ABC" || addr != "" {
		t.Fatalf("unexpected parse: name=%q addr=%q", name, addr)
	}
	if name, addr := ParseTarget("NaMe: RNode ABC "); name != "RNode ABC" || addr != "" {
		t.Fatalf("unexpected case-insensitive parse: name=%q addr=%q", name, addr)
	}
	if name, addr := ParseTarget("AA:BB:CC:DD:EE:FF"); name != "" || addr != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("unexpected parse: name=%q addr=%q", name, addr)
	}
	if name, addr := ParseTarget("RNode ABC"); name != "RNode ABC" || addr != "" {
		t.Fatalf("unexpected parse: name=%q addr=%q", name, addr)
	}
}
