// Package manager handles server process management for mcpd.
package manager

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// ProcessMode defines the lifecycle mode for server processes
type ProcessMode string

const (
	// ModeSingle starts one server process for all connections
	ModeSingle ProcessMode = "once"

	// ModePerConnection starts a new server process for each connection
	ModePerConnection ProcessMode = "per-connection"
)

// ServerManager manages server processes
type ServerManager struct {
	// Configuration
	Command   string
	Args      []string
	Mode      ProcessMode
	PidFile   string
	LogFile   string
	TraceFile *os.File

	// Active processes
	processes map[string]*exec.Cmd
	mu        sync.Mutex
}

// New creates a new ServerManager
func New(command string, args []string, mode ProcessMode) *ServerManager {
	return &ServerManager{
		Command:   command,
		Args:      args,
		Mode:      mode,
		processes: make(map[string]*exec.Cmd),
	}
}

// WithPidFile sets the PID file path
func (m *ServerManager) WithPidFile(path string) *ServerManager {
	m.PidFile = path
	return m
}

// WithLogFile sets the log file path
func (m *ServerManager) WithLogFile(path string) *ServerManager {
	m.LogFile = path
	return m
}

// WithTraceFile sets the trace file
func (m *ServerManager) WithTraceFile(file *os.File) *ServerManager {
	m.TraceFile = file
	return m
}

// StartServer starts a new server process
func (m *ServerManager) StartServer(ctx context.Context, id string) (*exec.Cmd, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If ModeSingle, check if we already have a process
	if m.Mode == ModeSingle && len(m.processes) > 0 {
		for _, cmd := range m.processes {
			return cmd, nil
		}
	}

	// Create command
	slog.Info("Starting server process", "id", id, "command", m.Command, "args", m.Args)
	cmd := exec.CommandContext(ctx, m.Command, m.Args...)

	// Place the server in its own process group to better manage signal propagation
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Don't set up pipes here - we'll do that when creating a session
	// The command itself will handle setting up the pipes when it's started

	// Set up stderr
	if m.LogFile != "" {
		// Open or create log file
		logFile, err := os.OpenFile(m.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", m.LogFile, err)
		}
		cmd.Stderr = logFile
	} else {
		// Default to stderr
		cmd.Stderr = os.Stderr
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Write PID file if requested
	if m.PidFile != "" {
		pid := cmd.Process.Pid

		// Ensure directory exists
		dir := filepath.Dir(m.PidFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			slog.Error("Failed to create PID file directory", "error", err)
		} else {
			// Write PID to file
			if err := os.WriteFile(m.PidFile, []byte(fmt.Sprintf("%d\n", pid)), 0644); err != nil {
				slog.Error("Failed to write PID file", "error", err)
			} else {
				slog.Info("Wrote server PID", "pid", pid, "file", m.PidFile)
			}
		}
	}

	// Store the process
	m.processes[id] = cmd

	slog.Info("Started server process",
		"id", id,
		"pid", cmd.Process.Pid,
		"command", m.Command,
		"mode", m.Mode)

	return cmd, nil
}

// StopServer stops a server process
func (m *ServerManager) StopServer(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd, ok := m.processes[id]
	if !ok {
		return fmt.Errorf("server with ID %s not found", id)
	}

	// Try to terminate gracefully first
	if cmd.Process != nil {
		pgid := cmd.Process.Pid // When Setpgid is true, child's PID is also its PGID
		pid := cmd.Process.Pid

		slog.Info("Stopping server gracefully", "id", id, "pid", pid)

		// First try SIGTERM to process group for graceful shutdown
		if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
			slog.Error("Failed to send SIGTERM to process group",
				"id", id,
				"pgid", pgid,
				"error", err)

			// Try sending to just the process
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				slog.Error("Failed to send SIGTERM to process",
					"id", id,
					"pid", pid,
					"error", err)
			}
		}

		// Wait for process to exit with timeout
		waitCh := make(chan error, 1)
		go func() {
			waitCh <- cmd.Wait()
		}()

		gracePeriod := 3 * time.Second
		select {
		case err := <-waitCh:
			// Process exited
			exitStatus := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitStatus = exitErr.ExitCode()
				}
			}
			slog.Info("Server process exited", "id", id, "pid", pid, "status", exitStatus)

		case <-time.After(gracePeriod):
			// Timeout, force kill the process group
			slog.Warn("Server did not exit within grace period, forcing termination",
				"id", id, "pid", pid, "grace_period_seconds", gracePeriod.Seconds())

			// Send SIGKILL to the process group
			if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
				slog.Error("Failed to send SIGKILL to process group",
					"id", id, "pgid", pgid, "error", err)

				// Last resort: kill the process directly
				if err := cmd.Process.Kill(); err != nil {
					slog.Error("Failed to kill process", "id", id, "pid", pid, "error", err)
				}
			}
		}
	}

	// Clean up PID file
	if m.PidFile != "" {
		if err := os.Remove(m.PidFile); err != nil && !os.IsNotExist(err) {
			slog.Error("Failed to remove PID file", "file", m.PidFile, "error", err)
		}
	}

	// Remove from processes map
	delete(m.processes, id)

	return nil
}

