// Package main provides mcp-repl, an interactive REPL for MCP servers
// with auto-completion, session management, multi-server support, and command history.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	"github.com/tmc/mcp"
)

const (
	// Version information
	Version = "1.0.0"
	Name    = "mcp-repl"

	// Default config
	DefaultHistoryFile = "~/.mcp-repl-history"
	DefaultConfigFile  = "~/.mcp-repl-config.json"
	MaxHistorySize     = 1000
	DefaultTimeout     = 30 * time.Second

	// ANSI color codes
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorGray    = "\033[37m"
	ColorBold    = "\033[1m"
)

// Config represents the REPL configuration
type Config struct {
	HistoryFile    string                  `json:"history_file"`
	AutoComplete   bool                    `json:"auto_complete"`
	ColorOutput    bool                    `json:"color_output"`
	DefaultTimeout time.Duration           `json:"default_timeout"`
	Servers        map[string]ServerConfig `json:"servers"`
	Aliases        map[string]string       `json:"aliases"`
}

// ServerConfig represents configuration for a server connection
type ServerConfig struct {
	Command     []string `json:"command"`
	Transport   string   `json:"transport"` // "stdio", "http", "sse"
	URL         string   `json:"url,omitempty"`
	Description string   `json:"description,omitempty"`
	AutoConnect bool     `json:"auto_connect"`
}

// Server represents a connected MCP server
type Server struct {
	Name         string
	Config       ServerConfig
	Client       *mcp.Client
	Connected    bool
	Capabilities mcp.ServerCapabilities
	Tools        []mcp.Tool
	Resources    []mcp.Resource
	Prompts      []mcp.Prompt
	LastUsed     time.Time
	mu           sync.RWMutex
}

// REPL represents the main REPL instance
type REPL struct {
	config    *Config
	servers   map[string]*Server
	current   string // current server name
	rl        *readline.Instance
	ctx       context.Context
	cancel    context.CancelFunc
	history   []string
	scripts   map[string]string
	variables map[string]interface{}
	mu        sync.RWMutex
}

// Global flags
var (
	configFile  = flag.String("config", DefaultConfigFile, "Configuration file path")
	historyFile = flag.String("history", DefaultHistoryFile, "History file path")
	noColor     = flag.Bool("no-color", false, "Disable colored output")
	debug       = flag.Bool("debug", false, "Enable debug mode")
	serverCmd   = flag.String("server", "", "Server command to connect to on startup")
	serverURL   = flag.String("url", "", "Server URL (for HTTP/SSE transport)")
	transport   = flag.String("transport", "stdio", "Transport type (stdio, http, sse)")
	interactive = flag.Bool("interactive", true, "Run in interactive mode")
	script      = flag.String("script", "", "Script file to execute")
	version     = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%s v%s\n", Name, Version)
		return
	}

	// Initialize REPL
	repl, err := NewREPL()
	if err != nil {
		log.Fatalf("Failed to initialize REPL: %v", err)
	}
	defer repl.Close()

	// Auto-connect to server if specified
	if *serverCmd != "" {
		serverArgs := strings.Fields(*serverCmd)
		config := ServerConfig{
			Command:   serverArgs,
			Transport: *transport,
			URL:       *serverURL,
		}

		serverName := "default"
		if err := repl.ConnectServer(serverName, config); err != nil {
			log.Fatalf("Failed to connect to server: %v", err)
		}
		repl.current = serverName
		repl.Printf("Connected to server: %s\n", serverName)
	}

	// Execute script if provided
	if *script != "" {
		if err := repl.ExecuteScript(*script); err != nil {
			log.Fatalf("Failed to execute script: %v", err)
		}
		if !*interactive {
			return
		}
	}

	// Start interactive session
	if *interactive {
		repl.Run()
	}
}

