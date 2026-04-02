package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	_ "embed"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const programVersion = "2.5.0"
const firmwareUpdateURL = "https://github.com/markqvist/RNode_Firmware/releases/download/"
const firmwareReleaseInfoURL = "https://github.com/markqvist/rnode_firmware/releases/latest/download/release.json"

type exitError struct {
	Code int
	Err  error
}

func (e exitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit %d", e.Code)
	}
	return e.Err.Error()
}

func asExitError(err error) (exitError, bool) {
	var ee exitError
	if err == nil {
		return ee, false
	}
	if errors.As(err, &ee) {
		return ee, true
	}
	return ee, false
}

type firmwareReleaseInfo map[string]struct {
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

type knownSigningKey struct {
	Vendor string
	DERHex string // DER SubjectPublicKeyInfo (PKIX) bytes, hex encoded (no delimiters)
}

// Matches upstream Python rnodeconf known_keys (vendor name + SPKI DER).
var knownSigningKeys = []knownSigningKey{
	{
		Vendor: "unsigned.io",
		DERHex: "30819f300d06092a864886f70d010101050003818d0030818902818100bf831ebd99f43b477caf1a094bec829389da40653e8f1f83fc14bf1b98a3e1cc70e759c213a43f71e5a47eb56a9ca487f241335b3e6ff7cdde0ee0a1c75c698574aeba0485726b6a9dfc046b4188e3520271ee8555a8f405cf21f81f2575771d0b0887adea5dd53c1f594f72c66b5f14904ffc2e72206a6698a490d51ba1105b0203010001",
	},
	{
		Vendor: "unsigned.io",
		DERHex: "30819f300d06092a864886f70d010101050003818d0030818902818100e5d46084e445595376bf7efd9c6ccf19d39abbc59afdb763207e4ff68b8d00ebffb63847aa2fe6dd10783d3ea63b55ac66f71ad885c20e223709f0d51ed5c6c0d0b093be9e1d165bb8a483a548b67a3f7a1e4580f50e75b306593fa6067ae259d3e297717bd7ff8c8f5b07f2bed89929a9a0321026cf3699524db98e2d18fb2d020300ff39",
	},
}

type PreparedFirmware struct {
	Version    string // "custom" or version number
	SourceURL  string // only for info
	LocalPath  string // downloaded/local firmware file (zip/hex)
	Extracted  bool   // extracted into paths.ExtractedDir
	IsZIP      bool
	FWFileName string
}

type storagePaths struct {
	ConfigDir           string
	UpdateDir           string
	FirmwareDir         string
	ExtractedDir        string
	TrustedKeysDir      string
	EEPROMDir           string
	RecoveryTool        string
	SignerKeyPath       string
	SignerKeyPathLegacy string
	DeviceKeyPath       string
	DeviceSerialCounter string
	DeviceDBDir         string
}

func prepareStoragePaths() (storagePaths, error) {
	base := strings.TrimSpace(os.Getenv("RNODECONF_DIR"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return storagePaths{}, fmt.Errorf("cannot resolve user home dir: %w", err)
		}
		base = filepath.Join(home, ".config", "rnodeconf")
	}
	paths := storagePaths{
		ConfigDir:           base,
		UpdateDir:           filepath.Join(base, "update"),
		FirmwareDir:         filepath.Join(base, "firmware"),
		ExtractedDir:        filepath.Join(base, "extracted"),
		TrustedKeysDir:      filepath.Join(base, "trusted_keys"),
		EEPROMDir:           filepath.Join(base, "eeprom"),
		RecoveryTool:        filepath.Join(base, "recovery_esptool.py"),
		SignerKeyPath:       filepath.Join(base, "signing.key"),
		SignerKeyPathLegacy: filepath.Join(base, "signing_key.der"),
		DeviceKeyPath:       filepath.Join(base, "device.key"),
		DeviceSerialCounter: filepath.Join(base, "device.serial"),
		DeviceDBDir:         filepath.Join(base, "device_db"),
	}

	for _, dir := range []string{
		paths.ConfigDir,
		paths.UpdateDir,
		paths.FirmwareDir,
		paths.ExtractedDir,
		paths.TrustedKeysDir,
		paths.EEPROMDir,
		paths.DeviceDBDir,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			// Python parity: failure to access/create config dir is fatal with code 99.
			return storagePaths{}, exitError{Code: 99, Err: fmt.Errorf("cannot create %s: %w", dir, err)}
		}
	}
	return paths, nil
}

func clearFirmwareCache(paths storagePaths) error {
	if err := os.RemoveAll(paths.UpdateDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(paths.UpdateDir, 0o755)
}

func verifyExtractedFirmware(paths storagePaths) error {
	required := []string{
		filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.version"),
		filepath.Join(paths.ExtractedDir, "extracted_console_image.bin"),
		filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.bin"),
		filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.boot_app0"),
		filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.bootloader"),
		filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.partitions"),
	}
	for _, p := range required {
		if !fileExists(p) {
			code := 184
			if strings.HasSuffix(p, "extracted_rnode_firmware.version") {
				code = 183
			}
			return exitError{Code: code, Err: fmt.Errorf("missing required extracted firmware file: %s", p)}
		}
	}
	return nil
}

func extractFirmwareZip(zipPath string, paths storagePaths) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		dst := filepath.Join(paths.ExtractedDir, filepath.Base(f.Name))
		rc, err := f.Open()
		if err != nil {
			return err
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o600); err != nil {
			return err
		}
	}
	return verifyExtractedFirmware(paths)
}

func localFirmwarePathFromURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	u, err := url.Parse(raw)
	if err == nil && u.Scheme == "file" {
		return u.Path, true
	}
	// Treat as local path if scheme missing.
	if err == nil && u.Scheme == "" {
		return raw, true
	}
	return "", false
}

func fetchReleaseInfo(urlStr string) (firmwareReleaseInfo, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var info firmwareReleaseInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return nil, err
	}
	return info, nil
}

func baseFirmwareURL(custom string) string {
	custom = strings.TrimSpace(custom)
	if custom == "" {
		return firmwareUpdateURL
	}
	if !strings.HasSuffix(custom, "/") {
		custom += "/"
	}
	return custom
}

func sha256HexFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeCachedFirmwareVersion(paths storagePaths, version, filename, hashHex string) {
	if strings.TrimSpace(version) == "" || strings.TrimSpace(filename) == "" || strings.TrimSpace(hashHex) == "" {
		return
	}
	dir := filepath.Join(paths.UpdateDir, version)
	_ = os.MkdirAll(dir, 0o755)
	versionPath := filepath.Join(dir, filename+".version")
	_ = os.WriteFile(versionPath, []byte(strings.TrimSpace(version)+" "+strings.TrimSpace(hashHex)), 0o600)
}

func readCachedFirmwareHash(paths storagePaths, version, filename string) (string, bool) {
	if strings.TrimSpace(version) == "" || strings.TrimSpace(filename) == "" {
		return "", false
	}
	versionPath := filepath.Join(paths.UpdateDir, version, filename+".version")
	data, err := os.ReadFile(versionPath)
	if err != nil {
		return "", false
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return "", false
	}
	hashHex := strings.TrimSpace(fields[1])
	if _, err := hex.DecodeString(hashHex); err != nil {
		return "", false
	}
	return hashHex, true
}

func downloadFile(urlStr, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	tmp := destPath + ".tmp"

	client := &http.Client{Timeout: 0}
	resp, err := client.Get(urlStr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, destPath)
}

func ensureOnlineFirmwarePrepared(paths storagePaths, node *RNode, fwURL, fwVersion string, updateFlag, autoinstallFlag, noCheck bool) (PreparedFirmware, error) {
	if node == nil {
		return PreparedFirmware{}, errors.New("nil device")
	}
	modelInfo, ok := Models[node.Model]
	if !ok || strings.TrimSpace(modelInfo.FWFile) == "" {
		return PreparedFirmware{}, fmt.Errorf("unknown/unsupported device model 0x%02x for online firmware", node.Model)
	}
	filename := modelInfo.FWFile

	version := strings.TrimSpace(fwVersion)
	var expectedHash string
	if version == "" && (updateFlag || autoinstallFlag) {
		if noCheck {
			return PreparedFirmware{}, exitError{Code: 98, Err: errors.New("--nocheck requires an explicit --fw-version or --fw-url")}
		}
		// Python parity: resolve per-board version via latest release.json.
		info, err := fetchReleaseInfo(firmwareReleaseInfoURL)
		if err != nil {
			return PreparedFirmware{}, err
		}
		variant, ok := info[filename]
		if !ok {
			// Python parity: "not" => 199 (no valid version for board).
			return PreparedFirmware{}, exitError{Code: 199, Err: errors.New("no valid version found for this board, exiting")}
		}
		version = strings.TrimSpace(variant.Version)
		expectedHash = strings.TrimSpace(variant.Hash)
		if version == "" || strings.EqualFold(version, "not") {
			return PreparedFirmware{}, exitError{Code: 199, Err: errors.New("no valid version found for this board, exiting")}
		}
	}
	if version == "" && strings.TrimSpace(fwURL) == "" {
		return PreparedFirmware{}, errors.New("no firmware version/url specified")
	}
	if version == "" && strings.TrimSpace(fwURL) != "" {
		version = "custom"
	}

	if version != "custom" && !noCheck && expectedHash == "" {
		// Python parity: for explicit version, pull release.json from that tag.
		u := baseFirmwareURL(fwURL) + version + "/release.json"
		info, err := fetchReleaseInfo(u)
		if err != nil {
			return PreparedFirmware{}, err
		}
		if v, ok := info[filename]; ok {
			expectedHash = strings.TrimSpace(v.Hash)
			writeCachedFirmwareVersion(paths, version, filename, expectedHash)
		}
	}

	var sourceURL string
	if strings.TrimSpace(fwURL) != "" {
		// Python expects fw_url to be a base URL where firmware lives under download/<version>/...,
		// but we accept either a direct file/url (handled earlier) or a base URL here.
		sourceURL = baseFirmwareURL(fwURL) + "download/" + version + "/" + filename
	} else {
		sourceURL = firmwareUpdateURL + version + "/" + filename
	}

	dest := filepath.Join(paths.UpdateDir, version, filename)
	if fileExists(dest) {
		fmt.Printf("Using cached firmware: %s\n", dest)
	} else {
		fmt.Printf("Downloading firmware: %s\n", sourceURL)
		if err := downloadFile(sourceURL, dest); err != nil {
			return PreparedFirmware{}, err
		}
		fmt.Printf("Saved firmware: %s\n", dest)
	}

	if version != "custom" {
		if noCheck {
			if cached, ok := readCachedFirmwareHash(paths, version, filename); ok {
				expectedHash = cached
			}
		}
		if expectedHash != "" {
			fmt.Println("Verifying firmware integrity...")
			fileHash, err := sha256HexFile(dest)
			if err != nil {
				return PreparedFirmware{}, exitError{Code: 95, Err: err}
			}
			if !strings.EqualFold(fileHash, expectedHash) {
				return PreparedFirmware{}, exitError{Code: 96, Err: fmt.Errorf("firmware hash %s but should be %s, possibly due to download corruption\nfirmware corrupt; try clearing the local firmware cache with: rnodeconf --clear-cache", fileHash, expectedHash)}
			}
		} else if !noCheck {
			return PreparedFirmware{}, exitError{Code: 97, Err: fmt.Errorf("no release hash found for %s; the firmware integrity could not be verified", filename)}
		}
	}

	if strings.HasSuffix(strings.ToLower(dest), ".zip") {
		// ESP32 flashing uses extracted parts, NRF52 uses the zip directly.
		if node.Platform == ROM_PLATFORM_NRF52 {
			return PreparedFirmware{
				Version:    version,
				SourceURL:  sourceURL,
				LocalPath:  dest,
				Extracted:  false,
				IsZIP:      true,
				FWFileName: filename,
			}, nil
		}
		if err := extractFirmwareZip(dest, paths); err != nil {
			return PreparedFirmware{}, err
		}
		return PreparedFirmware{
			Version:    version,
			SourceURL:  sourceURL,
			LocalPath:  dest,
			Extracted:  true,
			IsZIP:      true,
			FWFileName: filename,
		}, nil
	}

	return PreparedFirmware{
		Version:    version,
		SourceURL:  sourceURL,
		LocalPath:  dest,
		Extracted:  false,
		IsZIP:      false,
		FWFileName: filename,
	}, nil
}

func runRecoveryReadFlash(recoveryToolPath, portPath string, baudFlash int, outDir string) error {
	type part struct {
		name   string
		offset string
		size   string
		out    string
	}
	parts := []part{
		{"bootloader", "0x1000", "0x4650", filepath.Join(outDir, "extracted_rnode_firmware.bootloader")},
		{"partition table", "0x8000", "0xC00", filepath.Join(outDir, "extracted_rnode_firmware.partitions")},
		{"app boot", "0xE000", "0x2000", filepath.Join(outDir, "extracted_rnode_firmware.boot_app0")},
		{"application image", "0x10000", "0x200000", filepath.Join(outDir, "extracted_rnode_firmware.bin")},
		{"console image", "0x210000", "0x1F0000", filepath.Join(outDir, "extracted_console_image.bin")},
	}
	for _, p := range parts {
		args := []string{
			recoveryToolPath,
			"--chip", "esp32",
			"--port", portPath,
			"--baud", strconv.Itoa(baudFlash),
			"--before", "default_reset",
			"--after", "hard_reset",
			"read_flash",
			p.offset, p.size, p.out,
		}
		cmd := exec.Command("python", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return exitError{Code: 182, Err: fmt.Errorf("%s failed: %w", p.name, err)}
		}
	}
	return nil
}

func storeTrustedKey(hexString, dir string) (path string, fingerprint string, err error) {
	clean := strings.TrimSpace(hexString)
	if clean == "" {
		return "", "", errors.New("key string is empty")
	}
	keyBytes, err := hex.DecodeString(clean)
	if err != nil {
		return "", "", fmt.Errorf("invalid hex key: %w", err)
	}
	// Python parity: validate key bytes as DER SubjectPublicKeyInfo.
	pubAny, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		return "", "", fmt.Errorf("invalid trusted key (expected DER SubjectPublicKeyInfo): %w", err)
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return "", "", errors.New("invalid trusted key (expected RSA public key)")
	}
	// Normalise to PKIX DER (SPKI) format.
	keyBytes, err = x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(keyBytes)
	fingerprint = fmt.Sprintf("%x", sum)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	path = filepath.Join(dir, fingerprint+".pubkey")
	if err := os.WriteFile(path, keyBytes, 0o600); err != nil {
		return "", "", err
	}
	return path, fingerprint, nil
}

func ensureSignerKey(paths storagePaths) ([]byte, error) {
	if fileExists(paths.SignerKeyPath) {
		return os.ReadFile(paths.SignerKeyPath)
	}
	// Python parity: upstream signing keys are 1024-bit and signatures are 128 bytes.
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	// Python parity: upstream stores signing.key as PKCS8 DER (not PKCS1).
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(paths.SignerKeyPath, der, 0o600); err != nil {
		return nil, err
	}
	return der, nil
}

func parseSignerKeyDER(der []byte) (*rsa.PrivateKey, error) {
	if len(der) == 0 {
		return nil, errors.New("empty key data")
	}
	anyKey, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("invalid signing key format (expected PKCS8 DER): %w", err)
	}
	key, ok := anyKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("signing key is not RSA")
	}
	return key, nil
}

func hexWithColons(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	dst := make([]byte, 0, len(b)*3-1)
	const hextable = "0123456789abcdef"
	for i, v := range b {
		if i > 0 {
			dst = append(dst, ':')
		}
		dst = append(dst, hextable[v>>4], hextable[v&0x0f])
	}
	return string(dst)
}

