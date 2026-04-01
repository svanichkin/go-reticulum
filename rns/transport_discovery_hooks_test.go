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

func TestEnsureNetworkDestinations_AddsManagementDestinations(t *testing.T) {
	prevOwner := Owner
	prevNetworkIdentity := NetworkIdentity
	prevInstanceDestination := instanceDestination
	prevNetworkDestination := networkDestination
	prevMgmtDestinations := mgmtDestinations
	prevProbe := ProbeDestination
	prevRemoteManagementDest := remoteManagementDest
	prevRemoteManagementActive := remoteManagementActive
	prevDestinations := Destinations

	Owner = &Reticulum{}
	NetworkIdentity = nil
	instanceDestination = nil
	networkDestination = nil
	mgmtDestinations = nil
	ProbeDestination = nil
	remoteManagementDest = nil
	remoteManagementActive = false
	Destinations = nil

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
	})

	networkID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(network): %v", err)
	}
	NetworkIdentity = networkID

	ensureNetworkDestinations()
	refreshManagementDestinations()

	if instanceDestination == nil {
		t.Fatal("instanceDestination was not created")
	}
	if networkDestination == nil {
		t.Fatal("networkDestination was not created")
	}
	if len(mgmtDestinations) != 2 {
		t.Fatalf("mgmtDestinations len=%d, want 2", len(mgmtDestinations))
	}
	if mgmtDestinations[0] != instanceDestination || mgmtDestinations[1] != networkDestination {
		t.Fatal("network management destinations missing or misordered")
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
