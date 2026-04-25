package rns

import (
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	cryptography "github.com/svanichkin/go-reticulum/rns/cryptography"
	platformutils "github.com/svanichkin/go-reticulum/rns/vendor"
)

// Library version mirrors python/RNS/_version.py.
var Version = "1.1.5"
var compiledFlag = false

const (
	defaultLogTimeFormat     = "2006-01-02 15:04:05"
	defaultLogTimeFormatPrec = "15:04:05.000"
	identityAESBlockSize     = 16
)

// ---------- log levels ----------

const (
	LogNone     = -1
	LogCritical = 0
	LogError    = 1
	LogWarning  = 2
	LogNotice   = 3
	LogInfo     = 4
	LogVerbose  = 5
	LogDebug    = 6
	LogExtreme  = 7
)

type LogDestination int
type LogFileMarker = LogDestination

const (
	LogStdout   LogDestination = 0x91
	LogFile     LogDestination = 0x92
	LogCallback LogDestination = 0x93
)

const (
	LOG_NONE     = LogNone
	LOG_CRITICAL = LogCritical
	LOG_ERROR    = LogError
	LOG_WARNING  = LogWarning
	LOG_NOTICE   = LogNotice
	LOG_INFO     = LogInfo
	LOG_VERBOSE  = LogVerbose
	LOG_DEBUG    = LogDebug
	LOG_EXTREME  = LogExtreme

	LOG_STDOUT   = LogStdout
	LOG_FILE     = LogFile
	LOG_CALLBACK = LogCallback
)

const logMaxSize = 5 * 1024 * 1024

// LOG_MAXSIZE mirrors RNS.LOG_MAXSIZE in the Python implementation.
const LOG_MAXSIZE = logMaxSize

var (
	logLevel                             = LogNotice
	logDest               LogDestination = LogStdout
	logFilePath           string
	logCallback           func(level int, line string)
	logSuppressOutput     bool
	logTimeFmt            = defaultLogTimeFormat
	logTimeFmtPrec        = defaultLogTimeFormatPrec
	logTimePrecTrimMillis bool
	compactLogFmt         bool

	logMu              sync.Mutex
	randMu             sync.Mutex
	phyParamsMu        sync.RWMutex
	mtuMu              sync.Mutex
	profilerMu         sync.Mutex
	alwaysOverrideDest bool
	exitCalled         bool
	exitMu             sync.Mutex

	instanceRand = rand.New(rand.NewSource(seedRandom()))

	profilerRegistry = make(map[string]*Profiler)
	profilerTags     = make(map[string]*profilerTagEntry)
	profilerOrder    []string
	profilerHasRun   bool

	phyParamsSnapshot = PhyLayerParams{}

	// LinkMDU matches the classic Link.MDU from the Python implementation.
	LinkMDU = computeLinkMDU(DefaultMTU)
)

func init() {
	if strings.HasSuffix(os.Args[0], ".test") && os.Getenv("RNS_TEST_LOGS") != "1" {
		logSuppressOutput = true
	}
	mtuMu.Lock()
	recalcMTUDerivedLocked()
	mtuMu.Unlock()
}

func seedRandom() int64 {
	var buf [8]byte
	if _, err := crand.Read(buf[:]); err == nil {
		return int64(binary.LittleEndian.Uint64(buf[:]))
	}
	return time.Now().UnixNano()
}

// PhyLayerParams stores values printed by PhyParams().
type PhyLayerParams struct {
	PhysicalLayerMTU   int
	PacketPlainMDU     int
	PacketEncryptedMDU int
	LinkCurve          string
	LinkMDU            int
	LinkPublicKeyBits  int
	LinkPrivateKeyBits int
}

// ---------- helper getters ----------

func LogLevelName(level int) string {
	switch level {
	case LogCritical:
		return "[Critical]"
	case LogError:
		return "[Error]   "
	case LogWarning:
		return "[Warning] "
	case LogNotice:
		return "[Notice]  "
	case LogInfo:
		return "[Info]    "
	case LogVerbose:
		return "[Verbose] "
	case LogDebug:
		return "[Debug]   "
	case LogExtreme:
		return "[Extra]   "
	default:
		return "Unknown"
	}
}

func SetLogLevel(level int) {
	logMu.Lock()
	defer logMu.Unlock()
	logLevel = level
}

func LogLevel() int {
	logMu.Lock()
	defer logMu.Unlock()
	return logLevel
}

func UseStdoutLogging() {
	logMu.Lock()
	defer logMu.Unlock()
	logDest = LogStdout
	alwaysOverrideDest = false
}

func LogDest() LogDestination {
	logMu.Lock()
	defer logMu.Unlock()
	return logDest
}

// SetLogDest sets the active log destination, mirroring RNS.logdest in Python.
func SetLogDest(dest LogDestination) {
	logMu.Lock()
	defer logMu.Unlock()
	logDest = dest
	alwaysOverrideDest = false
}

