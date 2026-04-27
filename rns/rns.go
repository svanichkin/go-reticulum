package rns

import (
	crand "crypto/rand"
	"encoding/binary"
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

// moduleVersion mirrors python/RNS/_version.py::__version__.
var moduleVersion = "1.1.5"
var Compiled = false

const (
	defaultLogTimeFormat     = "2006-01-02 15:04:05"
	defaultLogTimeFormatPrec = "15:04:05.000000"
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
	Loglevel                         = LogNotice
	logDest           LogDestination = LogStdout
	logFilePath       string
	logCallback       func(line string)
	logSuppressOutput bool
	LogTimeFmt        = defaultLogTimeFormat
	LogTimeFmtPrec    = defaultLogTimeFormatPrec
	CompactLogFmt     bool

	logMu              sync.Mutex
	randMu             sync.Mutex
	phyParamsMu        sync.RWMutex
	mtuMu              sync.Mutex
	profilerMu         sync.Mutex
	alwaysOverrideDest bool
	exitCalled         bool
	exitMu             sync.Mutex

	instanceRand = func() *rand.Rand {
		var buf [10]byte
		if _, err := crand.Read(buf[:]); err != nil {
			panic(err)
		}
		seed := int64(binary.LittleEndian.Uint64(buf[:8]) ^ uint64(binary.LittleEndian.Uint16(buf[8:])))
		return rand.New(rand.NewSource(seed))
	}()

	profilerRegistry = make(map[string]*Profiler)
	profilerTags     = make(map[string]*profilerTagEntry)
	profilerOrder    []string
	profilerHasRun   bool
	profilerStart    = time.Now()

	phyParamsSnapshot = PhyLayerParams{}

	// LinkMDU matches the classic Link.MDU from the Python implementation.
	LinkMDU = func() int {
		payload := DefaultMTU - IFAC_MIN_SIZE - HEADER_MINSIZE - cryptography.Overhead
		if payload <= 0 {
			return 0
		}
		blocks := payload / identityAESBlockSize
		if blocks <= 0 {
			return 0
		}
		return blocks*identityAESBlockSize - 1
	}()
)

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

// ---------- version / platform ----------

func HostOS() string {
	return platformutils.GetPlatform()
}

func Version() string {
	return moduleVersion
}

func TimestampStr(seconds float64) string {
	nsec := int64(seconds * float64(time.Second))
	return time.Unix(0, nsec).In(time.Local).Format(LogTimeFmt)
}

func PreciseTimestampStr(seconds float64) string {
	// Python ignores the supplied timestamp and always uses "now" for
	// precise_timestamp_str(), since it is meant for log prefixes.
	_ = seconds
	now := time.Now()
	logMu.Lock()
	prec := LogTimeFmtPrec
	logMu.Unlock()
	s := now.Format(prec)
	if len(s) >= 3 {
		return s[:len(s)-3]
	}
	return s
}

// ---------- main logger ----------

func Log(msg any, level int, options ...bool) {
	overrideDest := false
	precise := false
	if len(options) > 0 {
		overrideDest = options[0]
	}
	if len(options) > 1 {
		precise = options[1]
	}

	logMu.Lock()
	currentLevel := Loglevel
	if currentLevel == LogNone || currentLevel < level {
		logMu.Unlock()
		return
	}
	compact := CompactLogFmt
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
		s := now.Format(LogTimeFmtPrec)
		if len(s) >= 3 {
			s = s[:len(s)-3]
		}
		prefix = "[" + s + "] " + LogLevelName(level) + " "
	} else {
		if compact {
			prefix = "[" + now.Format(LogTimeFmt) + "] "
		} else {
			prefix = "[" + now.Format(LogTimeFmt) + "] " + LogLevelName(level) + " "
		}
	}
	logLine := prefix + text

	switch {
	case dest == LogStdout || overrideAlways || overrideDest:
		if suppressOutput && cb == nil && !overrideAlways && !overrideDest {
			return
		}
		fmt.Println(logLine)
	case dest == LogFile && path != "":
		if suppressOutput && cb == nil {
			return
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			if _, err = fmt.Fprintln(f, logLine); err == nil {
				if closeErr := f.Close(); closeErr == nil {
					if info, statErr := os.Stat(path); statErr == nil && info.Size() > logMaxSize {
						prev := path + ".1"
						_ = os.Remove(prev)
						_ = os.Rename(path, prev)
					}
					break
				} else {
					err = closeErr
				}
			}
			_ = f.Close()
		}
		if err != nil {
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
			cb(logLine)
		}()
	}
}

func Rand() float64 {
	randMu.Lock()
	defer randMu.Unlock()
	return instanceRand.Float64()
}

