// Package main provides mcp-studio, a visual development environment for MCP servers
// with web-based interface, visual flow designer, real-time testing, and debugging capabilities.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/tmc/mcp"
)

const (
	// Version information
	Version = "1.0.0"
	Name    = "mcp-studio"

	// Default configuration
	DefaultPort         = 8080
	DefaultWorkspaceDir = "~/.mcp-studio"
	DefaultConfigFile   = "~/.mcp-studio-config.json"
	WebSocketTimeout    = 30 * time.Second
	ServerPingInterval  = 30 * time.Second
	MaxConnections      = 100
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

// Config represents the studio configuration
type Config struct {
	Port         int                   `json:"port"`
	WorkspaceDir string                `json:"workspace_dir"`
	Projects     map[string]*Project   `json:"projects"`
	Servers      map[string]*ServerDef `json:"servers"`
	Settings     *StudioSettings       `json:"settings"`
}

// StudioSettings represents global studio settings
type StudioSettings struct {
	Theme               string `json:"theme"`
	AutoSave            bool   `json:"auto_save"`
	AutoSaveInterval    int    `json:"auto_save_interval"`
	ShowGridLines       bool   `json:"show_grid_lines"`
	EnableCollaboration bool   `json:"enable_collaboration"`
	EnableDebugger      bool   `json:"enable_debugger"`
	MaxRecentProjects   int    `json:"max_recent_projects"`
	EnableHotReload     bool   `json:"enable_hot_reload"`
}

// Project represents a studio project
type Project struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Author      string                 `json:"author"`
	Flows       map[string]*Flow       `json:"flows"`
	Servers     map[string]*ServerDef  `json:"servers"`
	Variables   map[string]interface{} `json:"variables"`
	Settings    *ProjectSettings       `json:"settings"`
}

// ProjectSettings represents project-specific settings
type ProjectSettings struct {
	DefaultTimeout time.Duration `json:"default_timeout"`
	EnableTracing  bool          `json:"enable_tracing"`
	EnableMetrics  bool          `json:"enable_metrics"`
}

// Flow represents a visual flow of MCP operations
type Flow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Nodes       map[string]*FlowNode   `json:"nodes"`
	Edges       map[string]*FlowEdge   `json:"edges"`
	Variables   map[string]interface{} `json:"variables"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// FlowNode represents a node in a flow
type FlowNode struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "tool", "resource", "prompt", "condition", "loop", "variable"
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Position    *NodePosition          `json:"position"`
	Config      map[string]interface{} `json:"config"`
	Server      string                 `json:"server"`
	Status      string                 `json:"status"` // "idle", "running", "success", "error"
	LastRun     time.Time              `json:"last_run"`
	RunCount    int                    `json:"run_count"`
	Results     []interface{}          `json:"results"`
}

// NodePosition represents position in the visual editor
type NodePosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// FlowEdge represents a connection between nodes
type FlowEdge struct {
	ID     string                 `json:"id"`
	From   string                 `json:"from"`
	To     string                 `json:"to"`
	Label  string                 `json:"label"`
	Type   string                 `json:"type"` // "data", "control", "error"
	Config map[string]interface{} `json:"config"`
}

// ServerDef represents a server definition
type ServerDef struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Transport   string            `json:"transport"` // "stdio", "http", "sse"
	Command     []string          `json:"command"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	AutoStart   bool              `json:"auto_start"`
	HealthCheck *HealthCheck      `json:"health_check"`
}

// HealthCheck represents server health check configuration
type HealthCheck struct {
	Enabled  bool          `json:"enabled"`
	Interval time.Duration `json:"interval"`
	Timeout  time.Duration `json:"timeout"`
	Retries  int           `json:"retries"`
}

// Studio represents the main studio application
type Studio struct {
	config     *Config
	server     *http.Server
	router     *mux.Router
	wsUpgrader websocket.Upgrader
	clients    map[string]*Client
	servers    map[string]*ServerConnection
	projects   map[string]*Project
	templates  *template.Template
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	Conn     *websocket.Conn
	Project  string
	UserID   string
	LastSeen time.Time
	Send     chan []byte
	mu       sync.Mutex
}

