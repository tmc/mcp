// mcp-namespace - Plan9-style namespace server for MCP services
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

// Namespace represents a hierarchical namespace entry
type Namespace struct {
	mu      sync.RWMutex
	entries map[string]*Entry
	parent  *Namespace
	name    string
}

// Entry represents a service registration
type Entry struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"` // "local", "remote", "namespace"
	Transport string            `json:"transport,omitempty"`
	Address   string            `json:"address,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	LastSeen  time.Time         `json:"last_seen"`
	Children  *Namespace        `json:"-"` // For nested namespaces
}

// NamespaceServer handles namespace operations
type NamespaceServer struct {
	root *Namespace
	mu   sync.RWMutex
}

func NewNamespaceServer() *NamespaceServer {
	return &NamespaceServer{
		root: &Namespace{
			entries: make(map[string]*Entry),
			name:    "/",
		},
	}
}

// Register adds or updates an entry in the namespace
func (ns *NamespaceServer) Register(path string, entry *Entry) error {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return fmt.Errorf("invalid path")
	}

	ns.mu.Lock()
	defer ns.mu.Unlock()

	current := ns.root

	// Navigate to the parent namespace, creating as needed
	for i := 0; i < len(parts)-1; i++ {
		name := parts[i]
		current.mu.Lock()

		if existing, ok := current.entries[name]; ok {
			if existing.Type != "namespace" {
				current.mu.Unlock()
				return fmt.Errorf("path component %s is not a namespace", name)
			}
			current.mu.Unlock()
			current = existing.Children
		} else {
			// Create new namespace
			newNs := &Namespace{
				entries: make(map[string]*Entry),
				parent:  current,
				name:    name,
			}
			current.entries[name] = &Entry{
				Name:     name,
				Type:     "namespace",
				Children: newNs,
				LastSeen: time.Now(),
			}
			current.mu.Unlock()
			current = newNs
		}
	}

	// Register the entry
	name := parts[len(parts)-1]
	current.mu.Lock()
	entry.Name = name
	entry.LastSeen = time.Now()
	current.entries[name] = entry
	current.mu.Unlock()

	return nil
}

// Lookup finds an entry by path
func (ns *NamespaceServer) Lookup(path string) (*Entry, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path")
	}

	ns.mu.RLock()
	defer ns.mu.RUnlock()

	current := ns.root

	for i := 0; i < len(parts); i++ {
		name := parts[i]
		current.mu.RLock()

		if entry, ok := current.entries[name]; ok {
			if i == len(parts)-1 {
				current.mu.RUnlock()
				return entry, nil
			}

			if entry.Type != "namespace" {
				current.mu.RUnlock()
				return nil, fmt.Errorf("path component %s is not a namespace", name)
			}

			current.mu.RUnlock()
			current = entry.Children
		} else {
			current.mu.RUnlock()
			return nil, fmt.Errorf("not found: %s", path)
		}
	}

	return nil, fmt.Errorf("not found: %s", path)
}

// List returns entries in a namespace
func (ns *NamespaceServer) List(path string) ([]*Entry, error) {
	if path == "" || path == "/" {
		ns.mu.RLock()
		defer ns.mu.RUnlock()
		return ns.listNamespace(ns.root), nil
	}

	entry, err := ns.Lookup(path)
	if err != nil {
		return nil, err
	}

	if entry.Type != "namespace" {
		return nil, fmt.Errorf("not a namespace: %s", path)
	}

	return ns.listNamespace(entry.Children), nil
}

func (ns *NamespaceServer) listNamespace(namespace *Namespace) []*Entry {
	namespace.mu.RLock()
	defer namespace.mu.RUnlock()

	entries := make([]*Entry, 0, len(namespace.entries))
	for _, entry := range namespace.entries {
		entries = append(entries, entry)
	}
	return entries
}

// HTTP handlers

func (ns *NamespaceServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path  string `json:"path"`
		Entry Entry  `json:"entry"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := ns.Register(req.Path, &req.Entry); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (ns *NamespaceServer) handleLookup(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	entry, err := ns.Lookup(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func (ns *NamespaceServer) handleList(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	entries, err := ns.List(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// Plan9-style filesystem interface
func (ns *NamespaceServer) handle9P(w http.ResponseWriter, r *http.Request) {
	// This would implement a 9P-like protocol over HTTP
	// For now, we'll use a simplified REST-like interface

	path := strings.TrimPrefix(r.URL.Path, "/9p")

	switch r.Method {
	case http.MethodGet:
		// List directory or get file info
		entry, err := ns.Lookup(path)
		if err != nil {
			// Try listing as directory
			entries, err := ns.List(path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}

			// Return directory listing
			var names []string
			for _, e := range entries {
				names = append(names, e.Name)
			}

			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "%s\n", strings.Join(names, "\n"))
			return
		}

		// Return entry info
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)

	case http.MethodPost:
		// Create/update entry
		var entry Entry
		if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := ns.Register(path, &entry); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		// Remove entry (not implemented yet)
		http.Error(w, "Not implemented", http.StatusNotImplemented)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Mount mounts a namespace at a path (like Plan9 bind)
func (ns *NamespaceServer) Mount(source, target string) error {
	sourceEntry, err := ns.Lookup(source)
	if err != nil {
		return fmt.Errorf("source not found: %w", err)
	}

	if sourceEntry.Type != "namespace" {
		return fmt.Errorf("source is not a namespace")
	}

	// Create a reference at the target path
	return ns.Register(target, &Entry{
		Type:     "mount",
		Metadata: map[string]string{"source": source},
		LastSeen: time.Now(),
	})
}

func main() {
	var (
		addr    = flag.String("addr", ":9000", "Listen address")
		dataDir = flag.String("data", "", "Data directory for persistence")
	)
	flag.Parse()

	server := NewNamespaceServer()

	// Load from file if specified
	if *dataDir != "" {
		if err := server.LoadFromFile(path.Join(*dataDir, "namespace.json")); err != nil && !os.IsNotExist(err) {
			log.Printf("Failed to load namespace: %v", err)
		}
	}

	// HTTP routes
	http.HandleFunc("/register", server.handleRegister)
	http.HandleFunc("/lookup", server.handleLookup)
	http.HandleFunc("/list", server.handleList)
	http.HandleFunc("/9p/", server.handle9P)

	// Start periodic save if data directory is specified
	if *dataDir != "" {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				if err := server.SaveToFile(path.Join(*dataDir, "namespace.json")); err != nil {
					log.Printf("Failed to save namespace: %v", err)
				}
			}
		}()
	}

	log.Printf("Namespace server listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

// Persistence methods
func (ns *NamespaceServer) SaveToFile(filename string) error {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	data, err := json.MarshalIndent(ns.serializeNamespace(ns.root), "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func (ns *NamespaceServer) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}

	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.root = ns.deserializeNamespace(root, nil)
	return nil
}

func (ns *NamespaceServer) serializeNamespace(namespace *Namespace) map[string]interface{} {
	result := make(map[string]interface{})

	namespace.mu.RLock()
	defer namespace.mu.RUnlock()

	for name, entry := range namespace.entries {
		if entry.Type == "namespace" {
			result[name] = ns.serializeNamespace(entry.Children)
		} else {
			result[name] = entry
		}
	}

	return result
}

func (ns *NamespaceServer) deserializeNamespace(data map[string]interface{}, parent *Namespace) *Namespace {
	namespace := &Namespace{
		entries: make(map[string]*Entry),
		parent:  parent,
	}

	for name, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			// Check if it's an entry or a namespace
			if _, ok := v["type"]; ok {
				// It's an entry
				entryData, _ := json.Marshal(v)
				var entry Entry
				json.Unmarshal(entryData, &entry)
				entry.Name = name
				namespace.entries[name] = &entry
			} else {
				// It's a namespace
				child := ns.deserializeNamespace(v, namespace)
				child.name = name
				namespace.entries[name] = &Entry{
					Name:     name,
					Type:     "namespace",
					Children: child,
				}
			}
		}
	}

	return namespace
}
