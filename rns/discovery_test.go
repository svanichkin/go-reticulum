package rns

import (
	"bytes"
	"testing"
	"time"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

type fakeDiscoveryStamper struct {
	stamp []byte
	value int
}

func (f fakeDiscoveryStamper) StampSize() int {
	return len(f.stamp)
}

func (f fakeDiscoveryStamper) GenerateStamp(_ []byte, _ int, _ int) ([]byte, int, error) {
	return append([]byte(nil), f.stamp...), f.value, nil
}

func (f fakeDiscoveryStamper) StampWorkblock(_ []byte, _ int) []byte {
	return []byte("work")
}

func (f fakeDiscoveryStamper) StampValue(_ []byte, _ []byte) int {
	return f.value
}

func (f fakeDiscoveryStamper) StampValid(stamp []byte, requiredValue int, _ []byte) bool {
	return bytes.Equal(stamp, f.stamp) && f.value >= requiredValue
}

func TestInterfaceAnnouncer_GetInterfaceAnnounceData_TCPServer(t *testing.T) {
	prevStamper := DiscoveryStampProvider
	prevTransportIdentity := TransportIdentity
	prevNetworkIdentity := NetworkIdentity

	DiscoveryStampProvider = fakeDiscoveryStamper{stamp: []byte{1, 2, 3, 4}, value: 21}
	NetworkIdentity = nil
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	TransportIdentity = id

	t.Cleanup(func() {
		DiscoveryStampProvider = prevStamper
		TransportIdentity = prevTransportIdentity
		NetworkIdentity = prevNetworkIdentity
	})

	port := 4242
	ifc := &Interface{
		Type:                      "TCPServerInterface",
		Name:                      "srv0",
		Discoverable:              true,
		DiscoveryReachableOn:      "example.com",
		DiscoveryPort:             &port,
		DiscoveryPublishIFAC:      true,
		IFACNetnameVal:            "mesh",
		IFACNetkeyVal:             "secret",
		DiscoveryAnnounceInterval: 6 * time.Hour,
	}

	announcer := NewInterfaceAnnouncer()
	data, err := announcer.GetInterfaceAnnounceData(ifc)
	if err != nil {
		t.Fatalf("GetInterfaceAnnounceData: %v", err)
	}
	if len(data) < 1+4 {
		t.Fatalf("announce data too short: %d", len(data))
	}
	if data[0] != 0 {
		t.Fatalf("flags=%d, want 0", data[0])
	}

	raw := map[any]any{}
	if err := umsgpack.Unpackb(data[1:len(data)-4], &raw); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	if got := asDiscoveryString(discoveryRawGet(raw, discoveryFieldInterfaceType)); got != "TCPServerInterface" {
		t.Fatalf("interface type=%q, want TCPServerInterface", got)
	}
	if got := asDiscoveryString(discoveryRawGet(raw, discoveryFieldReachableOn)); got != "example.com" {
		t.Fatalf("reachable_on=%q, want example.com", got)
	}
	if got := asDiscoveryInt(discoveryRawGet(raw, discoveryFieldPort)); got != 4242 {
		t.Fatalf("port=%d, want 4242", got)
	}
	if got := asDiscoveryString(discoveryRawGet(raw, discoveryFieldIFACNetname)); got != "mesh" {
		t.Fatalf("ifac_netname=%q, want mesh", got)
	}
	if got := asDiscoveryString(discoveryRawGet(raw, discoveryFieldIFACNetkey)); got != "secret" {
		t.Fatalf("ifac_netkey=%q, want secret", got)
	}
}

func TestInterfaceDiscovery_PersistsDiscoveredInterfaceFromAnnounce(t *testing.T) {
	prevStamper := DiscoveryStampProvider
	prevTransportIdentity := TransportIdentity
	prevNetworkIdentity := NetworkIdentity
	prevOwner := Owner
	prevHandlers := announceHandlers

	DiscoveryStampProvider = fakeDiscoveryStamper{stamp: []byte{9, 9, 9, 9}, value: 22}
	announceHandlers = nil
	NetworkIdentity = nil
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	TransportIdentity = id
	Owner = &Reticulum{StoragePath: t.TempDir()}

	t.Cleanup(func() {
		DiscoveryStampProvider = prevStamper
		TransportIdentity = prevTransportIdentity
		NetworkIdentity = prevNetworkIdentity
		Owner = prevOwner
		announceHandlers = prevHandlers
	})

	discovery, err := NewInterfaceDiscovery(14, nil, true)
	if err != nil {
		t.Fatalf("NewInterfaceDiscovery: %v", err)
	}
	t.Cleanup(func() {
		if discovery.handler != nil {
			DeregisterAnnounceHandler(discovery.handler)
		}
	})

	port := 7777
	ifc := &Interface{
		Type:                 "TCPServerInterface",
		Name:                 "public-tcp",
		Discoverable:         true,
		DiscoveryReachableOn: "reticulum.example",
		DiscoveryPort:        &port,
	}
	announcer := NewInterfaceAnnouncer()
	appData, err := announcer.GetInterfaceAnnounceData(ifc)
	if err != nil {
		t.Fatalf("GetInterfaceAnnounceData: %v", err)
	}

	discoveryDest, err := NewDestination(TransportIdentity, DestinationIN, DestinationSINGLE, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	packet := discoveryDest.Announce(appData, false, nil, nil, false)
	if packet == nil {
		t.Fatal("Announce returned nil")
	}
	if err := packet.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}

	NotifyAnnounceHandlers(packet)

	list := discovery.ListDiscoveredInterfaces(false, false)
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
	if got := asDiscoveryInt(info["port"]); got != 7777 {
		t.Fatalf("port=%v, want 7777", info["port"])
	}
}
