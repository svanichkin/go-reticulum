package main

import (
	"testing"

	rns "github.com/svanichkin/go-reticulum/rns"
)

func TestEchoExample_RequestGetsProof(t *testing.T) {
	serverID, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(server): %v", err)
	}
	serverDest, err := rns.NewDestination(serverID, rns.DestinationIN, rns.DestinationSINGLE, appName, "echo", "request")
	if err != nil {
		t.Fatalf("NewDestination(server): %v", err)
	}
	_ = serverDest.SetProofStrategy(rns.DestinationPROVE_ALL)

	clientDest, err := rns.NewDestination(serverID, rns.DestinationOUT, rns.DestinationSINGLE, appName, "echo", "request")
	if err != nil {
		t.Fatalf("NewDestination(client): %v", err)
	}

	req := rns.NewPacket(clientDest, rns.IdentityGetRandomHash())
	if req == nil {
		t.Fatalf("NewPacket returned nil")
	}
	if err := req.Pack(); err != nil {
		t.Fatalf("pack: %v", err)
	}
	rc := rns.NewPacketReceipt(req)
	if rc == nil {
		t.Fatalf("NewPacketReceipt returned nil")
	}
	req.Receipt = rc

	packetHash := req.GetHash()
	sig, err := serverID.Sign(packetHash)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	proof := rns.NewPacket(req.GenerateProofDestination(), append(append([]byte(nil), packetHash...), sig...), rns.WithPacketType(rns.PacketTypeProof), rns.WithCreateReceipt(false))
	if proof == nil {
		t.Fatalf("proof packet nil")
	}
	if err := proof.Pack(); err != nil {
		t.Fatalf("proof pack: %v", err)
	}
	if ok := rc.ValidateProofPacket(proof); !ok {
		t.Fatalf("expected proof validation ok")
	}
	if rc.Status != rns.ReceiptDelivered {
		t.Fatalf("expected receipt delivered, got %d", rc.Status)
	}
	if rc.GetRTT() <= 0 {
		t.Fatalf("expected non-zero RTT")
	}
}