func ensureDeviceSignerKey(paths storagePaths) (*rns.Identity, error) {
	// Python parity:
	// - if device.key missing, create a new Identity and write it (exit 81 on failure)
	// - if device.key exists but can't be loaded, exit 82
	if fileExists(paths.DeviceKeyPath) {
		id, err := rns.IdentityFromFile(paths.DeviceKeyPath)
		if err != nil {
			return nil, exitError{Code: 82, Err: fmt.Errorf("could not load device signing key from %s: %w", paths.DeviceKeyPath, err)}
		}
		return id, nil
	}

	fmt.Println("Generating a new device signing key...")
	id, err := rns.NewIdentity()
	if err != nil {
		return nil, exitError{Code: 81, Err: fmt.Errorf("could not create new device signing key: %w", err)}
	}
	if err := id.Save(paths.DeviceKeyPath); err != nil {
		return nil, exitError{Code: 81, Err: fmt.Errorf("could not create new device signing key at %s: %w", paths.DeviceKeyPath, err)}
	}
	fmt.Printf("Device signing key written to %s\n", paths.DeviceKeyPath)
	return id, nil
}

const (
	romAddrProduct   = 0x00
	romAddrModel     = 0x01
	romAddrHWRev     = 0x02
	romAddrSerial    = 0x03
	romAddrMade      = 0x07
	romAddrChecksum  = 0x0B
	romAddrSignature = 0x1B
	romAddrInfoLock  = 0x9B

	romInfoLockByte = 0x73
)

type bootstrapOptions struct {
	Product byte
	Model   byte
	HWRev   byte
}

func parseByteSpec(s string) (byte, error) {
	raw := strings.TrimSpace(strings.ToLower(s))
	raw = strings.TrimPrefix(raw, "0x")
	if raw == "" {
		return 0, errors.New("empty value")
	}
	if strings.HasPrefix(raw, "0b") {
		val, err := strconv.ParseUint(strings.TrimPrefix(raw, "0b"), 2, 8)
		if err != nil {
			return 0, fmt.Errorf("invalid value %q", s)
		}
		return byte(val), nil
	}
	val, err := strconv.ParseUint(raw, 16, 8)
	if err == nil {
		return byte(val), nil
	}
	ival, err2 := strconv.ParseUint(strings.TrimSpace(s), 10, 8)
	if err2 != nil {
		return 0, fmt.Errorf("invalid value %q", s)
	}
	return byte(ival), nil
}

func nextDeviceSerial(counterPath string) (uint32, error) {
	data, err := os.ReadFile(counterPath)
	if err == nil {
		if v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 32); err == nil && v > 0 {
			next := uint32(v + 1)
			if err := os.WriteFile(counterPath, []byte(strconv.FormatUint(uint64(next), 10)), 0o600); err != nil {
				return 0, err
			}
			return uint32(v), nil
		}
	}
	// Start at 1 for parity with Python’s monotonic counter.
	if err := os.WriteFile(counterPath, []byte("2"), 0o600); err != nil {
		return 0, err
	}
	return 1, nil
}

func extractedFirmwareHash(paths storagePaths) ([]byte, error) {
	versionPath := filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.version")
	data, err := os.ReadFile(versionPath)
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid extracted firmware version file %s", versionPath)
	}
	raw, err := hex.DecodeString(strings.TrimSpace(fields[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid extracted firmware hash in %s: %w", versionPath, err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("invalid extracted firmware hash length %d in %s", len(raw), versionPath)
	}
	return raw, nil
}

func tryBackupDeviceEEPROM(paths storagePaths, serialNo uint32, eeprom []byte) {
	if len(eeprom) == 0 {
		return
	}
	name := fmt.Sprintf("%08x", serialNo)
	outPath := filepath.Join(paths.DeviceDBDir, name)
	if err := os.WriteFile(outPath, eeprom, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "WARNING: Could not backup device EEPROM to disk")
	}
}

func writeEEPROMWithDelay(n *RNode, addr, value byte) error {
	if err := n.WriteEEPROM(addr, value); err != nil {
		return err
	}
	time.Sleep(6 * time.Millisecond)
	return nil
}

func promptInt(label string) (int, error) {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, errors.New("empty input")
	}
	v, err := strconv.Atoi(line)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func promptString(label string) (string, error) {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", errors.New("empty input")
	}
	return line, nil
}

