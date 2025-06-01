// Package config handles configuration for mcpd.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ServerMode represents the server process lifecycle mode
type ServerMode string

const (
	// ModeSingle starts one server process for all connections
	ModeSingle ServerMode = "once"
	
	// ModePerConnection starts a new server process for each connection
	ModePerConnection ServerMode = "per-connection"
)

// Config holds the daemon configuration
type Config struct {
	// Server command
	ServerCommand string
	ServerArgs    []string

	// Network options
	ListenAddr    string
	SocketPath    string
	TCPAddr       string

	// Streaming options
	HTTPAddr      string
	EnableSSE     bool
	EnableWS      bool
	EnableStream  bool
	StreamTimeout time.Duration

	// Process management
	Mode          ServerMode
	Timeout       time.Duration

	// Logging options
	TraceFile     string
	ServerLogFile string
	PidFile       string
	ServerPidFile string
	Verbose       bool

	// Interactive mode
	Interactive   bool
	NoTTYPrompt   bool
}

// New creates a new default configuration
func New() *Config {
	return &Config{
		Mode:      ModeSingle,
		Verbose:   false,
	}
}

// SetServerCommand sets the server command and arguments
func (c *Config) SetServerCommand(cmd string, args []string) *Config {
	c.ServerCommand = cmd
	c.ServerArgs = args
	return c
}

// SetMode sets the server mode
func (c *Config) SetMode(mode string) error {
	switch mode {
	case "once":
		c.Mode = ModeSingle
	case "per-connection":
		c.Mode = ModePerConnection
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
	return nil
}

// SetListenAddr sets the listen address from either TCP or Unix socket
func (c *Config) SetListenAddr(tcpAddr, socketPath string) error {
	if tcpAddr != "" && socketPath != "" {
		return errors.New("cannot specify both TCP address and Unix socket path")
	}
	
	if tcpAddr != "" {
		// Use TCP
		c.TCPAddr = tcpAddr
		c.ListenAddr = "tcp://" + tcpAddr
		return nil
	}
	
	// Use Unix socket
	if socketPath == "" {
		// Generate a default socket path
		tmpDir := os.TempDir()
		socketPath = filepath.Join(tmpDir, fmt.Sprintf("mcpd-%d.sock", os.Getpid()))
	}
	
	c.SocketPath = socketPath
	c.ListenAddr = "unix://" + socketPath
	return nil
}

// SetTimeout sets the auto-termination timeout
func (c *Config) SetTimeout(timeout time.Duration) *Config {
	c.Timeout = timeout
	return c
}

// SetTraceFile sets the trace file path
func (c *Config) SetTraceFile(path string) *Config {
	c.TraceFile = path
	return c
}

// SetServerLogFile sets the server log file path
func (c *Config) SetServerLogFile(path string) *Config {
	c.ServerLogFile = path
	return c
}

// SetPidFile sets the PID file path
func (c *Config) SetPidFile(path string) *Config {
	c.PidFile = path
	return c
}

// SetServerPidFile sets the server PID file path
func (c *Config) SetServerPidFile(path string) *Config {
	c.ServerPidFile = path
	return c
}

// SetVerbose sets the verbose logging flag
func (c *Config) SetVerbose(verbose bool) *Config {
	c.Verbose = verbose
	return c
}

// SetInteractive sets the interactive mode flag
func (c *Config) SetInteractive(interactive bool) *Config {
	c.Interactive = interactive
	return c
}

// SetNoTTYPrompt sets the flag to disable TTY prompting
func (c *Config) SetNoTTYPrompt(noTTYPrompt bool) *Config {
	c.NoTTYPrompt = noTTYPrompt
	return c
}

// SetHTTPAddr sets the HTTP server address for streaming
func (c *Config) SetHTTPAddr(addr string) *Config {
	c.HTTPAddr = addr
	return c
}

// SetStreamingOptions sets the streaming options
func (c *Config) SetStreamingOptions(enableSSE, enableWS, enableStream bool) *Config {
	c.EnableSSE = enableSSE
	c.EnableWS = enableWS
	c.EnableStream = enableStream
	return c
}

// SetStreamTimeout sets the timeout for streaming connections
func (c *Config) SetStreamTimeout(timeout time.Duration) *Config {
	c.StreamTimeout = timeout
	return c
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Check server command
	if c.ServerCommand == "" {
		return errors.New("server command is required")
	}

	// Validate listen address
	if c.ListenAddr == "" {
		return errors.New("listen address is required")
	}

	// Validate mode
	switch c.Mode {
	case ModeSingle, ModePerConnection:
		// Valid modes
	default:
		return fmt.Errorf("invalid mode: %s", c.Mode)
	}

	// Validate streaming options
	if c.EnableSSE || c.EnableWS || c.EnableStream {
		if c.HTTPAddr == "" {
			return errors.New("HTTP address is required when streaming is enabled")
		}
	}

	// Set default stream timeout if not specified
	if (c.EnableSSE || c.EnableWS || c.EnableStream) && c.StreamTimeout == 0 {
		c.StreamTimeout = 1 * time.Hour // Default to 1 hour
	}

	return nil
}

// GetSocketPathForDiscovery returns the socket path for discovery
// It ensures it's an absolute path for reliable access
func (c *Config) GetSocketPathForDiscovery() (string, error) {
	if !strings.HasPrefix(c.ListenAddr, "unix://") {
		return c.ListenAddr, nil
	}
	
	// Extract path from unix:// prefix
	path := strings.TrimPrefix(c.ListenAddr, "unix://")
	
	// Convert to absolute path if not already
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		return "unix://" + absPath, nil
	}
	
	return c.ListenAddr, nil
}