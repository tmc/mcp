// Command mcpd manages and provides network access to MCP-compliant server commands
// that communicate over stdin/stdout.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/tmc/mcp/exp/cmd/mcpd/config"
	"github.com/tmc/mcp/exp/cmd/mcpd/daemon"
	"github.com/tmc/mcp/exp/cmd/mcpd/transport"
)

// Version information
const (
	Version = "0.1.0"
)

// Command line flags
var (
	// Server management
	serverMode = flag.String("mode", "once", "Server lifecycle mode: once (default), per-connection")

	// Network options
	socketPath = flag.String("socket", "", "Unix domain socket path (default: auto-generated)")
	tcpAddr    = flag.String("tcp", "", "TCP address to listen on (e.g., :8080)")

	// Streaming options
	httpAddr      = flag.String("http", "", "HTTP address to listen on for streaming endpoints (e.g., :8081)")
	enableSSE     = flag.Bool("enable-sse", false, "Enable Server-Sent Events endpoint (/sse)")
	enableWS      = flag.Bool("enable-ws", false, "Enable WebSocket support (/ws) - experimental")
	enableStream  = flag.Bool("enable-stream", false, "Enable HTTP streaming endpoint (/stream)")
	streamTimeout = flag.Duration("stream-timeout", 1*time.Hour, "Timeout for streaming connections")

	// Logging options
	logFile   = flag.String("log-file", "", "MCP trace file path (.mcp format)")
	serverLog = flag.String("server-log", "", "Log file for server stdout/stderr (default: stderr)")
	verbose   = flag.Bool("v", false, "Enable verbose logging")

	// Process management
	pidFile       = flag.String("pid-file", "", "Path to write PID file")
	serverPidFile = flag.String("server-pid-file", "", "Path to write server process PID file")
	timeout       = flag.Duration("timeout", 0, "Auto-terminate after specified duration (for testing)")

	// Interactive mode
	interactive = flag.Bool("i", false, "Run in interactive mode, handling prompts via TTY")
	noTTYPrompt = flag.Bool("no-tty-prompt", false, "Disable TTY prompting in interactive mode")

	// Authentication options
	enableOAuth     = flag.Bool("enable-oauth", false, "Enable OAuth authentication")
	oauthClientID   = flag.String("oauth-client-id", "", "OAuth Client ID")
	oauthSecret     = flag.String("oauth-secret", "", "OAuth Client Secret")
	oauthProvider   = flag.String("oauth-provider", "google", "OAuth provider: google, github, custom, local")
	oauthCallback   = flag.String("oauth-callback", "/auth/callback", "OAuth callback path")
	authorizedUsers = flag.String("authorized-users", "", "Comma-separated list of authorized user emails")

	// Local authentication options
	localAuthFile    = flag.String("local-auth-file", "", "Path to local users file (username:password format)")
	localAuthUsers   = flag.String("local-auth-users", "", "Local users in format 'user1:pass1,user2:pass2' (for development)")
	localAuthPersist = flag.String("local-auth-persist", "", "Path to persist local users as JSON file")
)

