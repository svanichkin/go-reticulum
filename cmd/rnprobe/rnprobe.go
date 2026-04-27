package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const (
	DefaultProbeSize = 16
	DefaultTimeout   = 12.0
)

const programName = "rnprobe"

type countFlag int

func (c *countFlag) String() string { return fmt.Sprint(int(*c)) }
func (c *countFlag) Set(string) error {
	*c++
	return nil
}
func (c *countFlag) Value() int       { return int(*c) }
func (c *countFlag) IsBoolFlag() bool { return true }

type lossError struct{ loss float64 }

func (e lossError) Error() string {
	return fmt.Sprintf("packet loss %.2f%%", e.loss)
}

type mtuError struct{}

func (mtuError) Error() string { return "probe payload exceeds MTU" }

type exitError struct {
	code int
	msg  string
}

func (e exitError) Error() string { return e.msg }

func main() {
	var (
		configDir string
		size      int
		probes    int
		timeout   float64
		wait      float64
		verbose   countFlag
		showVer   bool
		help      bool
	)

	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.IntVar(&size, "size", 0, "size of probe packet payload in bytes")
	flag.IntVar(&size, "s", 0, "size of probe packet payload in bytes")
	flag.IntVar(&probes, "probes", 1, "number of probes to send")
	flag.IntVar(&probes, "n", 1, "number of probes to send")
	flag.Float64Var(&timeout, "timeout", 0, "timeout before giving up (seconds)")
	flag.Float64Var(&timeout, "t", 0, "timeout before giving up (seconds)")
	flag.Float64Var(&wait, "wait", 0, "time between each probe (seconds)")
	flag.Float64Var(&wait, "w", 0, "time between each probe (seconds)")
	flag.Var(&verbose, "verbose", "increase verbosity")
	flag.Var(&verbose, "v", "increase verbosity")
	flag.BoolVar(&showVer, "version", false, "print program version and exit")
	flag.BoolVar(&help, "h", false, "show this help message and exit")
	flag.BoolVar(&help, "help", false, "show this help message and exit")

	flag.Usage = func() {
		fmt.Println("Reticulum Probe Utility")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  rnprobe [options] <full_name> <destination_hash>")
		fmt.Println()
		flag.PrintDefaults()
	}

	flag.Parse()

	if help {
		fmt.Println()
		flag.Usage()
		fmt.Println()
		return
	}

	if showVer {
		fmt.Printf("%s %s\n", programName, rns.Version())
		return
	}

	if flag.NArg() < 2 {
		fmt.Println()
		flag.Usage()
		fmt.Println()
		return
	}

	fullName := flag.Arg(0)
	destHex := flag.Arg(1)

	if err := programSetup(
		configDirOrNil(configDir),
		destHex,
		size,
		fullName,
		verbose.Value(),
		timeout,
		wait,
		probes,
	); err != nil {
		if ee, ok := err.(exitError); ok {
			if ee.msg != "" {
				fmt.Fprintln(os.Stdout, ee.msg)
			}
			os.Exit(ee.code)
		}
		switch err.(type) {
		case lossError:
			os.Exit(2)
		case mtuError:
			os.Exit(3)
		default:
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func configDirOrNil(path string) *string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return &path
}

func programSetup(
	configdir *string,
	destinationHex string,
	size int,
	fullName string,
	verbosity int,
	timeout float64,
	wait float64,
	probes int,
) error {
	if size <= 0 {
		size = DefaultProbeSize
	}
	if wait < 0 {
		wait = 0
	}
	if probes <= 0 {
		probes = 1
	}
	if strings.TrimSpace(fullName) == "" {
		// Python prints a message and exits without an error code.
		return exitError{code: 0, msg: "The full destination name including application name aspects must be specified for the destination"}
	}

	appName, aspects := (&rns.Destination{}).AppAndAspectsFromName(fullName)

	destLen := (rns.ReticulumTruncatedHashLength / 8) * 2
	if len(destinationHex) != destLen {
		return exitError{code: 0, msg: fmt.Sprintf("Destination length is invalid, must be %d hexadecimal characters (%d bytes).", destLen, destLen/2)}
	}
	destinationHash, err := hex.DecodeString(destinationHex)
	if err != nil {
		return exitError{code: 0, msg: "Invalid destination entered. Check your input."}
	}

	moreOutput := false
	if verbosity > 0 {
		moreOutput = true
		verbosity--
	} else {
		verbosity--
	}

	logLevel := 3 + verbosity
	ret, err := rns.NewReticulum(configdir, &logLevel, nil, nil, false, nil)
	if err != nil {
		return err
	}

	if !rns.HasPath(destinationHash) {
		rns.RequestPath(destinationHash, nil, nil, false)
		fmt.Print("Path to " + rns.PrettyHexRep(destinationHash) + " requested  ")
		os.Stdout.Sync()
	}

	firstHopTimeout := ret.GetFirstHopTimeout(destinationHash)
	useTimeout := effectiveTimeout(timeout, firstHopTimeout)

	deadline := time.Now().Add(time.Duration(useTimeout * float64(time.Second)))
	syms := []rune("⢄⢂⢁⡁⡈⡐⡠")
	spinnerIdx := 0

	for !rns.HasPath(destinationHash) && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("\b\b%c ", syms[spinnerIdx])
		os.Stdout.Sync()
		spinnerIdx = (spinnerIdx + 1) % len(syms)
	}

	if !rns.HasPath(destinationHash) {
		fmt.Print("\r                                                          \rPath request timed out\n")
		return exitError{code: 1, msg: ""}
	}

	serverIdentity := rns.IdentityRecall(destinationHash, false)
	if serverIdentity == nil {
		// Parity helper: allow recalling from identity hash if a full destination recall fails.
		serverIdentity = rns.IdentityRecall(destinationHash, true)
	}
	if serverIdentity == nil {
		return fmt.Errorf("could not recall server identity")
	}

	reqDest, err := rns.NewDestination(serverIdentity, rns.DestinationOUT, rns.DestinationSINGLE, appName, aspects...)
	if err != nil {
		return err
	}
	sent := 0
	replies := 0

	for probes > 0 {
		if sent > 0 {
			time.Sleep(time.Duration(wait * float64(time.Second)))
		}

		payload := make([]byte, size)
		if _, err := rand.Read(payload); err != nil {
			return err
		}
		probe := rns.NewPacket(reqDest, payload, rns.PacketTypeData, rns.PacketCtxNone, rns.Broadcast, rns.HeaderType1, nil, nil, true, rns.FlagUnset)
		if err := probe.Pack(); err != nil {
			fmt.Printf("Error: Probe packet size of %d bytes exceed MTU of %d bytes\n", len(probe.Raw), rns.MTU)
			return mtuError{}
		}

		receipt := probe.Send()
		sent++

		var more string
		if moreOutput {
			nh := ret.GetNextHop(destinationHash)
			if nh != nil {
				more = " via " + rns.PrettyHexRep(nh)
			}
			ifName := ret.GetNextHopIfName(destinationHash)
			if ifName != "" && ifName != "None" {
				more += " on " + ifName
			}
		}

		fmt.Printf("\rSent probe %d (%d bytes) to %s%s  ", sent, size, rns.PrettyHexRep(destinationHash), more)
		os.Stdout.Sync()

		firstHopTimeout = ret.GetFirstHopTimeout(destinationHash)
		useTimeout = effectiveTimeout(timeout, firstHopTimeout)
		respDeadline := time.Now().Add(time.Duration(useTimeout * float64(time.Second)))
		spinnerIdx = 0

		for receipt.GetStatus() == rns.PacketReceiptSENT && time.Now().Before(respDeadline) {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("\b\b%c ", syms[spinnerIdx])
			os.Stdout.Sync()
			spinnerIdx = (spinnerIdx + 1) % len(syms)
		}

		if receipt.GetStatus() == rns.PacketReceiptSENT {
			fmt.Print("\r                                                                \rProbe timed out\n")
		} else {
			fmt.Print("\b\b \n")
			os.Stdout.Sync()

			if receipt.GetStatus() == rns.PacketReceiptDELIVERED {
				replies++

				hops := rns.HopsTo(destinationHash)
				suffix := ""
				if hops != 1 {
					suffix = "s"
				}

				rtt := receipt.GetRTT()
				var rttStr string
				if rtt >= 1 {
					rtt = round(rtt, 3)
					rttStr = pyFloat(rtt) + " seconds"
				} else {
					rtt = round(rtt*1000, 3)
					rttStr = pyFloat(rtt) + " milliseconds"
				}

				receptionStats := ""
				if ret.IsConnectedToSharedInstance {
					if proof := receipt.ProofPacket; proof != nil {
						hash := proof.PacketHash
						if rssi := ret.GetPacketRSSI(hash); rssi != nil {
							receptionStats += " [RSSI " + pyFloat(*rssi) + " dBm]"
						}
						if snr := ret.GetPacketSNR(hash); snr != nil {
							receptionStats += " [SNR " + pyFloat(*snr) + " dB]"
						}
						if q := ret.GetPacketQ(hash); q != nil {
							receptionStats += " [Link Quality " + pyFloat(*q) + "%]"
						}
					}
				} else if proof := receipt.ProofPacket; proof != nil {
					if rssi := proof.RSSI; rssi != nil {
						receptionStats += " [RSSI " + pyFloat(*rssi) + " dBm]"
					}
					if snr := proof.SNR; snr != nil {
						receptionStats += " [SNR " + pyFloat(*snr) + " dB]"
					}
				}

				fmt.Printf(
					"Valid reply from %s\nRound-trip time is %s over %d hop%s%s\n\n",
					rns.PrettyHexRep(receipt.Destination.Hash),
					rttStr,
					hops,
					suffix,
					receptionStats,
				)
			} else {
				fmt.Print("\r                                                          \rProbe timed out\n")
			}
		}

		probes--
	}

	loss := 0.0
	if sent > 0 {
		loss = round((1.0-float64(replies)/float64(sent))*100.0, 2)
	}

	fmt.Printf("Sent %d, received %d, packet loss %s%%\n", sent, replies, pyFloat(loss))
	if loss > 0 {
		return lossError{loss: loss}
	}
	return nil
}

func effectiveTimeout(timeout, firstHopTimeout float64) float64 {
	// Parity with Python:
	// _timeout = now + (timeout or DEFAULT_TIMEOUT+reticulum.get_first_hop_timeout(...))
	if timeout > 0 {
		return timeout
	}
	return DefaultTimeout + firstHopTimeout
}

func round(v float64, decimals int) float64 {
	if decimals <= 0 {
		return math.Round(v)
	}
	factor := math.Pow(10, float64(decimals))
	return math.Round(v*factor) / factor
}

// pyFloat tries to mimic Python's str(float) behaviour:
// - no trailing zeros except keeping at least one decimal for integral floats (e.g. "1.0").
func pyFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}