func promptEnter() {
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func promptSelectSerialPort() (string, error) {
	ports, err := listSerialPorts()
	if err != nil {
		return "", err
	}
	if len(ports) == 0 {
		return "", errors.New("no serial ports detected")
	}
	if len(ports) == 1 {
		fmt.Printf("Detected serial port: %s\n", ports[0])
		fmt.Printf("Ok, using device on %s\n", ports[0])
		return ports[0], nil
	}

	fmt.Println("Detected serial ports:")
	for i, p := range ports {
		fmt.Printf("  [%d] %s\n", i+1, p)
	}
	fmt.Print("\nEnter the number of the serial port your device is connected to:\n? ")
	idx, err := promptInt("")
	if err != nil {
		return "", err
	}
	if idx < 1 || idx > len(ports) {
		return "", fmt.Errorf("that port does not exist")
	}
	selected := ports[idx-1]
	fmt.Printf("\nOk, using device on %s\n", selected)
	return selected, nil
}

func main() {
	// Match upstream rnodeconf logging defaults.
	rns.SetLogTimeFormat("%H:%M:%S")
	rns.SetCompactLogFormat(true)

	fs := flag.NewFlagSet("rnodeconf", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		showVersion bool
		listModels  bool
		modelQuery  string
		clearCache  bool
		trustKeyHex string
		portArg     string

		infoFlag        bool
		autoinstallFlag bool
		updateFlag      bool
		forceUpdateFlag bool
		flashFlag       bool
		romFlag         bool
		keyFlag         bool
		signFlag        bool
		showSigningKey  bool
		firmwareHash    string
		getTargetHash   bool
		getDeviceHash   bool
		bootstrapPlat   string
		bootstrapProd   string
		bootstrapModel  string
		bootstrapHWRev  int
		fwVersion       string
		fwURL           string
		noCheckFlag     bool
		extractFlag     bool
		useExtracted    bool
		baudFlash       string
		normalModeFlag  bool
		tncModeFlag     bool

		btOnFlag   bool
		btOffFlag  bool
		btPairFlag bool

		wifiMode    string
		wifiChannel string
		wifiSSID    string
		wifiPSK     string
		showPSK     bool
		wifiIP      string
		wifiNM      string

		displayIntensity   int
		displayTimeout     int
		displayRotation    int
		displayAddressHex  string
		reconditionDisplay bool
		neopixelIntensity  int
		freqHz             int
		bwHz               int
		txPower            int
		sfValue            int
		crValue            int
		iaEnable           bool
		iaDisable          bool
		configFlag         bool
		eepromBackupFlag   bool
		eepromDumpFlag     bool
		eepromWipeFlag     bool
	)

	fs.BoolVar(&showVersion, "version", false, "print program version and exit")
	fs.BoolVar(&listModels, "list-models", false, "list known hardware models and exit")
	fs.StringVar(&modelQuery, "show-model", "", "print detailed information about the specified model code (ex: A1 or 0xA1)")
	fs.BoolVar(&clearCache, "clear-cache", false, "clear locally cached firmware files and exit")
	fs.BoolVar(&clearCache, "C", false, "clear locally cached firmware files and exit")
	fs.StringVar(&trustKeyHex, "trust-key", "", "store trusted public key for device verification (hex DER)")
	fs.StringVar(&portArg, "port", "", "serial port for the RNode (can also be given as a positional argument)")

	fs.BoolVar(&infoFlag, "info", false, "show device info")
	fs.BoolVar(&infoFlag, "i", false, "show device info")
	fs.BoolVar(&autoinstallFlag, "autoinstall", false, "automatic installation on supported devices")
	fs.BoolVar(&autoinstallFlag, "a", false, "automatic installation on supported devices")
	fs.BoolVar(&updateFlag, "update", false, "update firmware to the latest version")
	fs.BoolVar(&updateFlag, "u", false, "update firmware to the latest version")
	fs.BoolVar(&forceUpdateFlag, "force-update", false, "update even if version matches or is older")
	fs.BoolVar(&forceUpdateFlag, "U", false, "update even if version matches or is older")
	fs.BoolVar(&flashFlag, "flash", false, "flash firmware and bootstrap EEPROM (offline/local firmware only)")
	fs.BoolVar(&flashFlag, "f", false, "flash firmware and bootstrap EEPROM (offline/local firmware only)")
	fs.BoolVar(&romFlag, "rom", false, "bootstrap EEPROM without flashing firmware")
	fs.BoolVar(&romFlag, "r", false, "bootstrap EEPROM without flashing firmware")
	fs.BoolVar(&keyFlag, "key", false, "generate a new signing key and exit")
	fs.BoolVar(&keyFlag, "k", false, "generate a new signing key and exit")
	fs.BoolVar(&signFlag, "sign", false, "sign attached device (store device signature in EEPROM)")
	fs.BoolVar(&signFlag, "S", false, "sign attached device (store device signature in EEPROM)")
	// Python parity: `-P/--public` prints the public part of the signing key.
	fs.BoolVar(&showSigningKey, "show-signing-key", false, "display public part of local signing key (hex DER)")
	fs.BoolVar(&showSigningKey, "public", false, "display public part of local signing key (hex DER)")
	fs.BoolVar(&showSigningKey, "P", false, "display public part of local signing key (hex DER)")
	fs.StringVar(&firmwareHash, "firmware-hash", "", "set installed firmware hash (hex)")
	fs.StringVar(&firmwareHash, "H", "", "set installed firmware hash (hex)")
	fs.BoolVar(&getTargetHash, "get-target-firmware-hash", false, "get target firmware hash from device")
	fs.BoolVar(&getTargetHash, "K", false, "get target firmware hash from device")
	fs.BoolVar(&getDeviceHash, "get-firmware-hash", false, "get calculated firmware hash from device")
	fs.BoolVar(&getDeviceHash, "L", false, "get calculated firmware hash from device")
	fs.StringVar(&bootstrapPlat, "platform", "", "platform specification for device bootstrap (accepted but currently informational)")
	fs.StringVar(&bootstrapProd, "product", "", "product byte for EEPROM bootstrap (hex or int)")
	fs.StringVar(&bootstrapModel, "model", "", "model code for EEPROM bootstrap (hex like a1/0xA1)")
	fs.IntVar(&bootstrapHWRev, "hwrev", -1, "hardware revision byte for EEPROM bootstrap (1-255)")
	fs.StringVar(&fwVersion, "fw-version", "", "use specific firmware version for update or autoinstall")
	fs.StringVar(&fwURL, "fw-url", "", "use alternate firmware download URL")
	fs.BoolVar(&noCheckFlag, "nocheck", false, "don't check for firmware updates online")
	fs.BoolVar(&extractFlag, "extract", false, "extract firmware from connected RNode for later use")
	fs.BoolVar(&extractFlag, "e", false, "extract firmware from connected RNode for later use")
	fs.BoolVar(&useExtracted, "use-extracted", false, "use previously extracted firmware for autoinstall or update")
	fs.BoolVar(&useExtracted, "E", false, "use previously extracted firmware for autoinstall or update")
	fs.StringVar(&baudFlash, "baud-flash", "921600", "set specific baud rate when flashing device")

	fs.BoolVar(&normalModeFlag, "normal", false, "switch device to normal mode")
	fs.BoolVar(&normalModeFlag, "N", false, "switch device to normal mode")
	fs.BoolVar(&tncModeFlag, "tnc", false, "switch device to TNC mode")
	fs.BoolVar(&tncModeFlag, "T", false, "switch device to TNC mode")

	fs.BoolVar(&btOnFlag, "bluetooth-on", false, "turn device bluetooth on")
	fs.BoolVar(&btOnFlag, "b", false, "turn device bluetooth on")
	fs.BoolVar(&btOffFlag, "bluetooth-off", false, "turn device bluetooth off")
	fs.BoolVar(&btOffFlag, "B", false, "turn device bluetooth off")
	fs.BoolVar(&btPairFlag, "bluetooth-pair", false, "put device into bluetooth pairing mode")
	fs.BoolVar(&btPairFlag, "p", false, "put device into bluetooth pairing mode")

	fs.StringVar(&wifiMode, "wifi", "", "set WiFi mode (OFF, AP or STATION)")
	fs.StringVar(&wifiMode, "w", "", "set WiFi mode (OFF, AP or STATION)")
	fs.StringVar(&wifiChannel, "channel", "", "set WiFi channel")
	fs.StringVar(&wifiSSID, "ssid", "", "set WiFi SSID (NONE to delete)")
	fs.StringVar(&wifiPSK, "psk", "", "set WiFi PSK (NONE to delete)")
	fs.BoolVar(&showPSK, "show-psk", false, "display stored WiFi PSK")
	fs.StringVar(&wifiIP, "ip", "", "set static WiFi IP address (NONE for DHCP)")
	fs.StringVar(&wifiNM, "nm", "", "set static WiFi netmask (NONE for DHCP)")

	fs.IntVar(&displayIntensity, "display", -1, "set display intensity (0-255)")
	fs.IntVar(&displayIntensity, "D", -1, "set display intensity (0-255)")
	fs.IntVar(&displayTimeout, "timeout", -1, "set display timeout in seconds, 0 to disable")
	fs.IntVar(&displayTimeout, "t", -1, "set display timeout in seconds, 0 to disable")
	fs.IntVar(&displayRotation, "rotation", -1, "set display rotation (0-3)")
	fs.IntVar(&displayRotation, "R", -1, "set display rotation (0-3)")
	fs.StringVar(&displayAddressHex, "display-addr", "", "set display address as hex byte (00-FF)")
	fs.BoolVar(&reconditionDisplay, "recondition-display", false, "start display reconditioning")

	fs.IntVar(&neopixelIntensity, "np", -1, "set NeoPixel intensity (0-255)")

	fs.IntVar(&freqHz, "freq", 0, "frequency in Hz for TNC mode")
	fs.IntVar(&bwHz, "bw", 0, "bandwidth in Hz for TNC mode")
	fs.IntVar(&txPower, "txp", 0, "TX power in dBm for TNC mode")
	fs.IntVar(&sfValue, "sf", 0, "spreading factor for TNC mode (7-12)")
	fs.IntVar(&crValue, "cr", 0, "coding rate for TNC mode (5-8)")

	fs.BoolVar(&iaEnable, "ia-enable", false, "enable interference avoidance")
	fs.BoolVar(&iaDisable, "ia-disable", false, "disable interference avoidance")
	// Python parity: short flags.
	fs.BoolVar(&iaEnable, "x", false, "enable interference avoidance")
	fs.BoolVar(&iaDisable, "X", false, "disable interference avoidance")

	fs.BoolVar(&configFlag, "config", false, "print device configuration")
	fs.BoolVar(&configFlag, "c", false, "print device configuration")
	fs.BoolVar(&eepromBackupFlag, "eeprom-backup", false, "backup EEPROM to file")
	fs.BoolVar(&eepromDumpFlag, "eeprom-dump", false, "dump EEPROM to console")
	fs.BoolVar(&eepromWipeFlag, "eeprom-wipe", false, "unlock and wipe EEPROM")

	var help bool
	fs.BoolVar(&help, "help", false, "show this help message and exit")
	fs.BoolVar(&help, "h", false, "show this help message and exit")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: rnodeconf [options] [port]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "RNode Configuration and firmware utility.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "options:")
		fs.VisitAll(func(f *flag.Flag) {
			// Hide these like Python argparse.SUPPRESS.
			if f.Name == "K" || f.Name == "get-target-firmware-hash" || f.Name == "L" || f.Name == "get-firmware-hash" {
				return
			}
			fmt.Fprintf(os.Stderr, "  -%s\t%s\n", f.Name, f.Usage)
		})
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		fs.Usage()
		msg := strings.TrimSpace(err.Error())
		if msg != "" {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(2)
	}
	if help {
		fs.Usage()
		os.Exit(0)
	}
	if portArg == "" && len(fs.Args()) > 0 {
		portArg = fs.Args()[0]
	}

	switch {
	case showVersion:
		fmt.Printf("rnodeconf %s\n", programVersion)
		return
	case listModels:
		printModelTable()
		return
	case modelQuery != "":
		if err := printModelInfo(modelQuery); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	paths, err := prepareStoragePaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error preparing local storage: %v\n", err)
		if ee, ok := asExitError(err); ok {
			os.Exit(ee.Code)
		}
		os.Exit(1)
	}

	switch {
	case clearCache:
		// Python parity: log-style output.
		fmt.Println("Clearing local firmware cache...")
		if err := clearFirmwareCache(paths); err != nil {
			fmt.Fprintf(os.Stderr, "error clearing cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Done")
		return
	case trustKeyHex != "":
		path, fingerprint, err := storeTrustedKey(trustKeyHex, paths.TrustedKeysDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error trusting key: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Trusting key: %s\nStored at: %s\n", fingerprint, path)
		return
	case keyFlag:
		if _, err := ensureDeviceSignerKey(paths); err != nil {
			if ee, ok := asExitError(err); ok {
				fmt.Fprintln(os.Stderr, ee.Err)
				os.Exit(ee.Code)
			}
			fmt.Fprintf(os.Stderr, "error generating device key: %v\n", err)
			os.Exit(1)
		}
		_, err := ensureSignerKey(paths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error generating signing key: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated signing key at %s\n", paths.SignerKeyPath)
		return
	case showSigningKey:
		// Python parity: print both the EEPROM signing public key (RSA, SPKI DER)
		// and the device signing public key (last 32 bytes of Identity public key,
		// colon-delimited).
		signerDER, err := os.ReadFile(paths.SignerKeyPath)
		if err != nil {
			fmt.Println("Could not load EEPROM signing key")
		} else {
			key, err := parseSignerKeyDER(signerDER)
			if err != nil {
				fmt.Println("Could not deserialize signing key")
				fmt.Println(err.Error())
			} else if pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey); err != nil {
				fmt.Println("Could not serialize signing key")
				fmt.Println(err.Error())
			} else {
				fmt.Println("EEPROM Signing Public key:")
				fmt.Println(hex.EncodeToString(pubDER))
			}
		}

		id, err := rns.IdentityFromFile(paths.DeviceKeyPath)
		if err != nil {
			fmt.Println("Could not load device signing key")
		} else {
			pubAll := id.GetPublicKey()
			if len(pubAll) < 64 {
				fmt.Println("Could not load device signing key")
			} else {
				fmt.Println("")
				fmt.Println("Device Signing Public key:")
				fmt.Println(hexWithColons(pubAll[32:]))
			}
		}
		return
	}

	selectedVersion, firmwareURL := "", ""
	if fwVersion != "" {
		if _, err := strconv.ParseFloat(fwVersion, 64); err != nil {
			fmt.Fprintf(os.Stderr, "invalid --fw-version %q: %v\n", fwVersion, err)
			os.Exit(1)
		}
		selectedVersion = fwVersion
	}
	if fwURL != "" {
		// Allow local paths / file:// URLs for offline parity.
		if localPath, ok := localFirmwarePathFromURL(fwURL); ok {
			firmwareURL = localPath
		} else {
			parsed, err := url.Parse(fwURL)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				fmt.Fprintf(os.Stderr, "invalid --fw-url %q\n", fwURL)
				os.Exit(1)
			}
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				fmt.Fprintf(os.Stderr, "unsupported --fw-url scheme %q (expected http/https/file)\n", parsed.Scheme)
				os.Exit(1)
			}
			firmwareURL = parsed.String()
		}
	}
	baudFlashVal, err := strconv.Atoi(baudFlash)
	if err != nil || baudFlashVal <= 0 {
		fmt.Fprintf(os.Stderr, "invalid --baud-flash value %q\n", baudFlash)
		os.Exit(1)
	}

	// Python parity: warn about cross-device extracted firmware before flashing/updating.
	if useExtracted && ((updateFlag && strings.TrimSpace(portArg) != "") || autoinstallFlag) {
		fmt.Println("")
		fmt.Println("You have specified that rnodeconf should use a firmware extracted")
		fmt.Println("from another device. Please note that this *only* works if you are")
		fmt.Println("targeting a device of the same type that the firmware came from!")
		fmt.Println("")
		fmt.Println("Flashing this firmware to a device it was not created for will most")
		fmt.Println("likely result in it being inoperable until it is updated with the")
		fmt.Println("correct firmware. Hit enter to continue.")
		promptEnter()
	}

	if useExtracted {
		if err := verifyExtractedFirmware(paths); err != nil {
			fmt.Fprintf(os.Stderr, "no extracted firmware is available: %v\n", err)
			fmt.Fprintln(os.Stderr, "Extract firmware first with --extract on an ESP32-based RNode.")
			if ee, ok := asExitError(err); ok {
				os.Exit(ee.Code)
			}
			os.Exit(2)
		}
		fmt.Printf("Using extracted firmware from %s\n", paths.ExtractedDir)
	}

	// Note: do not auto-extract local zip here, since the handling depends on device platform (ESP32 vs NRF52).

	if (autoinstallFlag || updateFlag || flashFlag) && !(useExtracted || firmwareURL != "" || selectedVersion != "") {
		if flashFlag && !autoinstallFlag && !updateFlag {
			fmt.Fprintln(os.Stderr, "Missing parameters, cannot continue")
			os.Exit(68)
		}
		fmt.Fprintln(os.Stderr, "autoinstall/update/flash needs either --use-extracted, --fw-url, or --fw-version (online download).")
		os.Exit(2)
	}
	// NOTE: online firmware/version logic is handled later after device detection.
	_ = forceUpdateFlag
	_ = noCheckFlag

	var (
		wifiModeNormalized string
	)

	if wifiMode != "" {
		var err error
		wifiModeNormalized, err = parseWifiMode(wifiMode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --wifi value: %v\n", err)
			os.Exit(1)
		}
	}

	var wifiChannelInt int
	if wifiChannel != "" {
		ch, err := strconv.Atoi(wifiChannel)
		if err != nil || ch < 1 || ch > 165 {
			fmt.Fprintf(os.Stderr, "invalid --channel value %q\n", wifiChannel)
			os.Exit(1)
		}
		wifiChannelInt = ch
	}

	if wifiIP != "" && !strings.EqualFold(strings.TrimSpace(wifiIP), "none") {
		if _, err := encodeIPv4(wifiIP); err != nil {
			fmt.Fprintf(os.Stderr, "invalid --ip value: %v\n", err)
			os.Exit(1)
		}
	}

	if wifiNM != "" && !strings.EqualFold(strings.TrimSpace(wifiNM), "none") {
		if _, err := encodeIPv4(wifiNM); err != nil {
			fmt.Fprintf(os.Stderr, "invalid --nm value: %v\n", err)
			os.Exit(1)
		}
	}

	if displayIntensity != -1 && (displayIntensity < 0 || displayIntensity > 255) {
		fmt.Fprintf(os.Stderr, "--display must be between 0 and 255\n")
		os.Exit(1)
	}
	if displayTimeout != -1 && displayTimeout < 0 {
		fmt.Fprintf(os.Stderr, "--timeout must be >= 0\n")
		os.Exit(1)
	}
	if displayRotation != -1 && (displayRotation < 0 || displayRotation > 3) {
		fmt.Fprintf(os.Stderr, "--rotation must be between 0 and 3\n")
		os.Exit(1)
	}
	var displayAddr byte
	if displayAddressHex != "" {
		parsed, err := strconv.ParseUint(strings.TrimPrefix(displayAddressHex, "0x"), 16, 8)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --display-addr value %q\n", displayAddressHex)
			os.Exit(1)
		}
		displayAddr = byte(parsed)
	}
	if neopixelIntensity != -1 && (neopixelIntensity < 0 || neopixelIntensity > 255) {
		fmt.Fprintf(os.Stderr, "--np must be between 0 and 255\n")
		os.Exit(1)
	}
	if sfValue != 0 && (sfValue < 7 || sfValue > 12) {
		fmt.Fprintf(os.Stderr, "--sf must be between 7 and 12\n")
		os.Exit(1)
	}
	if crValue != 0 && (crValue < 5 || crValue > 8) {
		fmt.Fprintf(os.Stderr, "--cr must be between 5 and 8\n")
		os.Exit(1)
	}
	if iaEnable && iaDisable {
		fmt.Fprintf(os.Stderr, "--ia-enable and --ia-disable are mutually exclusive\n")
		os.Exit(1)
	}

	var pendingOps []string
	if infoFlag {
		pendingOps = append(pendingOps, "--info")
	}
	if autoinstallFlag {
		pendingOps = append(pendingOps, "--autoinstall")
	}
	if updateFlag {
		pendingOps = append(pendingOps, "--update")
	}
	if extractFlag {
		pendingOps = append(pendingOps, "--extract")
	}
	if useExtracted {
		pendingOps = append(pendingOps, "--use-extracted")
	}
	if normalModeFlag {
		pendingOps = append(pendingOps, "--normal")
	}
	if tncModeFlag {
		pendingOps = append(pendingOps, "--tnc")
	}
	if btOnFlag {
		pendingOps = append(pendingOps, "--bluetooth-on")
	}
	if btOffFlag {
		pendingOps = append(pendingOps, "--bluetooth-off")
	}
	if btPairFlag {
		pendingOps = append(pendingOps, "--bluetooth-pair")
	}
	if wifiModeNormalized != "" {
		pendingOps = append(pendingOps, fmt.Sprintf("--wifi=%s", wifiModeNormalized))
	}
	if wifiChannel != "" {
		pendingOps = append(pendingOps, fmt.Sprintf("--channel=%d", wifiChannelInt))
	}
	if wifiSSID != "" {
		pendingOps = append(pendingOps, "--ssid")
	}
	if wifiPSK != "" {
		pendingOps = append(pendingOps, "--psk")
	}
	if showPSK {
		pendingOps = append(pendingOps, "--show-psk")
	}
	if wifiIP != "" {
		pendingOps = append(pendingOps, "--ip")
	}
	if wifiNM != "" {
		pendingOps = append(pendingOps, "--nm")
	}
	if displayIntensity != -1 {
		pendingOps = append(pendingOps, "--display")
	}
	if displayTimeout != -1 {
		pendingOps = append(pendingOps, "--timeout")
	}
	if displayRotation != -1 {
		pendingOps = append(pendingOps, "--rotation")
	}
	if displayAddressHex != "" {
		pendingOps = append(pendingOps, "--display-addr")
	}
	if reconditionDisplay {
		pendingOps = append(pendingOps, "--recondition-display")
	}
	if neopixelIntensity != -1 {
		pendingOps = append(pendingOps, "--np")
	}
	if freqHz != 0 {
		pendingOps = append(pendingOps, "--freq")
	}
	if bwHz != 0 {
		pendingOps = append(pendingOps, "--bw")
	}
	if txPower != 0 {
		pendingOps = append(pendingOps, "--txp")
	}
	if sfValue != 0 {
		pendingOps = append(pendingOps, "--sf")
	}
	if crValue != 0 {
		pendingOps = append(pendingOps, "--cr")
	}
	if iaEnable {
		pendingOps = append(pendingOps, "--ia-enable")
	}
	if iaDisable {
		pendingOps = append(pendingOps, "--ia-disable")
	}
	if configFlag {
		pendingOps = append(pendingOps, "--config")
	}
	if eepromBackupFlag {
		pendingOps = append(pendingOps, "--eeprom-backup")
	}
	if eepromDumpFlag {
		pendingOps = append(pendingOps, "--eeprom-dump")
	}
	if eepromWipeFlag {
		pendingOps = append(pendingOps, "--eeprom-wipe")
	}
	if strings.TrimSpace(firmwareHash) != "" {
		pendingOps = append(pendingOps, "--firmware-hash")
	}
	if getTargetHash {
		pendingOps = append(pendingOps, "--get-target-firmware-hash")
	}
	if getDeviceHash {
		pendingOps = append(pendingOps, "--get-firmware-hash")
	}
	if flashFlag {
		pendingOps = append(pendingOps, "--flash")
	}
	if romFlag {
		pendingOps = append(pendingOps, "--rom")
	}
	if signFlag {
		pendingOps = append(pendingOps, "--sign")
	}

	var bootOpts *bootstrapOptions
	if strings.TrimSpace(bootstrapProd) != "" || strings.TrimSpace(bootstrapModel) != "" || bootstrapHWRev >= 0 {
		if strings.TrimSpace(bootstrapProd) == "" || strings.TrimSpace(bootstrapModel) == "" || bootstrapHWRev < 0 || bootstrapHWRev > 255 {
			fmt.Fprintln(os.Stderr, "EEPROM bootstrap needs --product, --model and --hwrev together")
			os.Exit(2)
		}
		product, err := parseByteSpec(bootstrapProd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --product %q: %v\n", bootstrapProd, err)
			os.Exit(2)
		}
		model, err := parseModelCode(bootstrapModel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --model %q: %v\n", bootstrapModel, err)
			os.Exit(2)
		}
		bootOpts = &bootstrapOptions{Product: product, Model: model, HWRev: byte(bootstrapHWRev)}
		if strings.TrimSpace(bootstrapPlat) != "" {
			fmt.Fprintf(os.Stderr, "note: --platform=%s is currently informational in Go\n", strings.TrimSpace(bootstrapPlat))
		}
	}
	if bootOpts == nil && romFlag {
		fmt.Println("Please input EEPROM bootstrap parameters:")
		fmt.Println("")
		p, err := promptString("Product (hex or int):\t")
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid product: %v\n", err)
			os.Exit(2)
		}
		product, err := parseByteSpec(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid product: %v\n", err)
			os.Exit(2)
		}
		m, err := promptString("Model (hex, ex A1):\t")
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid model: %v\n", err)
			os.Exit(2)
		}
		model, err := parseModelCode(m)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid model: %v\n", err)
			os.Exit(2)
		}
		h, err := promptInt("HW revision (1-255):\t")
		if err != nil || h < 1 || h > 255 {
			fmt.Fprintf(os.Stderr, "invalid hwrev: %v\n", err)
			os.Exit(2)
		}
		fmt.Println("")
		bootOpts = &bootstrapOptions{Product: product, Model: model, HWRev: byte(h)}
	}

	// Operations that require an attached serial device.
	needsDevice := len(pendingOps) > 0
	if needsDevice {
		if strings.TrimSpace(portArg) == "" {
			selected, err := promptSelectSerialPort()
			if err != nil {
				fmt.Fprintf(os.Stderr, "missing serial port and could not select one: %v\n", err)
				os.Exit(2)
			}
			portArg = selected
		}
		sp, err := OpenSerialPort(portArg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not open serial port %q: %v\n", portArg, err)
			os.Exit(2)
		}
		defer sp.Close()

		node := NewRNode(sp)
		go node.ReadLoop()

		if err := node.Detect(); err != nil {
			fmt.Fprintf(os.Stderr, "could not send detect command: %v\n", err)
			os.Exit(2)
		}
		// Give the read loop time to populate fields.
		deadline := time.Now().Add(2 * time.Second)
		for !node.Detected && time.Now().Before(deadline) {
			time.Sleep(25 * time.Millisecond)
		}

		needEEPROM := infoFlag || configFlag || normalModeFlag || tncModeFlag || signFlag ||
			strings.TrimSpace(firmwareHash) != "" || getTargetHash || getDeviceHash
		if needEEPROM {
			if err := node.DownloadEEPROM(1200 * time.Millisecond); err == nil {
				node.verifyDeviceSignature(paths)
			}
		}

		// Basic info/config dumps.
		if infoFlag {
			printNodeInfo(node)
			if node.Provisioned && !node.SignatureValid {
				fmt.Println("")
				fmt.Println("     ")
				fmt.Println("     WARNING! This device is NOT verifiable and should NOT be trusted.")
				fmt.Println("     Someone could have added privacy-breaking or malicious code to it.")
				fmt.Println("     ")
				fmt.Println("     Please verify the signing key is present on this machine.")
				fmt.Println("     Autogenerated keys will not match another machine's signature.")
				fmt.Println("     ")
				fmt.Println("     Proceed at your own risk and responsibility! If you created this")
				fmt.Println("     device yourself, please read the documentation on how to sign your")
				fmt.Println("     device to avoid this warning.")
				fmt.Println("     ")
			}
		}
		if extractFlag {
			if err := runExtraction(node, sp.Name(), baudFlashVal, paths); err != nil {
				fmt.Fprintf(os.Stderr, "firmware extraction failed: %v\n", err)
				if ee, ok := asExitError(err); ok {
					os.Exit(ee.Code)
				}
				os.Exit(2)
			}
		}
		prepared := PreparedFirmware{}
		if useExtracted {
			prepared.Extracted = true
		}
		// Online firmware: if user asked for update/autoinstall/flash without extracted firmware,
		// prepare it now (needs model information from the device).
		if (autoinstallFlag || updateFlag || flashFlag) && !prepared.Extracted && (strings.HasPrefix(firmwareURL, "http") || selectedVersion != "") {
			fw, err := ensureOnlineFirmwarePrepared(paths, node, firmwareURL, selectedVersion, updateFlag, autoinstallFlag, noCheckFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "online firmware preparation failed: %v\n", err)
				if ee, ok := asExitError(err); ok {
					os.Exit(ee.Code)
				}
				os.Exit(2)
			}
			prepared = fw
		}
		// Local file:// or path --fw-url
		if (autoinstallFlag || updateFlag || flashFlag) && !prepared.Extracted && firmwareURL != "" && !strings.HasPrefix(firmwareURL, "http") {
			prepared.LocalPath = firmwareURL
			prepared.IsZIP = strings.HasSuffix(strings.ToLower(firmwareURL), ".zip")
			if prepared.IsZIP && node.Platform != ROM_PLATFORM_NRF52 {
				if err := extractFirmwareZip(firmwareURL, paths); err != nil {
					fmt.Fprintf(os.Stderr, "could not extract firmware zip: %v\n", err)
					os.Exit(2)
				}
				prepared.Extracted = true
			}
		}
		if (autoinstallFlag || updateFlag || flashFlag) && prepared.LocalPath == "" && !prepared.Extracted {
			// In case we have extracted firmware already on disk, but useExtracted wasn't set.
			if err := verifyExtractedFirmware(paths); err == nil {
				prepared.Extracted = true
			}
		}

		if updateFlag && prepared.Version != "" && prepared.Version != "custom" && prepared.Version == node.Version && !forceUpdateFlag {
			fmt.Printf("Installed firmware already matches version %s, skipping (use --force-update to override)\n", node.Version)
			return
		}

		if autoinstallFlag || updateFlag {
			if err := runFlash(node, sp.Name(), baudFlashVal, paths, prepared); err != nil {
				fmt.Fprintf(os.Stderr, "update/autoinstall failed: %v\n", err)
				if ee, ok := asExitError(err); ok {
					os.Exit(ee.Code)
				}
				os.Exit(2)
			}
			if hash, err := extractedFirmwareHash(paths); err == nil {
				_ = node.SetFirmwareHash(hash)
			}
			if autoinstallFlag && bootOpts != nil {
				if err := runROMBootstrap(node, paths, *bootOpts); err != nil {
					fmt.Fprintf(os.Stderr, "EEPROM bootstrap failed: %v\n", err)
					os.Exit(2)
				}
			}
		}
		if flashFlag {
			if prepared.LocalPath == "" && !prepared.Extracted {
				// If user requested explicit flash with already extracted firmware.
				prepared.Extracted = useExtracted
			}
			if err := runFlash(node, sp.Name(), baudFlashVal, paths, prepared); err != nil {
				fmt.Fprintf(os.Stderr, "flash failed: %v\n", err)
				if ee, ok := asExitError(err); ok {
					os.Exit(ee.Code)
				}
				os.Exit(2)
			}
			if prepared.Extracted {
				if hash, err := extractedFirmwareHash(paths); err == nil {
					_ = node.SetFirmwareHash(hash)
				}
			}
			if bootOpts != nil {
				if err := runROMBootstrap(node, paths, *bootOpts); err != nil {
					fmt.Fprintf(os.Stderr, "EEPROM bootstrap failed: %v\n", err)
					os.Exit(2)
				}
			}
		}
		if romFlag {
			if err := runROMBootstrap(node, paths, *bootOpts); err != nil {
				fmt.Fprintf(os.Stderr, "ROM bootstrap failed: %v\n", err)
				os.Exit(2)
			}
		}
		if signFlag {
			if !node.Provisioned {
				fmt.Fprintln(os.Stderr, "This device has not been provisioned yet, cannot create device signature")
				os.Exit(79)
			}
			if len(node.Checksum) != 16 {
				fmt.Fprintln(os.Stderr, "No EEPROM checksum present, cannot sign device")
				os.Exit(78)
			}
			der, err := ensureSignerKey(paths)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Could not load local signing key")
				os.Exit(78)
			}
			priv, err := parseSignerKeyDER(der)
			if err != nil || priv.N.BitLen() != 1024 {
				fmt.Fprintln(os.Stderr, "Could not deserialize local signing key")
				os.Exit(78)
			}
			checksumHash := sha256.Sum256(node.Checksum)
			sig, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, checksumHash[:], nil)
			if err != nil || len(sig) != 128 {
				fmt.Fprintln(os.Stderr, "Error while signing EEPROM")
				os.Exit(78)
			}
			if err := node.StoreSignature(sig); err != nil {
				fmt.Fprintf(os.Stderr, "failed to store device signature: %v\n", err)
				os.Exit(2)
			}
			_ = node.DownloadEEPROM(1200 * time.Millisecond)
			node.verifyDeviceSignature(paths)
			fmt.Println("Device signed")
		}
		if strings.TrimSpace(firmwareHash) != "" {
			if !node.Provisioned {
				fmt.Fprintln(os.Stderr, "This device has not been provisioned yet, cannot set firmware hash")
				os.Exit(77)
			}
			raw, err := hex.DecodeString(strings.TrimSpace(firmwareHash))
			if err != nil || len(raw) != 32 {
				fmt.Fprintln(os.Stderr, "The provided value was not a valid SHA256 hash")
				os.Exit(78)
			}
			if err := node.SetFirmwareHash(raw); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set firmware hash: %v\n", err)
				os.Exit(2)
			}
			fmt.Println("Firmware hash set")
		}
		if getTargetHash {
			if !node.Provisioned {
				fmt.Fprintln(os.Stderr, "This device has not been provisioned yet, cannot get firmware hash")
				os.Exit(77)
			}
			if len(node.FirmwareHashTarget) > 0 {
				fmt.Printf("The target firmware hash is: %x\n", node.FirmwareHashTarget)
			}
		}
		if getDeviceHash {
			if !node.Provisioned {
				fmt.Fprintln(os.Stderr, "This device has not been provisioned yet, cannot get firmware hash")
				os.Exit(77)
			}
			if len(node.FirmwareHash) > 0 {
				fmt.Printf("The actual firmware hash is: %x\n", node.FirmwareHash)
			}
		}

		// Modes.
		if normalModeFlag {
			if !node.Provisioned {
				fmt.Fprintln(os.Stderr, "This device contains a valid firmware, but EEPROM is invalid.")
				fmt.Fprintln(os.Stderr, "Probably the device has not been initialised, or the EEPROM has been erased.")
				fmt.Fprintln(os.Stderr, "Please correctly initialise the device and try again!")
				os.Exit(2)
			}
			if err := node.SetNormalMode(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set normal mode: %v\n", err)
				os.Exit(2)
			}
			fmt.Println("Device set to normal (host-controlled) operating mode")
		}
		if tncModeFlag {
			if !node.Provisioned {
				fmt.Fprintln(os.Stderr, "This device contains a valid firmware, but EEPROM is invalid.")
				fmt.Fprintln(os.Stderr, "Probably the device has not been initialised, or the EEPROM has been erased.")
				fmt.Fprintln(os.Stderr, "Please correctly initialise the device and try again!")
				os.Exit(2)
			}

			needsPrompt := freqHz == 0 || bwHz == 0 || txPower == 0 || sfValue == 0 || crValue == 0
			if needsPrompt {
				fmt.Println("Please input startup configuration:")
			}
			fmt.Println("")

			if freqHz == 0 {
				v, err := promptInt("Frequency in Hz:\t")
				if err != nil {
					fmt.Fprintf(os.Stderr, "invalid frequency: %v\n", err)
					os.Exit(2)
				}
				freqHz = v
			}
			if bwHz == 0 {
				v, err := promptInt("Bandwidth in Hz:\t")
				if err != nil {
					fmt.Fprintf(os.Stderr, "invalid bandwidth: %v\n", err)
					os.Exit(2)
				}
				bwHz = v
			}
			if txPower == 0 || txPower < 0 || txPower > 17 {
				v, err := promptInt("TX Power in dBm:\t")
				if err != nil {
					fmt.Fprintf(os.Stderr, "invalid tx power: %v\n", err)
					os.Exit(2)
				}
				txPower = v
			}
			if sfValue == 0 {
				v, err := promptInt("Spreading factor:\t")
				if err != nil {
					fmt.Fprintf(os.Stderr, "invalid spreading factor: %v\n", err)
					os.Exit(2)
				}
				sfValue = v
			}
			if crValue == 0 {
				v, err := promptInt("Coding rate:\t\t")
				if err != nil {
					fmt.Fprintf(os.Stderr, "invalid coding rate: %v\n", err)
					os.Exit(2)
				}
				crValue = v
			}
			fmt.Println("")

			_ = node.SetFrequency(freqHz)
			_ = node.SetBandwidth(bwHz)
			_ = node.SetTXPower(txPower)
			_ = node.SetSpreadingFactor(sfValue)
			_ = node.SetCodingRate(crValue)
			if err := node.InitRadio(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to initialise radio: %v\n", err)
				os.Exit(2)
			}
			if node.Serial == nil || !strings.HasPrefix(node.Serial.Name(), "sim://") {
				time.Sleep(500 * time.Millisecond)
			}
			if err := node.SetTNCMode(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set TNC mode: %v\n", err)
				os.Exit(2)
			}
			fmt.Println("Device set to TNC operating mode")
			if node.Serial == nil || !strings.HasPrefix(node.Serial.Name(), "sim://") {
				time.Sleep(1 * time.Second)
			}
		}

		// Bluetooth.
		if btOnFlag {
			if err := node.EnableBluetooth(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to enable bluetooth: %v\n", err)
				os.Exit(2)
			}
		}
		if btOffFlag {
			if err := node.DisableBluetooth(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to disable bluetooth: %v\n", err)
				os.Exit(2)
			}
		}
		if btPairFlag {
			if err := node.BluetoothPair(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to start bluetooth pairing: %v\n", err)
				os.Exit(2)
			}
		}

		// WiFi settings.
		if wifiModeNormalized != "" {
			var modeByte byte
			switch wifiModeNormalized {
			case "OFF":
				modeByte = 0x00
			case "AP":
				// Python parity: sta=0x01, ap=0x02
				modeByte = 0x02
			case "STATION":
				modeByte = 0x01
			}
			if err := node.SetWifiMode(modeByte); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set wifi mode: %v\n", err)
				os.Exit(2)
			}
		}
		if wifiChannelInt != 0 {
			if err := node.SetWifiChannel(wifiChannelInt); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set wifi channel: %v\n", err)
				os.Exit(2)
			}
		}
		if wifiSSID != "" {
			if strings.EqualFold(strings.TrimSpace(wifiSSID), "none") {
				wifiSSID = ""
			}
			if err := node.SetWifiSSID(wifiSSID); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set wifi ssid: %v\n", err)
				os.Exit(2)
			}
		}
		if wifiPSK != "" {
			if strings.EqualFold(strings.TrimSpace(wifiPSK), "none") {
				wifiPSK = ""
			}
			if err := node.SetWifiPSK(wifiPSK); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set wifi psk: %v\n", err)
				os.Exit(2)
			}
		}
		if wifiIP != "" {
			if err := node.SetWifiIP(wifiIP); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set wifi ip: %v\n", err)
				os.Exit(2)
			}
		}
		if wifiNM != "" {
			if err := node.SetWifiNetmask(wifiNM); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set wifi netmask: %v\n", err)
				os.Exit(2)
			}
		}

		// Display / LEDs.
		if displayIntensity != -1 {
			if err := node.SetDisplayIntensity(byte(displayIntensity)); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set display intensity: %v\n", err)
				os.Exit(2)
			}
		}
		if displayTimeout != -1 {
			if err := node.SetDisplayBlanking(byte(displayTimeout)); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set display timeout: %v\n", err)
				os.Exit(2)
			}
		}
		if displayRotation != -1 {
			if err := node.SetDisplayRotation(byte(displayRotation)); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set display rotation: %v\n", err)
				os.Exit(2)
			}
		}
		if displayAddressHex != "" {
			if err := node.SetDisplayAddress(displayAddr); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set display address: %v\n", err)
				os.Exit(2)
			}
		}
		if reconditionDisplay {
			if err := node.ReconditionDisplay(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to recondition display: %v\n", err)
				os.Exit(2)
			}
		}
		if neopixelIntensity != -1 {
			if err := node.SetNeopixelIntensity(byte(neopixelIntensity)); err != nil {
				fmt.Fprintf(os.Stderr, "failed to set neopixel intensity: %v\n", err)
				os.Exit(2)
			}
		}

		// Radio config (TNC mode).
		if freqHz != 0 {
			_ = node.SetFrequency(freqHz)
		}
		if bwHz != 0 {
			_ = node.SetBandwidth(bwHz)
		}
		if txPower != 0 {
			_ = node.SetTXPower(txPower)
		}
		if sfValue != 0 {
			_ = node.SetSpreadingFactor(sfValue)
		}
		if crValue != 0 {
			_ = node.SetCodingRate(crValue)
		}
		if freqHz != 0 || bwHz != 0 || txPower != 0 || sfValue != 0 || crValue != 0 {
			if err := node.InitRadio(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to initialise radio: %v\n", err)
				os.Exit(2)
			}
		}

		if iaEnable {
			_ = node.SetDisableInterferenceAvoidance(false)
		}
		if iaDisable {
			_ = node.SetDisableInterferenceAvoidance(true)
		}

		// Device config output (Python parity): print after applying modifications.
		if configFlag {
			if err := node.DownloadEEPROM(1200 * time.Millisecond); err != nil {
				fmt.Fprintf(os.Stderr, "failed to read EEPROM: %v\n", err)
				os.Exit(2)
			}
			node.verifyDeviceSignature(paths)
			if err := node.DownloadCfgSector(800 * time.Millisecond); err != nil {
				fmt.Fprintf(os.Stderr, "could not read config sector: %v\n", err)
				os.Exit(2)
			}
			printDeviceConfig(node, showPSK)
		}

		// EEPROM operations.
		if eepromDumpFlag || eepromBackupFlag {
			if err := node.DownloadEEPROM(1200 * time.Millisecond); err != nil {
				fmt.Fprintf(os.Stderr, "failed to read EEPROM: %v\n", err)
				os.Exit(2)
			}
			if eepromDumpFlag {
				fmt.Printf("EEPROM (%d bytes):\n%s\n", len(node.EEPROM), hex.Dump(node.EEPROM))
			}
			if eepromBackupFlag {
				name := fmt.Sprintf("eeprom_%s_%d.bin", sanitizeFilename(sp.Name()), time.Now().Unix())
				outPath := filepath.Join(paths.EEPROMDir, name)
				if err := os.WriteFile(outPath, node.EEPROM, 0o600); err != nil {
					fmt.Fprintf(os.Stderr, "failed to write EEPROM backup: %v\n", err)
					os.Exit(2)
				}
				fmt.Printf("EEPROM backed up to %s\n", outPath)
			}
		}
		if eepromWipeFlag {
			if err := node.WipeEEPROM(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to wipe EEPROM: %v\n", err)
				os.Exit(2)
			}
			fmt.Println("EEPROM wipe completed")
		}

		return
	}

	flag.Usage()
}

