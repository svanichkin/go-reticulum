package interfaces

import (
	"bytes"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestI2PIntegration_SpawnIncoming_InheritsIngressControl(t *testing.T) {
	prevSpawn := SpawnHandler
	t.Cleanup(func() { SpawnHandler = prevSpawn })

	parent := &Interface{Name: "i2p0", Type: "I2PInterface", IngressControl: false, Bitrate: 1234, HWMTU: 4096}
	driver := &I2PClientDriver{iface: parent}

	serverSide, clientSide := net.Pipe()
	defer clientSide.Close()

	got := make(chan *Interface, 1)
	SpawnHandler = func(ifc *Interface) { got <- ifc }

	driver.spawnIncoming(serverSide)

	select {
	case peerIface := <-got:
		if peerIface.IngressControl != parent.IngressControl {
			t.Fatalf("peer IngressControl=%v, want %v", peerIface.IngressControl, parent.IngressControl)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SpawnHandler")
	}
}

func TestI2PIntegration_NewInitiatorPeer_InheritsIngressControl(t *testing.T) {
	parent := &Interface{Name: "i2p0", Type: "I2PInterface", IngressControl: false, Bitrate: 1234, HWMTU: 4096}
	driver := &I2PClientDriver{iface: parent}

	peerIface, err := driver.NewInitiatorPeer("peer.example.b32.i2p")
	if err != nil {
		t.Fatalf("NewInitiatorPeer: %v", err)
	}
	if peerIface.IngressControl != parent.IngressControl {
		t.Fatalf("peer IngressControl=%v, want %v", peerIface.IngressControl, parent.IngressControl)
	}
}

func TestI2PIntegration_Peer_ProcessOutgoing_FramesHDLC(t *testing.T) {
	// Does not require an I2P daemon; validates framing on the wire.

	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()

	parent := &Interface{Name: "i2p0", Type: "I2PInterface"}
	iface := &Interface{Name: "i2p0peer", Type: "I2PInterfacePeer", Parent: parent, HWMTU: 1 << 16, Online: true}
	peer := &I2PPeer{iface: iface, conn: serverSide, kissFraming: false, parentCount: true}
	peer.connGen.Store(1)

	payload := []byte{0x01, hdlcFlag, 0x02, hdlcEsc, 0x03}
	want := append([]byte{hdlcFlag}, hdlcEscape(payload)...)
	want = append(want, hdlcFlag)

	gotCh := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 64)
		_ = clientSide.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, _ := clientSide.Read(buf)
		gotCh <- buf[:n]
	}()

	peer.ProcessOutgoing(payload)

	got := <-gotCh
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected frame:\nwant %x\ngot  %x", want, got)
	}
	if iface.TXB != uint64(len(want)) {
		t.Fatalf("unexpected TXB: want %d got %d", len(want), iface.TXB)
	}
	if parent.TXB != uint64(len(want)) {
		t.Fatalf("unexpected parent TXB: want %d got %d", len(want), parent.TXB)
	}
}

func TestI2PIntegration_Driver_Close_TeardownSpawnedPeers(t *testing.T) {
	// Does not require an I2P daemon; validates Close() cleans up peers.

	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() { RemoveInterfaceHandler = oldRemove })
	RemoveInterfaceHandler = func(*Interface) {}

	driver := &I2PClientDriver{iface: &Interface{Name: "i2p0", Type: "I2PInterface"}, stopCh: make(chan struct{})}

	serverSide, clientSide := net.Pipe()
	defer clientSide.Close()

	peerIface := &Interface{Name: "peer0", Type: "I2PInterfacePeer", Parent: driver.iface, HWMTU: 1 << 16, Online: true}
	peer := &I2PPeer{parent: driver, iface: peerIface, conn: serverSide, initiator: false}
	peerIface.i2pPeer = peer

	driver.mu.Lock()
	driver.spawned = append(driver.spawned, peerIface)
	driver.mu.Unlock()

	driver.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if peer.closed.Load() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected spawned peer to be closed on driver.Close()")
}

func TestI2PIntegration_ConnectableEndpoint_SetsB32(t *testing.T) {
	// Requires a local I2P router with SAM enabled.
	// Enable by setting: RUN_I2P_INTEGRATION=1 (optionally I2P_SAM_HOST, I2P_SAM_PORT).

	if os.Getenv("RUN_I2P_INTEGRATION") != "1" {
		t.Skip("set RUN_I2P_INTEGRATION=1 to run")
	}

	samHost := os.Getenv("I2P_SAM_HOST")
	if samHost == "" {
		samHost = "127.0.0.1"
	}
	samPort := os.Getenv("I2P_SAM_PORT")
	if samPort == "" {
		samPort = "7656"
	}
	addr := net.JoinHostPort(samHost, samPort)
	if c, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err != nil {
		t.Skipf("SAM not reachable at %s: %v", addr, err)
	} else {
		_ = c.Close()
	}

	kv := map[string]string{
		"storagepath":  t.TempDir(),
		"connectable":  "true",
		"sam_host":     samHost,
		"sam_port":     samPort,
		"kiss_framing": "false",
	}
	ifc, err := NewI2PInterface("i2p-test", kv)
	if err != nil {
		t.Fatalf("NewI2PInterface: %v", err)
	}
	t.Cleanup(func() {
		if ifc != nil && ifc.i2pClient != nil {
			ifc.i2pClient.Close()
		}
	})

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if b32 := ifc.I2PB32(); b32 != nil {
			if strings.TrimSpace(*b32) == "" {
				t.Fatalf("got empty b32")
			}
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for b32; check SAM/I2P router logs")
}
