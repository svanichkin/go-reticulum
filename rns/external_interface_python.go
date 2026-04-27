package rns

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	ifaces "github.com/svanichkin/go-reticulum/rns/interfaces"
)

//go:embed pyhelpers/external_interface_bridge.py
var externalInterfaceBridgeScript string

var externalInterfacePythonLookup = func() (string, error) {
	if candidate := strings.TrimSpace(os.Getenv("RNS_PYTHON")); candidate != "" {
		return exec.LookPath(candidate)
	}
	return exec.LookPath("python3")
}

type externalInterfaceEvent struct {
	Type          string `json:"type"`
	Name          string `json:"name,omitempty"`
	InterfaceType string `json:"interface_type,omitempty"`
	Message       string `json:"message,omitempty"`
	Data          string `json:"data,omitempty"`
	Level         *int   `json:"level,omitempty"`
	Online        *bool  `json:"online,omitempty"`
	Bitrate       *int   `json:"bitrate,omitempty"`
	HWMTU         *int   `json:"hw_mtu,omitempty"`
	IFACSize      *int   `json:"ifac_size,omitempty"`
}

type externalInterfaceCommand struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
}

type pythonExternalInterfaceDriver struct {
	ifc        *Interface
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	scriptPath string

	mu     sync.Mutex
	closed bool
}

func loadExternalInterfacePython(interfacePath, ifType, name string, kv map[string]string) (*Interface, bool, error) {
	if strings.TrimSpace(interfacePath) == "" || strings.TrimSpace(ifType) == "" {
		return nil, false, nil
	}

	pyPath := filepath.Join(interfacePath, ifType+".py")
	if st, err := os.Stat(pyPath); err != nil || st.IsDir() {
		return nil, false, nil
	}

	pythonBin, err := externalInterfacePythonLookup()
	if err != nil {
		return nil, true, fmt.Errorf("python3 is not available: %w", err)
	}

	scriptPath, err := materialiseExternalInterfaceBridgeScript()
	if err != nil {
		return nil, true, err
	}

	cfg := make(map[string]string, len(kv))
	for k, v := range kv {
		cfg[k] = v
	}
	if strings.TrimSpace(cfg["name"]) == "" {
		cfg["name"] = strings.TrimSpace(name)
	}
	if strings.TrimSpace(cfg["type"]) == "" {
		cfg["type"] = strings.TrimSpace(ifType)
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		_ = os.Remove(scriptPath)
		return nil, true, fmt.Errorf("could not serialise external interface config: %w", err)
	}

	cmd := exec.Command(pythonBin, scriptPath, pyPath, string(cfgJSON))
	cmd.Dir = interfacePath
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = os.Remove(scriptPath)
		return nil, true, fmt.Errorf("could not open stdin for external interface %q: %w", ifType, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		_ = os.Remove(scriptPath)
		return nil, true, fmt.Errorf("could not open stdout for external interface %q: %w", ifType, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = os.Remove(scriptPath)
		return nil, true, fmt.Errorf("could not open stderr for external interface %q: %w", ifType, err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = os.Remove(scriptPath)
		return nil, true, fmt.Errorf("could not start external interface %q: %w", ifType, err)
	}

	go logExternalInterfaceStderr(ifType, name, stderr)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*ifaces.MaxFrameLength)

	ready, err := readExternalInterfaceEvent(scanner)
	if err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		_ = os.Remove(scriptPath)
		return nil, true, fmt.Errorf("external interface %s / %s did not initialise: %w", ifType, name, err)
	}
	if ready.Type != "ready" {
		_ = stdin.Close()
		_ = cmd.Wait()
		_ = os.Remove(scriptPath)
		return nil, true, fmt.Errorf("external interface %s / %s returned unexpected initial event %q", ifType, name, ready.Type)
	}

	ifc := &Interface{
		Name:              strings.TrimSpace(firstNonEmpty(ready.Name, name)),
		Type:              strings.TrimSpace(firstNonEmpty(ready.InterfaceType, ifType)),
		DriverImplemented: true,
	}
	if ready.Bitrate != nil {
		ifc.Bitrate = *ready.Bitrate
	}
	if ready.HWMTU != nil {
		ifc.HWMTU = *ready.HWMTU
	}
	if ready.Online != nil {
		ifc.Online = *ready.Online
	}
	if ready.IFACSize != nil {
		ifc.IFACSize = *ready.IFACSize
	}

	driver := &pythonExternalInterfaceDriver{
		ifc:        ifc,
		cmd:        cmd,
		stdin:      stdin,
		scriptPath: scriptPath,
	}
	ifc.SetProcessOutgoingFunc(driver.processOutgoing)
	ifc.SetDetachFunc(driver.detach)

	go driver.readLoop(scanner)
	go driver.waitLoop()

	return ifc, true, nil
}