func TraceException(e any) {
	if e == nil {
		return
	}
	Log(fmt.Sprintf("An unhandled %T exception occurred: %v", e, e), LogError)
	normal := fmt.Sprint(e)
	verbose := fmt.Sprintf("%+v", e)
	if strings.TrimSpace(verbose) != "" && verbose != normal {
		Log(verbose, LogError)
	} else {
		Log(normal, LogError)
	}
}

// ---------- hex representation ----------

func HexRep(data any, delimit ...bool) string {
	sep := ":"
	if len(delimit) > 0 && !delimit[0] {
		sep = ""
	}

	value := reflect.ValueOf(data)
	if !value.IsValid() {
		return ""
	}

	var items []reflect.Value
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		items = make([]reflect.Value, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			items = append(items, value.Index(i))
		}
	default:
		items = []reflect.Value{value}
	}

	var sb strings.Builder
	for i, item := range items {
		var v byte
		switch item.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			v = byte(item.Uint())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v = byte(item.Int())
		default:
			return ""
		}
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(fmt.Sprintf("%02x", v))
	}
	return sb.String()
}

func PrettyHexRep(data any) string {
	return "<" + HexRep(data, false) + ">"
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
		return "0 Hz"
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
		var secondsText string
		if compact {
			secondsText = strconv.FormatFloat(seconds, 'f', 0, 64)
		} else {
			// Python: str(round(sec, 2)) always keeps at least one decimal digit
			// e.g. 5.0 → "5.0", 5.1 → "5.1", 5.12 → "5.12"
			secondsText = strings.TrimRight(strconv.FormatFloat(seconds, 'f', 2, 64), "0")
			if strings.HasSuffix(secondsText, ".") {
				secondsText += "0"
			}
		}
		if verbose {
			suf := "seconds"
			if seconds == 1 {
				suf = "second"
			}
			components = append(components, fmt.Sprintf("%s %s", secondsText, suf))
		} else {
			components = append(components, fmt.Sprintf("%s%s", secondsText, "s"))
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
		var microText string
		if compact {
			microText = strconv.FormatFloat(micro, 'f', 0, 64)
		} else {
			// Python: str(round(time, 2)) always keeps at least one decimal digit
			microText = strings.TrimRight(strconv.FormatFloat(micro, 'f', 2, 64), "0")
			if strings.HasSuffix(microText, ".") {
				microText += "0"
			}
		}
		if verbose {
			suf := "microseconds"
			if micro == 1 {
				suf = "microsecond"
			}
			components = append(components, fmt.Sprintf("%s %s", microText, suf))
		} else {
			components = append(components, fmt.Sprintf("%sµs", microText))
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

// ---------- PHY parameters ----------

func PhyParams() {
	fmt.Printf("Required Physical Layer MTU : %d bytes\n", MTU)
	fmt.Printf("Plaintext Packet MDU        : %d bytes\n", PacketPlainMDU)
	fmt.Printf("Encrypted Packet MDU        : %d bytes\n", PacketEncryptedMDU)
	fmt.Printf("Link Curve                  : %s\n", linkCurveName)
	fmt.Printf("Link Packet MDU             : %d bytes\n", LinkMDU)
	fmt.Printf("Link Public Key Size        : %d bits\n", linkEcPubSize*8)
	fmt.Printf("Link Private Key Size       : %d bits\n", linkKeySize*8)
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
	currentStart float64
	captures     []float64
}

// ---------- Profiler ----------

type Profiler struct {
	tag           string
	superTag      string
	paused        bool
	pauseStarted  float64
	pauseTime     float64
	superProfiler *Profiler
	pauseSuper    func(...float64)
	resumeSuper   func()
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
		pauseSuper: func(...float64) {
		},
		resumeSuper: func() {
		},
	}

	if parentTag != "" {
		if parent, ok := profilerRegistry[parentTag]; ok {
			prof.superProfiler = parent
			prof.pauseSuper = parent.pause
			prof.resumeSuper = parent.resume
		}
	}

	profilerRegistry[tag] = prof
	return prof
}

var Profile = GetProfiler

func (p *Profiler) Enter() {
	if p == nil {
		return
	}
	p.pauseSuper()

	var gid int64
	{
		var buf [64]byte
		n := runtime.Stack(buf[:], false)
		fields := strings.Fields(string(buf[:n]))
		if len(fields) >= 2 && fields[0] == "goroutine" {
			if id, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				gid = id
			}
		}
	}
	profilerMu.Lock()
	entry := profilerTags[p.tag]
	if entry == nil {
		entry = &profilerTagEntry{
			super:   p.superTag,
			threads: make(map[int64]*profilerThreadEntry),
		}
		profilerTags[p.tag] = entry
		profilerOrder = append(profilerOrder, p.tag)
	}
	thread := entry.threads[gid]
	if thread == nil {
		thread = &profilerThreadEntry{}
		entry.threads[gid] = thread
	}
	thread.currentStart = time.Since(profilerStart).Seconds()
	profilerMu.Unlock()

	p.resumeSuper()
}

func (p *Profiler) Exit() {
	if p == nil {
		return
	}
	p.pauseSuper()

	var gid int64
	{
		var buf [64]byte
		n := runtime.Stack(buf[:], false)
		fields := strings.Fields(string(buf[:n]))
		if len(fields) >= 2 && fields[0] == "goroutine" {
			if id, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				gid = id
			}
		}
	}
	now := time.Since(profilerStart).Seconds()

	profilerMu.Lock()
	if entry, ok := profilerTags[p.tag]; ok {
		if thread, ok := entry.threads[gid]; ok {
			if thread.currentStart != 0 {
				duration := now - thread.currentStart - p.pauseTime
				if duration < 0 {
					duration = 0
				}
				thread.captures = append(thread.captures, duration)
				thread.currentStart = 0
				profilerHasRun = true
			}
		}
	}
	p.pauseTime = 0
	p.pauseStarted = 0
	profilerMu.Unlock()

	p.resumeSuper()
}