// StopAllServers stops all running server processes
func (m *ServerManager) StopAllServers() {
	m.mu.Lock()

	// Create list of processes to stop
	type serverProcess struct {
		id   string
		cmd  *exec.Cmd
		pgid int
		pid  int
	}

	serversToStop := []serverProcess{}

	for id, cmd := range m.processes {
		if cmd.Process != nil {
			// When Setpgid is true, child's PID is also its PGID
			pgid := cmd.Process.Pid
			pid := cmd.Process.Pid

			serversToStop = append(serversToStop, serverProcess{
				id:   id,
				cmd:  cmd,
				pgid: pgid,
				pid:  pid,
			})
		}

		// Remove from processes map immediately to prevent new connections
		delete(m.processes, id)
	}

	// Release the lock before stopping processes
	m.mu.Unlock()

	if len(serversToStop) == 0 {
		return
	}

	slog.Info("Stopping all server processes", "count", len(serversToStop))

	// First, send SIGTERM to all process groups
	for _, sp := range serversToStop {
		// Try sending SIGTERM to the process group
		if err := syscall.Kill(-sp.pgid, syscall.SIGTERM); err != nil {
			slog.Error("Failed to send SIGTERM to process group",
				"id", sp.id, "pgid", sp.pgid, "error", err)

			// Try sending to just the process
			if err := sp.cmd.Process.Signal(syscall.SIGTERM); err != nil {
				slog.Error("Failed to send SIGTERM to process",
					"id", sp.id, "pid", sp.pid, "error", err)
			}
		}
	}

	// Give processes time to shut down gracefully
	gracePeriod := 2 * time.Second
	deadline := time.After(gracePeriod)

	// Set up WaitGroup to track process exits
	var wg sync.WaitGroup
	for _, sp := range serversToStop {
		wg.Add(1)

		go func(sp serverProcess) {
			defer wg.Done()

			// Set up channel for Wait result
			doneCh := make(chan struct{})

			go func() {
				_ = sp.cmd.Wait()
				close(doneCh)
			}()

			// Wait for either process exit or deadline
			select {
			case <-doneCh:
				slog.Info("Server process exited gracefully", "id", sp.id, "pid", sp.pid)
			case <-deadline:
				// Force kill the process if still running
				slog.Warn("Server did not exit within grace period, forcing termination",
					"id", sp.id, "pid", sp.pid)

				// Try to kill the process group
				if err := syscall.Kill(-sp.pgid, syscall.SIGKILL); err != nil {
					slog.Error("Failed to kill process group",
						"id", sp.id, "pgid", sp.pgid, "error", err)

					// Last resort
					_ = sp.cmd.Process.Kill()
				}
			}
		}(sp)
	}

	// Wait for all process handling to complete
	// with a timeout in case something goes wrong
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("All server processes stopped")
	case <-time.After(gracePeriod + 1*time.Second):
		slog.Error("Timed out waiting for server processes to stop")
	}

	// Clean up PID file
	if m.PidFile != "" {
		if err := os.Remove(m.PidFile); err != nil && !os.IsNotExist(err) {
			slog.Error("Failed to remove PID file", "file", m.PidFile, "error", err)
		}
	}
}

// CreateServerSession creates pipes for communicating with a server process
func (m *ServerManager) CreateServerSession(id string) (io.WriteCloser, io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd, ok := m.processes[id]
	if !ok {
		return nil, nil, fmt.Errorf("server with ID %s not found", id)
	}

	// Create pipes before starting the process
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		// Close the stdin pipe to avoid leaks
		stdin.Close()
		return nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Check if the process is already running
	if cmd.Process == nil {
		// Start the command if it's not running yet
		if err := cmd.Start(); err != nil {
			// Close the pipes to avoid leaks
			stdin.Close()
			return nil, nil, fmt.Errorf("failed to start server process: %w", err)
		}

		slog.Info("Started server process",
			"id", id,
			"pid", cmd.Process.Pid,
			"command", m.Command)
	}

	return stdin, stdout, nil
}