func SetLogFile(path string) {
	logMu.Lock()
	defer logMu.Unlock()
	logFilePath = path
	logDest = LogFile
	alwaysOverrideDest = false
}

// SetLogDestFile is an alias for compatibility with Reticulum code.
func SetLogDestFile(path string) {
	SetLogFile(path)
}

func SetLogDestCallback(cb func(level int, msg string)) {
	logMu.Lock()
	defer logMu.Unlock()
	logCallback = cb
	if cb != nil {
		logDest = LogCallback
	} else {
		logDest = LogStdout
	}
	alwaysOverrideDest = false
}

func SetLogCallback(cb func(string)) {
	if cb == nil {
		SetLogDestCallback(nil)
		return
	}
	SetLogDestCallback(func(_ int, msg string) {
		cb(msg)
	})
}

func SetCompactLogFormat(on bool) {
	logMu.Lock()
	defer logMu.Unlock()
	compactLogFmt = on
}

// SetLogOutputSuppressed controls whether non-callback log destinations should
// emit output. It is primarily used to keep go test output quiet while still
// allowing explicit callback-based log assertions in tests.
func SetLogOutputSuppressed(on bool) {
	logMu.Lock()
	defer logMu.Unlock()
	logSuppressOutput = on
}

// SetLogTimeFormat allows customising the timestamp prefix used in logs,
// mirroring RNS.logtimefmt in the Python implementation. Passing an empty
// string resets the format to the default value.
func SetLogTimeFormat(format string) {
	logMu.Lock()
	defer logMu.Unlock()
	if strings.TrimSpace(format) == "" {
		logTimeFmt = defaultLogTimeFormat
		return
	}
	logTimeFmt, _ = normalizeTimeFormat(format, false)
}

// SetPreciseLogTimeFormat sets the format used when LogPrecise is called.
// An empty string resets it back to the default.
func SetPreciseLogTimeFormat(format string) {
	logMu.Lock()
	defer logMu.Unlock()
	if strings.TrimSpace(format) == "" {
		logTimeFmtPrec = defaultLogTimeFormatPrec
		logTimePrecTrimMillis = false
		return
	}
	logTimeFmtPrec, logTimePrecTrimMillis = normalizeTimeFormat(format, true)
}

// LogTimeFormats returns the current standard and precise timestamp formats.
func LogTimeFormats() (standard string, precise string) {
	logMu.Lock()
	defer logMu.Unlock()
	return logTimeFmt, logTimeFmtPrec
}

// ---------- version / platform ----------

func GetVersion() string { return Version }

// VersionString mirrors Python's RNS.version() helper.
func VersionString() string { return Version }

// SetVersion allows overriding the exposed library version string at runtime.
// Useful when embedding the Go port inside other binaries that manage versioning.
func SetVersion(ver string) {
	if strings.TrimSpace(ver) == "" {
		return
	}
	Version = ver
}

// Compiled reports whether the current build is considered "compiled" (mirrors
// the Python flag RNS.compiled). It defaults to true for the Go port, but can
// be overridden if embedding environments need to signal otherwise.
func Compiled() bool { return compiledFlag }

// SetCompiled lets callers override the compiled flag returned by Compiled().
func SetCompiled(flag bool) { compiledFlag = flag }

func HostOS() string {
	return platformutils.GetPlatform()
}

// ---------- time / formatting ----------

func timestampStr(t time.Time) string {
	return t.Format(logTimeFmt)
}

func preciseTimestampStr(t time.Time) string {
	s := t.Format(logTimeFmtPrec)
	if logTimePrecTrimMillis && len(s) >= 3 {
		return s[:len(s)-3]
	}
	return s
}

func TimestampStr(seconds float64) string {
	nsec := int64(seconds * float64(time.Second))
	return time.Unix(0, nsec).In(time.Local).Format(logTimeFmt)
}

func PreciseTimestampStr(seconds float64) string {
	// Python ignores the supplied timestamp and always uses "now" for
	// precise_timestamp_str(), since it is meant for log prefixes.
	_ = seconds
	now := time.Now()
	logMu.Lock()
	prec := logTimeFmtPrec
	trim := logTimePrecTrimMillis
	logMu.Unlock()
	s := now.Format(prec)
	if trim && len(s) >= 3 {
		return s[:len(s)-3]
	}
	return s
}