func runExtraction(node *RNode, portPath string, baudFlash int, paths storagePaths) error {
	if node == nil {
		return errors.New("nil device")
	}
	if portPath == "" {
		return errors.New("missing port path")
	}

	fmt.Println()
	fmt.Println("RNode Firmware Extraction")
	fmt.Println()
	fmt.Println("Probing device...")

	if !node.Detected {
		return errors.New("no answer from device")
	}
	if node.Platform != ROM_PLATFORM_ESP32 {
		return exitError{Code: 170, Err: errors.New("firmware extraction is currently only supported on ESP32-based RNodes")}
	}

	// Simulation shortcut (no external tool execution).
	if strings.HasPrefix(portPath, "sim://") {
		u, _ := url.Parse(portPath)
		switch u.Query().Get("extract") {
		case "fail182":
			return exitError{Code: 182, Err: errors.New("simulated extraction part failure")}
		case "fail180":
			return exitError{Code: 180, Err: errors.New("simulated extraction failed")}
		default:
			// Create expected extracted files for downstream flows.
			_ = os.MkdirAll(paths.ExtractedDir, 0o755)
			versionLine := fmt.Sprintf("%s %s\n", node.Version, hex.EncodeToString(node.FirmwareHash))
			_ = os.WriteFile(filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.version"), []byte(versionLine), 0o600)
			parts := []string{
				"extracted_rnode_firmware.bootloader",
				"extracted_rnode_firmware.partitions",
				"extracted_rnode_firmware.boot_app0",
				"extracted_rnode_firmware.bin",
				"extracted_console_image.bin",
			}
			for _, name := range parts {
				_ = os.WriteFile(filepath.Join(paths.ExtractedDir, name), []byte("sim"), 0o600)
			}
			fmt.Println("Firmware successfully extracted!")
			return nil
		}
	}

	// This matches Python behaviour: extraction depends on firmware hashes being available.
	if len(node.FirmwareHash) == 0 || node.Version == "" {
		fmt.Println("Warning: firmware hash/version not available from device.")
		fmt.Println("Try running --info first, then rerun --extract.")
	}

	versionLine := fmt.Sprintf("%s %s\n", node.Version, hex.EncodeToString(node.FirmwareHash))
	versionPath := filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.version")
	if err := os.WriteFile(versionPath, []byte(versionLine), 0o600); err != nil {
		return fmt.Errorf("could not write extracted firmware version file: %w", err)
	}

	fmt.Println("Ready to extract firmware images from the RNode.")
	fmt.Printf("Wrote version file: %s\n", versionPath)
	fmt.Println()
	if err := ensureRecoveryTool(paths); err != nil {
		return exitError{Code: 181, Err: fmt.Errorf("Error: Could not extract recovery ESP-Tool. The contained exception was: %w", err)}
	}

	fmt.Printf("Using recovery tool: %s\n", paths.RecoveryTool)
	fmt.Println("Attempting to run recovery tool to read flash regions...")
	if err := runRecoveryReadFlash(paths.RecoveryTool, portPath, baudFlash, paths.ExtractedDir); err != nil {
		return exitError{Code: 180, Err: fmt.Errorf("RNode firmware extraction failed: %w", err)}
	}
	fmt.Println("Firmware successfully extracted!")
	return nil
}

func runFlash(node *RNode, portPath string, baudFlash int, paths storagePaths, fw PreparedFirmware) error {
	if node == nil || portPath == "" {
		return errors.New("missing device/port")
	}

	// Simulation shortcut: allow deterministic branch coverage without external tools.
	if strings.HasPrefix(portPath, "sim://") {
		u, _ := url.Parse(portPath)
		switch u.Query().Get("flash") {
		case "fail1":
			return exitError{Code: 1, Err: errors.New("simulated flash failure")}
		case "missing_avrdude":
			// Python parity: missing flasher tool prints guidance and exits via graceful_exit() (default 0).
			return exitError{Code: 0, Err: errors.New("you do not currently have the \"avrdude\" program installed on your system\nunfortunately, that means we can't proceed, since it is needed to flash your\nboard. Please install \"avrdude\" and try again")}
		case "missing_nrfutil":
			return exitError{Code: 0, Err: errors.New("you do not currently have the \"adafruit-nrfutil\" program installed on your system\nunfortunately, that means we can't proceed, since it is needed to flash your\nboard.\n\n  pip3 install --user adafruit-nrfutil\n\nPlease install \"adafruit-nrfutil\" and try again")}
		case "flasher_error_baud":
			return exitError{Code: 0, Err: errors.New("error from flasher (1) while writing\nsome boards have trouble flashing at high speeds, and you can\ntry flashing with a lower baud rate, as in this example:\nrnodeconf --autoinstall --baud-flash 115200")}
		default:
			fmt.Println("Done flashing")
			return nil
		}
	}

	switch node.Platform {
	case ROM_PLATFORM_ESP32:
		if err := ensureRecoveryTool(paths); err != nil {
			return exitError{Code: 181, Err: fmt.Errorf("Error: Could not extract recovery ESP-Tool. The contained exception was: %w", err)}
		}
		if !fw.Extracted {
			return errors.New("ESP32 flashing requires extracted firmware (use --use-extracted, a firmware zip, or online download)")
		}
		if !fileExists(paths.RecoveryTool) {
			return fmt.Errorf("missing recovery tool %s (run Python rnodeconf once to extract it)", paths.RecoveryTool)
		}
		if err := verifyExtractedFirmware(paths); err != nil {
			return err
		}

		fmt.Println("Flashing extracted firmware images...")

		type part struct {
			offset string
			file   string
		}
		parts := []part{
			{"0x1000", filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.bootloader")},
			{"0xe000", filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.boot_app0")},
			{"0x8000", filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.partitions")},
			{"0x10000", filepath.Join(paths.ExtractedDir, "extracted_rnode_firmware.bin")},
			{"0x210000", filepath.Join(paths.ExtractedDir, "extracted_console_image.bin")},
		}
		for _, p := range parts {
			if !fileExists(p.file) {
				return fmt.Errorf("missing extracted firmware part %s", p.file)
			}
		}

		args := []string{
			paths.RecoveryTool,
			"--chip", "esp32",
			"--port", portPath,
			"--baud", strconv.Itoa(baudFlash),
			"--before", "default_reset",
			"--after", "hard_reset",
			"write_flash",
		}
		for _, p := range parts {
			args = append(args, p.offset, p.file)
		}
		cmd := exec.Command("python", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return exitError{Code: 1, Err: err}
		}

		// Best-effort: signal firmware update to the running firmware (if still responsive).
		_ = node.IndicateFirmwareUpdate()
		fmt.Println("Flash completed.")
		return nil

	case ROM_PLATFORM_AVR:
		if strings.TrimSpace(fw.LocalPath) == "" {
			return errors.New("AVR flashing requires a local firmware file")
		}
		avrdude, err := exec.LookPath("avrdude")
		if err != nil {
			// Python parity: prints guidance and exits via graceful_exit() default (0).
			return exitError{Code: 0, Err: errors.New("you do not currently have the \"avrdude\" program installed on your system\nunfortunately, that means we can't proceed, since it is needed to flash your\nboard. Please install \"avrdude\" and try again")}
		}
		args := []string{avrdude}
		// Parity with Python `get_flasher_call`
		switch filepath.Base(fw.LocalPath) {
		case "rnode_firmware.hex":
			args = append(args, "-P", portPath, "-p", "m1284p", "-c", "arduino", "-b", "115200", "-U", "flash:w:"+fw.LocalPath+":i")
		case "rnode_firmware_m2560.hex":
			args = append(args, "-P", portPath, "-p", "atmega2560", "-c", "wiring", "-D", "-b", "115200", "-U", "flash:w:"+fw.LocalPath)
		default:
			return fmt.Errorf("unknown AVR firmware filename %q", filepath.Base(fw.LocalPath))
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				code := ee.ExitCode()
				return exitError{Code: 0, Err: fmt.Errorf("error from flasher (%d) while writing\nsome boards have trouble flashing at high speeds, and you can\ntry flashing with a lower baud rate, as in this example:\nrnodeconf --autoinstall --baud-flash 115200", code)}
			}
			return exitError{Code: 1, Err: err}
		}
		return nil

	case ROM_PLATFORM_NRF52:
		if strings.TrimSpace(fw.LocalPath) == "" {
			return errors.New("NRF52 flashing requires a local firmware file")
		}
		nrfutil, err := exec.LookPath("adafruit-nrfutil")
		if err != nil {
			// Python parity: prints guidance and exits via graceful_exit() default (0).
			return exitError{Code: 0, Err: errors.New("you do not currently have the \"adafruit-nrfutil\" program installed on your system\nunfortunately, that means we can't proceed, since it is needed to flash your\nboard. You can install it via your package manager, for example:\n\n  pip3 install --user adafruit-nrfutil\n\nPlease install \"adafruit-nrfutil\" and try again")}
		}
		cmd := exec.Command(nrfutil, "dfu", "serial", "--package", fw.LocalPath, "-p", portPath, "-b", "115200", "-t", "1200")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				code := ee.ExitCode()
				return exitError{Code: 0, Err: fmt.Errorf("Error from flasher (%d) while writing.\nSome boards have trouble flashing at high speeds, and you can\ntry flashing with a lower baud rate, as in this example:\nrnodeconf --autoinstall --baud-flash 115200", code)}
			}
			return exitError{Code: 1, Err: err}
		}
		return nil
	default:
		return fmt.Errorf("unsupported platform 0x%02x", node.Platform)
	}
}

//go:embed recovery_esptool.b64
var recoveryToolB64 string

func ensureRecoveryTool(paths storagePaths) error {
	if fileExists(paths.RecoveryTool) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(paths.RecoveryTool), 0o755); err != nil {
		return err
	}
	decoded, err := base64.StdEncoding.DecodeString(recoveryToolB64)
	if err != nil {
		return err
	}
	reader, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return err
	}
	defer reader.Close()

	out, err := os.Create(paths.RecoveryTool)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, reader); err != nil {
		return err
	}
	if err := out.Chmod(0o755); err != nil {
		return err
	}
	return nil
}

