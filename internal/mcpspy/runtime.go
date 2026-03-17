package mcpspy

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	heartbeatInterval = 2 * time.Second
	staleAfter        = 20 * time.Second
)

// InstanceInfo describes a running mcpspy instance.
type InstanceInfo struct {
	ID              string    `json:"id"`
	PID             int       `json:"pid"`
	PPID            int       `json:"ppid"`
	ParentMCPSpyPID int       `json:"parent_mcpspy_pid,omitempty"`
	SessionID       string    `json:"session_id"`
	Name            string    `json:"name,omitempty"`
	Command         string    `json:"command,omitempty"`
	CWD             string    `json:"cwd,omitempty"`
	StartedAt       time.Time `json:"started_at"`
	EndedAt         time.Time `json:"ended_at,omitempty"`
	OutputFile      string    `json:"output_file,omitempty"`
	UIURL           string    `json:"ui_url,omitempty"`
	ControlSocket   string    `json:"control_socket,omitempty"`
	Heartbeat       time.Time `json:"heartbeat"`
}

// RuntimeOptions configures runtime registration and control.
type RuntimeOptions struct {
	Name            string
	SessionID       string
	ParentMCPSpyPID int
	Command         []string
	OutputFile      string
	CWD             string
}

// SnapshotResponse is returned by the local control socket.
type SnapshotResponse struct {
	Info   InstanceInfo `json:"info"`
	Events []Event      `json:"events"`
}

// Runtime manages local discovery and control.
type Runtime struct {
	recorder     *Recorder
	dir          string
	registryPath string
	socketPath   string

	mu     sync.Mutex
	info   InstanceInfo
	openUI func() (string, error)

	listener net.Listener
	stop     chan struct{}
	done     chan struct{}
}

// NewRuntime constructs a Runtime.
func NewRuntime(recorder *Recorder, opts RuntimeOptions) (*Runtime, error) {
	dir, err := runtimeDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create runtime dir: %w", err)
	}
	id := newID()
	now := time.Now()
	command := strings.Join(opts.Command, " ")
	if opts.CWD == "" {
		opts.CWD, _ = os.Getwd()
	}
	info := InstanceInfo{
		ID:              id,
		PID:             os.Getpid(),
		PPID:            os.Getppid(),
		ParentMCPSpyPID: opts.ParentMCPSpyPID,
		SessionID:       opts.SessionID,
		Name:            opts.Name,
		Command:         command,
		CWD:             opts.CWD,
		StartedAt:       now,
		OutputFile:      opts.OutputFile,
		Heartbeat:       now,
	}
	socketPath, err := controlSocketPath(dir, id)
	if err != nil {
		return nil, err
	}
	return &Runtime{
		recorder:     recorder,
		dir:          dir,
		registryPath: filepath.Join(dir, id+".json"),
		socketPath:   socketPath,
		info:         info,
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}, nil
}

// Start registers the runtime and starts the local control socket.
func (r *Runtime) Start(openUI func() (string, error)) error {
	r.mu.Lock()
	r.openUI = openUI
	r.info.ControlSocket = r.socketPath
	r.mu.Unlock()

	if err := pruneStale(r.dir); err != nil {
		return err
	}
	if err := r.writeRegistry(); err != nil {
		return err
	}

	_ = os.Remove(r.socketPath)
	ln, err := net.Listen("unix", r.socketPath)
	if err != nil {
		return fmt.Errorf("listen control socket: %w", err)
	}
	if err := os.Chmod(r.socketPath, 0600); err != nil {
		ln.Close()
		return fmt.Errorf("chmod control socket: %w", err)
	}
	r.listener = ln

	go r.serveControl()
	go r.heartbeat()
	return nil
}

// Close stops the runtime. The final registry entry is left on disk until stale cleanup.
func (r *Runtime) Close() error {
	close(r.stop)
	if r.listener != nil {
		r.listener.Close()
	}
	<-r.done
	_ = os.Remove(r.socketPath)
	r.mu.Lock()
	r.info.EndedAt = time.Now()
	r.info.ControlSocket = ""
	r.info.Heartbeat = time.Now()
	r.mu.Unlock()
	return r.writeRegistry()
}

// Status returns the current instance info.
func (r *Runtime) Status() InstanceInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.info
}

// UpdateUIURL records a resolved UI URL.
func (r *Runtime) UpdateUIURL(url string) error {
	r.mu.Lock()
	r.info.UIURL = url
	r.info.Heartbeat = time.Now()
	r.mu.Unlock()
	return r.writeRegistry()
}

// Peers returns known local peers, sorted by relationship.
func (r *Runtime) Peers() ([]InstanceInfo, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}
	self := r.Status()
	var peers []InstanceInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := readRegistry(filepath.Join(r.dir, entry.Name()))
		if err != nil {
			continue
		}
		if info.ID == self.ID {
			continue
		}
		peers = append(peers, info)
	}
	sort.Slice(peers, func(i, j int) bool {
		return peerRank(self, peers[i]) < peerRank(self, peers[j])
	})
	return peers, nil
}