// normalizeTimeFormat accepts either a Go time layout (default for this port)
// or a Python strftime-style format (used by the upstream implementation).
// If the input contains '%' directives, a best-effort conversion is applied.
func normalizeTimeFormat(format string, precise bool) (layout string, trimMillis bool) {
	f := strings.TrimSpace(format)
	if f == "" {
		if precise {
			return defaultLogTimeFormatPrec, false
		}
		return defaultLogTimeFormat, false
	}

	// If it looks like Python strftime, convert the directives we use upstream.
	// Python defaults: "%Y-%m-%d %H:%M:%S" and "%H:%M:%S.%f" (then trimmed to ms).
	if strings.Contains(f, "%") {
		trim := false
		out := f
		// Order matters: replace "%f" before generic "%".
		if strings.Contains(out, "%f") {
			// Go has microseconds at 6 digits with ".000000". Python later slices to ms.
			out = strings.ReplaceAll(out, "%f", "000000")
			if precise {
				trim = true
			}
		}
		out = strings.ReplaceAll(out, "%Y", "2006")
		out = strings.ReplaceAll(out, "%m", "01")
		out = strings.ReplaceAll(out, "%d", "02")
		out = strings.ReplaceAll(out, "%H", "15")
		out = strings.ReplaceAll(out, "%M", "04")
		out = strings.ReplaceAll(out, "%S", "05")
		return out, trim
	}

	return f, false
}

// ---------- main logger ----------

func Log(msg any, level int) {
	logInternal(msg, level, false, false)
}

func LogPrecise(msg any, level int) {
	logInternal(msg, level, true, false)
}

// LogOverride forces the message to be printed to stdout regardless of the configured destination.
// It mirrors the internal _override_destination flag of Python's RNS.log(...).
func LogOverride(msg any, level int) {
	logInternal(msg, level, false, true)
}

// LogPreciseOverride forces precise logging to stdout regardless of the configured destination.
func LogPreciseOverride(msg any, level int) {
	logInternal(msg, level, true, true)
}

func Logf(level int, format string, args ...any) {
	logInternal(fmt.Sprintf(format, args...), level, false, false)
}

func logInternal(msg any, level int, precise, overrideDest bool) {
	logMu.Lock()
	currentLevel := logLevel
	if currentLevel == LogNone || currentLevel < level {
		logMu.Unlock()
		return
	}
	compact := compactLogFmt
	dest := logDest
	overrideAlways := alwaysOverrideDest
	path := logFilePath
	cb := logCallback
	suppressOutput := logSuppressOutput
	logMu.Unlock()

	text := fmt.Sprint(msg)
	now := time.Now()

	var prefix string
	if precise {
		prefix = "[" + preciseTimestampStr(now) + "] " + LogLevelName(level) + " "
	} else if compact {
		prefix = "[" + timestampStr(now) + "] "
	} else {
		prefix = "[" + timestampStr(now) + "] " + LogLevelName(level) + " "
	}

	logLine := prefix + text
	toStdout := dest == LogStdout || overrideAlways || overrideDest

	switch {
	case toStdout:
		if suppressOutput && cb == nil && !overrideDest && !overrideAlways {
			return
		}
		fmt.Println(logLine)

	case dest == LogFile && path != "":
		if suppressOutput && cb == nil {
			return
		}
		if err := appendLogFile(path, logLine); err != nil {
			logMu.Lock()
			alwaysOverrideDest = true
			logMu.Unlock()
			Log(fmt.Sprintf("Exception occurred while writing log message to log file: %v", err), LogCritical)
			Log("Dumping future log events to console!", LogCritical)
			Log(msg, level)
		}

	case dest == LogCallback && cb != nil:
		func() {
			defer func() {
				if r := recover(); r != nil {
					logMu.Lock()
					alwaysOverrideDest = true
					logMu.Unlock()
					Log(fmt.Sprintf("Exception occurred while calling external log handler: %v", r), LogCritical)
					Log("Dumping future log events to console!", LogCritical)
					Log(msg, level)
				}
			}()
			cb(level, logLine)
		}()
	}
}

func appendLogFile(path, line string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f, line); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if info.Size() <= logMaxSize {
		return nil
	}

	prev := path + ".1"
	_ = os.Remove(prev)
	return os.Rename(path, prev)
}

// ---------- random ----------

func Rand() float64 {
	randMu.Lock()
	defer randMu.Unlock()
	return instanceRand.Float64()
}

func TraceException(e any) {
	if e == nil {
		return
	}
	LogOverride(fmt.Sprintf("An unhandled %T exception occurred: %v", e, e), LogError)
	trace := formattedExceptionTrace(e)
	if strings.TrimSpace(trace) != "" {
		LogOverride(trace, LogError)
	}
}

func formattedExceptionTrace(e any) string {
	if e == nil {
		return ""
	}

	// pkg/errors and similar wrappers often expose stack-aware formatting
	// through %+v. Prefer that output before falling back to the current stack.
	normal := fmt.Sprint(e)
	verbose := fmt.Sprintf("%+v", e)
	if strings.TrimSpace(verbose) != "" && verbose != normal {
		return verbose
	}
	return ""
}

// ---------- hex representation ----------