// NewREPL creates a new REPL instance
func NewREPL() (*REPL, error) {
	config, err := LoadConfig(*configFile)
	if err != nil {
		// Use default config if file doesn't exist
		config = &Config{
			HistoryFile:    expandPath(*historyFile),
			AutoComplete:   true,
			ColorOutput:    !*noColor,
			DefaultTimeout: DefaultTimeout,
			Servers:        make(map[string]ServerConfig),
			Aliases:        make(map[string]string),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	repl := &REPL{
		config:    config,
		servers:   make(map[string]*Server),
		ctx:       ctx,
		cancel:    cancel,
		scripts:   make(map[string]string),
		variables: make(map[string]interface{}),
	}

	// Initialize readline
	if err := repl.initReadline(); err != nil {
		return nil, fmt.Errorf("failed to initialize readline: %w", err)
	}

	// Load history
	if err := repl.loadHistory(); err != nil && *debug {
		log.Printf("Warning: failed to load history: %v", err)
	}

	// Auto-connect to configured servers
	for name, serverConfig := range config.Servers {
		if serverConfig.AutoConnect {
			if err := repl.ConnectServer(name, serverConfig); err != nil {
				if *debug {
					log.Printf("Warning: failed to auto-connect to %s: %v", name, err)
				}
			}
		}
	}

	return repl, nil
}

// initReadline initializes the readline instance with completions
func (r *REPL) initReadline() error {
	completer := &Completer{repl: r}

	config := &readline.Config{
		Prompt:            r.getPrompt(),
		HistoryFile:       r.config.HistoryFile,
		AutoComplete:      completer,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	}

	var err error
	r.rl, err = readline.NewEx(config)
	if err != nil {
		return err
	}

	return nil
}

// getPrompt returns the current prompt string
func (r *REPL) getPrompt() string {
	if r.config.ColorOutput {
		if r.current != "" {
			return fmt.Sprintf("%s[%s]%s mcp> ", ColorBlue, r.current, ColorReset)
		}
		return fmt.Sprintf("%smcp> %s", ColorGreen, ColorReset)
	}

	if r.current != "" {
		return fmt.Sprintf("[%s] mcp> ", r.current)
	}
	return "mcp> "
}

// Run starts the interactive REPL loop
func (r *REPL) Run() {
	r.Printf("Welcome to %s v%s\n", Name, Version)
	r.Printf("Type 'help' for available commands, 'exit' to quit.\n\n")

	for {
		// Update prompt
		r.rl.SetPrompt(r.getPrompt())

		line, err := r.rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if len(line) == 0 {
					break
				} else {
					continue
				}
			} else if err == io.EOF {
				break
			}
			r.Printf("Error reading input: %v\n", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Add to history
		r.history = append(r.history, line)
		if len(r.history) > MaxHistorySize {
			r.history = r.history[1:]
		}

		// Execute command
		if err := r.ExecuteCommand(line); err != nil {
			r.Printf("Error: %v\n", err)
		}
	}
}

// ExecuteCommand executes a single command
func (r *REPL) ExecuteCommand(line string) error {
	// Parse command and arguments
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	command := parts[0]
	args := parts[1:]

	// Check for aliases
	if alias, exists := r.config.Aliases[command]; exists {
		aliasParts := strings.Fields(alias)
		command = aliasParts[0]
		args = append(aliasParts[1:], args...)
	}

	// Execute command
	switch command {
	case "help", "?":
		return r.showHelp(args)
	case "exit", "quit":
		return errors.New("exit")
	case "connect":
		return r.connectCommand(args)
	case "disconnect":
		return r.disconnectCommand(args)
	case "servers", "list":
		return r.listServers(args)
	case "use":
		return r.useServer(args)
	case "tools":
		return r.listTools(args)
	case "resources":
		return r.listResources(args)
	case "prompts":
		return r.listPrompts(args)
	case "call":
		return r.callTool(args)
	case "read":
		return r.readResource(args)
	case "prompt":
		return r.getPrompt_(args)
	case "ping":
		return r.pingServer(args)
	case "info":
		return r.serverInfo(args)
	case "save":
		return r.saveSession(args)
	case "load":
		return r.loadSession(args)
	case "script":
		return r.runScript(args)
	case "set":
		return r.setVariable(args)
	case "get":
		return r.getVariable(args)
	case "history":
		return r.showHistory(args)
	case "clear":
		return r.clearScreen(args)
	case "config":
		return r.showConfig(args)
	case "alias":
		return r.manageAlias(args)
	default:
		return fmt.Errorf("unknown command: %s (type 'help' for available commands)", command)
	}
}

// connectCommand connects to a server
func (r *REPL) connectCommand(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: connect <name> <command...> [options]")
	}

	name := args[0]
	command := args[1:]

	// Parse options
	transport := "stdio"
	var url string

	// Simple option parsing
	for i := 0; i < len(command); i++ {
		if command[i] == "--transport" && i+1 < len(command) {
			transport = command[i+1]
			command = append(command[:i], command[i+2:]...)
			i -= 2
		} else if command[i] == "--url" && i+1 < len(command) {
			url = command[i+1]
			command = append(command[:i], command[i+2:]...)
			i -= 2
		}
	}

	config := ServerConfig{
		Command:   command,
		Transport: transport,
		URL:       url,
	}

	return r.ConnectServer(name, config)
}

// ConnectServer connects to a server with the given configuration
func (r *REPL) ConnectServer(name string, config ServerConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already connected
	if server, exists := r.servers[name]; exists && server.Connected {
		return fmt.Errorf("server %s is already connected", name)
	}

	// Create transport
	var transport mcp.Transport
	switch config.Transport {
	case "stdio":
		if len(config.Command) == 0 {
			return errors.New("stdio transport requires a command")
		}
		transport = mcp.NewStdioTransport(config.Command[0], config.Command[1:]...)
	case "http":
		if config.URL == "" {
			return errors.New("http transport requires a URL")
		}
		transport = mcp.NewHTTPTransport(config.URL)
	case "sse":
		if config.URL == "" {
			return errors.New("sse transport requires a URL")
		}
		transport = mcp.NewSSETransport(config.URL)
	default:
		return fmt.Errorf("unsupported transport: %s", config.Transport)
	}

	// Create client
	client, err := mcp.NewClient(transport)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Initialize connection
	ctx, cancel := context.WithTimeout(r.ctx, r.config.DefaultTimeout)
	defer cancel()

	initReq := mcp.InitializeRequest{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ClientInfo: mcp.Implementation{
			Name:    Name,
			Version: Version,
		},
	}

	_, err = client.Initialize(ctx, initReq)
	if err != nil {
		return fmt.Errorf("failed to initialize connection: %w", err)
	}

	// Create server instance
	server := &Server{
		Name:      name,
		Config:    config,
		Client:    client,
		Connected: true,
		LastUsed:  time.Now(),
	}

	// Load server capabilities and data
	if err := r.refreshServerData(server); err != nil {
		if *debug {
			log.Printf("Warning: failed to load server data: %v", err)
		}
	}

	r.servers[name] = server
	r.Printf("Connected to server: %s\n", name)

	return nil
}

// refreshServerData loads tools, resources, and prompts from the server
func (r *REPL) refreshServerData(server *Server) error {
	ctx, cancel := context.WithTimeout(r.ctx, r.config.DefaultTimeout)
	defer cancel()

	// Load tools
	if toolsResp, err := server.Client.ListTools(ctx, mcp.ListToolsRequest{}); err == nil {
		server.Tools = toolsResp.Tools
	}

	// Load resources
	if resourcesResp, err := server.Client.ListResources(ctx, mcp.ListResourcesRequest{}); err == nil {
		server.Resources = resourcesResp.Resources
	}

	// Load prompts
	if promptsResp, err := server.Client.ListPrompts(ctx, mcp.ListPromptsRequest{}); err == nil {
		server.Prompts = promptsResp.Prompts
	}

	return nil
}

// Completer provides auto-completion for the REPL
type Completer struct {
	repl *REPL
}

// Do implements the readline.AutoCompleter interface
func (c *Completer) Do(line []rune, pos int) ([][]rune, int, int) {
	lineStr := string(line)
	parts := strings.Fields(lineStr)

	if len(parts) == 0 || (len(parts) == 1 && pos == len(line)) {
		// Complete command names
		commands := []string{
			"help", "exit", "quit", "connect", "disconnect", "servers", "list",
			"use", "tools", "resources", "prompts", "call", "read", "prompt",
			"ping", "info", "save", "load", "script", "set", "get", "history",
			"clear", "config", "alias",
		}

		var completions [][]rune
		prefix := ""
		if len(parts) > 0 {
			prefix = parts[0]
		}

		for _, cmd := range commands {
			if strings.HasPrefix(cmd, prefix) {
				completions = append(completions, []rune(cmd))
			}
		}

		return completions, 0, pos
	}

	// Command-specific completions
	command := parts[0]
	switch command {
	case "use", "disconnect", "info", "ping":
		return c.completeServerNames(parts, pos)
	case "call":
		return c.completeToolNames(parts, pos)
	case "read":
		return c.completeResourceNames(parts, pos)
	case "prompt":
		return c.completePromptNames(parts, pos)
	}

	return nil, 0, 0
}

// completeServerNames provides completion for server names
func (c *Completer) completeServerNames(parts []string, pos int) ([][]rune, int, int) {
	if len(parts) < 2 {
		return nil, 0, 0
	}

	prefix := parts[1]
	var completions [][]rune

	c.repl.mu.RLock()
	for name := range c.repl.servers {
		if strings.HasPrefix(name, prefix) {
			completions = append(completions, []rune(name))
		}
	}
	c.repl.mu.RUnlock()

	return completions, 0, pos
}

// completeToolNames provides completion for tool names
func (c *Completer) completeToolNames(parts []string, pos int) ([][]rune, int, int) {
	if len(parts) < 2 {
		return nil, 0, 0
	}

	prefix := parts[1]
	var completions [][]rune

	c.repl.mu.RLock()
	if c.repl.current != "" {
		if server, exists := c.repl.servers[c.repl.current]; exists {
			for _, tool := range server.Tools {
				if strings.HasPrefix(tool.Name, prefix) {
					completions = append(completions, []rune(tool.Name))
				}
			}
		}
	}
	c.repl.mu.RUnlock()

	return completions, 0, pos
}

// completeResourceNames provides completion for resource names
func (c *Completer) completeResourceNames(parts []string, pos int) ([][]rune, int, int) {
	if len(parts) < 2 {
		return nil, 0, 0
	}

	prefix := parts[1]
	var completions [][]rune

	c.repl.mu.RLock()
	if c.repl.current != "" {
		if server, exists := c.repl.servers[c.repl.current]; exists {
			for _, resource := range server.Resources {
				if strings.HasPrefix(resource.URI, prefix) {
					completions = append(completions, []rune(resource.URI))
				}
			}
		}
	}
	c.repl.mu.RUnlock()

	return completions, 0, pos
}

// completePromptNames provides completion for prompt names
func (c *Completer) completePromptNames(parts []string, pos int) ([][]rune, int, int) {
	if len(parts) < 2 {
		return nil, 0, 0
	}

	prefix := parts[1]
	var completions [][]rune

	c.repl.mu.RLock()
	if c.repl.current != "" {
		if server, exists := c.repl.servers[c.repl.current]; exists {
			for _, prompt := range server.Prompts {
				if strings.HasPrefix(prompt.Name, prefix) {
					completions = append(completions, []rune(prompt.Name))
				}
			}
		}
	}
	c.repl.mu.RUnlock()

	return completions, 0, pos
}

// Printf prints formatted output with optional color support
func (r *REPL) Printf(format string, args ...interface{}) {
	if r.rl != nil {
		fmt.Fprintf(r.rl.Stderr(), format, args...)
	} else {
		fmt.Printf(format, args...)
	}
}

// Command implementations

// showHelp displays help information
func (r *REPL) showHelp(args []string) error {
	if len(args) > 0 {
		// Show help for specific command
		return r.showCommandHelp(args[0])
	}

	r.Printf("Available commands:\n\n")
	r.Printf("  Connection Management:\n")
	r.Printf("    connect <name> <cmd...>  - Connect to a server\n")
	r.Printf("    disconnect <name>        - Disconnect from a server\n")
	r.Printf("    servers                  - List connected servers\n")
	r.Printf("    use <name>               - Switch to a server\n")
	r.Printf("    ping [name]              - Ping a server\n")
	r.Printf("    info [name]              - Show server information\n")
	r.Printf("\n")
	r.Printf("  Server Operations:\n")
	r.Printf("    tools                    - List available tools\n")
	r.Printf("    resources                - List available resources\n")
	r.Printf("    prompts                  - List available prompts\n")
	r.Printf("    call <tool> [args...]    - Call a tool\n")
	r.Printf("    read <resource>          - Read a resource\n")
	r.Printf("    prompt <name> [args...]  - Get a prompt\n")
	r.Printf("\n")
	r.Printf("  Session Management:\n")
	r.Printf("    save <file>              - Save session to file\n")
	r.Printf("    load <file>              - Load session from file\n")
	r.Printf("    script <file>            - Execute script file\n")
	r.Printf("    history                  - Show command history\n")
	r.Printf("\n")
	r.Printf("  Variables and Configuration:\n")
	r.Printf("    set <var> <value>        - Set a variable\n")
	r.Printf("    get <var>                - Get a variable\n")
	r.Printf("    config                   - Show configuration\n")
	r.Printf("    alias <name> <command>   - Create command alias\n")
	r.Printf("\n")
	r.Printf("  Utility:\n")
	r.Printf("    clear                    - Clear screen\n")
	r.Printf("    help [command]           - Show help\n")
	r.Printf("    exit                     - Exit REPL\n")
	r.Printf("\n")
	r.Printf("Use 'help <command>' for detailed help on a specific command.\n")

	return nil
}

// showCommandHelp displays help for a specific command
func (r *REPL) showCommandHelp(command string) error {
	switch command {
	case "connect":
		r.Printf("connect <name> <command...> [options]\n\n")
		r.Printf("Connect to an MCP server.\n\n")
		r.Printf("Arguments:\n")
		r.Printf("  name      - Name for the server connection\n")
		r.Printf("  command   - Command to execute the server\n\n")
		r.Printf("Options:\n")
		r.Printf("  --transport <type>  - Transport type (stdio, http, sse)\n")
		r.Printf("  --url <url>         - URL for HTTP/SSE transport\n\n")
		r.Printf("Examples:\n")
		r.Printf("  connect myserver go run ./examples/servers/mcp-time-server\n")
		r.Printf("  connect httpserver --transport http --url http://localhost:8080\n")
	case "call":
		r.Printf("call <tool> [args...]\n\n")
		r.Printf("Call a tool on the current server.\n\n")
		r.Printf("Arguments:\n")
		r.Printf("  tool  - Name of the tool to call\n")
		r.Printf("  args  - Arguments to pass (key=value format)\n\n")
		r.Printf("Example:\n")
		r.Printf("  call get_time\n")
		r.Printf("  call calculate a=5 b=3 operation=add\n")
	default:
		return fmt.Errorf("no help available for command: %s", command)
	}
	return nil
}

// disconnectCommand disconnects from a server
func (r *REPL) disconnectCommand(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: disconnect <name>")
	}

	name := args[0]

	r.mu.Lock()
	server, exists := r.servers[name]
	if !exists {
		r.mu.Unlock()
		return fmt.Errorf("server %s not found", name)
	}

	if server.Connected {
		server.Client.Close()
		server.Connected = false
	}

	delete(r.servers, name)
	if r.current == name {
		r.current = ""
	}
	r.mu.Unlock()

	r.Printf("Disconnected from server: %s\n", name)
	return nil
}

