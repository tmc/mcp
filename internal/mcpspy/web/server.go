// Package web serves the embedded mcpspy UI.
package web

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/tmc/mcp/internal/mcpspy"
)

//go:generate sh -c "cd app && npm install --package-lock=false && npm run build"

//go:embed dist/*
var assets embed.FS

// Options configures the embedded web server.
type Options struct {
	Addr       string
	OutputFile string
}

// Server serves the embedded mcpspy UI.
type Server struct {
	recorder *mcpspy.Recorder
	runtime  *mcpspy.Runtime
	spec     *mcpspy.SpecTracker
	opts     Options

	mu       sync.Mutex
	listener net.Listener
	server   *http.Server
	url      string
}

// New creates a Server.
func New(recorder *mcpspy.Recorder, spec *mcpspy.SpecTracker, rt *mcpspy.Runtime, opts Options) *Server {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:0"
	}
	return &Server{
		recorder: recorder,
		spec:     spec,
		runtime:  rt,
		opts:     opts,
	}
}

// Start starts the server lazily and returns its URL.
func (s *Server) Start() (string, error) {
	s.mu.Lock()
	if s.url != "" {
		defer s.mu.Unlock()
		return s.url, nil
	}
	s.mu.Unlock()

	ln, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return "", err
	}
	url := "http://" + ln.Addr().String()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/info", s.handleInfo)
	mux.HandleFunc("/api/snapshot", s.handleSnapshot)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/file", s.handleFile)
	mux.HandleFunc("/api/spec", s.handleSpec)
	mux.HandleFunc("/api/peers", s.handlePeers)
	mux.HandleFunc("/api/peers/open", s.handleOpenPeer)
	mux.Handle("/", s.staticHandler())

	srv := &http.Server{
		Handler: mux,
	}

	s.mu.Lock()
	s.listener = ln
	s.server = srv
	s.url = url
	s.mu.Unlock()

	if err := s.runtime.UpdateUIURL(url); err != nil {
		ln.Close()
		return "", err
	}

	go srv.Serve(ln)
	return url, nil
}

// Close shuts down the server.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server == nil {
		return nil
	}
	err := s.server.Close()
	s.server = nil
	s.listener = nil
	s.url = ""
	return err
}

// OpenBrowser opens the specified URL in the user's browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		if _, err := exec.LookPath("xdg-open"); err != nil {
			return errors.New("xdg-open not found")
		}
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func (s *Server) handleInfo(w http.ResponseWriter, _ *http.Request) {
	peers, _ := s.runtime.Peers()
	respondJSON(w, map[string]any{
		"self":        s.runtime.Status(),
		"peers":       peers,
		"url":         s.urlString(),
		"output_file": s.opts.OutputFile,
	})
}

func (s *Server) handleSnapshot(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, map[string]any{
		"events": s.recorder.Snapshot(),
		"file":   currentFile(s.opts.OutputFile),
		"spec":   s.specSnapshot(),
	})
}

func (s *Server) handleFile(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, currentFile(s.opts.OutputFile))
}

func (s *Server) handleSpec(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, s.specSnapshot())
}

func (s *Server) handlePeers(w http.ResponseWriter, _ *http.Request) {
	peers, _ := s.runtime.Peers()
	respondJSON(w, peers)
}

func (s *Server) handleOpenPeer(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	peers, _ := s.runtime.Peers()
	for _, peer := range peers {
		if peer.ID != id {
			continue
		}
		url, err := mcpspy.OpenPeerUI(peer)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		respondJSON(w, map[string]string{"url": url})
		return
	}
	http.Error(w, "peer not found", http.StatusNotFound)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	events, cancel := s.recorder.Subscribe()
	defer cancel()
	specs, cancelSpec := s.specSubscribe()
	defer cancelSpec()
	peerTicker := time.NewTicker(2 * time.Second)
	pingTicker := time.NewTicker(10 * time.Second)
	defer peerTicker.Stop()
	defer pingTicker.Stop()

	lastPeers := ""
	send := func(v any) bool {
		data, err := json.Marshal(v)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	peers, _ := s.runtime.Peers()
	if b, err := json.Marshal(peers); err == nil {
		lastPeers = string(b)
	}
	if !send(map[string]any{"type": "peers", "peers": peers}) {
		return
	}
	if spec := s.specSnapshot(); spec.Path != "" || spec.Text != "" || spec.Spec.Server.Name != "" || len(spec.Spec.Tools) > 0 || len(spec.Spec.Resources) > 0 || len(spec.Spec.Prompts) > 0 {
		if !send(map[string]any{"type": "spec", "spec": spec}) {
			return
		}
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if !send(map[string]any{"type": "event", "event": ev}) {
				return
			}
		case spec, ok := <-specs:
			if !ok {
				specs = nil
				continue
			}
			if !send(map[string]any{"type": "spec", "spec": spec}) {
				return
			}
		case <-peerTicker.C:
			peers, _ := s.runtime.Peers()
			b, err := json.Marshal(peers)
			if err != nil {
				continue
			}
			if string(b) == lastPeers {
				continue
			}
			lastPeers = string(b)
			if !send(map[string]any{"type": "peers", "peers": peers}) {
				return
			}
		case <-pingTicker.C:
			if !send(map[string]any{"type": "ping", "time": time.Now()}) {
				return
			}
		}
	}
}

func (s *Server) specSnapshot() mcpspy.SpecSnapshot {
	if s.spec == nil {
		return mcpspy.SpecSnapshot{}
	}
	return s.spec.Snapshot()
}

func (s *Server) specSubscribe() (<-chan mcpspy.SpecSnapshot, func()) {
	if s.spec == nil {
		ch := make(chan mcpspy.SpecSnapshot)
		close(ch)
		return ch, func() {}
	}
	return s.spec.Subscribe()
}

func (s *Server) staticHandler() http.Handler {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		})
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			r.URL.Path = "/index.html"
		}
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) urlString() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.url
}

type fileSnapshot struct {
	Path    string    `json:"path,omitempty"`
	Size    int64     `json:"size,omitempty"`
	ModTime time.Time `json:"mod_time,omitempty"`
	Lines   []string  `json:"lines,omitempty"`
}

func currentFile(path string) fileSnapshot {
	if path == "" {
		return fileSnapshot{}
	}
	info, err := os.Stat(path)
	if err != nil {
		return fileSnapshot{Path: path}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fileSnapshot{Path: path, Size: info.Size(), ModTime: info.ModTime()}
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	return fileSnapshot{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Lines:   out,
	}
}

func respondJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(buf.Bytes())
}