func materialiseExternalInterfaceBridgeScript() (string, error) {
	f, err := os.CreateTemp("", "rns-external-interface-bridge-*.py")
	if err != nil {
		return "", fmt.Errorf("could not create external interface bridge helper: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, bytes.NewBufferString(externalInterfaceBridgeScript)); err != nil {
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("could not write external interface bridge helper: %w", err)
	}
	return f.Name(), nil
}

func readExternalInterfaceEvent(scanner *bufio.Scanner) (externalInterfaceEvent, error) {
	for scanner.Scan() {
		var event externalInterfaceEvent
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if err := json.Unmarshal(line, &event); err != nil {
			return externalInterfaceEvent{}, fmt.Errorf("could not decode bridge message %q: %w", string(line), err)
		}
		return event, nil
	}
	if err := scanner.Err(); err != nil {
		return externalInterfaceEvent{}, err
	}
	return externalInterfaceEvent{}, io.EOF
}

func logExternalInterfaceStderr(ifType, name string, r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 4096), 256*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		Log(fmt.Sprintf("external python interface %s / %s: %s", ifType, name, line), LogNotice)
	}
}

func (d *pythonExternalInterfaceDriver) processOutgoing(data []byte) error {
	if d == nil || d.ifc == nil || len(data) == 0 {
		return nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.stdin == nil {
		return errors.New("external interface is offline")
	}

	cmd := externalInterfaceCommand{
		Type: "outgoing",
		Data: base64.StdEncoding.EncodeToString(data),
	}
	frame, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	frame = append(frame, '\n')
	if _, err := d.stdin.Write(frame); err != nil {
		d.ifc.Online = false
		return err
	}
	atomic.AddUint64(&d.ifc.TXB, uint64(len(data)))
	if d.ifc.Parent != nil {
		atomic.AddUint64(&d.ifc.Parent.TXB, uint64(len(data)))
	}
	return nil
}

func (d *pythonExternalInterfaceDriver) detach() {
	if d == nil {
		return
	}

	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	stdin := d.stdin
	cmd := d.cmd
	d.stdin = nil
	d.mu.Unlock()

	if stdin != nil {
		frame, _ := json.Marshal(externalInterfaceCommand{Type: "detach"})
		frame = append(frame, '\n')
		_, _ = stdin.Write(frame)
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func (d *pythonExternalInterfaceDriver) waitLoop() {
	if d == nil || d.cmd == nil {
		return
	}
	_ = d.cmd.Wait()
	if d.ifc != nil {
		d.ifc.Online = false
	}
	if d.scriptPath != "" {
		_ = os.Remove(d.scriptPath)
	}
}

func (d *pythonExternalInterfaceDriver) readLoop(scanner *bufio.Scanner) {
	if d == nil || d.ifc == nil {
		return
	}

	for {
		event, err := readExternalInterfaceEvent(scanner)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				Log(fmt.Sprintf("external python interface %s: %v", d.ifc, err), LogError)
			}
			return
		}

		switch event.Type {
		case "inbound":
			payload, err := base64.StdEncoding.DecodeString(event.Data)
			if err != nil {
				Log(fmt.Sprintf("could not decode inbound payload from external interface %s: %v", d.ifc, err), LogError)
				continue
			}
			atomic.AddUint64(&d.ifc.RXB, uint64(len(payload)))
			if d.ifc.Parent != nil {
				atomic.AddUint64(&d.ifc.Parent.RXB, uint64(len(payload)))
			}
			if ifaces.InboundHandler != nil {
				ifaces.InboundHandler(payload, d.ifc)
			}
		case "log":
			level := LogInfo
			if event.Level != nil {
				level = *event.Level
			}
			Log(fmt.Sprintf("external python interface %s: %s", d.ifc, event.Message), level)
		case "error":
			Log(fmt.Sprintf("external python interface %s: %s", d.ifc, event.Message), LogError)
		case "state":
			if event.Online != nil {
				d.ifc.Online = *event.Online
			}
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
