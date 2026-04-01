package rns

import (
	ifaces "github.com/svanichkin/go-reticulum/rns/interfaces"
	"testing"
	"time"
)

func TestRunInterfaceJobs_RespectsPythonInterval(t *testing.T) {
	prevInterfaces := Interfaces
	prevLast := InterfaceLastJobs
	prevInterval := InterfaceJobsInterval
	prevInbound := ifaces.InboundHandler

	slow := &Interface{Name: "slow", Bitrate: 10}
	fast := &Interface{Name: "fast", Bitrate: 100}
	Interfaces = []*Interface{slow, fast}
	InterfaceJobsInterval = 5 * time.Second

	released := make(chan []byte, 1)
	ifaces.InboundHandler = func(raw []byte, _ *Interface) {
		released <- append([]byte(nil), raw...)
	}

	destHash := make([]byte, truncatedHashBytes)
	raw := []byte{0x01, 0x02, 0x03}
	slow.HoldAnnounce(raw, nil, destHash, 1)

	t.Cleanup(func() {
		Interfaces = prevInterfaces
		InterfaceLastJobs = prevLast
		InterfaceJobsInterval = prevInterval
		ifaces.InboundHandler = prevInbound
	})

	now := time.Now()
	InterfaceLastJobs = now

	if ran := runInterfaceJobs(now.Add(time.Second)); ran {
		t.Fatal("runInterfaceJobs() ran before interface_jobs_interval elapsed")
	}
	if slow.HeldAnnouncesCount() != 1 {
		t.Fatalf("held announces=%d, want 1 before interval", slow.HeldAnnouncesCount())
	}
	if Interfaces[0] != slow || Interfaces[1] != fast {
		t.Fatal("interfaces were reprioritized before interval elapsed")
	}

	if ran := runInterfaceJobs(now.Add(InterfaceJobsInterval + time.Millisecond)); !ran {
		t.Fatal("runInterfaceJobs() did not run after interface_jobs_interval elapsed")
	}
	if slow.HeldAnnouncesCount() != 0 {
		t.Fatalf("held announces=%d, want 0 after interval", slow.HeldAnnouncesCount())
	}
	if Interfaces[0] != fast || Interfaces[1] != slow {
		t.Fatal("interfaces were not reprioritized by bitrate when interface jobs ran")
	}

	select {
	case got := <-released:
		if string(got) != string(raw) {
			t.Fatalf("released raw=%x, want %x", got, raw)
		}
	case <-time.After(time.Second):
		t.Fatal("held announce was not released")
	}
}