func HexRep(data any, delimit ...bool) string {
	sep := ":"
	if len(delimit) > 0 && !delimit[0] {
		sep = ""
	}

	values := hexValues(data)
	if len(values) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, v := range values {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(fmt.Sprintf("%02x", v))
	}
	return sb.String()
}

func PrettyHexRep(data any) string {
	values := hexValues(data)
	var sb strings.Builder
	sb.WriteByte('<')
	for _, v := range values {
		sb.WriteString(fmt.Sprintf("%02x", v))
	}
	sb.WriteByte('>')
	return sb.String()
}

func PrettyHex(data any) string {
	return PrettyHexRep(data)
}

func PrettyHash(data any) string {
	return PrettyHex(data)
}

func hexValues(data any) []uint64 {
	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case []byte:
		out := make([]uint64, len(v))
		for i, b := range v {
			out[i] = uint64(b)
		}
		return out
	case string:
		return hexValues([]byte(v))
	case byte:
		return []uint64{uint64(v)}
	case int:
		return []uint64{uint64(absInt64(int64(v)))}
	case int64:
		return []uint64{uint64(absInt64(v))}
	case int32:
		return []uint64{uint64(absInt64(int64(v)))}
	case int16:
		return []uint64{uint64(absInt64(int64(v)))}
	case int8:
		return []uint64{uint64(absInt64(int64(v)))}
	case uint16:
		return []uint64{uint64(v)}
	case uint32:
		return []uint64{uint64(v)}
	case uint64:
		return []uint64{v}
	case uint:
		return []uint64{uint64(v)}
	case []int:
		out := make([]uint64, len(v))
		for i, val := range v {
			out[i] = uint64(absInt64(int64(val)))
		}
		return out
	case []uint16:
		out := make([]uint64, len(v))
		for i, val := range v {
			out[i] = uint64(val)
		}
		return out
	case []uint32:
		out := make([]uint64, len(v))
		for i, val := range v {
			out[i] = uint64(val)
		}
		return out
	case []uint64:
		out := make([]uint64, len(v))
		copy(out, v)
		return out
	}

	val := reflect.ValueOf(data)
	if !val.IsValid() {
		return nil
	}

	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil
		}
		return hexValues(val.Elem().Interface())
	}

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		length := val.Len()
		out := make([]uint64, 0, length)
		for i := 0; i < length; i++ {
			out = append(out, valueToUint64(val.Index(i)))
		}
		return out
	default:
		return []uint64{valueToUint64(val)}
	}
}

func valueToUint64(val reflect.Value) uint64 {
	switch val.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return val.Uint()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint64(absInt64(val.Int()))
	case reflect.Bool:
		if val.Bool() {
			return 1
		}
		return 0
	case reflect.Float32, reflect.Float64:
		f := val.Float()
		if f < 0 {
			f = -f
		}
		return uint64(f)
	default:
		return 0
	}
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// ---------- human-readable sizes/speeds ----------

func PrettySize(num float64, suffix ...string) string {
	suf := "B"
	if len(suffix) > 0 && suffix[0] != "" {
		suf = suffix[0]
	}

	units := []string{"", "K", "M", "G", "T", "P", "E", "Z"}
	lastUnit := "Y"
	value := num

	if suf == "b" {
		value *= 8
	}

	for _, unit := range units {
		if math.Abs(value) < 1000.0 {
			if unit == "" {
				return fmt.Sprintf("%.0f %s%s", value, unit, suf)
			}
			return fmt.Sprintf("%.2f %s%s", value, unit, suf)
		}
		value /= 1000.0
	}
	return fmt.Sprintf("%.2f%s%s", value, lastUnit, suf)
}

func PrettySpeed(num float64, suffix ...string) string {
	suf := "b"
	if len(suffix) > 0 && suffix[0] != "" {
		suf = suffix[0]
	}
	return PrettySize(num/8, suf) + "ps"
}

func PrettyFrequency(hz float64, suffix ...string) string {
	suf := "Hz"
	if len(suffix) > 0 && suffix[0] != "" {
		suf = suffix[0]
	}
	if hz == 0 {
		return "0 " + suf
	}
	num := hz * 1e6
	units := []string{"µ", "m", "", "K", "M", "G", "T", "P", "E", "Z"}
	lastUnit := "Y"
	for _, unit := range units {
		if math.Abs(num) < 1000.0 {
			return fmt.Sprintf("%.2f %s%s", num, unit, suf)
		}
		num /= 1000.0
	}
	return fmt.Sprintf("%.2f%s%s", num, lastUnit, suf)
}

