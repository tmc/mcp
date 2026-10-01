// Package daemon implements the mcpd daemon.
package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/tmc/mcp/exp/cmd/mcpd/config"
	"github.com/tmc/mcp/exp/cmd/mcpd/manager"
	"github.com/tmc/mcp/exp/cmd/mcpd/transport"
)

// Daemon is the main daemon that manages server processes and connections
type Daemon struct {
	Config        *config.Config
	ServerManager *manager.ServerManager
	Listener      *transport.Listener

	// Optional streaming transport
	StreamTransport *transport.StreamingTransport

	traceFile             *os.File
	wg                    sync.WaitGroup
	nextConnID            int
	mu                    sync.Mutex
	promptHandlerCallback func(handler *transport.InteractivePromptHandler)
}

// New creates a new daemon with the given configuration
func New(cfg *config.Config) (*Daemon, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	d := &Daemon{
		Config: cfg,
	}

	return d, nil
}

// WithPromptHandlerCallback sets a callback for when the prompt handler is created
func (d *Daemon) WithPromptHandlerCallback(callback func(handler *transport.InteractivePromptHandler)) *Daemon {
	d.promptHandlerCallback = callback
	return d
}

// Start starts the daemon
func (d *Daemon) Start(ctx context.Context) error {
	// Set up tracing if requested
	if d.Config.TraceFile != "" {
		if err := d.setupTracing(); err != nil {
			return fmt.Errorf("failed to set up tracing: %w", err)
		}
	}

	// Set up server manager
	d.ServerManager = manager.New(
		d.Config.ServerCommand,
		d.Config.ServerArgs,
		manager.ProcessMode(d.Config.Mode),
	)

	// Configure server manager
	d.ServerManager.WithPidFile(d.Config.ServerPidFile)
	d.ServerManager.WithLogFile(d.Config.ServerLogFile)

	// Create listener
	var err error
	d.Listener, err = transport.NewListener(d.Config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Set up trace logger if tracing is enabled
	if d.traceFile != nil {
		traceLogger := transport.NewTraceLogger(d.traceFile)
		d.Listener.WithTraceLogger(traceLogger)
	}

	// Set up interactive prompt handler if enabled
	if d.Config.Interactive {
		promptHandler := transport.NewInteractivePromptHandler(!d.Config.NoTTYPrompt)
		d.Listener.WithPromptHandler(promptHandler)

		// Invoke callback if registered
		if d.promptHandlerCallback != nil {
			d.promptHandlerCallback(promptHandler)
		}

		if d.Config.Verbose {
			if promptHandler.IsInteractive() {
				slog.Info("Interactive prompt handling enabled")
			} else {
				slog.Info("Interactive mode enabled but no TTY available for prompting")
			}
		}
	}

	// Set up streaming transport if HTTP address is provided
	if d.Config.HTTPAddr != "" {
		// Create streaming transport using the base listener
		d.StreamTransport = transport.NewStreamingTransport(d.Listener, d.Config.HTTPAddr)

		// Enable requested streaming features
		if d.Config.EnableSSE {
			d.StreamTransport.EnableSSE()
		}

		if d.Config.EnableWS {
			d.StreamTransport.EnableWebSockets()
		}

		if d.Config.EnableStream {
			d.StreamTransport.EnableHTTPStreaming()
		}

		// Set timeout for streaming connections
		if d.Config.StreamTimeout > 0 {
			d.StreamTransport.SetStreamTimeout(d.Config.StreamTimeout)
		}

		// Start the streaming transport
		if err := d.StreamTransport.Start(ctx); err != nil {
			return fmt.Errorf("failed to start streaming transport: %w", err)
		}

		slog.Info("Streaming enabled",
			"http_addr", d.Config.HTTPAddr,
			"sse", d.Config.EnableSSE,
			"websocket", d.Config.EnableWS,
			"http_stream", d.Config.EnableStream,
		)
	} else {
		// Start listening on the regular listener
		if err := d.Listener.Listen(); err != nil {
			return fmt.Errorf("failed to start listener: %w", err)
		}
	}

	slog.Info("Daemon started",
		"listen_addr", d.Config.ListenAddr,
		"server_command", d.Config.ServerCommand,
		"mode", d.Config.Mode,
		"interactive", d.Config.Interactive,
	)

	// Print listener address to stdout for discovery
	discoveryAddr, err := d.Config.GetSocketPathForDiscovery()
	if err != nil {
		slog.Error("Failed to get discovery address", "error", err)
	} else {
		fmt.Println(discoveryAddr)
	}

	// Auto-terminate for testing if timeout is set
	if d.Config.Timeout > 0 {
		go func() {
			slog.Info("Auto-termination enabled", "timeout", d.Config.Timeout)
			time.Sleep(d.Config.Timeout)
			slog.Info("Auto-terminating due to timeout")
			d.Stop()
		}()
	}

	// Start accepting connections
	return d.acceptConnections(ctx)
}

// setupTracing sets up the trace file
func (d *Daemon) setupTracing() error {
	// Ensure directory exists
	traceDir := filepath.Dir(d.Config.TraceFile)
	if err := os.MkdirAll(traceDir, 0755); err != nil {
		return fmt.Errorf("failed to create trace directory: %w", err)
	}

	// Open trace file
	var err error
	d.traceFile, err = os.OpenFile(d.Config.TraceFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open trace file: %w", err)
	}

	// Write trace header
	if _, err := d.traceFile.WriteString("# mcptrace:v1\n"); err != nil {
		return fmt.Errorf("failed to write trace header: %w", err)
	}

	slog.Info("Tracing enabled", "file", d.Config.TraceFile)
	return nil
}

// acceptConnections accepts and handles incoming connections
func (d *Daemon) acceptConnections(ctx context.Context) error {
	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create a single server instance if in Single mode, but don't start it yet
	if d.Config.Mode == config.ModeSingle {
		slog.Info("Creating single server instance for all connections")

		// Generate a connection ID
		connID := "shared"

		// Create the server entry but don't start it yet - it will start on first connection
		cmd, err := d.ServerManager.StartServer(ctx, connID)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		// Set up monitor for when the process starts
		go func() {
			// The server might not be started yet, so wait until it is
			for cmd.Process == nil {
				select {
				case <-ctx.Done():
					return
				case <-time.After(100 * time.Millisecond):
					// Keep checking
				}
			}

			// Now we can wait for it to finish
			err := cmd.Wait()
			if err != nil {
				slog.Error("Server process exited with error", "error", err)
			} else {
				slog.Info("Server process exited cleanly")
			}

			// Cancel the context to trigger shutdown if the context is still valid
			if ctx.Err() == nil {
				cancel()
			}
		}()
	}

	// Accept connections in a loop
	for {
		// Check if context is canceled
		select {
		case <-ctx.Done():
			slog.Info("Context canceled, shutting down daemon")
			return nil
		default:
			// Continue accepting
		}

		// Accept a connection
		conn, err := d.Listener.Accept()
		if err != nil {
			// Check if listener was closed
			if os.IsTimeout(err) || ctx.Err() != nil {
				return nil
			}

			slog.Error("Failed to accept connection", "error", err)
			continue
		}

		// Generate a unique connection ID
		d.mu.Lock()
		d.nextConnID++
		connID := fmt.Sprintf("conn-%d", d.nextConnID)
		d.mu.Unlock()

		// Start a server for this connection if in PerConnection mode
		var cmd *exec.Cmd
		if d.Config.Mode == config.ModePerConnection {
			slog.Info("Starting new server instance for connection", "id", connID)

			// Start server process
			var err error
			cmd, err = d.ServerManager.StartServer(ctx, connID)
			if err != nil {
				slog.Error("Failed to start server for connection", "error", err)
				conn.Close()
				continue
			}
		} else {
			// In Single mode, use the shared server
			cmd, err = d.ServerManager.StartServer(ctx, "shared")
			if err != nil {
				slog.Error("Failed to get shared server", "error", err)
				conn.Close()
				continue
			}
		}

		// Create session with the server manager
		stdin, stdout, err := d.ServerManager.CreateServerSession(connID)
		if err != nil {
			slog.Error("Failed to create server session", "error", err)
			conn.Close()
			continue
		}

		// Handle the connection in a goroutine
		d.wg.Add(1)
		slog.Info("Handling connection", "id", connID)
		go func(connID string, mode config.ServerMode, cmd *exec.Cmd) {
			defer d.wg.Done()

			slog.Info("Connection handler started", "id", connID)
			// Handle connection
			d.Listener.HandleConnection(ctx, conn, stdin, stdout)

			slog.Info("Connection handler", "stopping", "id", connID, slog.String("mode", fmt.Sprintf("%v", mode)))
			// If per-connection mode, stop the server when the connection closes
			if mode == config.ModePerConnection {
				slog.Info("Connection closed, stopping server", "id", connID)
				d.ServerManager.StopServer(connID)
			}
		}(connID, d.Config.Mode, cmd)
	}
}

// Stop stops the daemon
func (d *Daemon) Stop() error {
	// Close the streaming transport if it exists
	if d.StreamTransport != nil {
		if err := d.StreamTransport.Close(); err != nil {
			slog.Error("Failed to close streaming transport", "error", err)
		}
	} else if d.Listener != nil {
		// If we didn't use streaming transport, close the listener directly
		if err := d.Listener.Close(); err != nil {
			slog.Error("Failed to close listener", "error", err)
		}
	}

	// Stop all server processes
	if d.ServerManager != nil {
		d.ServerManager.StopAllServers()
	}

	// Wait for all connection handlers to complete
	d.wg.Wait()

	// Close trace file if open
	if d.traceFile != nil {
		if err := d.traceFile.Close(); err != nil {
			slog.Error("Failed to close trace file", "error", err)
		}
	}

	slog.Info("Daemon stopped")
	return nil
}
