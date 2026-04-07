package rns

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

type fakeRPCConn struct {
	recv   map[string]any
	sent   []any
	closed bool
}

func (c *fakeRPCConn) Recv(v interface{}) error {
	call, ok := v.(*map[string]any)
	if !ok {
		return nil
	}
	*call = c.recv
	return nil
}

func (c *fakeRPCConn) Send(v interface{}) error {
	c.sent = append(c.sent, v)
	return nil
}

func (c *fakeRPCConn) Close() error {
	c.closed = true
	return nil
}

func TestReticulumBlackholeIdentity_LocalAPI(t *testing.T) {
	prevOwner := Owner
	prevTransportIdentity := TransportIdentity
	prevBlackholed := blackholedIdentities

	dir := t.TempDir()
	Owner = &Reticulum{StoragePath: dir}
	TransportIdentity, _ = NewIdentity()
	blackholedIdentities = make(map[hashKey]*blackholeEntry)

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
	if ok := r.BlackholeIdentity(victim.Hash, &until, &reason); !ok {
		t.Fatal("BlackholeIdentity() = false, want true")
	}

	snapshot := r.GetBlackholedIdentities()
	key, _ := makeHashKey(victim.Hash)
	entry, exists := snapshot[key]
	if !exists {
		t.Fatal("expected blackholed identity in snapshot")
	}
	if got, _ := entry["reason"].(string); got != "local-api" {
		t.Fatalf("snapshot reason=%q, want local-api", got)
	}

	if ok := r.UnblackholeIdentity(victim.Hash); !ok {
		t.Fatal("UnblackholeIdentity() = false, want true")
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
	if got, ok := blackholeConn.sent[0].(bool); !ok || !got {
		t.Fatalf("blackhole RPC response=%#v, want true", blackholeConn.sent[0])
	}

	getConn := &fakeRPCConn{recv: map[string]any{"get": "blackholed_identities"}}
	r.handleRPC(getConn)
	if len(getConn.sent) != 1 {
		t.Fatalf("get RPC sent %d responses, want 1", len(getConn.sent))
	}
	snapshot, ok := getConn.sent[0].(map[hashKey]map[string]any)
	if !ok {
		t.Fatalf("get RPC response type=%T, want map[hashKey]map[string]any", getConn.sent[0])
	}
	key, _ := makeHashKey(victim.Hash)
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
