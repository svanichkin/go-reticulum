package rns

import (
	"os"
	"strings"
	"testing"
)

func TestDestinationSetProofStrategy_InvalidPanics(t *testing.T) {
	d, err := NewDestination(nil, DestinationIN, DestinationGROUP, "test", "proof", "strategy")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected panic for unsupported proof strategy")
		}
		typeErr, ok := rec.(*DestinationTypeError)
		if !ok {
			t.Fatalf("panic type=%T, want *DestinationTypeError", rec)
		}
		if typeErr.Message != "Unsupported proof strategy" {
			t.Fatalf("panic message=%q, want %q", typeErr.Message, "Unsupported proof strategy")
		}
	}()

	d.SetProofStrategy(0x99)
}

func TestDestinationGroupEncryptDecrypt_WithoutKeyPanics(t *testing.T) {
	d, err := NewDestination(nil, DestinationIN, DestinationGROUP, "test", "group", "missingkey")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	func() {
		defer func() {
			rec := recover()
			if rec == nil {
				t.Fatal("expected panic from Encrypt without group key")
			}
		}()
		_ = d.Encrypt([]byte("hello"))
	}()

	func() {
		defer func() {
			rec := recover()
			if rec == nil {
				t.Fatal("expected panic from Decrypt without group key")
			}
			valueErr, ok := rec.(*DestinationValueError)
			if !ok {
				t.Fatalf("panic type=%T, want *DestinationValueError", rec)
			}
			if valueErr.Message != "No private key held by GROUP destination. Did you create or load one?" {
				t.Fatalf("panic message=%q", valueErr.Message)
			}
		}()
		_ = d.Decrypt([]byte("cipher"))
	}()
}

func TestDestinationReloadRatchets_CorruptFileLogsRecoveryGuidance(t *testing.T) {
	d, err := NewDestination(nil, DestinationIN, DestinationSINGLE, "test", "ratchet", "corrupt")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	path := t.TempDir() + "/ratchets"
	if err := os.WriteFile(path, []byte("not-msgpack"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var got []string
	SetLogDestCallback(func(_ int, msg string) {
		got = append(got, msg)
	})
	t.Cleanup(func() {
		SetLogDestCallback(nil)
	})

	if err := d.reloadRatchets(path); err == nil {
		t.Fatal("expected reloadRatchets to fail for corrupt file")
	}

	joined := strings.Join(got, "\n")
	wantSnippets := []string{
		"First ratchet reload attempt for " + str(d) + " failed. Possible I/O conflict. Retrying in 500ms.",
		"The ratchet file located at " + path + " could not be loaded. This could indicate that the ratchet file has become corrupt.",
		"You can attempt to manually recover the ratchet file, or simply remove it to have Reticulum recreate it on the next use.",
		"If re-initialize this ratchet file, make sure to send an announce for the relevant destination as soon as possible,",
		"so that the new ratchet information is synchronized to the network.",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(joined, snippet) {
			t.Fatalf("missing log snippet %q in logs:\n%s", snippet, joined)
		}
	}
}