func runROMBootstrap(node *RNode, paths storagePaths, opts bootstrapOptions) error {
	// Implements the same EEPROM provisioning fields as Python:
	// product/model/hwrev/serial/made/checksum/signature/lock.
	if node == nil {
		return errors.New("nil device")
	}
	if opts.HWRev == 0 || opts.Product == 0 || opts.Model == 0 {
		return errors.New("invalid bootstrap parameters")
	}

	serialNo, err := nextDeviceSerial(paths.DeviceSerialCounter)
	if err != nil {
		return fmt.Errorf("could not allocate device serial: %w", err)
	}
	timestamp := uint32(time.Now().Unix())

	infoChunk := make([]byte, 0, 11)
	infoChunk = append(infoChunk, opts.Product, opts.Model, opts.HWRev)
	infoChunk = append(infoChunk,
		byte(serialNo>>24), byte(serialNo>>16), byte(serialNo>>8), byte(serialNo),
		byte(timestamp>>24), byte(timestamp>>16), byte(timestamp>>8), byte(timestamp),
	)

	sum := md5.Sum(infoChunk)
	checksum := sum[:]

	der, err := ensureSignerKey(paths)
	if err != nil {
		return fmt.Errorf("could not load signing key: %w", err)
	}
	priv, err := parseSignerKeyDER(der)
	if err != nil {
		return fmt.Errorf("could not parse signing key: %w", err)
	}
	if priv.N.BitLen() != 1024 {
		return fmt.Errorf("signing key must be 1024-bit (got %d); delete %s and rerun to regenerate", priv.N.BitLen(), paths.SignerKeyPath)
	}
	checksumHash := sha256.Sum256(checksum)
	sig, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, checksumHash[:], nil)
	if err != nil {
		return fmt.Errorf("could not sign EEPROM checksum: %w", err)
	}
	if len(sig) != 128 {
		return fmt.Errorf("unexpected signature length %d (expected 128)", len(sig))
	}

	fmt.Println("Bootstrapping device EEPROM...")
	if err := writeEEPROMWithDelay(node, romAddrProduct, opts.Product); err != nil {
		return err
	}
	if err := writeEEPROMWithDelay(node, romAddrModel, opts.Model); err != nil {
		return err
	}
	if err := writeEEPROMWithDelay(node, romAddrHWRev, opts.HWRev); err != nil {
		return err
	}
	for i := 0; i < 4; i++ {
		if err := writeEEPROMWithDelay(node, romAddrSerial+byte(i), infoChunk[3+i]); err != nil {
			return err
		}
	}
	for i := 0; i < 4; i++ {
		if err := writeEEPROMWithDelay(node, romAddrMade+byte(i), infoChunk[7+i]); err != nil {
			return err
		}
	}
	for i := 0; i < 16; i++ {
		if err := writeEEPROMWithDelay(node, romAddrChecksum+byte(i), checksum[i]); err != nil {
			return err
		}
	}
	for i := 0; i < 128; i++ {
		if err := writeEEPROMWithDelay(node, romAddrSignature+byte(i), sig[i]); err != nil {
			return err
		}
	}
	if err := writeEEPROMWithDelay(node, romAddrInfoLock, romInfoLockByte); err != nil {
		return err
	}
	if node.Platform == ROM_PLATFORM_NRF52 {
		time.Sleep(3 * time.Second)
	}

	fmt.Println("EEPROM written! Validating...")
	if err := node.DownloadEEPROM(1200 * time.Millisecond); err != nil {
		return err
	}
	if !node.Provisioned {
		return errors.New("device did not report provisioned after bootstrap")
	}

	tryBackupDeviceEEPROM(paths, serialNo, node.EEPROM)

	// Save public key for external verification (Python stores it implicitly as trusted key).
	pubDERAny, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubDER := pubDERAny
	_, _, _ = storeTrustedKey(hex.EncodeToString(pubDER), paths.TrustedKeysDir)

	_ = node.IndicateFirmwareUpdate()
	fmt.Println("ROM bootstrap completed.")
	return nil
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, string(os.PathSeparator), "_")
	name = strings.ReplaceAll(name, " ", "_")
	if name == "" {
		return "rnode"
	}
	return name
}

func printNodeInfo(n *RNode) {
	fmt.Printf("Device: %s\n", n)
	if n.Version != "" {
		fmt.Printf("  Firmware: %s\n", n.Version)
	}
	if n.Platform != 0 {
		fmt.Printf("  Platform: 0x%02X\n", n.Platform)
	}
	if n.MCU != 0 {
		fmt.Printf("  MCU     : 0x%02X\n", n.MCU)
	}
	if n.Board != 0 {
		fmt.Printf("  Board   : 0x%02X\n", n.Board)
	}
	if n.Provisioned {
		sigStr := "Unverified"
		if n.SignatureValid {
			if n.LocallySigned {
				sigStr = "Validated - Local signature"
			} else if strings.TrimSpace(n.Vendor) != "" {
				sigStr = "Genuine board, vendor is " + n.Vendor
			} else {
				sigStr = "Validated"
			}
		}
		fmt.Printf("  Device signature: %s\n", sigStr)
	}
	if len(n.DeviceHash) > 0 {
		fmt.Printf("  Device hash: %x\n", n.DeviceHash)
	}
	if len(n.FirmwareHashTarget) > 0 {
		fmt.Printf("  FW hash (target): %x\n", n.FirmwareHashTarget)
	}
	if len(n.FirmwareHash) > 0 {
		fmt.Printf("  FW hash (device): %x\n", n.FirmwareHash)
	}
}

// ================== Device config output (Python parity) ==================

const (
	romAddrConfBT   = 0xB0
	romAddrConfDInt = 0xB2
	romAddrConfDAdr = 0xB3
	romAddrConfDBlk = 0xB4
	romAddrConfPSet = 0xB5
	romAddrConfPInt = 0xB6
	romAddrConfBSet = 0xB7
	romAddrConfDRot = 0xB8
	romAddrConfDIA  = 0xB9
	romAddrConfWiFi = 0xBA
	romAddrConfWChn = 0xBB
)