// listServers lists all connected servers
func (r *REPL) listServers(args []string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.servers) == 0 {
		r.Printf("No servers connected.\n")
		return nil
	}

	r.Printf("Connected servers:\n")
	for name, server := range r.servers {
		status := "connected"
		if !server.Connected {
			status = "disconnected"
		}

		current := ""
		if name == r.current {
			current = " (current)"
		}

		r.Printf("  %s - %s%s\n", name, status, current)
		if server.Config.Description != "" {
			r.Printf("    Description: %s\n", server.Config.Description)
		}
		r.Printf("    Transport: %s\n", server.Config.Transport)
		r.Printf("    Tools: %d, Resources: %d, Prompts: %d\n",
			len(server.Tools), len(server.Resources), len(server.Prompts))
	}

	return nil
}

// useServer switches to a different server
func (r *REPL) useServer(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: use <name>")
	}

	name := args[0]

	r.mu.Lock()
	server, exists := r.servers[name]
	if !exists {
		r.mu.Unlock()
		return fmt.Errorf("server %s not found", name)
	}

	if !server.Connected {
		r.mu.Unlock()
		return fmt.Errorf("server %s is not connected", name)
	}

	r.current = name
	server.LastUsed = time.Now()
	r.mu.Unlock()

	r.Printf("Switched to server: %s\n", name)
	return nil
}

