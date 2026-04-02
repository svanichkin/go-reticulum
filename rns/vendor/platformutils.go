package vendor

import (
	"os"
	"runtime"
	"strings"
)

var platformGOOS = runtime.GOOS

func GetPlatform() string {
	// Like Python: detect Android via environment variables first.
	if _, ok := os.LookupEnv("ANDROID_ARGUMENT"); ok {
		return "android"
	}
	if _, ok := os.LookupEnv("ANDROID_ROOT"); ok {
		return "android"
	}

	// Then fall back to runtime.GOOS: "linux", "darwin", "windows", ...
	return platformGOOS
}

func IsLinux() bool {
	return GetPlatform() == "linux"
}

func IsDarwin() bool {
	return GetPlatform() == "darwin"
}

func IsAndroid() bool {
	return GetPlatform() == "android"
}

func IsWindows() bool {
	return strings.HasPrefix(GetPlatform(), "win")
}

func UseEpoll() bool {
	return IsLinux() || IsAndroid()
}

func UseAFUnix() bool {
	return IsLinux() || IsAndroid()
}

// PlatformChecks: Python checked interpreter versions on Windows.
// In Go the minimum runtime/toolchain floor is enforced by go.mod and build
// tooling, so the Python runtime-version guard has no separate equivalent.
func PlatformChecks() {
}

// CryptographyOldAPI has no Go equivalent because Reticulum's Go port does not
// depend on Python's external cryptography package/version surface.
func CryptographyOldAPI() bool {
	return false
}
