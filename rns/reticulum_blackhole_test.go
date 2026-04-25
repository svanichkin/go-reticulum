package rns

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

type fakeRPCConn struct {
	recv     map[string]any
	sent     []any
	closed   bool
	read     *bytes.Reader
	writeBuf []byte
}

type fakeAddr string

func (a fakeAddr) Network() string { return "fake" }
func (a fakeAddr) String() string  { return string(a) }

func (c *fakeRPCConn) Recv(v interface{}) error {
	call, ok := v.(*map[string]any)
	if !ok {
		return nil
	}
	*call = c.recv
	return nil
}

func (c *fakeRPCConn) Read(p []byte) (int, error) {
	if c.read == nil {
		data, err := umsgpack.PickleDumps(c.recv)
		if err != nil {
			return 0, err
		}
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
		c.read = bytes.NewReader(append(hdr[:], data...))
	}
	return c.read.Read(p)
}

func (c *fakeRPCConn) Write(p []byte) (int, error) {
	c.writeBuf = append(c.writeBuf, p...)
	for len(c.writeBuf) >= 4 {
		n := int(binary.BigEndian.Uint32(c.writeBuf[:4]))
		if len(c.writeBuf) < 4+n {
			break
		}
		payload := make([]byte, n)
		copy(payload, c.writeBuf[4:4+n])
		c.writeBuf = c.writeBuf[4+n:]
		v, err := umsgpack.PickleLoads(payload)
		if err != nil {
			return len(p), err
		}
		c.sent = append(c.sent, v)
	}
	return len(p), nil
}

func (c *fakeRPCConn) Send(v interface{}) error {
	c.sent = append(c.sent, v)
	return nil
}

func (c *fakeRPCConn) Close() error {
	c.closed = true
	return nil
}

func (c *fakeRPCConn) LocalAddr() net.Addr              { return fakeAddr("local") }
func (c *fakeRPCConn) RemoteAddr() net.Addr             { return fakeAddr("remote") }
func (c *fakeRPCConn) SetDeadline(time.Time) error      { return nil }
func (c *fakeRPCConn) SetReadDeadline(time.Time) error  { return nil }
func (c *fakeRPCConn) SetWriteDeadline(time.Time) error { return nil }

func TestReticulumBlackholeIdentity_LocalAPI(t *testing.T) {
	prevOwner := Owner
	prevTransportIdentity := TransportIdentity
	prevBlackholed := blackholedIdentities

	dir := t.TempDir()
	Owner = &Reticulum{StoragePath: dir}
	TransportIdentity, _ = NewIdentity()
	blackholedIdentities = make(map[hashKey]*blackholeEntry)
	if err := os.MkdirAll(filepath.Join(dir, "blackhole"), 0o755); err != nil {
		t.Fatalf("MkdirAll(blackhole): %v", err)
	}

	t.Cleanup(func() {
		Owner = prevOwner
		TransportIdentity = prevTransportIdentity
		blackholedIdentities = prevBlackholed
	})

	r := &Reticulum{StoragePath: dir}
	victim, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(victim): %v", err)
	}

	reason := "local-api"
	until := time.Now().Add(time.Hour)
	if ok := r.BlackholeIdentity(victim.Hash, &until, &reason); ok != true {
		t.Fatalf("BlackholeIdentity() = %#v, want true", ok)
	}

	snapshot := r.GetBlackholedIdentities()
	key, _ := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(victim.Hash)
	entry, exists := snapshot[key]
	if !exists {
		t.Fatal("expected blackholed identity in snapshot")
	}
	if got, _ := entry["reason"].(string); got != "local-api" {
		t.Fatalf("snapshot reason=%q, want local-api", got)
	}

	if ok := r.UnblackholeIdentity(victim.Hash); ok != true {
		t.Fatalf("UnblackholeIdentity() = %#v, want true", ok)
	}
	if ok := r.UnblackholeIdentity(victim.Hash); ok != nil {
		t.Fatalf("UnblackholeIdentity() duplicate = %#v, want nil", ok)
	}
	if len(r.GetBlackholedIdentities()) != 0 {
		t.Fatalf("expected empty blackhole snapshot after unblackhole, got %#v", r.GetBlackholedIdentities())
	}

	raw, err := os.ReadFile(filepath.Join(dir, "blackhole", "local"))
	if err != nil {
		t.Fatalf("ReadFile(local blackhole): %v", err)
	}
	var persisted map[any]any
	if err := umsgpack.Unpackb(raw, &persisted); err != nil {
		t.Fatalf("Unpackb(local blackhole): %v", err)
	}
	if _, exists := persistedBlackholeLookup(persisted, victim.Hash); exists {
		t.Fatal("expected persisted local blackhole list to drop removed identity")
	}
}

func TestReticulumHandleRPC_BlackholeSurface(t *testing.T) {
	prevOwner := Owner
	prevTransportIdentity := TransportIdentity
	prevBlackholed := blackholedIdentities

	Owner = &Reticulum{StoragePath: t.TempDir()}
	TransportIdentity, _ = NewIdentity()
	blackholedIdentities = make(map[hashKey]*blackholeEntry)
	if err := os.MkdirAll(filepath.Join(Owner.StoragePath, "blackhole"), 0o755); err != nil {
		t.Fatalf("MkdirAll(blackhole): %v", err)
	}

	t.Cleanup(func() {
		Owner = prevOwner
		TransportIdentity = prevTransportIdentity
		blackholedIdentities = prevBlackholed
	})

	r := &Reticulum{}
	victim, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(victim): %v", err)
	}

	reason := "rpc"
	blackholeConn := &fakeRPCConn{recv: map[string]any{
		"blackhole_identity": victim.Hash,
		"reason":             reason,
	}}
	r.handleRPC(blackholeConn)
	if len(blackholeConn.sent) != 1 {
		t.Fatalf("blackhole RPC sent %d responses, want 1", len(blackholeConn.sent))
	}
	if got := blackholeConn.sent[0]; got != true {
		t.Fatalf("blackhole RPC response=%#v, want true", blackholeConn.sent[0])
	}

	getConn := &fakeRPCConn{recv: map[string]any{"get": "blackholed_identities"}}
	r.handleRPC(getConn)
	if len(getConn.sent) != 1 {
		t.Fatalf("get RPC sent %d responses, want 1", len(getConn.sent))
	}
	snapshot, ok := getConn.sent[0].(map[string]any)
	if !ok {
		t.Fatalf("get RPC response type=%T, want map[string]any", getConn.sent[0])
	}
	key := hex.EncodeToString(victim.Hash[:truncatedHashBytes])
	if _, exists := snapshot[key]; !exists {
		t.Fatal("expected blackholed identity in RPC snapshot")
	}

	unblackholeConn := &fakeRPCConn{recv: map[string]any{"unblackhole_identity": victim.Hash}}
	r.handleRPC(unblackholeConn)
	if len(unblackholeConn.sent) != 1 {
		t.Fatalf("unblackhole RPC sent %d responses, want 1", len(unblackholeConn.sent))
	}
	if got, ok := unblackholeConn.sent[0].(bool); !ok || !got {
		t.Fatalf("unblackhole RPC response=%#v, want true", unblackholeConn.sent[0])
	}
	if len(blackholedIdentities) != 0 {
		t.Fatalf("expected empty blackhole state after RPC unblackhole, got %#v", blackholedIdentities)
	}
}
