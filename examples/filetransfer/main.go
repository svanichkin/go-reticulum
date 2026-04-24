package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

const (
	appName    = "example_utilities"
	appTimeout = 45 * time.Second
)

func main() {
	var (
		serveDir  string
		destHex   string
		configDir string
		outDir    string
	)
	flag.StringVar(&serveDir, "serve", "", "serve a directory of files to clients")
	flag.StringVar(&destHex, "destination", "", "hexadecimal hash of the server destination (client mode)")
	flag.StringVar(&outDir, "out", ".", "directory to write downloaded files (client mode)")
	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.Parse()

	var configArg *string
	if strings.TrimSpace(configDir) != "" {
		configArg = &configDir
	}

	if strings.TrimSpace(serveDir) != "" {
		if err := runServer(configArg, serveDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if strings.TrimSpace(destHex) == "" {
		fmt.Fprintln(os.Stderr, "missing -serve or -destination")
		flag.Usage()
		os.Exit(2)
	}

	if err := runClient(configArg, destHex, outDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ---------------- server ----------------

func runServer(configDir *string, serveDir string) error {
	if _, err := os.Stat(serveDir); err != nil {
		return fmt.Errorf("serve dir: %w", err)
	}

	_, err := rns.NewReticulum(configDir, nil, nil, nil, false, nil)
	if err != nil {
		return fmt.Errorf("NewReticulum: %w", err)
	}

	serverID, err := rns.NewIdentity()
	if err != nil {
		return fmt.Errorf("NewIdentity: %w", err)
	}

	serverDest, err := rns.NewDestination(serverID, rns.DestinationIN, rns.DestinationSINGLE, appName, "filetransfer", "server")
	if err != nil {
		return fmt.Errorf("NewDestination(server): %w", err)
	}

	serveDir = filepath.Clean(serveDir)
	if err := serverDest.RegisterRequestHandler(
		"/files",
		func(path string, data any, requestID []byte, linkID []byte, remoteIdentity *rns.Identity, requestedAt time.Time) any {
			files, err := listFiles(serveDir)
			if err != nil {
				return []string{}
			}
			return files
		},
		rns.DestinationALLOW_ALL,
		nil,
		true,
	); err != nil {
		return fmt.Errorf("RegisterRequestHandler(/files): %w", err)
	}

	if err := serverDest.RegisterRequestHandler(
		"/get",
		func(path string, data any, requestID []byte, linkID []byte, remoteIdentity *rns.Identity, requestedAt time.Time) any {
			filename, _ := data.(string)
			filename = strings.TrimSpace(filename)
			if filename == "" || !fileExistsInDir(serveDir, filename) {
				return "ERR"
			}

			link := findActiveLinkByID(linkID)
			if link == nil {
				return "ERR"
			}

			f, err := os.Open(filepath.Join(serveDir, filename))
			if err != nil {
				return "ERR"
			}

			timeoutSeconds := appTimeout.Seconds()
			res, err := rns.NewResource(
				nil,
				f,
				link,
				map[string]any{"filename": filename},
				true,
				true,
				func(res *rns.Resource) {
					defer func() { _ = f.Close() }()
					if res == nil {
						return
					}
					if res.Status == rns.ResourceComplete {
						rns.Log("Done sending "+filename+" to client", rns.LogInfo)
					} else {
						rns.Log("Sending "+filename+" to client failed", rns.LogError)
					}
				},
				nil,
				&timeoutSeconds,
				0,
				nil,
				nil,
				false,
				0,
			)
			if err != nil || res == nil {
				_ = f.Close()
				return "ERR"
			}
			return "OK"
		},
		rns.DestinationALLOW_ALL,
		nil,
		true,
	); err != nil {
		return fmt.Errorf("RegisterRequestHandler(/get): %w", err)
	}

	rns.Log("File server "+rns.PrettyHexRep(serverDest.Hash)+" running", rns.LogInfo)
	rns.Log("Hit enter to manually send an announce (Ctrl-C to quit)", rns.LogInfo)

	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		serverDest.Announce(nil, false, nil, nil, true)
		rns.Log("Sent announce from "+rns.PrettyHexRep(serverDest.Hash), rns.LogInfo)
	}
	return in.Err()
}

func listFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		out = append(out, name)
	}
	slices.Sort(out)
	return out, nil
}

func packFileListChunks(files []string, maxPayload int) ([][]byte, error) {
	if maxPayload <= 0 {
		return nil, errors.New("max payload must be >0")
	}
	if len(files) == 0 {
		b, err := umsgpack.Packb([]string{})
		if err != nil {
			return nil, err
		}
		return [][]byte{b}, nil
	}

	var chunks [][]byte
	var cur []string
	for _, name := range files {
		try := append(cur, name)
		b, err := umsgpack.Packb(try)
		if err != nil {
			return nil, err
		}
		if len(b) <= maxPayload {
			cur = try
			continue
		}
		if len(cur) == 0 {
			return nil, fmt.Errorf("single filename %q exceeds max payload (%d)", name, maxPayload)
		}
		prev, err := umsgpack.Packb(cur)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, prev)
		cur = []string{name}
	}
	if len(cur) > 0 {
		b, err := umsgpack.Packb(cur)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, b)
	}
	return chunks, nil
}