// listTools lists available tools
func (r *REPL) listTools(args []string) error {
	server := r.getCurrentServer()
	if server == nil {
		return errors.New("no server selected")
	}

	if len(server.Tools) == 0 {
		r.Printf("No tools available.\n")
		return nil
	}

	r.Printf("Available tools:\n")
	for _, tool := range server.Tools {
		r.Printf("  %s - %s\n", tool.Name, tool.Description)
		if tool.InputSchema != nil {
			r.Printf("    Input schema: %s\n", string(tool.InputSchema))
		}
	}

	return nil
}

// listResources lists available resources
func (r *REPL) listResources(args []string) error {
	server := r.getCurrentServer()
	if server == nil {
		return errors.New("no server selected")
	}

	if len(server.Resources) == 0 {
		r.Printf("No resources available.\n")
		return nil
	}

	r.Printf("Available resources:\n")
	for _, resource := range server.Resources {
		r.Printf("  %s - %s\n", resource.URI, resource.Name)
		if resource.Description != "" {
			r.Printf("    Description: %s\n", resource.Description)
		}
		if resource.MimeType != "" {
			r.Printf("    MIME type: %s\n", resource.MimeType)
		}
	}

	return nil
}

// listPrompts lists available prompts
func (r *REPL) listPrompts(args []string) error {
	server := r.getCurrentServer()
	if server == nil {
		return errors.New("no server selected")
	}

	if len(server.Prompts) == 0 {
		r.Printf("No prompts available.\n")
		return nil
	}

	r.Printf("Available prompts:\n")
	for _, prompt := range server.Prompts {
		r.Printf("  %s - %s\n", prompt.Name, prompt.Description)
		if len(prompt.Arguments) > 0 {
			r.Printf("    Arguments: %v\n", prompt.Arguments)
		}
	}

	return nil
}