func PrettyDistance(m float64, suffix ...string) string {
	suf := "m"
	if len(suffix) > 0 && suffix[0] != "" {
		suf = suffix[0]
	}
	num := m * 1e6
	units := []string{"µ", "m", "c", ""}
	lastUnit := "K"

	for _, unit := range units {
		div := 1000.0
		if unit == "m" {
			div = 10
		}
		if unit == "c" {
			div = 100
		}

		if math.Abs(num) < div {
			return fmt.Sprintf("%.2f %s%s", num, unit, suf)
		}
		num /= div
	}
	return fmt.Sprintf("%.2f %s%s", num, lastUnit, suf)
}

// ---------- human-readable time ----------

func PrettyTime(sec float64, verbose, compact bool) string {
	neg := false
	if sec < 0 {
		neg = true
		sec = -sec
	}

	days := int(sec) / (24 * 3600)
	sec = math.Mod(sec, 24*3600)
	hours := int(sec) / 3600
	sec = math.Mod(sec, 3600)
	minutes := int(sec) / 60
	sec = math.Mod(sec, 60)

	var seconds float64
	if compact {
		seconds = float64(int(sec))
	} else {
		seconds = math.Round(sec*100) / 100
	}

	components := []string{}
	shown := 0

	add := func(v int, one, many string) {
		if v <= 0 {
			return
		}
		if compact && shown >= 2 {
			return
		}
		lbl := many
		if v == 1 {
			lbl = one
		}
		if verbose {
			components = append(components, fmt.Sprintf("%d %s", v, lbl))
		} else {
			components = append(components, fmt.Sprintf("%d%s", v, string(one[0])))
		}
		shown++
	}

	add(days, "day", "days")
	add(hours, "hour", "hours")
	add(minutes, "minute", "minutes")

	if seconds > 0 && (!compact || shown < 2) {
		if verbose {
			suf := "seconds"
			if seconds == 1 {
				suf = "second"
			}
			components = append(components, fmt.Sprintf("%s %s", formatNumber(seconds, compact), suf))
		} else {
			components = append(components, fmt.Sprintf("%s%s", formatNumber(seconds, compact), "s"))
		}
	}

	if len(components) == 0 {
		return "0s"
	}

	// join "a, b and c"
	out := ""
	for i, c := range components {
		switch {
		case i == 0:
			out += c
		case i < len(components)-1:
			out += ", " + c
		default:
			out += " and " + c
		}
	}

	if neg {
		return "-" + out
	}
	return out
}

func PrettyShortTime(sec float64, verbose, compact bool) string {
	// input is seconds, internal is microseconds
	neg := false
	if sec < 0 {
		neg = true
		sec = -sec
	}
	us := sec * 1e6

	seconds := int(us) / 1_000_000
	us = math.Mod(us, 1_000_000)
	milliseconds := int(us) / 1_000
	us = math.Mod(us, 1_000)

	var micro float64
	if compact {
		micro = float64(int(us))
	} else {
		micro = math.Round(us*100) / 100
	}

	components := []string{}
	shown := 0

	add := func(v int, one, many, short string) {
		if v <= 0 {
			return
		}
		if compact && shown >= 2 {
			return
		}
		if verbose {
			lbl := many
			if v == 1 {
				lbl = one
			}
			components = append(components, fmt.Sprintf("%d %s", v, lbl))
		} else {
			components = append(components, fmt.Sprintf("%d%s", v, short))
		}
		shown++
	}

	add(seconds, "second", "seconds", "s")
	add(milliseconds, "millisecond", "milliseconds", "ms")
	if micro > 0 && (!compact || shown < 2) {
		if verbose {
			suf := "microseconds"
			if micro == 1 {
				suf = "microsecond"
			}
			components = append(components, fmt.Sprintf("%s %s", formatNumber(micro, compact), suf))
		} else {
			components = append(components, fmt.Sprintf("%sµs", formatNumber(micro, compact)))
		}
	}

	if len(components) == 0 {
		return "0us"
	}

	out := ""
	for i, c := range components {
		switch {
		case i == 0:
			out += c
		case i < len(components)-1:
			out += ", " + c
		default:
			out += " and " + c
		}
	}

	if neg {
		return "-" + out
	}
	return out
}

func formatNumber(value float64, compact bool) string {
	decimals := 2
	if compact {
		decimals = 0
	}
	format := fmt.Sprintf("%%.%df", decimals)
	text := fmt.Sprintf(format, value)
	if decimals > 0 {
		text = strings.TrimRight(text, "0")
		text = strings.TrimRight(text, ".")
	}
	if text == "" {
		return "0"
	}
	return text
}

// ---------- PHY parameters ----------

func SetPhyLayerParams(params PhyLayerParams) {
	phyParamsMu.Lock()
	defer phyParamsMu.Unlock()
	phyParamsSnapshot = params
}

