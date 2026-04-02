package vendor

import (
	"os"
	"testing"
)

func withPlatformGOOS(t *testing.T, goos string) {
	t.Helper()
	orig := platformGOOS
	platformGOOS = goos
	t.Cleanup(func() { platformGOOS = orig })
}

func TestGetPlatform_AndroidOverride(t *testing.T) {
	restore := func(k, v string, ok bool) {
		if ok {
			_ = os.Setenv(k, v)
		} else {
			_ = os.Unsetenv(k)
		}
	}

	origArg, okArg := os.LookupEnv("ANDROID_ARGUMENT")
	origRoot, okRoot := os.LookupEnv("ANDROID_ROOT")
	t.Cleanup(func() {
		restore("ANDROID_ARGUMENT", origArg, okArg)
		restore("ANDROID_ROOT", origRoot, okRoot)
	})

	_ = os.Unsetenv("ANDROID_ARGUMENT")
	_ = os.Unsetenv("ANDROID_ROOT")

	_ = os.Setenv("ANDROID_ROOT", "1")
	if got := GetPlatform(); got != "android" {
		t.Fatalf("GetPlatform=%q, want %q", got, "android")
	}
	if !IsAndroid() {
		t.Fatalf("IsAndroid=false, want true")
	}
	if !UseEpoll() {
		t.Fatalf("UseEpoll=false, want true on android override")
	}
	if !UseAFUnix() {
		t.Fatalf("UseAFUnix=false, want true on android override")
	}
}

func TestUseAFUnix_StrictPythonParity(t *testing.T) {
	restore := func(k, v string, ok bool) {
		if ok {
			_ = os.Setenv(k, v)
		} else {
			_ = os.Unsetenv(k)
		}
	}
	origArg, okArg := os.LookupEnv("ANDROID_ARGUMENT")
	origRoot, okRoot := os.LookupEnv("ANDROID_ROOT")
	t.Cleanup(func() {
		restore("ANDROID_ARGUMENT", origArg, okArg)
		restore("ANDROID_ROOT", origRoot, okRoot)
	})
	_ = os.Unsetenv("ANDROID_ARGUMENT")
	_ = os.Unsetenv("ANDROID_ROOT")

	withPlatformGOOS(t, "darwin")
	if UseAFUnix() {
		t.Fatalf("UseAFUnix=true on darwin, want false for python parity")
	}

	withPlatformGOOS(t, "linux")
	if !UseAFUnix() {
		t.Fatalf("UseAFUnix=false on linux, want true")
	}
}

func TestPlatformChecksAndCryptographyOldAPI_GoSemantics(t *testing.T) {
	withPlatformGOOS(t, "windows")
	PlatformChecks()
	if CryptographyOldAPI() {
		t.Fatalf("CryptographyOldAPI=true, want false")
	}
}