// callTool calls a tool with arguments
func (r *REPL) callTool(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: call <tool> [args...]")
	}

	server := r.getCurrentServer()
	if server == nil {
		return errors.New("no server selected")
	}

	toolName := args[0]
	toolArgs := make(map[string]interface{})

	// Parse arguments in key=value format
	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid argument format: %s (expected key=value)", arg)
		}

		key := parts[0]
		value := parts[1]

		// Try to parse as number or boolean
		if v, err := strconv.Atoi(value); err == nil {
			toolArgs[key] = v
		} else if v, err := strconv.ParseFloat(value, 64); err == nil {
			toolArgs[key] = v
		} else if v, err := strconv.ParseBool(value); err == nil {
			toolArgs[key] = v
		} else {
			toolArgs[key] = value
		}
	}

	ctx, cancel := context.WithTimeout(r.ctx, r.config.DefaultTimeout)
	defer cancel()

	request := mcp.CallToolRequest{
		Name:      toolName,
		Arguments: toolArgs,
	}

	result, err := server.Client.CallTool(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to call tool: %w", err)
	}

	r.Printf("Tool call result:\n")
	r.printContent(result.Content)

	return nil
}

// readResource reads a resource
func (r *REPL) readResource(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: read <resource>")
	}

	server := r.getCurrentServer()
	if server == nil {
		return errors.New("no server selected")
	}

	resourceURI := args[0]

	ctx, cancel := context.WithTimeout(r.ctx, r.config.DefaultTimeout)
	defer cancel()

	request := mcp.ReadResourceRequest{
		URI: resourceURI,
	}

	contents, err := server.Client.ReadResource(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to read resource: %w", err)
	}

	r.Printf("Resource contents:\n")
	for _, content := range contents {
		r.Printf("URI: %s\n", content.URI)
		if content.MimeType != "" {
			r.Printf("MIME type: %s\n", content.MimeType)
		}
		r.Printf("Content: %s\n", string(content.Text))
		r.Printf("---\n")
	}

	return nil
}