func (p *Profiler) pause(start ...float64) {
	if p == nil || p.paused {
		return
	}
	p.paused = true
	if len(start) > 0 && start[0] != 0 {
		p.pauseStarted = start[0]
	} else {
		p.pauseStarted = time.Since(profilerStart).Seconds()
	}
	p.pauseSuper(p.pauseStarted)
}

func (p *Profiler) resume() {
	if p == nil || !p.paused {
		return
	}
	p.pauseTime += time.Since(profilerStart).Seconds() - p.pauseStarted
	p.paused = false
	p.resumeSuper()
}

func (Profiler) Ran() bool {
	profilerMu.Lock()
	defer profilerMu.Unlock()
	return profilerHasRun
}

func (Profiler) Results() {
	profilerMu.Lock()
	results := make(map[string]profilerResult, len(profilerTags))
	for tag, entry := range profilerTags {
		var captures []float64
		for _, thread := range entry.threads {
			captures = append(captures, thread.captures...)
		}
		if len(captures) == 0 {
			continue
		}
		sum := 0.0
		for _, v := range captures {
			sum += v
		}
		mean := sum / float64(len(captures))

		sortedVals := append([]float64(nil), captures...)
		sort.Slice(sortedVals, func(i, j int) bool { return sortedVals[i] < sortedVals[j] })
		mid := len(sortedVals) / 2
		var median float64
		if len(sortedVals)%2 == 0 {
			median = (sortedVals[mid-1] + sortedVals[mid]) / 2
		} else {
			median = sortedVals[mid]
		}

		var stddev *float64
		if len(captures) > 1 {
			variance := 0.0
			for _, v := range captures {
				diff := v - mean
				variance += diff * diff
			}
			variance /= float64(len(captures) - 1)
			v := math.Sqrt(variance)
			stddev = &v
		}

		results[tag] = profilerResult{
			Name:   tag,
			Super:  entry.super,
			Count:  len(captures),
			Mean:   mean,
			Median: median,
			StdDev: stddev,
		}
	}
	profilerMu.Unlock()

	if len(results) == 0 {
		fmt.Print("\nProfiler results:\n\n")
		return
	}

	fmt.Print("\nProfiler results:\n\n")
	orderForParent := func(parent string) []string {
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
	var printResultsRecursive func(profilerResult, int)
	printResultsRecursive = func(res profilerResult, level int) {
		indent := strings.Repeat("  ", level)
		fmt.Printf("%s%s\n", indent, res.Name)
		fmt.Printf("%s  Samples  : %d\n", indent, res.Count)
		if res.StdDev != nil {
			fmt.Printf("%s  Mean     : %s\n", indent, PrettyShortTime(res.Mean, false, false))
			fmt.Printf("%s  Median   : %s\n", indent, PrettyShortTime(res.Median, false, false))
			fmt.Printf("%s  St.dev.  : %s\n", indent, PrettyShortTime(*res.StdDev, false, false))
		}
		fmt.Printf("%s  Total    : %s\n\n", indent, PrettyShortTime(res.Mean*float64(res.Count), false, false))

		children := orderForParent(res.Name)
		for _, name := range children {
			printResultsRecursive(results[name], level+1)
		}
	}

	rootNames := orderForParent("")
	for _, name := range rootNames {
		printResultsRecursive(results[name], 1)
	}
}

type profilerResult struct {
	Name   string
	Super  string
	Count  int
	Mean   float64
	Median float64
	StdDev *float64
}
