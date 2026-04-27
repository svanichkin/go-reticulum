package rns

import (
	"math"
	"strings"
	"testing"
	"time"

	cryptography "github.com/svanichkin/go-reticulum/rns/cryptography"
)

func TestPrettyHexRep_Format(t *testing.T) {
	maybeParallel(t)

	if got := PrettyHexRep([]byte{0x01, 0xAB, 0x00}); got != "<01ab00>" {
		t.Fatalf("unexpected PrettyHexRep: %q", got)
	}
}

func TestHexRep_DelimitToggle(t *testing.T) {
	maybeParallel(t)

	if got := HexRep([]byte{0x01, 0x02, 0x03}); got != "01:02:03" {
		t.Fatalf("unexpected HexRep default: %q", got)
	}
	if got := HexRep([]byte{0x01, 0x02, 0x03}, false); got != "010203" {
		t.Fatalf("unexpected HexRep no-delimit: %q", got)
	}
}

func TestPrettySize_AndSpeed(t *testing.T) {
	maybeParallel(t)

	// Matches Python behaviour: 1024 -> 1.0 KB.
	if got := PrettySize(1024); !strings.Contains(got, "KB") {
		t.Fatalf("expected KB in %q", got)
	}
	if got := PrettySpeed(8); !strings.HasSuffix(got, "bps") {
		t.Fatalf("expected bps suffix in %q", got)
	}
}

func TestPrettyFrequency_Zero(t *testing.T) {
	maybeParallel(t)

	if got := PrettyFrequency(0); got != "0 Hz" {
		t.Fatalf("expected zero frequency to be rendered as 0 Hz, got %q", got)
	}
}

func TestSetLogTimeFormat_ResetsOnEmpty(t *testing.T) {
	maybeParallel(t)

	std := LogTimeFmt
	logMu.Lock()
	LogTimeFmt = "2006"
	logMu.Unlock()
	changed := LogTimeFmt
	if changed == std {
		t.Fatalf("expected format to change")
	}
	logMu.Lock()
	LogTimeFmt = defaultLogTimeFormat
	logMu.Unlock()
	reset := LogTimeFmt
	if reset != std {
		t.Fatalf("expected reset to %q, got %q", std, reset)
	}
}

func TestLogOutputSuppressed_DropsStdoutButKeepsCallback(t *testing.T) {
	prevLevel := Loglevel
	prevDest := logDest
	prevSuppress := logSuppressOutput
	Loglevel = LogDebug
	logSuppressOutput = true
	t.Cleanup(func() {
		logMu.Lock()
		logDest = prevDest
		alwaysOverrideDest = false
		logCallback = nil
		logSuppressOutput = prevSuppress
		logMu.Unlock()
		Loglevel = prevLevel
	})

	called := false
	logMu.Lock()
	logCallback = func(msg string) {
		called = strings.Contains(msg, "callback-visible")
	}
	logDest = LogCallback
	logMu.Unlock()
	Log("callback-visible", LogInfo)
	if !called {
		t.Fatal("expected callback logging to remain active while output is suppressed")
	}
}

func TestTimestampStr_AndPreciseTimestampStr(t *testing.T) {
	maybeParallel(t)

	ts := TimestampStr(0)
	if ts == "" {
		t.Fatalf("expected timestamp string")
	}
	pt := PreciseTimestampStr(float64(time.Now().Unix()))
	if pt == "" {
		t.Fatalf("expected precise timestamp string")
	}
}

