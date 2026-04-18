package rns

import "testing"

type countingTransportWorker struct {
	starts *int
}

func (w *countingTransportWorker) Start() {
	if w != nil && w.starts != nil {
		(*w.starts)++
	}
}

func TestSetNetworkIdentity_OnlySetsOnce(t *testing.T) {
	prevNetworkIdentity := NetworkIdentity
	NetworkIdentity = nil
	t.Cleanup(func() {
		NetworkIdentity = prevNetworkIdentity
	})

	first, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(first): %v", err)
	}
	second, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(second): %v", err)
	}

	if HasNetworkIdentity() {
		t.Fatal("HasNetworkIdentity() = true before set")
	}

	SetNetworkIdentity(first)
	SetNetworkIdentity(second)

	if !HasNetworkIdentity() {
		t.Fatal("HasNetworkIdentity() = false after set")
	}
	if NetworkIdentity != first {
		t.Fatal("SetNetworkIdentity() did not preserve first identity")
	}
}

func TestSharedConnectionDisappeared_AddsManagementDestinations(t *testing.T) {
	prevOwner := Owner
	prevNetworkIdentity := NetworkIdentity
	prevInstanceDestination := instanceDestination
	prevNetworkDestination := networkDestination
	prevMgmtDestinations := mgmtDestinations
	prevProbe := ProbeDestination
	prevRemoteManagementDest := remoteManagementDest
	prevRemoteManagementActive := remoteManagementActive
	prevDestinations := Destinations
	prevAnnounceTable := announceTable
	prevPathTable := pathTable
	prevReverseTable := reverseTable
	prevLinkTable := linkTable
	prevHeldAnnounces := heldAnnounces
	prevTunnels := tunnels

	Owner = &Reticulum{}
	NetworkIdentity = nil
	instanceDestination = nil
	networkDestination = nil
	mgmtDestinations = nil
	ProbeDestination = nil
	remoteManagementDest = nil
	remoteManagementActive = false
	Destinations = nil
	announceTable = make(map[hashKey]*announceEntry)
	pathTable = make(map[hashKey]*PathEntry)
	reverseTable = make(map[hashKey]*reverseEntry)
	linkTable = make(map[hashKey]*linkEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)
	tunnels = make(map[string]*tunnelEntry)
	var junkKey hashKey
	junkKey[0] = 1
	announceTable[junkKey] = &announceEntry{}
	pathTable[junkKey] = &PathEntry{}
	reverseTable[junkKey] = &reverseEntry{}
	linkTable[junkKey] = &linkEntry{}
	heldAnnounces[junkKey] = &heldAnnounce{}
	tunnels["junk"] = &tunnelEntry{}

	t.Cleanup(func() {
		Owner = prevOwner
		NetworkIdentity = prevNetworkIdentity
		instanceDestination = prevInstanceDestination
		networkDestination = prevNetworkDestination
		mgmtDestinations = prevMgmtDestinations
		ProbeDestination = prevProbe
		remoteManagementDest = prevRemoteManagementDest
		remoteManagementActive = prevRemoteManagementActive
		Destinations = prevDestinations
		announceTable = prevAnnounceTable
		pathTable = prevPathTable
		reverseTable = prevReverseTable
		linkTable = prevLinkTable
		heldAnnounces = prevHeldAnnounces
		tunnels = prevTunnels
	})

	networkID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(network): %v", err)
	}
	NetworkIdentity = networkID

	SharedConnectionDisappeared()

	if len(announceTable) != 0 {
		t.Fatalf("announceTable len=%d, want 0", len(announceTable))
	}
	if len(pathTable) != 0 {
		t.Fatalf("pathTable len=%d, want 0", len(pathTable))
	}
	if len(reverseTable) != 0 {
		t.Fatalf("reverseTable len=%d, want 0", len(reverseTable))
	}
	if len(linkTable) != 0 {
		t.Fatalf("linkTable len=%d, want 0", len(linkTable))
	}
	if len(heldAnnounces) != 0 {
		t.Fatalf("heldAnnounces len=%d, want 0", len(heldAnnounces))
	}
	if len(tunnels) != 0 {
		t.Fatalf("tunnels len=%d, want 0", len(tunnels))
	}
	if instanceDestination != nil {
		t.Fatal("instanceDestination should not be created by shared_connection_disappeared")
	}
	if networkDestination != nil {
		t.Fatal("networkDestination should not be created by shared_connection_disappeared")
	}
	if len(mgmtDestinations) != 0 {
		t.Fatalf("mgmtDestinations len=%d, want 0", len(mgmtDestinations))
	}
}

func TestDiscoveryHooks_InstantiateOnlyOnce(t *testing.T) {
	prevInterfaceAnnouncer := interfaceAnnouncer
	prevDiscoveryHandler := discoveryHandler
	prevBlackholeUpdater := blackholeUpdater
	prevInterfaceAnnouncerFactory := interfaceAnnouncerFactory
	prevDiscoveryHandlerFactory := discoveryHandlerFactory
	prevBlackholeUpdaterFactory := blackholeUpdaterFactory
	prevDiscoveryRequiredValue := discoveryRequiredValue

	interfaceAnnouncer = nil
	discoveryHandler = nil
	blackholeUpdater = nil
	discoveryRequiredValue = 23

	var announcerStarts int
	var updaterStarts int
	var handlerCalls int
	var gotRequiredValue int
	var gotDiscoverInterfaces bool

	interfaceAnnouncerFactory = func() transportBackgroundWorker {
		return &countingTransportWorker{starts: &announcerStarts}
	}
	discoveryHandlerFactory = func(requiredValue int, discoverInterfaces bool) any {
		handlerCalls++
		gotRequiredValue = requiredValue
		gotDiscoverInterfaces = discoverInterfaces
		return struct{}{}
	}
	blackholeUpdaterFactory = func() transportBackgroundWorker {
		return &countingTransportWorker{starts: &updaterStarts}
	}

	t.Cleanup(func() {
		interfaceAnnouncer = prevInterfaceAnnouncer
		discoveryHandler = prevDiscoveryHandler
		blackholeUpdater = prevBlackholeUpdater
		interfaceAnnouncerFactory = prevInterfaceAnnouncerFactory
		discoveryHandlerFactory = prevDiscoveryHandlerFactory
		blackholeUpdaterFactory = prevBlackholeUpdaterFactory
		discoveryRequiredValue = prevDiscoveryRequiredValue
	})

	EnableDiscovery()
	EnableDiscovery()
	DiscoverInterfaces()
	DiscoverInterfaces()
	EnableBlackholeUpdater()
	EnableBlackholeUpdater()

	if announcerStarts != 1 {
		t.Fatalf("interface announcer starts=%d, want 1", announcerStarts)
	}
	if updaterStarts != 1 {
		t.Fatalf("blackhole updater starts=%d, want 1", updaterStarts)
	}
	if handlerCalls != 1 {
		t.Fatalf("discovery handler factory calls=%d, want 1", handlerCalls)
	}
	if gotRequiredValue != 23 {
		t.Fatalf("discovery required value=%d, want 23", gotRequiredValue)
	}
	if !gotDiscoverInterfaces {
		t.Fatal("discoverInterfaces flag=false, want true")
	}
}