const (
	cfgAddrSSID = 0x00
	cfgAddrPSK  = 0x21
	cfgAddrIP   = 0x42
	cfgAddrNM   = 0x46
)

const confOKByte = 0x73

func printDeviceConfig(n *RNode, showPSK bool) {
	if len(n.EEPROM) == 0 && len(n.CfgSector) == 0 {
		fmt.Println("No device configuration data received")
		return
	}

	readEEPROM := func(addr int) (byte, bool) {
		if addr < 0 || addr >= len(n.EEPROM) {
			return 0, false
		}
		return n.EEPROM[addr], true
	}

	ecBT, _ := readEEPROM(romAddrConfBT)
	ecDInt, _ := readEEPROM(romAddrConfDInt)
	ecDAdr, _ := readEEPROM(romAddrConfDAdr)
	ecDBlk, _ := readEEPROM(romAddrConfDBlk)
	ecDRot, _ := readEEPROM(romAddrConfDRot)
	ecPSet, _ := readEEPROM(romAddrConfPSet)
	ecPInt, _ := readEEPROM(romAddrConfPInt)
	ecBSet, _ := readEEPROM(romAddrConfBSet)
	ecDIA, _ := readEEPROM(romAddrConfDIA)
	ecWiFi, _ := readEEPROM(romAddrConfWiFi)
	ecWChn, _ := readEEPROM(romAddrConfWChn)

	if ecWChn < 1 || ecWChn > 14 {
		ecWChn = 1
	}

	var (
		ecSSID string
		ecPSK  string
		ecIP   string
		ecNM   string
	)
	if len(n.CfgSector) > 0 {
		ecSSID = readCStringFF(n.CfgSector, cfgAddrSSID, 32)
		ecPSK = readCStringFF(n.CfgSector, cfgAddrPSK, 32)
		ecIP = readIPv4OrEmpty(n.CfgSector, cfgAddrIP)
		ecNM = readIPv4OrEmpty(n.CfgSector, cfgAddrNM)

		if ecWiFi == 0x02 {
			ecIP = "10.0.0.1"
			ecNM = "255.255.255.0"
		}

		if !showPSK && ecPSK != "" {
			ecPSK = strings.Repeat("*", len(ecPSK))
		}
	}

	fmt.Println("\nDevice configuration:")
	if ecBT == confOKByte {
		fmt.Println("  Bluetooth              : Enabled")
	} else {
		fmt.Println("  Bluetooth              : Disabled")
	}

	switch ecWiFi {
	case 0x01:
		fmt.Println("  WiFi                   : Enabled (Station)")
	case 0x02:
		fmt.Println("  WiFi                   : Enabled (AP)")
	default:
		fmt.Println("  WiFi                   : Disabled")
	}

	if ecWiFi == 0x01 || ecWiFi == 0x02 {
		fmt.Printf("    Channel              : %d\n", ecWChn)
		if ecSSID == "" {
			fmt.Println("    SSID                 : Not set")
		} else {
			fmt.Printf("    SSID                 : %s\n", ecSSID)
		}
		if ecPSK == "" {
			fmt.Println("    PSK                  : Not set")
		} else {
			fmt.Printf("    PSK                  : %s\n", ecPSK)
		}
		if ecIP == "" {
			fmt.Println("    IP Address           : DHCP")
		} else {
			fmt.Printf("    IP Address           : %s\n", ecIP)
		}
		if ecIP != "" && ecNM != "" {
			fmt.Printf("    Network Mask         : %s\n", ecNM)
		}
	}

	if ecDIA == 0x00 {
		fmt.Println("  Interference avoidance : Enabled")
	} else {
		fmt.Println("  Interference avoidance : Disabled")
	}

	fmt.Printf("  Display brightness     : %d\n", ecDInt)
	if ecDAdr == 0xFF {
		fmt.Println("  Display address        : Default")
	} else {
		fmt.Printf("  Display address        : %02x\n", ecDAdr)
	}
	if ecBSet == confOKByte && ecDBlk != 0x00 {
		fmt.Printf("  Display blanking       : %ds\n", ecDBlk)
	} else {
		fmt.Println("  Display blanking       : Disabled")
	}
	if ecDRot != 0xFF {
		rstr := "Unknown"
		switch ecDRot {
		case 0x00:
			rstr = "Landscape"
		case 0x01:
			rstr = "Portrait"
		case 0x02:
			rstr = "Landscape 180"
		case 0x03:
			rstr = "Portrait 180"
		}
		fmt.Printf("  Display rotation       : %s\n", rstr)
	} else {
		fmt.Println("  Display rotation       : Default")
	}
	if ecPSet == confOKByte {
		fmt.Printf("  Neopixel Intensity     : %d\n", ecPInt)
	}
	fmt.Println("")
}

func readCStringFF(buf []byte, offset, max int) string {
	if offset < 0 || max <= 0 || offset >= len(buf) {
		return ""
	}
	end := offset + max
	if end > len(buf) {
		end = len(buf)
	}
	out := make([]byte, 0, end-offset)
	for i := offset; i < end; i++ {
		b := buf[i]
		if b == 0xFF {
			b = 0x00
		}
		if b == 0x00 {
			break
		}
		out = append(out, b)
	}
	return strings.TrimSpace(string(out))
}

func readIPv4OrEmpty(buf []byte, offset int) string {
	if offset < 0 || offset+4 > len(buf) {
		return ""
	}
	ip := fmt.Sprintf("%d.%d.%d.%d", buf[offset], buf[offset+1], buf[offset+2], buf[offset+3])
	if ip == "255.255.255.255" || ip == "0.0.0.0" {
		return ""
	}
	return ip
}

// ================== KISS ==================

type KISSConst byte

const (
	KISS_FEND  = 0xC0
	KISS_FESC  = 0xDB
	KISS_TFEND = 0xDC
	KISS_TFESC = 0xDD

	KISS_CMD_UNKNOWN     = 0xFE
	KISS_CMD_DATA        = 0x00
	KISS_CMD_FREQUENCY   = 0x01
	KISS_CMD_BANDWIDTH   = 0x02
	KISS_CMD_TXPOWER     = 0x03
	KISS_CMD_SF          = 0x04
	KISS_CMD_CR          = 0x05
	KISS_CMD_RADIO_STATE = 0x06
	KISS_CMD_RADIO_LOCK  = 0x07
	KISS_CMD_DETECT      = 0x08
	KISS_CMD_LEAVE       = 0x0A
	KISS_CMD_READY       = 0x0F

	KISS_CMD_STAT_RX   = 0x21
	KISS_CMD_STAT_TX   = 0x22
	KISS_CMD_STAT_RSSI = 0x23
	KISS_CMD_STAT_SNR  = 0x24

	KISS_CMD_BLINK     = 0x30
	KISS_CMD_RANDOM    = 0x40
	KISS_CMD_DISP_INT  = 0x45
	KISS_CMD_NP_INT    = 0x65
	KISS_CMD_DISP_ADR  = 0x63
	KISS_CMD_DISP_BLNK = 0x64
	KISS_CMD_DISP_ROT  = 0x67
	KISS_CMD_DISP_RCND = 0x68

	KISS_CMD_BT_CTRL = 0x46
	KISS_CMD_BT_PIN  = 0x62

	KISS_CMD_DIS_IA      = 0x69
	KISS_CMD_WIFI_MODE   = 0x6A
	KISS_CMD_WIFI_SSID   = 0x6B
	KISS_CMD_WIFI_PSK    = 0x6C
	KISS_CMD_WIFI_CHN    = 0x6E
	KISS_CMD_WIFI_IP     = 0x84
	KISS_CMD_WIFI_NM     = 0x85
	KISS_CMD_BOARD       = 0x47
	KISS_CMD_PLATFORM    = 0x48
	KISS_CMD_MCU         = 0x49
	KISS_CMD_FW_VERSION  = 0x50
	KISS_CMD_CFG_READ    = 0x6D
	KISS_CMD_ROM_READ    = 0x51
	KISS_CMD_ROM_WRITE   = 0x52
	KISS_CMD_ROM_WIPE    = 0x59
	KISS_CMD_CONF_SAVE   = 0x53
	KISS_CMD_CONF_DELETE = 0x54
	KISS_CMD_RESET       = 0x55
	KISS_CMD_DEV_HASH    = 0x56
	KISS_CMD_DEV_SIG     = 0x57
	KISS_CMD_HASHES      = 0x60
	KISS_CMD_FW_HASH     = 0x58
	KISS_CMD_FW_UPD      = 0x61

	KISS_DETECT_REQ  = 0x73
	KISS_DETECT_RESP = 0x46

	KISS_RADIO_STATE_OFF = 0x00
	KISS_RADIO_STATE_ON  = 0x01
	KISS_RADIO_STATE_ASK = 0xFF

	KISS_CMD_ERROR           = 0x90
	KISS_ERROR_INITRADIO     = 0x01
	KISS_ERROR_TXFAILED      = 0x02
	KISS_ERROR_EEPROM_LOCKED = 0x03
)

// Escape mirrors Python behaviour.
func kissEscape(data []byte) []byte {
	data = bytesReplace(data, []byte{KISS_FESC}, []byte{KISS_FESC, KISS_TFESC})
	data = bytesReplace(data, []byte{KISS_FEND}, []byte{KISS_FESC, KISS_TFEND})
	return data
}

// small helper to avoid pulling in bytes.ReplaceAll repeatedly
func bytesReplace(src, old, new []byte) []byte {
	out := make([]byte, 0, len(src))
	for i := 0; i < len(src); {
		if i+len(old) <= len(src) && equalBytes(src[i:i+len(old)], old) {
			out = append(out, new...)
			i += len(old)
		} else {
			out = append(out, src[i])
			i++
		}
	}
	return out
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ================== ROM / models ==================

const (
	ROM_PLATFORM_AVR   = 0x90
	ROM_PLATFORM_ESP32 = 0x80
	ROM_PLATFORM_NRF52 = 0x70

	ROM_MCU_1284P = 0x91
	ROM_MCU_2560  = 0x92
	ROM_MCU_ESP32 = 0x81
	ROM_MCU_NRF52 = 0x71

	ROM_PRODUCT_RNODE = 0x03
	ROM_MODEL_A1      = 0xA1
	ROM_MODEL_A6      = 0xA6
	ROM_MODEL_A4      = 0xA4
	ROM_MODEL_A9      = 0xA9
	ROM_MODEL_A3      = 0xA3
	ROM_MODEL_A8      = 0xA8
	ROM_MODEL_A2      = 0xA2
	ROM_MODEL_A7      = 0xA7
	ROM_MODEL_A5      = 0xA5
	ROM_MODEL_AA      = 0xAA
	ROM_MODEL_AC      = 0xAC
	// ... other ROM constants from Python (PRODUCT_T32_10/20/21, H32_V2/3/4, TBEAM, TDECK, etc.)

	ROM_BOARD_RNODE         = 0x31
	ROM_BOARD_HMBRW         = 0x32
	ROM_BOARD_TBEAM         = 0x33
	ROM_BOARD_TDECK         = 0x3B
	ROM_BOARD_HUZZAH32      = 0x34
	ROM_BOARD_GENERIC_ESP32 = 0x35
	ROM_BOARD_LORA32_V2_0   = 0x36
	ROM_BOARD_LORA32_V2_1   = 0x37
	ROM_BOARD_TECHO         = 0x43
	ROM_BOARD_RAK4631       = 0x51
)

// For brevity, the full ROM block is not duplicated here; it can be copied from Python
// 1:1 as consts above.

// For models, in Go it's convenient to use a struct:
type ModelInfo struct {
	MinFreq    int64  // Hz
	MaxFreq    int64  // Hz
	MaxOutput  int    // dBm
	BandString string // "410-525 MHz"
	FWFile     string // firmware filename or ""
	Chip       string // SX1278/SX1262/...
}

// Full table like models = {...} in Python:
var Models = map[byte]ModelInfo{
	0xA4: {410000000, 525000000, 14, "410 - 525 MHz", "rnode_firmware.hex", "SX1278"},
	0xA9: {820000000, 1020000000, 17, "820 - 1020 MHz", "rnode_firmware.hex", "SX1276"},
	0xA1: {410000000, 525000000, 22, "410 - 525 MHz", "rnode_firmware_t3s3.zip", "SX1268"},
	0xA6: {820000000, 1020000000, 22, "820 - 960 MHz", "rnode_firmware_t3s3.zip", "SX1262"},
	0xA5: {410000000, 525000000, 17, "410 - 525 MHz", "rnode_firmware_t3s3_sx127x.zip", "SX1278"},
	0xAA: {820000000, 1020000000, 17, "820 - 960 MHz", "rnode_firmware_t3s3_sx127x.zip", "SX1276"},
	0xAC: {2400000000, 2500000000, 20, "2.4 - 2.5 GHz", "rnode_firmware_t3s3_sx1280_pa.zip", "SX1280"},
	0xA2: {410000000, 525000000, 17, "410 - 525 MHz", "rnode_firmware_ng21.zip", "SX1278"},
	0xA7: {820000000, 1020000000, 17, "820 - 1020 MHz", "rnode_firmware_ng21.zip", "SX1276"},
	0xA3: {410000000, 525000000, 17, "410 - 525 MHz", "rnode_firmware_ng20.zip", "SX1278"},
	0xA8: {820000000, 1020000000, 17, "820 - 1020 MHz", "rnode_firmware_ng20.zip", "SX1276"},
	0xB3: {420000000, 520000000, 17, "420 - 520 MHz", "rnode_firmware_lora32v20.zip", "SX1278"},
	0xB8: {850000000, 950000000, 17, "850 - 950 MHz", "rnode_firmware_lora32v20.zip", "SX1276"},
	0xB4: {420000000, 520000000, 17, "420 - 520 MHz", "rnode_firmware_lora32v21.zip", "SX1278"},
	0xB9: {850000000, 950000000, 17, "850 - 950 MHz", "rnode_firmware_lora32v21.zip", "SX1276"},
	0x04: {420000000, 520000000, 17, "420 - 520 MHz", "rnode_firmware_lora32v21_tcxo.zip", "SX1278"},
	0x09: {850000000, 950000000, 17, "850 - 950 MHz", "rnode_firmware_lora32v21_tcxo.zip", "SX1276"},
	0xBA: {420000000, 520000000, 17, "420 - 520 MHz", "rnode_firmware_lora32v10.zip", "SX1278"},
	0xBB: {850000000, 950000000, 17, "850 - 950 MHz", "rnode_firmware_lora32v10.zip", "SX1276"},
	0xC4: {420000000, 520000000, 17, "420 - 520 MHz", "rnode_firmware_heltec32v2.zip", "SX1278"},
	0xC9: {850000000, 950000000, 17, "850 - 950 MHz", "rnode_firmware_heltec32v2.zip", "SX1276"},
	0xC5: {420000000, 520000000, 22, "420 - 520 MHz", "rnode_firmware_heltec32v3.zip", "SX1268"},
	0xCA: {850000000, 950000000, 22, "850 - 950 MHz", "rnode_firmware_heltec32v3.zip", "SX1262"},
	0xC8: {860000000, 930000000, 28, "850 - 950 MHz", "rnode_firmware_heltec32v4pa.zip", "SX1262"},
	0xC6: {420000000, 520000000, 22, "420 - 520 MHz", "rnode_firmware_heltec_t114.zip", "SX1268"},
	0xC7: {850000000, 950000000, 22, "850 - 950 MHz", "rnode_firmware_heltec_t114.zip", "SX1262"},
	0xE4: {420000000, 520000000, 17, "420 - 520 MHz", "rnode_firmware_tbeam.zip", "SX1278"},
	0xE9: {850000000, 950000000, 17, "850 - 950 MHz", "rnode_firmware_tbeam.zip", "SX1276"},
	0xD4: {420000000, 520000000, 22, "420 - 520 MHz", "rnode_firmware_tdeck.zip", "SX1268"},
	0xD9: {850000000, 950000000, 22, "850 - 950 MHz", "rnode_firmware_tdeck.zip", "SX1262"},
	0xDB: {420000000, 520000000, 22, "420 - 520 MHz", "rnode_firmware_tbeam_supreme.zip", "SX1268"},
	0xDC: {850000000, 950000000, 22, "850 - 950 MHz", "rnode_firmware_tbeam_supreme.zip", "SX1262"},
	0xE3: {420000000, 520000000, 22, "420 - 520 MHz", "rnode_firmware_tbeam_sx1262.zip", "SX1268"},
	0xE8: {850000000, 950000000, 22, "850 - 950 MHz", "rnode_firmware_tbeam_sx1262.zip", "SX1262"},
	0x11: {430000000, 510000000, 22, "430 - 510 MHz", "rnode_firmware_rak4631.zip", "SX1262"},
	0x12: {779000000, 928000000, 22, "779 - 928 MHz", "rnode_firmware_rak4631.zip", "SX1262"},
	0x13: {430000000, 510000000, 22, "430 - 510 MHz", "rnode_firmware_rak4631_sx1280.zip", "SX1262 + SX1280"},
	0x14: {779000000, 928000000, 22, "779 - 928 MHz", "rnode_firmware_rak4631_sx1280.zip", "SX1262 + SX1280"},
	0x16: {779000000, 928000000, 22, "430 - 510 Mhz", "rnode_firmware_techo.zip", "SX1262"},
	0x17: {779000000, 928000000, 22, "779 - 928 Mhz", "rnode_firmware_techo.zip", "SX1262"},
	0x21: {820000000, 960000000, 22, "820 - 960 MHz", "rnode_firmware_opencom_xl.zip", "SX1262 + SX1280"},
	0xDE: {420000000, 520000000, 22, "420 - 520 MHz", "rnode_firmware_xiao_esp32s3.zip", "SX1262"},
	0xDD: {850000000, 950000000, 22, "850 - 950 MHz", "rnode_firmware_xiao_esp32s3.zip", "SX1262"},
	0xFE: {100000000, 1100000000, 17, "(Band capabilities unknown)", "", "Unknown"},
	0xFF: {100000000, 1100000000, 14, "(Band capabilities unknown)", "", "Unknown"},
}

// ================== Serial abstraction ==================

type SerialPort interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Close() error

	// these two methods must be adapted to the chosen library
	BytesAvailable() (int, error)
	IsOpen() bool
	Name() string

	// Optional serial control lines (Python pyserial exposes these).
	// Implementations may return an error if unsupported.
	SetDTR(on bool) error
	SetRTS(on bool) error
}