func PhyParams() {
	params := resolvedPhyLayerParams()

	fmt.Printf("Required Physical Layer MTU : %s\n", bytesOrUnknown(params.PhysicalLayerMTU))
	fmt.Printf("Plaintext Packet MDU        : %s\n", bytesOrUnknown(params.PacketPlainMDU))
	fmt.Printf("Encrypted Packet MDU        : %s\n", bytesOrUnknown(params.PacketEncryptedMDU))
	fmt.Printf("Link Curve                  : %s\n", stringOrUnknown(params.LinkCurve))
	fmt.Printf("Link Packet MDU             : %s\n", bytesOrUnknown(params.LinkMDU))
	fmt.Printf("Link Public Key Size        : %s\n", bitsOrUnknown(params.LinkPublicKeyBits))
	fmt.Printf("Link Private Key Size       : %s\n", bitsOrUnknown(params.LinkPrivateKeyBits))
}

func bytesOrUnknown(value int) string {
	if value > 0 {
		return fmt.Sprintf("%d bytes", value)
	}
	return "unknown"
}

func bitsOrUnknown(value int) string {
	if value > 0 {
		return fmt.Sprintf("%d bits", value)
	}
	return "unknown"
}

func stringOrUnknown(value string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return "unknown"
}

// SetMTU overrides the global Reticulum MTU at runtime and recalculates all
// derived payload sizes (packet/link MDUs, resource hash capacities, etc).
// The supplied MTU must leave room for protocol headers and resource metadata.
func SetMTU(mtu int) error {
	if mtu <= 0 {
		return errors.New("MTU must be positive")
	}

	plain := computePlainMDU(mtu)
	if plain <= 0 {
		return fmt.Errorf("MTU %d leaves no room for packet payloads", mtu)
	}

	link := computeLinkMDU(mtu)
	if link <= 0 {
		return fmt.Errorf("MTU %d leaves no room for link payloads", mtu)
	}

	hashLen := int(math.Floor(float64(link-AdvOverhead) / float64(MapHashLen)))
	if hashLen <= 0 {
		return fmt.Errorf("MTU %d leaves no room for resource hashmaps", mtu)
	}

	encrypted := computeEncryptedPacketMDU(plain)

	mtuMu.Lock()
	defer mtuMu.Unlock()

	MTU = mtu
	applyMTUDerivedValuesLocked(plain, encrypted, link, hashLen)
	return nil
}

func recalcMTUDerivedLocked() {
	plain := computePlainMDU(MTU)
	link := computeLinkMDU(MTU)
	hashLen := int(math.Floor(float64(link-AdvOverhead) / float64(MapHashLen)))
	encrypted := computeEncryptedPacketMDU(plain)
	applyMTUDerivedValuesLocked(plain, encrypted, link, hashLen)
}

func applyMTUDerivedValuesLocked(plain, encrypted, link, hashLen int) {
	prevPlain := PacketPlainMDU
	prevEncrypted := PacketEncryptedMDU
	prevLink := LinkMDU

	if plain < 0 {
		plain = 0
	}
	if encrypted < 0 {
		encrypted = 0
	}
	if link < 0 {
		link = 0
	}
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
		go propagateMTUChange(prevPlain, plain, prevLink, link)
	}
}

func computePlainMDU(mtu int) int {
	value := mtu - HEADER_MAXSIZE - IFAC_MIN_SIZE
	if value < 0 {
		return 0
	}
	return value
}

func computeEncryptedPacketMDU(plain int) int {
	usable := plain - cryptography.Overhead - (identityPubKeyLen / 2)
	if usable <= 0 {
		return 0
	}
	blocks := usable / identityAESBlockSize
	if blocks <= 0 {
		return 0
	}
	return blocks*identityAESBlockSize - 1
}

func computeLinkMDU(mtu int) int {
	payload := mtu - IFAC_MIN_SIZE - HEADER_MINSIZE - cryptography.Overhead
	if payload <= 0 {
		return 0
	}
	blocks := payload / identityAESBlockSize
	if blocks <= 0 {
		return 0
	}
	return blocks*identityAESBlockSize - 1
}

func propagateMTUChange(prevPlain, newPlain, prevLink, newLink int) {
	for _, dst := range Destinations {
		if dst == nil {
			continue
		}
		if dst.mtu == 0 || dst.mtu == prevPlain {
			dst.mtu = newPlain
		}
	}

	linkMu.Lock()
	updateLinkMTU(PendingLinks, prevLink, newLink)
	updateLinkMTU(ActiveLinks, prevLink, newLink)
	linkMu.Unlock()
}

func updateLinkMTU(links []*Link, prevLink, newLink int) {
	for _, l := range links {
		if l == nil {
			continue
		}
		if prevLink == 0 || l.MTU == prevLink {
			l.MTU = newLink
			l.updateMDU()
		}
	}
}