func main() {
	// Parse flags and separate server command
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] -- <server_command> [server_args...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	// Find -- separator
	args := os.Args[1:]
	separatorIdx := -1
	for i, arg := range args {
		if arg == "--" {
			separatorIdx = i
			break
		}
	}

	var cmdArgs []string
	if separatorIdx >= 0 {
		// Args before -- are flags, after are command and its args
		flag.CommandLine.Parse(args[:separatorIdx])
		cmdArgs = args[separatorIdx+1:]
	} else {
		// No --, all are flags
		flag.Parse()
		cmdArgs = flag.Args()
	}

	// Make sure we have a server command
	if len(cmdArgs) == 0 {
		fmt.Fprintf(os.Stderr, "Error: server command is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Configure logging
	setupLogging()

	// Create a cancellable context for graceful shutdown on signals
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	defer cancel()

	// Create a shared struct for storing prompt handler reference
	// This is needed because the prompt handler is created later in daemon initialization
	// but we need to access it from the signal handlers
	type signalHandlerState struct {
		promptHandler *transport.InteractivePromptHandler
		mu            sync.Mutex
	}
	sigHandlerState := &signalHandlerState{}

	// Set up additional signal handling for terminal signals if in interactive mode
	if *interactive {
		termSignals := make(chan os.Signal, 1)

		// Handle terminal-specific signals
		signal.Notify(termSignals,
			syscall.SIGWINCH, // Window size change
			syscall.SIGTSTP,  // Ctrl+Z (suspend)
			syscall.SIGCONT,  // Continue after suspension
			syscall.SIGTTIN,  // Terminal read from background
			syscall.SIGTTOU)  // Terminal write from background

		go func() {
			for sig := range termSignals {
				sigHandlerState.mu.Lock()
				promptHandler := sigHandlerState.promptHandler
				sigHandlerState.mu.Unlock()

				switch sig {
				case syscall.SIGWINCH:
					// Terminal window size changed
					slog.Debug("Terminal window size changed (SIGWINCH)")
					if promptHandler != nil {
						promptHandler.HandleResize()
					}

				case syscall.SIGTSTP:
					// Terminal suspend (Ctrl+Z)
					slog.Debug("Terminal suspend requested (SIGTSTP)")

					// Prepare terminal for suspension if we have a prompt handler
					if promptHandler != nil {
						promptHandler.HandleSuspend()
					}

					// Re-send the signal to ourself to actually suspend
					// But first unblock the signal
					signal.Stop(termSignals)
					syscall.Kill(os.Getpid(), syscall.SIGTSTP)
					signal.Notify(termSignals, syscall.SIGTSTP)

				case syscall.SIGCONT:
					// Continue after suspension
					slog.Debug("Terminal continuing after suspension (SIGCONT)")

					// Restore terminal state that was saved before suspension
					if promptHandler != nil {
						promptHandler.HandleResume()
					}

				case syscall.SIGTTIN, syscall.SIGTTOU:
					// Terminal I/O from background process
					slog.Debug("Terminal I/O from background (SIGTTIN/SIGTTOU)")
				}
			}
		}()

		// Clean up this signal handler when the context is done
		go func() {
			<-ctx.Done()
			signal.Stop(termSignals)
			close(termSignals)
		}()
	}

	// Write PID file if requested
	if *pidFile != "" {
		writePidFile(*pidFile)
		defer os.Remove(*pidFile)
	}

	// Create configuration
	cfg := config.New()
	cfg.SetServerCommand(cmdArgs[0], cmdArgs[1:])

	// Set server mode
	if err := cfg.SetMode(*serverMode); err != nil {
		log.Fatalf("Invalid server mode: %v", err)
	}

	// Set listen address
	if err := cfg.SetListenAddr(*tcpAddr, *socketPath); err != nil {
		log.Fatalf("Failed to set listen address: %v", err)
	}

	// Set other options
	cfg.SetTraceFile(*logFile).
		SetServerLogFile(*serverLog).
		SetPidFile(*pidFile).
		SetServerPidFile(*serverPidFile).
		SetVerbose(*verbose).
		SetTimeout(*timeout).
		SetInteractive(*interactive).
		SetNoTTYPrompt(*noTTYPrompt)

	// Set streaming options if HTTP address is provided
	if *httpAddr != "" {
		cfg.SetHTTPAddr(*httpAddr).
			SetStreamingOptions(*enableSSE, *enableWS, *enableStream).
			SetStreamTimeout(*streamTimeout)
	}

	// Create the daemon
	d, err := daemon.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	// Set up prompt handler callback if in interactive mode
	if *interactive {
		d.WithPromptHandlerCallback(func(handler *transport.InteractivePromptHandler) {
			// Store the prompt handler in the shared state for signal handlers
			sigHandlerState.mu.Lock()
			sigHandlerState.promptHandler = handler
			sigHandlerState.mu.Unlock()

			if *verbose {
				slog.Info("Signal handler connected to prompt handler")
			}
		})
	}

	// Start the daemon
	if err := d.Start(ctx); err != nil {
		if err != context.Canceled {
			log.Fatalf("Daemon error: %v", err)
		}

		log.Println("Daemon stopped due to context cancellation")
	}

	// Stop the daemon
	if err := d.Stop(); err != nil {
		log.Printf("Error stopping daemon: %v", err)
	}
}

// setupLogging configures the logger
func setupLogging() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Configure structured logging
	logOpts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}

	if *verbose {
		logOpts.Level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, logOpts)
	slog.SetDefault(slog.New(handler))

	slog.Info("mcpd starting", "version", Version)
}

// writePidFile writes the current process ID to the specified file
func writePidFile(path string) {
	pid := os.Getpid()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create PID file directory: %v", err)
	}

	// Write PID to file
	if err := os.WriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), 0644); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}

	log.Printf("Wrote PID %d to %s", pid, path)
}