// getPrompt_ gets a prompt (underscore to avoid conflict with getPrompt method)
func (r *REPL) getPrompt_(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: prompt <name> [args...]")
	}

	server := r.getCurrentServer()
	if server == nil {
		return errors.New("no server selected")
	}

	promptName := args[0]
	promptArgs := make(map[string]interface{})

	// Parse arguments in key=value format
	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid argument format: %s (expected key=value)", arg)
		}

		key := parts[0]
		value := parts[1]

		// Try to parse as number or boolean
		if v, err := strconv.Atoi(value); err == nil {
			promptArgs[key] = v
		} else if v, err := strconv.ParseFloat(value, 64); err == nil {
			promptArgs[key] = v
		} else if v, err := strconv.ParseBool(value); err == nil {
			promptArgs[key] = v
		} else {
			promptArgs[key] = value
		}
	}

	ctx, cancel := context.WithTimeout(r.ctx, r.config.DefaultTimeout)
	defer cancel()

	request := mcp.GetPromptRequest{
		Name:      promptName,
		Arguments: promptArgs,
	}

	result, err := server.Client.GetPrompt(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	r.Printf("Prompt result:\n")
	r.Printf("Description: %s\n", result.Description)
	r.Printf("Messages:\n")
	for _, message := range result.Messages {
		r.Printf("  Role: %s\n", message.Role)
		r.printContent(message.Content)
	}

	return nil
}

// pingServer pings a server
func (r *REPL) pingServer(args []string) error {
	var server *Server

	if len(args) == 0 {
		server = r.getCurrentServer()
		if server == nil {
			return errors.New("no server selected")
		}
	} else {
		r.mu.RLock()
		var exists bool
		server, exists = r.servers[args[0]]
		r.mu.RUnlock()

		if !exists {
			return fmt.Errorf("server %s not found", args[0])
		}
	}

	ctx, cancel := context.WithTimeout(r.ctx, r.config.DefaultTimeout)
	defer cancel()

	start := time.Now()
	_, err := server.Client.Ping(ctx, mcp.PingRequest{})
	duration := time.Since(start)

	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	r.Printf("Ping successful: %v\n", duration)
	return nil
}