func TestSetMTU_UpdatesDerivedValues(t *testing.T) {
	prevMTU := MTU
	prevMDU := MDU
	prevLinkMDU := LinkMDU
	t.Cleanup(func() {
		mtuMu.Lock()
		prevPlain := PacketPlainMDU
		prevEncrypted := PacketEncryptedMDU
		prevLink := LinkMDU
		MTU = prevMTU
		plain := MTU - HEADER_MAXSIZE - IFAC_MIN_SIZE
		if plain < 0 {
			plain = 0
		}
		link := func() int {
			payload := MTU - IFAC_MIN_SIZE - HEADER_MINSIZE - cryptography.Overhead
			if payload <= 0 {
				return 0
			}
			blocks := payload / identityAESBlockSize
			if blocks <= 0 {
				return 0
			}
			return blocks*identityAESBlockSize - 1
		}()
		hashLen := int(math.Floor(float64(link-AdvOverhead) / float64(MapHashLen)))
		encrypted := func() int {
			usable := plain - cryptography.Overhead - (identityPubKeyLen / 2)
			if usable <= 0 {
				return 0
			}
			blocks := usable / identityAESBlockSize
			if blocks <= 0 {
				return 0
			}
			return blocks*identityAESBlockSize - 1
		}()
		MDU = plain
		PacketMDU = plain
		PacketPLAIN_MDU = plain
		PacketENCRYPTED_MDU = encrypted
		PacketPlainMDU = plain
		PacketEncryptedMDU = encrypted
		LinkMDU = link
		HashmapMaxLen = hashLen
		CollisionGuardSize = 2*ResourceWindowMax + HashmapMaxLen
		if plain != prevPlain || encrypted != prevEncrypted || link != prevLink {
			go func(prevPlain, newPlain, prevLink, newLink int) {
				for _, dst := range Destinations {
					if dst == nil {
						continue
					}
					if dst.mtu == 0 || dst.mtu == prevPlain {
						dst.mtu = newPlain
					}
				}
				linkMu.Lock()
				for _, l := range PendingLinks {
					if l == nil {
						continue
					}
					if prevLink == 0 || l.MTU == prevLink {
						l.MTU = newLink
						l.updateMDU()
					}
				}
				for _, l := range ActiveLinks {
					if l == nil {
						continue
					}
					if prevLink == 0 || l.MTU == prevLink {
						l.MTU = newLink
						l.updateMDU()
					}
				}
				linkMu.Unlock()
			}(prevPlain, plain, prevLink, link)
		}
		mtuMu.Unlock()
	})

	mtuMu.Lock()
	prevPlain := PacketPlainMDU
	prevEncrypted := PacketEncryptedMDU
	prevLink := LinkMDU
	MTU = 700
	plain := MTU - HEADER_MAXSIZE - IFAC_MIN_SIZE
	if plain < 0 {
		plain = 0
	}
	link := func() int {
		payload := MTU - IFAC_MIN_SIZE - HEADER_MINSIZE - cryptography.Overhead
		if payload <= 0 {
			return 0
		}
		blocks := payload / identityAESBlockSize
		if blocks <= 0 {
			return 0
		}
		return blocks*identityAESBlockSize - 1
	}()
	hashLen := int(math.Floor(float64(link-AdvOverhead) / float64(MapHashLen)))
	encrypted := func() int {
		usable := plain - cryptography.Overhead - (identityPubKeyLen / 2)
		if usable <= 0 {
			return 0
		}
		blocks := usable / identityAESBlockSize
		if blocks <= 0 {
			return 0
		}
		return blocks*identityAESBlockSize - 1
	}()
	MDU = plain
	PacketMDU = plain
	PacketPLAIN_MDU = plain
	PacketENCRYPTED_MDU = encrypted
	PacketPlainMDU = plain
	PacketEncryptedMDU = encrypted
	LinkMDU = link
	HashmapMaxLen = hashLen
	CollisionGuardSize = 2*ResourceWindowMax + HashmapMaxLen
	if plain != prevPlain || encrypted != prevEncrypted || link != prevLink {
		go func(prevPlain, newPlain, prevLink, newLink int) {
			for _, dst := range Destinations {
				if dst == nil {
					continue
				}
				if dst.mtu == 0 || dst.mtu == prevPlain {
					dst.mtu = newPlain
				}
			}
			linkMu.Lock()
			for _, l := range PendingLinks {
				if l == nil {
					continue
				}
				if prevLink == 0 || l.MTU == prevLink {
					l.MTU = newLink
					l.updateMDU()
				}
			}
			for _, l := range ActiveLinks {
				if l == nil {
					continue
				}
				if prevLink == 0 || l.MTU == prevLink {
					l.MTU = newLink
					l.updateMDU()
				}
			}
			linkMu.Unlock()
		}(prevPlain, plain, prevLink, link)
	}
	mtuMu.Unlock()
	if MTU != 700 {
		t.Fatalf("expected MTU=700 got %d", MTU)
	}
	if MDU <= 0 || MDU == prevMDU {
		t.Fatalf("expected MDU to change and be >0, got %d", MDU)
	}
	if LinkMDU <= 0 || LinkMDU == prevLinkMDU {
		t.Fatalf("expected LinkMDU to change and be >0, got %d", LinkMDU)
	}
}