// ================== RNode ==================

type RNode struct {
	Serial  SerialPort
	Timeout time.Duration

	RFrequency int
	RBandwidth int
	RTxPower   byte
	RSF        byte
	RCR        byte
	RState     byte
	RLock      byte

	SF        byte
	CR        byte
	TxPower   byte
	Frequency int
	Bandwidth int

	Detected     bool
	USBSerialID  string
	Platform     byte
	MCU          byte
	EEPROM       []byte
	CfgSector    []byte
	MajorVersion byte
	MinorVersion byte
	Version      string

	Provisioned bool
	Product     byte
	Board       byte
	Model       byte
	HWRev       byte
	Made        []byte
	SerialNo    []byte
	Checksum    []byte
	DeviceHash  []byte

	FirmwareHash       []byte
	FirmwareHashTarget []byte
	Signature          []byte
	SignatureValid     bool
	LocallySigned      bool
	Vendor             string
	MinFreq, MaxFreq   int
	MaxOutput          int
	Configured         bool
	ConfSF, ConfCR     byte
	ConfTxPower        byte
	ConfFrequency      int
	ConfBandwidth      int
	Bitrate            float64
	BitrateKbps        float64
}

// NewRNode is the constructor.
func NewRNode(sp SerialPort) *RNode {
	return &RNode{
		Serial:  sp,
		Timeout: 100 * time.Millisecond,
	}
}

func (n *RNode) String() string {
	if n.Serial != nil {
		return fmt.Sprintf("RNode(%s)", n.Serial.Name())
	}
	return "RNode(?)"
}

func (n *RNode) Disconnect() {
	n.Leave()
	if n.Serial != nil {
		_ = n.Serial.Close()
	}
}

// ================== ReadLoop (skeleton) ==================

// The upstream version uses non-blocking serial.in_waiting.
// Here it's simplified via BytesAvailable() + Read(1).
func (n *RNode) ReadLoop() {
	defer func() {
		if r := recover(); r != nil {
			rns.Logf(rns.LogError, "RNode read loop panic: %v", r)
		}
	}()

	inFrame := false
	escape := false
	cmd := byte(KISS_CMD_UNKNOWN)
	dataBuf := make([]byte, 0, 1024)
	cmdBuf := make([]byte, 0, 128)
	lastReadMs := time.Now()

	for n.Serial != nil && n.Serial.IsOpen() {
		avail, err := n.Serial.BytesAvailable()
		if err != nil {
			avail = 0
		}
		if avail > 0 {
			b := make([]byte, 1)
			_, err := n.Serial.Read(b)
			if err != nil {
				continue
			}
			byteVal := b[0]
			lastReadMs = time.Now()

			switch {
			case inFrame && byteVal == KISS_FEND && cmd == KISS_CMD_ROM_READ:
				n.EEPROM = append([]byte(nil), dataBuf...)
				inFrame = false
				dataBuf = dataBuf[:0]
				cmdBuf = cmdBuf[:0]

			case inFrame && byteVal == KISS_FEND && cmd == KISS_CMD_CFG_READ:
				n.CfgSector = append([]byte(nil), dataBuf...)
				inFrame = false
				dataBuf = dataBuf[:0]
				cmdBuf = cmdBuf[:0]

			case byteVal == KISS_FEND:
				inFrame = true
				cmd = KISS_CMD_UNKNOWN
				dataBuf = dataBuf[:0]
				cmdBuf = cmdBuf[:0]

			case inFrame && len(dataBuf) < 1024:
				// first byte is the command
				if len(dataBuf) == 0 && cmd == KISS_CMD_UNKNOWN {
					cmd = byteVal
				} else {
					n.handleKISSByte(&cmd, &escape, &dataBuf, &cmdBuf, byteVal)
				}
			}
		} else {
			if len(dataBuf) > 0 && time.Since(lastReadMs) > n.Timeout {
				rns.Logf(rns.LogDebug, "%s serial read timeout", n)
				dataBuf = dataBuf[:0]
				inFrame = false
				cmd = KISS_CMD_UNKNOWN
				escape = false
			}
			time.Sleep(80 * time.Millisecond)
		}
	}
}

// Byte parsing by command (trimmed to what's most commonly needed).
// The rest can be added from Python if needed.
func (n *RNode) handleKISSByte(
	cmd *byte,
	escape *bool,
	dataBuf *[]byte,
	cmdBuf *[]byte,
	b byte,
) {
	switch *cmd {
	case KISS_CMD_ROM_READ, KISS_CMD_CFG_READ, KISS_CMD_DATA,
		KISS_CMD_FREQUENCY, KISS_CMD_BANDWIDTH,
		KISS_CMD_BT_PIN, KISS_CMD_DEV_HASH, KISS_CMD_HASHES,
		KISS_CMD_FW_VERSION, KISS_CMD_STAT_RX, KISS_CMD_STAT_TX:
		// shared escaping logic
		if b == KISS_FESC {
			*escape = true
			return
		}
		if *escape {
			if b == KISS_TFEND {
				b = KISS_FEND
			} else if b == KISS_TFESC {
				b = KISS_FESC
			}
			*escape = false
		}
		*cmdBuf = append(*cmdBuf, b)
		// then handle per-command lengths
		switch *cmd {
		case KISS_CMD_FREQUENCY:
			if len(*cmdBuf) == 4 {
				n.RFrequency = int((*cmdBuf)[0])<<24 |
					int((*cmdBuf)[1])<<16 |
					int((*cmdBuf)[2])<<8 |
					int((*cmdBuf)[3])
				rns.Logf(rns.LogInfo, "Radio reporting frequency is %.3f MHz", float64(n.RFrequency)/1_000_000)
				n.updateBitrate()
			}
		case KISS_CMD_BANDWIDTH:
			if len(*cmdBuf) == 4 {
				n.RBandwidth = int((*cmdBuf)[0])<<24 |
					int((*cmdBuf)[1])<<16 |
					int((*cmdBuf)[2])<<8 |
					int((*cmdBuf)[3])
				rns.Logf(rns.LogInfo, "Radio reporting bandwidth is %.3f kHz", float64(n.RBandwidth)/1000)
				n.updateBitrate()
			}
		case KISS_CMD_BT_PIN:
			if len(*cmdBuf) == 4 {
				pin := int((*cmdBuf)[0])<<24 |
					int((*cmdBuf)[1])<<16 |
					int((*cmdBuf)[2])<<8 |
					int((*cmdBuf)[3])
				rns.Logf(rns.LogInfo, "Bluetooth pairing PIN is: %06d", pin)
			}
		case KISS_CMD_DEV_HASH:
			if len(*cmdBuf) == 32 {
				n.DeviceHash = append([]byte(nil), (*cmdBuf)...)
			}
		case KISS_CMD_HASHES:
			if len(*cmdBuf) == 33 {
				switch (*cmdBuf)[0] {
				case 0x01:
					n.FirmwareHashTarget = append([]byte(nil), (*cmdBuf)[1:]...)
				case 0x02:
					n.FirmwareHash = append([]byte(nil), (*cmdBuf)[1:]...)
				}
			}
		case KISS_CMD_FW_VERSION:
			if len(*cmdBuf) == 2 {
				n.MajorVersion = (*cmdBuf)[0]
				n.MinorVersion = (*cmdBuf)[1]
				n.updateVersion()
			}
		case KISS_CMD_STAT_RX:
			if len(*cmdBuf) == 4 {
				// if needed, persist RX counter
			}
		case KISS_CMD_STAT_TX:
			if len(*cmdBuf) == 4 {
				// TX counter
			}
		default:
			// ROM_READ / CFG_READ / DATA: just accumulate into dataBuf
			*dataBuf = append(*dataBuf, b)
		}
	case KISS_CMD_BOARD:
		n.Board = b
	case KISS_CMD_PLATFORM:
		n.Platform = b
	case KISS_CMD_MCU:
		n.MCU = b
	case KISS_CMD_TXPOWER:
		n.RTxPower = b
		rns.Logf(rns.LogInfo, "Radio reporting TX power is %d dBm", n.RTxPower)
	case KISS_CMD_SF:
		n.RSF = b
		rns.Logf(rns.LogInfo, "Radio reporting spreading factor is %d", n.RSF)
		n.updateBitrate()
	case KISS_CMD_CR:
		n.RCR = b
		rns.Logf(rns.LogInfo, "Radio reporting coding rate is %d", n.RCR)
		n.updateBitrate()
	case KISS_CMD_RADIO_STATE:
		n.RState = b
	case KISS_CMD_RADIO_LOCK:
		n.RLock = b
	case KISS_CMD_STAT_RSSI:
		// RSSI offset -157
		rssi := int(b) - 157
		_ = rssi
	case KISS_CMD_STAT_SNR:
		// signed 8-bit * 0.25
		s := int(int8(b)) * 25 / 100
		_ = s
	case KISS_CMD_ERROR:
		switch b {
		case KISS_ERROR_INITRADIO:
			rns.Logf(rns.LogError, "%s hardware initialisation error (code 0x%02x)", n, b)
		case KISS_ERROR_TXFAILED:
			rns.Logf(rns.LogError, "%s hardware TX error (code 0x%02x)", n, b)
		default:
			rns.Logf(rns.LogError, "%s hardware error (code 0x%02x)", n, b)
		}
	case KISS_CMD_DETECT:
		n.Detected = (b == KISS_DETECT_RESP)
	}
}

// ================== helper methods ==================

func (n *RNode) updateBitrate() {
	defer func() {
		if r := recover(); r != nil {
			n.Bitrate = 0
		}
	}()
	if n.RSF == 0 || n.RCR == 0 || n.RBandwidth == 0 {
		return
	}
	// formula like Python
	b := float64(n.RSF) * ((4.0 / float64(n.RCR)) / (math.Pow(2, float64(n.RSF)) / (float64(n.RBandwidth) / 1000.0))) * 1000.0
	n.Bitrate = b
	n.BitrateKbps = math.Round(b/10.0) / 100.0
}

func (n *RNode) updateVersion() {
	min := int(n.MinorVersion)
	if min < 10 {
		n.Version = fmt.Sprintf("%d0%d", n.MajorVersion, min)
	} else {
		n.Version = fmt.Sprintf("%d.%d", n.MajorVersion, min)
	}
}

// ================== high-level KISS commands ==================

func (n *RNode) Detect() error {
	cmd := []byte{
		KISS_FEND, KISS_CMD_DETECT, KISS_DETECT_REQ, KISS_FEND,
		KISS_FEND, KISS_CMD_FW_VERSION, 0x00, KISS_FEND,
		KISS_FEND, KISS_CMD_PLATFORM, 0x00, KISS_FEND,
		KISS_FEND, KISS_CMD_MCU, 0x00, KISS_FEND,
		KISS_FEND, KISS_CMD_BOARD, 0x00, KISS_FEND,
		KISS_FEND, KISS_CMD_DEV_HASH, 0x01, KISS_FEND,
		KISS_FEND, KISS_CMD_HASHES, 0x01, KISS_FEND,
		KISS_FEND, KISS_CMD_HASHES, 0x02, KISS_FEND,
	}
	w, err := n.Serial.Write(cmd)
	if err != nil || w != len(cmd) {
		return errors.New("error writing detect command")
	}
	return nil
}

func (n *RNode) Leave() error {
	if n.Serial == nil {
		return nil
	}
	cmd := []byte{KISS_FEND, KISS_CMD_LEAVE, 0xFF, KISS_FEND}
	w, err := n.Serial.Write(cmd)
	if err != nil || w != len(cmd) {
		return errors.New("error sending leave")
	}
	time.Sleep(1 * time.Second)
	return nil
}

func (n *RNode) SetDisplayIntensity(intensity byte) error {
	data := []byte{intensity}
	k := []byte{KISS_FEND, KISS_CMD_DISP_INT}
	k = append(k, data...)
	k = append(k, KISS_FEND)
	_, err := n.Serial.Write(k)
	return err
}

func (n *RNode) SetDisplayBlanking(timeoutSec byte) error {
	data := []byte{timeoutSec}
	k := []byte{KISS_FEND, KISS_CMD_DISP_BLNK}
	k = append(k, data...)
	k = append(k, KISS_FEND)
	_, err := n.Serial.Write(k)
	return err
}

func (n *RNode) SetDisplayRotation(rot byte) error {
	data := []byte{rot}
	k := []byte{KISS_FEND, KISS_CMD_DISP_ROT}
	k = append(k, data...)
	k = append(k, KISS_FEND)
	_, err := n.Serial.Write(k)
	return err
}

func (n *RNode) ReconditionDisplay() error {
	k := []byte{KISS_FEND, KISS_CMD_DISP_RCND, 0x01, KISS_FEND}
	_, err := n.Serial.Write(k)
	return err
}

func (n *RNode) SetDisableInterferenceAvoidance(disable bool) error {
	var v byte
	if disable {
		v = 0x01
	}
	k := []byte{KISS_FEND, KISS_CMD_DIS_IA, v, KISS_FEND}
	_, err := n.Serial.Write(k)
	return err
}

func (n *RNode) SetNeopixelIntensity(intensity byte) error {
	k := []byte{KISS_FEND, KISS_CMD_NP_INT, intensity, KISS_FEND}
	_, err := n.Serial.Write(k)
	return err
}

func (n *RNode) EnableBluetooth() error {
	k := []byte{KISS_FEND, KISS_CMD_BT_CTRL, 0x01, KISS_FEND}
	_, err := n.Serial.Write(k)
	return err
}