// ServerConnection represents a connection to an MCP server
type ServerConnection struct {
	ID           string
	Name         string
	Client       *mcp.Client
	Connected    bool
	LastPing     time.Time
	Tools        []mcp.Tool
	Resources    []mcp.Resource
	Prompts      []mcp.Prompt
	Capabilities mcp.ServerCapabilities
	mu           sync.RWMutex
}

// Message represents a WebSocket message
type Message struct {
	Type      string      `json:"type"`
	ID        string      `json:"id,omitempty"`
	ProjectID string      `json:"project_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Global flags
var (
	port         = flag.Int("port", DefaultPort, "Port to listen on")
	workspaceDir = flag.String("workspace", DefaultWorkspaceDir, "Workspace directory")
	configFile   = flag.String("config", DefaultConfigFile, "Configuration file")
	debug        = flag.Bool("debug", false, "Enable debug mode")
	version      = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%s v%s\n", Name, Version)
		return
	}

	// Initialize studio
	studio, err := NewStudio()
	if err != nil {
		log.Fatalf("Failed to initialize studio: %v", err)
	}

	// Start studio server
	log.Printf("Starting %s v%s on port %d", Name, Version, *port)
	if err := studio.Start(); err != nil {
		log.Fatalf("Failed to start studio: %v", err)
	}
}

// NewStudio creates a new studio instance
func NewStudio() (*Studio, error) {
	// Load configuration
	config, err := LoadConfig(*configFile)
	if err != nil {
		// Use default config if file doesn't exist
		config = &Config{
			Port:         *port,
			WorkspaceDir: expandPath(*workspaceDir),
			Projects:     make(map[string]*Project),
			Servers:      make(map[string]*ServerDef),
			Settings: &StudioSettings{
				Theme:               "light",
				AutoSave:            true,
				AutoSaveInterval:    30,
				ShowGridLines:       true,
				EnableCollaboration: true,
				EnableDebugger:      true,
				MaxRecentProjects:   10,
				EnableHotReload:     true,
			},
		}
	}

	// Ensure workspace directory exists
	if err := os.MkdirAll(config.WorkspaceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Load templates
	templates, err := template.ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	studio := &Studio{
		config:    config,
		clients:   make(map[string]*Client),
		servers:   make(map[string]*ServerConnection),
		projects:  make(map[string]*Project),
		templates: templates,
		ctx:       ctx,
		cancel:    cancel,
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}

	// Setup router
	studio.setupRouter()

	// Load projects
	if err := studio.loadProjects(); err != nil {
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}

	// Start background services
	go studio.startHealthChecker()
	go studio.startAutoSaver()

	return studio, nil
}

// setupRouter sets up the HTTP router
func (s *Studio) setupRouter() {
	s.router = mux.NewRouter()

	// Static files
	s.router.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticFiles)))

	// WebSocket endpoint
	s.router.HandleFunc("/ws", s.handleWebSocket)

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/projects", s.handleGetProjects).Methods("GET")
	api.HandleFunc("/projects", s.handleCreateProject).Methods("POST")
	api.HandleFunc("/projects/{id}", s.handleGetProject).Methods("GET")
	api.HandleFunc("/projects/{id}", s.handleUpdateProject).Methods("PUT")
	api.HandleFunc("/projects/{id}", s.handleDeleteProject).Methods("DELETE")
	api.HandleFunc("/projects/{id}/flows", s.handleGetFlows).Methods("GET")
	api.HandleFunc("/projects/{id}/flows", s.handleCreateFlow).Methods("POST")
	api.HandleFunc("/projects/{id}/flows/{flowId}", s.handleGetFlow).Methods("GET")
	api.HandleFunc("/projects/{id}/flows/{flowId}", s.handleUpdateFlow).Methods("PUT")
	api.HandleFunc("/projects/{id}/flows/{flowId}", s.handleDeleteFlow).Methods("DELETE")
	api.HandleFunc("/projects/{id}/flows/{flowId}/run", s.handleRunFlow).Methods("POST")
	api.HandleFunc("/servers", s.handleGetServers).Methods("GET")
	api.HandleFunc("/servers", s.handleCreateServer).Methods("POST")
	api.HandleFunc("/servers/{id}", s.handleGetServer).Methods("GET")
	api.HandleFunc("/servers/{id}", s.handleUpdateServer).Methods("PUT")
	api.HandleFunc("/servers/{id}", s.handleDeleteServer).Methods("DELETE")
	api.HandleFunc("/servers/{id}/connect", s.handleConnectServer).Methods("POST")
	api.HandleFunc("/servers/{id}/disconnect", s.handleDisconnectServer).Methods("POST")
	api.HandleFunc("/servers/{id}/ping", s.handlePingServer).Methods("POST")
	api.HandleFunc("/servers/{id}/tools", s.handleGetServerTools).Methods("GET")
	api.HandleFunc("/servers/{id}/resources", s.handleGetServerResources).Methods("GET")
	api.HandleFunc("/servers/{id}/prompts", s.handleGetServerPrompts).Methods("GET")
	api.HandleFunc("/settings", s.handleGetSettings).Methods("GET")
	api.HandleFunc("/settings", s.handleUpdateSettings).Methods("PUT")

	// Web UI routes
	s.router.HandleFunc("/", s.handleIndex)
	s.router.HandleFunc("/project/{id}", s.handleProject)
	s.router.HandleFunc("/project/{id}/flow/{flowId}", s.handleFlowEditor)
	s.router.HandleFunc("/servers", s.handleServersPage)
	s.router.HandleFunc("/settings", s.handleSettingsPage)
}

// Start starts the studio server
func (s *Studio) Start() error {
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: s.router,
	}

	log.Printf("Studio available at http://localhost:%d", s.config.Port)
	return s.server.ListenAndServe()
}

// Stop stops the studio server
func (s *Studio) Stop() error {
	s.cancel()

	// Close all WebSocket connections
	s.mu.Lock()
	for _, client := range s.clients {
		client.Conn.Close()
	}
	s.mu.Unlock()

	// Disconnect all servers
	for _, server := range s.servers {
		if server.Connected {
			server.Client.Close()
		}
	}

	// Stop HTTP server
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}

	return nil
}

// WebSocket handler
func (s *Studio) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		ID:       generateID(),
		Conn:     conn,
		LastSeen: time.Now(),
		Send:     make(chan []byte, 256),
	}

	s.mu.Lock()
	s.clients[client.ID] = client
	s.mu.Unlock()

	// Start client message handlers
	go s.handleClientMessages(client)
	go s.handleClientWrites(client)

	// Send initial state
	s.sendToClient(client, &Message{
		Type:      "connected",
		ID:        client.ID,
		Timestamp: time.Now(),
	})
}

// handleClientMessages handles incoming WebSocket messages
func (s *Studio) handleClientMessages(client *Client) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, client.ID)
		s.mu.Unlock()
		client.Conn.Close()
	}()

	for {
		var msg Message
		if err := client.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		client.LastSeen = time.Now()
		s.handleMessage(client, &msg)
	}
}

// handleClientWrites handles outgoing WebSocket messages
func (s *Studio) handleClientWrites(client *Client) {
	defer client.Conn.Close()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}
}

// handleMessage handles a WebSocket message
func (s *Studio) handleMessage(client *Client, msg *Message) {
	switch msg.Type {
	case "ping":
		s.sendToClient(client, &Message{
			Type:      "pong",
			ID:        msg.ID,
			Timestamp: time.Now(),
		})

	case "subscribe_project":
		if projectID, ok := msg.Data.(string); ok {
			client.Project = projectID
		}

	case "run_flow":
		s.handleRunFlowMessage(client, msg)

	case "update_node":
		s.handleUpdateNodeMessage(client, msg)

	case "update_flow":
		s.handleUpdateFlowMessage(client, msg)

	default:
		s.sendToClient(client, &Message{
			Type:      "error",
			ID:        msg.ID,
			Error:     fmt.Sprintf("Unknown message type: %s", msg.Type),
			Timestamp: time.Now(),
		})
	}
}

// sendToClient sends a message to a specific client
func (s *Studio) sendToClient(client *Client, msg *Message) {
	client.mu.Lock()
	defer client.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	select {
	case client.Send <- data:
	default:
		close(client.Send)
		delete(s.clients, client.ID)
	}
}

// broadcastToProject sends a message to all clients in a project
func (s *Studio) broadcastToProject(projectID string, msg *Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		if client.Project == projectID {
			s.sendToClient(client, msg)
		}
	}
}

// HTTP handlers

// handleIndex serves the main index page
func (s *Studio) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title    string
		Projects map[string]*Project
		Settings *StudioSettings
	}{
		Title:    "MCP Studio",
		Projects: s.projects,
		Settings: s.config.Settings,
	}

	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleProject serves the project page
func (s *Studio) handleProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	project, exists := s.projects[projectID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Title   string
		Project *Project
		Servers map[string]*ServerConnection
	}{
		Title:   fmt.Sprintf("Project: %s", project.Name),
		Project: project,
		Servers: s.servers,
	}

	if err := s.templates.ExecuteTemplate(w, "project.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleFlowEditor serves the flow editor page
func (s *Studio) handleFlowEditor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]
	flowID := vars["flowId"]

	project, exists := s.projects[projectID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	flow, exists := project.Flows[flowID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Title   string
		Project *Project
		Flow    *Flow
		Servers map[string]*ServerConnection
	}{
		Title:   fmt.Sprintf("Flow: %s", flow.Name),
		Project: project,
		Flow:    flow,
		Servers: s.servers,
	}

	if err := s.templates.ExecuteTemplate(w, "flow-editor.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// API handlers

// handleGetProjects returns all projects
func (s *Studio) handleGetProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.projects)
}

// handleCreateProject creates a new project
func (s *Studio) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var project Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	project.ID = generateID()
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()
	project.Flows = make(map[string]*Flow)
	project.Servers = make(map[string]*ServerDef)
	project.Variables = make(map[string]interface{})

	if project.Settings == nil {
		project.Settings = &ProjectSettings{
			DefaultTimeout: 30 * time.Second,
			EnableTracing:  true,
			EnableMetrics:  true,
		}
	}

	s.projects[project.ID] = &project
	s.saveProject(&project)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&project)
}

// handleGetProject returns a specific project
func (s *Studio) handleGetProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	project, exists := s.projects[projectID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// handleUpdateProject updates a project
func (s *Studio) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	project, exists := s.projects[projectID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	var updates Project
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update fields
	if updates.Name != "" {
		project.Name = updates.Name
	}
	if updates.Description != "" {
		project.Description = updates.Description
	}
	if updates.Variables != nil {
		project.Variables = updates.Variables
	}
	if updates.Settings != nil {
		project.Settings = updates.Settings
	}

	project.UpdatedAt = time.Now()
	s.saveProject(project)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// handleDeleteProject deletes a project
func (s *Studio) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	if _, exists := s.projects[projectID]; !exists {
		http.NotFound(w, r)
		return
	}

	delete(s.projects, projectID)

	// Remove project file
	projectFile := filepath.Join(s.config.WorkspaceDir, "projects", projectID+".json")
	os.Remove(projectFile)

	w.WriteHeader(http.StatusNoContent)
}

// Background services

// startHealthChecker starts the server health checker
func (s *Studio) startHealthChecker() {
	ticker := time.NewTicker(ServerPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkServerHealth()
		case <-s.ctx.Done():
			return
		}
	}
}

// startAutoSaver starts the auto-saver
func (s *Studio) startAutoSaver() {
	if !s.config.Settings.AutoSave {
		return
	}

	interval := time.Duration(s.config.Settings.AutoSaveInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.saveAllProjects()
		case <-s.ctx.Done():
			return
		}
	}
}

// checkServerHealth checks the health of all servers
func (s *Studio) checkServerHealth() {
	for _, server := range s.servers {
		if server.Connected {
			go s.pingServer(server)
		}
	}
}

// pingServer pings a server to check its health
func (s *Studio) pingServer(server *ServerConnection) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	_, err := server.Client.Ping(ctx, mcp.PingRequest{})

	server.mu.Lock()
	if err != nil {
		server.Connected = false
		log.Printf("Server %s is not responding: %v", server.Name, err)
	} else {
		server.LastPing = time.Now()
	}
	server.mu.Unlock()
}

// saveAllProjects saves all projects to disk
func (s *Studio) saveAllProjects() {
	for _, project := range s.projects {
		s.saveProject(project)
	}
}

// saveProject saves a project to disk
func (s *Studio) saveProject(project *Project) error {
	projectsDir := filepath.Join(s.config.WorkspaceDir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(projectsDir, project.ID+".json")
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// loadProjects loads all projects from disk
func (s *Studio) loadProjects() error {
	projectsDir := filepath.Join(s.config.WorkspaceDir, "projects")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			filename := filepath.Join(projectsDir, entry.Name())
			data, err := os.ReadFile(filename)
			if err != nil {
				continue
			}

			var project Project
			if err := json.Unmarshal(data, &project); err != nil {
				continue
			}

			s.projects[project.ID] = &project
		}
	}

	return nil
}

// Utility functions

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// LoadConfig loads configuration from file
func LoadConfig(filename string) (*Config, error) {
	filename = expandPath(filename)

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(filename string, config *Config) error {
	filename = expandPath(filename)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// Additional handlers for flow operations, server management, etc.
// These are placeholder implementations that would need to be completed

func (s *Studio) handleGetFlows(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting flows
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"flows": []interface{}{}})
}

func (s *Studio) handleCreateFlow(w http.ResponseWriter, r *http.Request) {
	// Implementation for creating flows
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *Studio) handleGetFlow(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting a specific flow
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"flow": nil})
}

func (s *Studio) handleUpdateFlow(w http.ResponseWriter, r *http.Request) {
	// Implementation for updating flows
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *Studio) handleDeleteFlow(w http.ResponseWriter, r *http.Request) {
	// Implementation for deleting flows
	w.WriteHeader(http.StatusNoContent)
}

func (s *Studio) handleRunFlow(w http.ResponseWriter, r *http.Request) {
	// Implementation for running flows
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *Studio) handleRunFlowMessage(client *Client, msg *Message) {
	// Implementation for running flows via WebSocket
}

func (s *Studio) handleUpdateNodeMessage(client *Client, msg *Message) {
	// Implementation for updating nodes via WebSocket
}

func (s *Studio) handleUpdateFlowMessage(client *Client, msg *Message) {
	// Implementation for updating flows via WebSocket
}

func (s *Studio) handleGetServers(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting servers
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.servers)
}

func (s *Studio) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	// Implementation for creating servers
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *Studio) handleGetServer(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting a specific server
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"server": nil})
}

func (s *Studio) handleUpdateServer(w http.ResponseWriter, r *http.Request) {
	// Implementation for updating servers
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *Studio) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	// Implementation for deleting servers
	w.WriteHeader(http.StatusNoContent)
}

func (s *Studio) handleConnectServer(w http.ResponseWriter, r *http.Request) {
	// Implementation for connecting to servers
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *Studio) handleDisconnectServer(w http.ResponseWriter, r *http.Request) {
	// Implementation for disconnecting from servers
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *Studio) handlePingServer(w http.ResponseWriter, r *http.Request) {
	// Implementation for pinging servers
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *Studio) handleGetServerTools(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting server tools
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tools": []interface{}{}})
}

func (s *Studio) handleGetServerResources(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting server resources
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"resources": []interface{}{}})
}

func (s *Studio) handleGetServerPrompts(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting server prompts
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"prompts": []interface{}{}})
}

func (s *Studio) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting settings
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.config.Settings)
}

func (s *Studio) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	// Implementation for updating settings
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *Studio) handleServersPage(w http.ResponseWriter, r *http.Request) {
	// Implementation for servers page
	data := struct {
		Title   string
		Servers map[string]*ServerConnection
	}{
		Title:   "Servers",
		Servers: s.servers,
	}

	if err := s.templates.ExecuteTemplate(w, "servers.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Studio) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	// Implementation for settings page
	data := struct {
		Title    string
		Settings *StudioSettings
	}{
		Title:    "Settings",
		Settings: s.config.Settings,
	}

	if err := s.templates.ExecuteTemplate(w, "settings.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