// serverInfo shows server information
func (r *REPL) serverInfo(args []string) error {
	var server *Server

	if len(args) == 0 {
		server = r.getCurrentServer()
		if server == nil {
			return errors.New("no server selected")
		}
	} else {
		r.mu.RLock()
		var exists bool
		server, exists = r.servers[args[0]]
		r.mu.RUnlock()

		if !exists {
			return fmt.Errorf("server %s not found", args[0])
		}
	}

	r.Printf("Server: %s\n", server.Name)
	r.Printf("Status: %s\n", map[bool]string{true: "connected", false: "disconnected"}[server.Connected])
	r.Printf("Transport: %s\n", server.Config.Transport)
	if server.Config.URL != "" {
		r.Printf("URL: %s\n", server.Config.URL)
	}
	if server.Config.Description != "" {
		r.Printf("Description: %s\n", server.Config.Description)
	}
	r.Printf("Command: %v\n", server.Config.Command)
	r.Printf("Last used: %v\n", server.LastUsed.Format(time.RFC3339))
	r.Printf("Tools: %d\n", len(server.Tools))
	r.Printf("Resources: %d\n", len(server.Resources))
	r.Printf("Prompts: %d\n", len(server.Prompts))

	return nil
}

// saveSession saves the current session to a file
func (r *REPL) saveSession(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: save <file>")
	}

	filename := expandPath(args[0])

	session := SessionData{
		Timestamp: time.Now(),
		Current:   r.current,
		Servers:   make(map[string]ServerConfig),
		Variables: r.variables,
		History:   r.history,
	}

	r.mu.RLock()
	for name, server := range r.servers {
		session.Servers[name] = server.Config
	}
	r.mu.RUnlock()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	r.Printf("Session saved to: %s\n", filename)
	return nil
}

// loadSession loads a session from a file
func (r *REPL) loadSession(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: load <file>")
	}

	filename := expandPath(args[0])

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read session file: %w", err)
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Disconnect existing servers
	r.mu.Lock()
	for name, server := range r.servers {
		if server.Connected {
			server.Client.Close()
		}
	}
	r.servers = make(map[string]*Server)
	r.mu.Unlock()

	// Reconnect servers
	for name, config := range session.Servers {
		if err := r.ConnectServer(name, config); err != nil {
			r.Printf("Warning: failed to reconnect to %s: %v\n", name, err)
		}
	}

	// Restore state
	r.current = session.Current
	r.variables = session.Variables
	r.history = session.History

	r.Printf("Session loaded from: %s\n", filename)
	return nil
}

// runScript executes a script file
func (r *REPL) runScript(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: script <file>")
	}

	filename := expandPath(args[0])
	return r.ExecuteScript(filename)
}

// ExecuteScript executes a script file
func (r *REPL) ExecuteScript(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		r.Printf("Executing: %s\n", line)
		if err := r.ExecuteCommand(line); err != nil {
			return fmt.Errorf("script error at line %d: %w", i+1, err)
		}
	}

	return nil
}

// setVariable sets a variable
func (r *REPL) setVariable(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: set <name> <value>")
	}

	name := args[0]
	value := args[1]

	// Try to parse as number or boolean
	if v, err := strconv.Atoi(value); err == nil {
		r.variables[name] = v
	} else if v, err := strconv.ParseFloat(value, 64); err == nil {
		r.variables[name] = v
	} else if v, err := strconv.ParseBool(value); err == nil {
		r.variables[name] = v
	} else {
		r.variables[name] = value
	}

	r.Printf("Variable set: %s = %v\n", name, r.variables[name])
	return nil
}

// getVariable gets a variable
func (r *REPL) getVariable(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: get <name>")
	}

	name := args[0]
	value, exists := r.variables[name]
	if !exists {
		return fmt.Errorf("variable %s not found", name)
	}

	r.Printf("%s = %v\n", name, value)
	return nil
}

// showHistory shows command history
func (r *REPL) showHistory(args []string) error {
	if len(r.history) == 0 {
		r.Printf("No command history.\n")
		return nil
	}

	r.Printf("Command history:\n")
	for i, cmd := range r.history {
		r.Printf("%4d: %s\n", i+1, cmd)
	}

	return nil
}