func peerRank(self, peer InstanceInfo) string {
	var rank string
	switch {
	case peer.SessionID == self.SessionID:
		rank = "0"
	case peer.ParentMCPSpyPID == self.PID || self.ParentMCPSpyPID == peer.PID:
		rank = "1"
	default:
		rank = "2"
	}
	return rank + ":" + peer.StartedAt.Format(time.RFC3339Nano) + ":" + peer.ID
}

func (r *Runtime) writeRegistry() error {
	r.mu.Lock()
	info := r.info
	r.mu.Unlock()

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.registryPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, r.registryPath)
}

func (r *Runtime) heartbeat() {
	ticker := time.NewTicker(heartbeatInterval)
	defer func() {
		ticker.Stop()
		close(r.done)
	}()
	for {
		select {
		case <-ticker.C:
			r.mu.Lock()
			r.info.Heartbeat = time.Now()
			r.mu.Unlock()
			_ = r.writeRegistry()
			_ = pruneStale(r.dir)
		case <-r.stop:
			return
		}
	}
}

func (r *Runtime) serveControl() {
	for {
		conn, err := r.listener.Accept()
		if err != nil {
			select {
			case <-r.stop:
				return
			default:
				continue
			}
		}
		go r.handleConn(conn)
	}
}

type controlRequest struct {
	Method string `json:"method"`
}

type controlResponse struct {
	Info     *InstanceInfo     `json:"info,omitempty"`
	URL      string            `json:"url,omitempty"`
	Snapshot *SnapshotResponse `json:"snapshot,omitempty"`
	Error    string            `json:"error,omitempty"`
}

func (r *Runtime) handleConn(conn net.Conn) {
	defer conn.Close()
	var req controlRequest
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(controlResponse{Error: err.Error()})
		return
	}
	resp := controlResponse{}
	switch req.Method {
	case "status":
		info := r.Status()
		resp.Info = &info
	case "open_ui":
		r.mu.Lock()
		openUI := r.openUI
		r.mu.Unlock()
		if openUI == nil {
			resp.Error = "ui unavailable"
			break
		}
		url, err := openUI()
		if err != nil {
			resp.Error = err.Error()
			break
		}
		resp.URL = url
	case "snapshot":
		resp.Snapshot = &SnapshotResponse{
			Info:   r.Status(),
			Events: r.recorder.Snapshot(),
		}
	default:
		resp.Error = "unknown method"
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

// OpenPeerUI asks a peer to start its UI lazily.
func OpenPeerUI(info InstanceInfo) (string, error) {
	if info.UIURL != "" {
		return info.UIURL, nil
	}
	if info.ControlSocket == "" {
		return "", errors.New("peer has no control socket")
	}
	conn, err := net.Dial("unix", info.ControlSocket)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if err := json.NewEncoder(conn).Encode(controlRequest{Method: "open_ui"}); err != nil {
		return "", err
	}
	var resp controlResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}
	if resp.URL == "" {
		return "", errors.New("peer did not return a ui url")
	}
	return resp.URL, nil
}

func readRegistry(path string) (InstanceInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return InstanceInfo{}, err
	}
	var info InstanceInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return InstanceInfo{}, err
	}
	return info, nil
}

func pruneStale(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".sock") {
			continue
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		path := filepath.Join(dir, name)
		info, err := readRegistry(path)
		if err != nil {
			_ = os.Remove(path)
			continue
		}
		if processAlive(info.PID) {
			continue
		}
		if now.Sub(info.Heartbeat) < staleAfter {
			continue
		}
		_ = os.Remove(path)
		if info.ControlSocket != "" {
			_ = os.Remove(info.ControlSocket)
		}
	}
	return nil
}

func runtimeDir() (string, error) {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "mcpspy"), nil
	}
	return filepath.Join("/tmp", fmt.Sprintf("mcpspy-%d", os.Getuid())), nil
}

func controlSocketPath(dir, id string) (string, error) {
	path := filepath.Join(dir, id+".sock")
	if len(path) < 100 {
		return path, nil
	}
	shortDir := filepath.Join("/tmp", fmt.Sprintf("mcpspy-%d", os.Getuid()))
	if err := os.MkdirAll(shortDir, 0700); err != nil {
		return "", fmt.Errorf("create short socket dir: %w", err)
	}
	shortID := id
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return filepath.Join(shortDir, shortID+".sock"), nil
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		now := time.Now().UnixNano()
		return fmt.Sprintf("%x", now)
	}
	var out [16]byte
	hex.Encode(out[:], b[:])
	return string(bytes.ToLower(out[:]))
}