func fileExistsInDir(dir, name string) bool {
	if name == "" || strings.Contains(name, string(filepath.Separator)) || strings.Contains(name, "..") {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, name))
	return err == nil && info.Mode().IsRegular()
}

// ---------------- client ----------------

type clientState struct {
	mu            sync.Mutex
	serverFiles   []string
	serverLink    *rns.Link
	currentRes    *rns.Resource
	currentName   string
	downloadStart time.Time
	outDir        string
}

func runClient(configDir *string, destinationHexHash string, outDir string) error {
	destinationHash, err := parseTruncatedHashHex(destinationHexHash)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	if _, err := rns.NewReticulum(configDir, nil, nil, nil, false, nil); err != nil {
		return fmt.Errorf("NewReticulum: %w", err)
	}

	if !rns.HasPath(destinationHash) {
		rns.Log("Destination is not yet known. Requesting path and waiting for announce to arrive...", rns.LogInfo)
		rns.RequestPath(destinationHash, nil, nil, false)
		deadline := time.Now().Add(appTimeout)
		for time.Now().Before(deadline) && !rns.HasPath(destinationHash) {
			time.Sleep(100 * time.Millisecond)
		}
		if !rns.HasPath(destinationHash) {
			return errors.New("timed out waiting for path to destination")
		}
	}

	serverID := rns.IdentityRecall(destinationHash)
	if serverID == nil {
		return errors.New("could not recall server identity")
	}

	serverDest, err := rns.NewDestination(serverID, rns.DestinationOUT, rns.DestinationSINGLE, appName, "filetransfer", "server")
	if err != nil {
		return fmt.Errorf("NewDestination(server out): %w", err)
	}
	serverDest.SetProofStrategy(rns.DestinationPROVE_ALL)

	state := &clientState{outDir: outDir}
	link, err := rns.NewLink(serverDest, nil, rns.LinkModeDefault, func(l *rns.Link) {
		state.mu.Lock()
		state.serverLink = l
		state.mu.Unlock()

		rns.Log("Link established with server", rns.LogInfo)
		rns.Log("Requesting filelist...", rns.LogInfo)
		l.SetResourceStrategy(rns.LinkAcceptAll)
		l.SetResourceStartedCallback(state.downloadBegan)
		l.SetResourceConcludedCallback(state.downloadConcluded)

		l.Request("/files", nil, func(rr *rns.RequestReceipt) {
			state.fileListReceived(rr)
		}, nil, nil, appTimeout.Seconds())
	}, func(l *rns.Link) {
		rns.Log("Link closed, exiting now", rns.LogInfo)
		os.Exit(0)
	})
	if err != nil || link == nil {
		return fmt.Errorf("NewOutgoingLink: %v", err)
	}

	return state.menu()
}