// clearScreen clears the screen
func (r *REPL) clearScreen(args []string) error {
	if r.config.ColorOutput {
		r.Printf("\033[2J\033[H")
	} else {
		// Try to use the clear command
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	return nil
}

// showConfig shows current configuration
func (r *REPL) showConfig(args []string) error {
	r.Printf("Configuration:\n")
	r.Printf("  History file: %s\n", r.config.HistoryFile)
	r.Printf("  Auto-complete: %t\n", r.config.AutoComplete)
	r.Printf("  Color output: %t\n", r.config.ColorOutput)
	r.Printf("  Default timeout: %v\n", r.config.DefaultTimeout)
	r.Printf("  Configured servers: %d\n", len(r.config.Servers))
	r.Printf("  Aliases: %d\n", len(r.config.Aliases))

	if len(r.config.Aliases) > 0 {
		r.Printf("  Aliases:\n")
		for name, command := range r.config.Aliases {
			r.Printf("    %s = %s\n", name, command)
		}
	}

	return nil
}

// manageAlias manages command aliases
func (r *REPL) manageAlias(args []string) error {
	if len(args) == 0 {
		// List aliases
		if len(r.config.Aliases) == 0 {
			r.Printf("No aliases defined.\n")
			return nil
		}

		r.Printf("Aliases:\n")
		for name, command := range r.config.Aliases {
			r.Printf("  %s = %s\n", name, command)
		}
		return nil
	}

	if len(args) == 1 {
		// Remove alias
		name := args[0]
		if _, exists := r.config.Aliases[name]; !exists {
			return fmt.Errorf("alias %s not found", name)
		}
		delete(r.config.Aliases, name)
		r.Printf("Alias removed: %s\n", name)
		return nil
	}

	if len(args) >= 2 {
		// Set alias
		name := args[0]
		command := strings.Join(args[1:], " ")
		r.config.Aliases[name] = command
		r.Printf("Alias set: %s = %s\n", name, command)
		return nil
	}

	return errors.New("usage: alias [name] [command...]")
}

// getCurrentServer returns the current server
func (r *REPL) getCurrentServer() *Server {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.current == "" {
		return nil
	}

	return r.servers[r.current]
}

// printContent prints content with proper formatting
func (r *REPL) printContent(content []interface{}) {
	for _, item := range content {
		switch v := item.(type) {
		case map[string]interface{}:
			if contentType, ok := v["type"].(string); ok {
				switch contentType {
				case "text":
					if text, ok := v["text"].(string); ok {
						r.Printf("%s\n", text)
					}
				case "image":
					if data, ok := v["data"].(string); ok {
						r.Printf("[Image: %s]\n", data[:min(50, len(data))])
					}
				case "resource":
					if uri, ok := v["uri"].(string); ok {
						r.Printf("[Resource: %s]\n", uri)
					}
				default:
					r.Printf("Unknown content type: %s\n", contentType)
				}
			}
		case string:
			r.Printf("%s\n", v)
		default:
			r.Printf("%v\n", v)
		}
	}
}

// loadHistory loads command history from file
func (r *REPL) loadHistory() error {
	filename := r.config.HistoryFile
	if filename == "" {
		return nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			r.history = append(r.history, line)
		}
	}

	// Limit history size
	if len(r.history) > MaxHistorySize {
		r.history = r.history[len(r.history)-MaxHistorySize:]
	}

	return nil
}

// saveHistory saves command history to file
func (r *REPL) saveHistory() error {
	filename := r.config.HistoryFile
	if filename == "" {
		return nil
	}

	data := strings.Join(r.history, "\n")
	return os.WriteFile(filename, []byte(data), 0644)
}

// Close closes the REPL and cleans up resources
func (r *REPL) Close() error {
	// Save history
	if err := r.saveHistory(); err != nil && *debug {
		log.Printf("Warning: failed to save history: %v", err)
	}

	// Close readline
	if r.rl != nil {
		r.rl.Close()
	}

	// Disconnect servers
	r.mu.Lock()
	for _, server := range r.servers {
		if server.Connected {
			server.Client.Close()
		}
	}
	r.mu.Unlock()

	// Cancel context
	if r.cancel != nil {
		r.cancel()
	}

	return nil
}

// SessionData represents a saved session
type SessionData struct {
	Timestamp time.Time               `json:"timestamp"`
	Current   string                  `json:"current"`
	Servers   map[string]ServerConfig `json:"servers"`
	Variables map[string]interface{}  `json:"variables"`
	History   []string                `json:"history"`
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

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
