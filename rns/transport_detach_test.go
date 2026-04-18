package rns

import (
	"sync"
	"testing"
	"time"
)

func TestDetachInterfaces_MirrorsPythonOrder(t *testing.T) {
	prevInterfaces := Interfaces
	prevLocalClients := LocalClientInterfaces
	Interfaces = nil
	LocalClientInterfaces = nil
	t.Cleanup(func() {
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClients
	})

	var mu sync.Mutex
	order := make([]string, 0, 4)
	record := func(label string) {
		mu.Lock()
		order = append(order, label)
		mu.Unlock()
	}

	nonLocalStarted := make(chan struct{})
	nonLocalRelease := make(chan struct{})
	nonLocal := &Interface{Name: "udp0", Type: "UDPInterface"}
	nonLocal.SetDetachFunc(func() {
		close(nonLocalStarted)
		<-nonLocalRelease
		record("nonlocal")
	})

	localClient := &Interface{Name: "local-client", Type: "LocalInterface", LocalIsSharedClient: true}
	localClient.SetDetachFunc(func() {
		record("local-client")
	})

	sharedMaster := &Interface{Name: "shared-master", Type: "LocalInterface"}
	sharedMaster.SetDetachFunc(func() {
		record("shared-master")
	})

	Interfaces = append(Interfaces, nonLocal, localClient, sharedMaster)
	LocalClientInterfaces = append(LocalClientInterfaces, localClient)

	done := make(chan struct{})
	go func() {
		DetachInterfaces()
		close(done)
	}()

	select {
	case <-nonLocalStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("non-local detach did not start")
	}

	select {
	case <-done:
		t.Fatal("DetachInterfaces returned before non-local detach completed")
	case <-time.After(100 * time.Millisecond):
	}

	close(nonLocalRelease)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("DetachInterfaces did not complete")
	}

	mu.Lock()
	got := append([]string(nil), order...)
	mu.Unlock()

	want := []string{"nonlocal", "local-client", "local-client", "shared-master"}
	if len(got) != len(want) {
		t.Fatalf("unexpected detach order length: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected detach order: got %v want %v", got, want)
		}
	}
}