func (s *clientState) fileListReceived(rr *rns.RequestReceipt) {
	if rr == nil {
		return
	}
	resp := rr.Response()
	if resp == nil {
		return
	}

	var files []string
	switch v := resp.(type) {
	case []string:
		files = v
	case []any:
		for _, it := range v {
			if s, ok := it.(string); ok {
				files = append(files, s)
			}
		}
	default:
		if b, ok := resp.([]byte); ok {
			_ = umsgpack.Unpackb(b, &files)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range files {
		if f == "" {
			continue
		}
		if !slices.Contains(s.serverFiles, f) {
			s.serverFiles = append(s.serverFiles, f)
		}
	}
	slices.Sort(s.serverFiles)
}

func (s *clientState) menu() error {
	deadline := time.Now().Add(appTimeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		ready := len(s.serverFiles) > 0 && s.serverLink != nil
		s.mu.Unlock()
		if ready {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	s.mu.Lock()
	files := slices.Clone(s.serverFiles)
	link := s.serverLink
	s.mu.Unlock()
	if link == nil || len(files) == 0 {
		return errors.New("timed out waiting for file list")
	}

	rns.Log("Ready!", rns.LogInfo)

	for {
		fmt.Println("")
		for i, name := range files {
			fmt.Printf("%d) %s\n", i, name)
		}
		fmt.Println("")
		fmt.Print("Select a file to download by entering name or number, or q to quit\n> ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				link.Teardown()
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "q" || line == "quit" || line == "exit" {
			fmt.Println("")
			link.Teardown()
			return nil
		}

		filename := ""
		if idx, ok := parseIndex(line, len(files)); ok {
			filename = files[idx]
		} else if slices.Contains(files, line) {
			filename = line
		}
		if filename == "" {
			continue
		}

		s.requestDownload(filename)
	}
}

func parseIndex(s string, max int) (int, bool) {
	var idx int
	_, err := fmt.Sscanf(s, "%d", &idx)
	if err != nil {
		return 0, false
	}
	if idx < 0 || idx >= max {
		return 0, false
	}
	return idx, true
}

func (s *clientState) requestDownload(filename string) {
	s.mu.Lock()
	link := s.serverLink
	s.currentName = filename
	s.currentRes = nil
	s.downloadStart = time.Time{}
	s.mu.Unlock()

	_ = link.Request("/get", filename, nil, nil, nil, appTimeout.Seconds())
	fmt.Println("")
	fmt.Printf("Requested %q from server, waiting for download to begin...\n", filename)
}

func (s *clientState) downloadBegan(res *rns.Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.downloadStart.IsZero() {
		s.downloadStart = time.Now()
	}
	s.currentRes = res
}

func (s *clientState) downloadConcluded(res *rns.Resource) {
	s.mu.Lock()
	filename := s.currentName
	start := s.downloadStart
	outDir := s.outDir
	s.mu.Unlock()

	if res == nil {
		return
	}

	if res.Status != rns.ResourceComplete {
		fmt.Println("")
		fmt.Println("The download failed! Press enter to return to the menu.")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	src := res.DataFile
	dst := uniquePath(filepath.Join(outDir, filename))
	if err := copyFile(dst, src); err != nil {
		fmt.Println("")
		fmt.Println("Could not write downloaded file to disk")
		return
	}

	dt := time.Since(start)
	fmt.Println("")
	fmt.Printf("Saved to %s (%s)\n", dst, dt.Round(10*time.Millisecond))
	fmt.Println("Press enter to return to the menu.")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	base := path
	for i := 1; ; i++ {
		try := fmt.Sprintf("%s.%d", base, i)
		if _, err := os.Stat(try); errors.Is(err, os.ErrNotExist) {
			return try
		}
	}
}

func copyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func parseTruncatedHashHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	destLen := (rns.TRUNCATED_HASHLENGTH / 8) * 2
	if len(s) != destLen {
		return nil, fmt.Errorf("invalid destination length: got %d want %d", len(s), destLen)
	}
	return hex.DecodeString(s)
}

func findActiveLinkByID(linkID []byte) *rns.Link {
	if len(linkID) == 0 {
		return nil
	}
	for _, l := range rns.ActiveLinks {
		if l != nil && bytes.Equal(l.LinkID, linkID) {
			return l
		}
	}
	return nil
}
