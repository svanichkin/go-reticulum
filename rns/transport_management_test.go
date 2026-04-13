package rns

import (
	"testing"
	"time"
)

type managementAnnounceBackend struct {
	outboundCalls int
}

func (b *managementAnnounceBackend) Outbound(p *Packet) bool {
	if p != nil && p.PacketType == PacketANNOUNCE {
		b.outboundCalls++
	}
	return true
}

func (b *managementAnnounceBackend) HopsTo(_ []byte) int {
	return PathfinderMaxHops
}

func (b *managementAnnounceBackend) GetFirstHopTimeout(_ []byte) time.Duration {
	return 0
}

func (b *managementAnnounceBackend) GetPacketRSSI(_ []byte) *float64 {
	return nil
}

func (b *managementAnnounceBackend) GetPacketSNR(_ []byte) *float64 {
	return nil
}

func (b *managementAnnounceBackend) GetPacketQ(_ []byte) *float64 {
	return nil
}

func TestRefreshManagementDestinations_IncludesRemoteAndProbe(t *testing.T) {
	prevRemote := remoteManagementDest
	prevProbe := ProbeDestination
	prevMgmt := mgmtDestinations
	prevActive := remoteManagementActive

	remoteManagementDest = &Destination{}
	ProbeDestination = &Destination{}
	remoteManagementActive = true
	mgmtDestinations = nil

	t.Cleanup(func() {
		remoteManagementDest = prevRemote
		ProbeDestination = prevProbe
		mgmtDestinations = prevMgmt
		remoteManagementActive = prevActive
	})

	refreshManagementDestinations()

	if len(mgmtDestinations) != 2 {
		t.Fatalf("mgmtDestinations len=%d, want 2", len(mgmtDestinations))
	}
	if mgmtDestinations[0] != remoteManagementDest || mgmtDestinations[1] != ProbeDestination {
		t.Fatalf("unexpected management destinations ordering/content")
	}
}

func TestConfigureControlDestinations_DoesNotCreateProbeWhenTransportDisabled(t *testing.T) {
	prevOwner := Owner
	prevTransportEnabled := transportEnabled
	prevAllowProbes := allowProbes
	prevIdentity := TransportIdentity
	prevProbe := ProbeDestination
	prevDests := Destinations
	prevMgmt := mgmtDestinations

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	Owner = &Reticulum{}
	transportEnabled = false
	allowProbes = true
	TransportIdentity = id
	ProbeDestination = nil
	Destinations = nil
	mgmtDestinations = nil

	t.Cleanup(func() {
		Owner = prevOwner
		transportEnabled = prevTransportEnabled
		allowProbes = prevAllowProbes
		TransportIdentity = prevIdentity
		ProbeDestination = prevProbe
		Destinations = prevDests
		mgmtDestinations = prevMgmt
	})

	configureControlDestinations()

	if ProbeDestination != nil {
		t.Fatalf("ProbeDestination was created with transport disabled")
	}
}

func TestDueManagementDestinations_DeferredInitialAnnounce(t *testing.T) {
	prevMgmt := mgmtDestinations
	prevLast := LastMgmtAnnounce

	mgmtDestinations = []*Destination{{}}
	now := time.Now()
	LastMgmtAnnounce = now.Add(-(mgmtAnnounceInterval - initialMgmtAnnounceWait))

	t.Cleanup(func() {
		mgmtDestinations = prevMgmt
		LastMgmtAnnounce = prevLast
	})

	if due := dueManagementDestinations(now); len(due) != 0 {
		t.Fatalf("dueManagementDestinations() len=%d, want 0 before initial delay", len(due))
	}

	later := now.Add(initialMgmtAnnounceWait + time.Millisecond)
	if due := dueManagementDestinations(later); len(due) != 1 {
		t.Fatalf("dueManagementDestinations() len=%d, want 1 after initial delay", len(due))
	}
}

func TestAnnounceManagementDestinations_SendsAnnounces(t *testing.T) {
	prevTransport := Transport

	backend := &managementAnnounceBackend{}
	Transport = backend

	t.Cleanup(func() {
		Transport = prevTransport
	})

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	remote, err := NewDestination(id, DestinationIN, DestinationSINGLE, TransportAppName, "remote", "management")
	if err != nil {
		t.Fatalf("NewDestination(remote): %v", err)
	}
	probe, err := NewDestination(id, DestinationIN, DestinationSINGLE, TransportAppName, "probe")
	if err != nil {
		t.Fatalf("NewDestination(probe): %v", err)
	}

	announceManagementDestinations([]*Destination{remote, probe})

	if backend.outboundCalls != 2 {
		t.Fatalf("announce outbound calls=%d, want 2", backend.outboundCalls)
	}
}