func resolvedPhyLayerParams() PhyLayerParams {
	defaults := PhyLayerParams{
		PhysicalLayerMTU:   MTU,
		PacketPlainMDU:     PacketPlainMDU,
		PacketEncryptedMDU: PacketEncryptedMDU,
		LinkCurve:          linkCurveName,
		LinkMDU:            LinkMDU,
		LinkPublicKeyBits:  linkEcPubSize * 8,
		LinkPrivateKeyBits: linkKeySize * 8,
	}

	phyParamsMu.RLock()
	override := phyParamsSnapshot
	phyParamsMu.RUnlock()

	if override.PhysicalLayerMTU > 0 {
		defaults.PhysicalLayerMTU = override.PhysicalLayerMTU
	}
	if override.PacketPlainMDU > 0 {
		defaults.PacketPlainMDU = override.PacketPlainMDU
	}
	if override.PacketEncryptedMDU > 0 {
		defaults.PacketEncryptedMDU = override.PacketEncryptedMDU
	}
	if strings.TrimSpace(override.LinkCurve) != "" {
		defaults.LinkCurve = override.LinkCurve
	}
	if override.LinkMDU > 0 {
		defaults.LinkMDU = override.LinkMDU
	}
	if override.LinkPublicKeyBits > 0 {
		defaults.LinkPublicKeyBits = override.LinkPublicKeyBits
	}
	if override.LinkPrivateKeyBits > 0 {
		defaults.LinkPrivateKeyBits = override.LinkPrivateKeyBits
	}
	return defaults
}

// ---------- panic / exit ----------

// exitCallback is the process-level exit callback.
var exitCallback func()

func Panic() {
	os.Exit(255)
}

func Exit(code ...int) {
	exitMu.Lock()
	if exitCalled {
		exitMu.Unlock()
		return
	}
	exitCalled = true
	handler := exitCallback
	exitMu.Unlock()

	if handler != nil {
		handler()
	}

	exitCode := 0
	if len(code) > 0 {
		exitCode = code[0]
	}
	os.Exit(exitCode)
}

type profilerTagEntry struct {
	super   string
	threads map[int64]*profilerThreadEntry
}

type profilerThreadEntry struct {
	currentStart time.Time
	captures     []time.Duration
}

// ---------- Profiler ----------

type Profiler struct {
	tag           string
	superTag      string
	paused        bool
	pauseStarted  time.Time
	pauseTime     time.Duration
	superProfiler *Profiler
}

func GetProfiler(tag string, superTag ...string) *Profiler {
	profilerMu.Lock()
	defer profilerMu.Unlock()

	if prof, ok := profilerRegistry[tag]; ok {
		return prof
	}

	var parentTag string
	if len(superTag) > 0 {
		parentTag = superTag[0]
	}

	prof := &Profiler{
		tag:      tag,
		superTag: parentTag,
	}

	if parentTag != "" {
		if parent, ok := profilerRegistry[parentTag]; ok {
			prof.superProfiler = parent
		}
	}

	profilerRegistry[tag] = prof
	profilerOrder = append(profilerOrder, tag)
	return prof
}

var Profile = GetProfiler

func (p *Profiler) Enter() {
	if p == nil {
		return
	}
	if p.superProfiler != nil {
		p.superProfiler.pause()
	}

	gid := goroutineID()
	profilerMu.Lock()
	entry := profilerTags[p.tag]
	if entry == nil {
		entry = &profilerTagEntry{
			super:   p.superTag,
			threads: make(map[int64]*profilerThreadEntry),
		}
		profilerTags[p.tag] = entry
	}
	thread := entry.threads[gid]
	if thread == nil {
		thread = &profilerThreadEntry{}
		entry.threads[gid] = thread
	}
	thread.currentStart = time.Now()
	profilerMu.Unlock()

	if p.superProfiler != nil {
		p.superProfiler.resume()
	}
}

func (p *Profiler) Exit() {
	if p == nil {
		return
	}
	if p.superProfiler != nil {
		p.superProfiler.pause()
	}

	gid := goroutineID()
	now := time.Now()

	profilerMu.Lock()
	if entry, ok := profilerTags[p.tag]; ok {
		if thread, ok := entry.threads[gid]; ok {
			if !thread.currentStart.IsZero() {
				duration := now.Sub(thread.currentStart) - p.pauseTime
				if duration < 0 {
					duration = 0
				}
				thread.captures = append(thread.captures, duration)
				thread.currentStart = time.Time{}
				profilerHasRun = true
			}
		}
	}
	p.pauseTime = 0
	p.pauseStarted = time.Time{}
	profilerMu.Unlock()

	if p.superProfiler != nil {
		p.superProfiler.resume()
	}
}

func (p *Profiler) Pause() {
	p.pause()
}

func (p *Profiler) Resume() {
	p.resume()
}