func (n *RNode) DisableBluetooth() error {
	k := []byte{KISS_FEND, KISS_CMD_BT_CTRL, 0x00, KISS_FEND}
	_, err := n.Serial.Write(k)
	return err
}

func (n *RNode) BluetoothPair() error {
	k := []byte{KISS_FEND, KISS_CMD_BT_CTRL, 0x02, KISS_FEND}
	_, err := n.Serial.Write(k)
	return err
}

// sendCommand wraps a payload in a KISS frame, optionally escaping it,
// and writes the result to the attached serial port.
func (n *RNode) sendCommand(cmd byte, payload []byte, escape bool) error {
	if n.Serial == nil {
		return errors.New("serial port not configured")
	}
	frame := make([]byte, 0, len(payload)+3)
	frame = append(frame, byte(KISS_FEND), cmd)
	if len(payload) > 0 {
		if escape {
			payload = kissEscape(payload)
		}
		frame = append(frame, payload...)
	}
	frame = append(frame, byte(KISS_FEND))

	written, err := n.Serial.Write(frame)
	if err != nil {
		return err
	}
	if written != len(frame) {
		return fmt.Errorf("short write, expected %d bytes, wrote %d", len(frame), written)
	}
	return nil
}

func (n *RNode) SetDisplayAddress(addr byte) error {
	return n.sendCommand(KISS_CMD_DISP_ADR, []byte{addr}, false)
}

func (n *RNode) StoreSignature(signature []byte) error {
	if len(signature) == 0 {
		return errors.New("signature is empty")
	}
	return n.sendCommand(KISS_CMD_DEV_SIG, signature, true)
}

func (n *RNode) SetFirmwareHash(hash []byte) error {
	if len(hash) == 0 {
		return errors.New("firmware hash is empty")
	}
	return n.sendCommand(KISS_CMD_FW_HASH, hash, true)
}

func (n *RNode) IndicateFirmwareUpdate() error {
	return n.sendCommand(KISS_CMD_FW_UPD, []byte{0x01}, false)
}

func (n *RNode) SetWifiMode(mode byte) error {
	return n.sendCommand(KISS_CMD_WIFI_MODE, []byte{mode}, false)
}

func (n *RNode) SetWifiChannel(channel int) error {
	if channel < 1 || channel > 14 {
		return fmt.Errorf("invalid Wi-Fi channel %d", channel)
	}
	return n.sendCommand(KISS_CMD_WIFI_CHN, []byte{byte(channel)}, true)
}

func (n *RNode) SetWifiIP(ip string) error {
	data, err := encodeIPv4(ip)
	if err != nil {
		return err
	}
	return n.sendCommand(KISS_CMD_WIFI_IP, data, true)
}

func (n *RNode) SetWifiNetmask(mask string) error {
	data, err := encodeIPv4(mask)
	if err != nil {
		return err
	}
	return n.sendCommand(KISS_CMD_WIFI_NM, data, true)
}

func (n *RNode) SetWifiSSID(ssid string) error {
	if len(ssid) == 0 {
		return n.sendCommand(KISS_CMD_WIFI_SSID, []byte{0x00}, true)
	}
	if len(ssid) > 32 {
		return errors.New("ssid must be 1-32 characters")
	}
	payload := append([]byte(ssid), 0x00)
	return n.sendCommand(KISS_CMD_WIFI_SSID, payload, true)
}

func (n *RNode) SetWifiPSK(psk string) error {
	if len(psk) == 0 {
		return n.sendCommand(KISS_CMD_WIFI_PSK, []byte{0x00}, true)
	}
	if len(psk) < 8 || len(psk) > 32 {
		return errors.New("psk must be 8-32 characters")
	}
	payload := append([]byte(psk), 0x00)
	return n.sendCommand(KISS_CMD_WIFI_PSK, payload, true)
}

func (n *RNode) InitRadio() error {
	if n.Frequency == 0 || n.Bandwidth == 0 || n.TxPower == 0 || n.SF == 0 || n.CR == 0 {
		return errors.New("radio parameters not fully configured")
	}
	if err := n.SetFrequency(n.Frequency); err != nil {
		return err
	}
	if err := n.SetBandwidth(n.Bandwidth); err != nil {
		return err
	}
	if err := n.SetTXPower(int(n.TxPower)); err != nil {
		return err
	}
	if err := n.SetSpreadingFactor(int(n.SF)); err != nil {
		return err
	}
	if err := n.SetCodingRate(int(n.CR)); err != nil {
		return err
	}
	return n.SetRadioState(KISS_RADIO_STATE_ON)
}

func (n *RNode) SetFrequency(freq int) error {
	if freq <= 0 {
		return errors.New("invalid frequency")
	}
	n.Frequency = freq
	return n.sendCommand(KISS_CMD_FREQUENCY, intToBytes(freq), true)
}

func (n *RNode) SetBandwidth(bw int) error {
	if bw <= 0 {
		return errors.New("invalid bandwidth")
	}
	n.Bandwidth = bw
	return n.sendCommand(KISS_CMD_BANDWIDTH, intToBytes(bw), true)
}

func (n *RNode) SetTXPower(txp int) error {
	if txp < 0 || txp > 0xFF {
		return fmt.Errorf("invalid tx power %d", txp)
	}
	n.TxPower = byte(txp)
	return n.sendCommand(KISS_CMD_TXPOWER, []byte{byte(txp)}, false)
}

func (n *RNode) SetSpreadingFactor(sf int) error {
	if sf < 5 || sf > 12 {
		return fmt.Errorf("invalid spreading factor %d", sf)
	}
	n.SF = byte(sf)
	return n.sendCommand(KISS_CMD_SF, []byte{byte(sf)}, false)
}

func (n *RNode) SetCodingRate(cr int) error {
	if cr < 5 || cr > 8 {
		return fmt.Errorf("invalid coding rate %d", cr)
	}
	n.CR = byte(cr)
	return n.sendCommand(KISS_CMD_CR, []byte{byte(cr)}, false)
}

func (n *RNode) SetRadioState(state byte) error {
	return n.sendCommand(KISS_CMD_RADIO_STATE, []byte{state}, false)
}

func (n *RNode) SetNormalMode() error {
	return n.sendCommand(KISS_CMD_CONF_DELETE, []byte{0x00}, false)
}

func (n *RNode) SetTNCMode() error {
	if err := n.sendCommand(KISS_CMD_CONF_SAVE, []byte{0x00}, false); err != nil {
		return err
	}
	if n.Platform == ROM_PLATFORM_ESP32 {
		return n.HardReset()
	}
	return nil
}

func (n *RNode) WipeEEPROM() error {
	if err := n.sendCommand(KISS_CMD_ROM_WIPE, []byte{0xF8}, false); err != nil {
		return err
	}
	// In tests and simulated ports, don't sleep for firmware EEPROM erase timing.
	if n.Serial != nil && strings.HasPrefix(n.Serial.Name(), "sim://") {
		return nil
	}
	time.Sleep(13 * time.Second)
	if n.Board == ROM_BOARD_RAK4631 {
		time.Sleep(10 * time.Second)
	}
	return nil
}

func (n *RNode) HardReset() error {
	if err := n.sendCommand(KISS_CMD_RESET, []byte{0xF8}, false); err != nil {
		return err
	}
	if n.Serial != nil && strings.HasPrefix(n.Serial.Name(), "sim://") {
		return nil
	}
	time.Sleep(2 * time.Second)
	return nil
}

func (n *RNode) WriteEEPROM(addr, value byte) error {
	payload := []byte{addr, value}
	return n.sendCommand(KISS_CMD_ROM_WRITE, payload, true)
}

func (n *RNode) DownloadEEPROM(wait time.Duration) error {
	n.EEPROM = nil
	if err := n.sendCommand(KISS_CMD_ROM_READ, []byte{0x00}, false); err != nil {
		return err
	}
	if wait <= 0 {
		wait = 600 * time.Millisecond
	}
	deadline := time.Now().Add(wait)
	for {
		if n.EEPROM != nil {
			n.parseEEPROM()
			return nil
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	return errors.New("timed out waiting for EEPROM data")
}

func (n *RNode) parseEEPROM() {
	// Minimal parity with Python: determine provisioning + extract checksum/signature fields.
	n.Provisioned = false
	n.SignatureValid = false
	n.LocallySigned = false
	n.Vendor = ""

	if len(n.EEPROM) < int(romAddrSignature)+128 || len(n.EEPROM) <= int(romAddrInfoLock) {
		return
	}

	n.Product = n.EEPROM[romAddrProduct]
	n.Model = n.EEPROM[romAddrModel]
	n.HWRev = n.EEPROM[romAddrHWRev]
	n.SerialNo = append([]byte(nil), n.EEPROM[romAddrSerial:romAddrSerial+4]...)
	n.Made = append([]byte(nil), n.EEPROM[romAddrMade:romAddrMade+4]...)
	n.Checksum = append([]byte(nil), n.EEPROM[romAddrChecksum:romAddrChecksum+16]...)
	n.Signature = append([]byte(nil), n.EEPROM[romAddrSignature:romAddrSignature+128]...)

	if n.EEPROM[romAddrInfoLock] != romInfoLockByte {
		return
	}

	checksummedInfo := make([]byte, 0, 11)
	checksummedInfo = append(checksummedInfo, n.Product, n.Model, n.HWRev)
	checksummedInfo = append(checksummedInfo, n.SerialNo...)
	checksummedInfo = append(checksummedInfo, n.Made...)
	sum := md5.Sum(checksummedInfo)
	if !bytes.Equal(n.Checksum, sum[:]) {
		return
	}

	n.Provisioned = true
}

func (n *RNode) verifyDeviceSignature(paths storagePaths) {
	n.SignatureValid = false
	n.LocallySigned = false
	n.Vendor = ""

	if !n.Provisioned || len(n.Signature) != 128 || len(n.Checksum) != 16 {
		return
	}

	checksumHash := sha256.Sum256(n.Checksum)

	var candidates []struct {
		vendor string
		pub    *rsa.PublicKey
		local  bool
	}

	// Local signing key (if present).
	if fileExists(paths.SignerKeyPath) {
		if der, err := os.ReadFile(paths.SignerKeyPath); err == nil {
			if priv, err := parseSignerKeyDER(der); err == nil {
				if priv.N.BitLen() == 1024 {
					candidates = append(candidates, struct {
						vendor string
						pub    *rsa.PublicKey
						local  bool
					}{vendor: "LOCAL", pub: &priv.PublicKey, local: true})
				}
			}
		}
	}

	// Trusted keys.
	entries, _ := os.ReadDir(paths.TrustedKeysDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pubkey") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(paths.TrustedKeysDir, e.Name()))
		if err != nil {
			continue
		}
		pubAny, err := x509.ParsePKIXPublicKey(b)
		if err != nil {
			continue
		}
		pub, ok := pubAny.(*rsa.PublicKey)
		if !ok {
			continue
		}
		candidates = append(candidates, struct {
			vendor string
			pub    *rsa.PublicKey
			local  bool
		}{vendor: "LOCAL", pub: pub, local: true})
	}

	// Built-in known vendor keys.
	for _, k := range knownSigningKeys {
		b, err := hex.DecodeString(k.DERHex)
		if err != nil {
			continue
		}
		pubAny, err := x509.ParsePKIXPublicKey(b)
		if err != nil {
			continue
		}
		pub, ok := pubAny.(*rsa.PublicKey)
		if !ok {
			continue
		}
		candidates = append(candidates, struct {
			vendor string
			pub    *rsa.PublicKey
			local  bool
		}{vendor: k.Vendor, pub: pub})
	}

	for _, c := range candidates {
		if c.pub == nil || c.pub.N.BitLen() != 1024 {
			continue
		}
		if err := rsa.VerifyPSS(c.pub, crypto.SHA256, checksumHash[:], n.Signature, nil); err == nil {
			n.SignatureValid = true
			n.Vendor = c.vendor
			if c.local {
				n.LocallySigned = true
			}
			return
		}
	}
}

func (n *RNode) DownloadCfgSector(wait time.Duration) error {
	n.CfgSector = nil
	if err := n.sendCommand(KISS_CMD_CFG_READ, []byte{0x00}, false); err != nil {
		return err
	}
	if wait <= 0 {
		wait = 600 * time.Millisecond
	}
	deadline := time.Now().Add(wait)
	for {
		if n.CfgSector != nil {
			return nil
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	return errors.New("timed out waiting for config sector")
}

// The remaining parts of the Python utility (firmware downloads, signatures, CLI, etc.)
// are still pending porting.

func intToBytes(v int) []byte {
	return []byte{
		byte((v >> 24) & 0xFF),
		byte((v >> 16) & 0xFF),
		byte((v >> 8) & 0xFF),
		byte(v & 0xFF),
	}
}

func encodeIPv4(raw string) ([]byte, error) {
	if strings.EqualFold(strings.TrimSpace(raw), "none") || strings.TrimSpace(raw) == "" {
		return []byte{0x00, 0x00, 0x00, 0x00}, nil
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid IPv4 address %q", raw)
	}
	res := make([]byte, 4)
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("invalid IPv4 octet %q", part)
		}
		val, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid IPv4 octet %q: %w", part, err)
		}
		if val < 0 || val > 255 {
			return nil, fmt.Errorf("invalid IPv4 octet value %d", val)
		}
		res[i] = byte(val)
	}
	return res, nil
}

func parseWifiMode(val string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(val)) {
	case "OFF":
		return "OFF", nil
	case "AP":
		return "AP", nil
	case "STATION", "STA":
		return "STATION", nil
	default:
		return "", fmt.Errorf("expected OFF, AP or STATION, got %q", val)
	}
}

// ----- CLI helpers -----

func printModelTable() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Code\tBand\tMax Output\tChipset")
	for _, code := range sortedModelCodes() {
		info := Models[code]
		fmt.Fprintf(
			w,
			"0x%02X\t%s\t%s\t%s\n",
			code,
			info.BandString,
			formatPower(info.MaxOutput),
			info.Chip,
		)
	}
	w.Flush()
	fmt.Printf("\nTotal %d models known.\n", len(Models))
}

func printModelInfo(query string) error {
	code, err := parseModelCode(query)
	if err != nil {
		return err
	}
	info, ok := Models[code]
	if !ok {
		return fmt.Errorf("model code 0x%02X is not known", code)
	}
	fmt.Printf("Model 0x%02X\n", code)
	fmt.Printf("  Frequency band : %s\n", info.BandString)
	if info.MinFreq > 0 || info.MaxFreq > 0 {
		fmt.Printf("  Range          : %.3f - %.3f MHz\n", hzToMHz(info.MinFreq), hzToMHz(info.MaxFreq))
	}
	if info.MaxOutput > 0 {
		fmt.Printf("  Max RF output  : %d dBm\n", info.MaxOutput)
	}
	if info.Chip != "" {
		fmt.Printf("  Chipset        : %s\n", info.Chip)
	}
	if info.FWFile != "" {
		fmt.Printf("  Firmware file  : %s\n", info.FWFile)
	}
	return nil
}

func parseModelCode(query string) (byte, error) {
	s := strings.TrimSpace(strings.ToLower(query))
	s = strings.TrimPrefix(s, "0x")
	if len(s) == 0 {
		return 0, errors.New("empty model code")
	}
	val, err := strconv.ParseUint(s, 16, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid model code %q", query)
	}
	return byte(val), nil
}

func hzToMHz(hz int64) float64 {
	if hz == 0 {
		return 0
	}
	return math.Round((float64(hz)/1_000_000.0)*1000) / 1000
}

func sortedModelCodes() []byte {
	keys := make([]int, 0, len(Models))
	for k := range Models {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	res := make([]byte, len(keys))
	for i, v := range keys {
		res[i] = byte(v)
	}
	return res
}

func formatPower(dbm int) string {
	if dbm == 0 {
		return "-"
	}
	return fmt.Sprintf("%d dBm", dbm)
}