func (p *Profiler) pause(start ...time.Time) {
	if p == nil || p.paused {
		return
	}
	p.paused = true
	if len(start) > 0 && !start[0].IsZero() {
		p.pauseStarted = start[0]
	} else {
		p.pauseStarted = time.Now()
	}
	if p.superProfiler != nil {
		p.superProfiler.pause(p.pauseStarted)
	}
}

func (p *Profiler) resume() {
	if p == nil || !p.paused {
		return
	}
	p.pauseTime += time.Since(p.pauseStarted)
	p.paused = false
	if p.superProfiler != nil {
		p.superProfiler.resume()
	}
}

func ProfilerRan() bool {
	profilerMu.Lock()
	defer profilerMu.Unlock()
	return profilerHasRun
}

func ProfilerResults() {
	profilerMu.Lock()
	results := make(map[string]profilerResult, len(profilerTags))
	for tag, entry := range profilerTags {
		var captures []time.Duration
		for _, thread := range entry.threads {
			captures = append(captures, thread.captures...)
		}
		if len(captures) == 0 {
			continue
		}
		stats := computeProfilerStats(captures)
		results[tag] = profilerResult{
			Name:   tag,
			Super:  entry.super,
			Count:  len(captures),
			Mean:   stats.mean,
			Median: stats.median,
			StdDev: stats.stddev,
			HasStd: stats.hasStd,
			Total:  stats.total,
		}
	}
	profilerMu.Unlock()

	if len(results) == 0 {
		fmt.Print("\nProfiler results:\n\n")
		return
	}

	fmt.Print("\nProfiler results:\n\n")
	rootNames := orderedProfilerNames(results, "")
	for _, name := range rootNames {
		printProfilerResultsRecursive(results[name], results, 0)
	}
}

type profilerResult struct {
	Name   string
	Super  string
	Count  int
	Mean   time.Duration
	Median time.Duration
	StdDev time.Duration
	HasStd bool
	Total  time.Duration
}

type profilerStats struct {
	mean   time.Duration
	median time.Duration
	stddev time.Duration
	hasStd bool
	total  time.Duration
}

func computeProfilerStats(values []time.Duration) profilerStats {
	if len(values) == 0 {
		return profilerStats{}
	}
	sum := 0.0
	total := time.Duration(0)
	for _, v := range values {
		sum += float64(v)
		total += v
	}
	mean := time.Duration(sum / float64(len(values)))

	sortedVals := append([]time.Duration(nil), values...)
	sort.Slice(sortedVals, func(i, j int) bool { return sortedVals[i] < sortedVals[j] })
	var median time.Duration
	mid := len(sortedVals) / 2
	if len(sortedVals)%2 == 0 {
		median = (sortedVals[mid-1] + sortedVals[mid]) / 2
	} else {
		median = sortedVals[mid]
	}

	variance := 0.0
	for _, v := range values {
		diff := float64(v - mean)
		variance += diff * diff
	}

	stats := profilerStats{
		mean:   mean,
		median: median,
	}
	stats.total = total
	if len(values) > 1 {
		variance /= float64(len(values) - 1)
		stats.stddev = time.Duration(math.Sqrt(variance))
		stats.hasStd = true
	}
	return stats
}

func printProfilerResultsRecursive(res profilerResult, results map[string]profilerResult, level int) {
	indent := strings.Repeat("  ", level)
	fmt.Printf("%s%s\n", indent, res.Name)
	fmt.Printf("%s  Samples  : %d\n", indent, res.Count)
	if res.HasStd {
		fmt.Printf("%s  Mean     : %s\n", indent, PrettyShortTime(res.Mean.Seconds(), false, false))
		fmt.Printf("%s  Median   : %s\n", indent, PrettyShortTime(res.Median.Seconds(), false, false))
		fmt.Printf("%s  St.dev.  : %s\n", indent, PrettyShortTime(res.StdDev.Seconds(), false, false))
	} else {
		fmt.Printf("%s  Mean     : %s\n", indent, PrettyShortTime(res.Mean.Seconds(), false, false))
		fmt.Printf("%s  Median   : %s\n", indent, PrettyShortTime(res.Median.Seconds(), false, false))
	}
	fmt.Printf("%s  Total    : %s\n\n", indent, PrettyShortTime(res.Total.Seconds(), false, false))

	children := orderedProfilerNames(results, res.Name)
	for _, name := range children {
		printProfilerResultsRecursive(results[name], results, level+1)
	}
}

func orderedProfilerNames(results map[string]profilerResult, parent string) []string {
	var names []string
	for _, name := range profilerOrder {
		res, ok := results[name]
		if !ok {
			continue
		}
		if parent == "" && res.Super == "" {
			names = append(names, name)
		} else if res.Super == parent {
			names = append(names, name)
		}
	}
	return names
}

func goroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	fields := strings.Fields(string(buf[:n]))
	if len(fields) >= 2 && fields[0] == "goroutine" {
		if id, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
			return id
		}
	}
	return 0
}
