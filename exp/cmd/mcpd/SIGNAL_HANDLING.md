# Signal Handling for MCP Tools

This document describes the signal handling patterns and best practices for mcpd and related MCP tools.

## Overview

Proper signal handling is essential for MCP tools to:
1. Gracefully shut down when requested
2. Clean up resources (sockets, file handles, PID files)
3. Propagate signals appropriately to child processes
4. Respond to terminal state changes (for interactive applications)

## Signal Types

### Critical Signals (Must Handle)

| Signal   | Description                   | Action                                     |
|----------|-------------------------------|-------------------------------------------|
| SIGINT   | Interrupt from keyboard (^C)  | Graceful shutdown                         |
| SIGTERM  | Termination signal            | Graceful shutdown                         |
| SIGQUIT  | Quit from keyboard (^\)       | Graceful shutdown with stack trace option |
| SIGHUP   | Terminal disconnected         | Graceful shutdown or reload config        |

### Process Management Signals

| Signal   | Description                   | Action                                     |
|----------|-------------------------------|-------------------------------------------|
| SIGCHLD  | Child process terminated      | Reap child processes to prevent zombies   |
| SIGUSR1  | User-defined signal 1         | Reload configuration                      |
| SIGUSR2  | User-defined signal 2         | Toggle debug mode/dump state              |

### Terminal-Related Signals (for Interactive Mode)

| Signal   | Description                   | Action                                     |
|----------|-------------------------------|-------------------------------------------|
| SIGWINCH | Window size change            | Adjust terminal UI if applicable          |
| SIGTSTP  | Stop typed at terminal (^Z)   | Suspend if appropriate                    |
| SIGCONT  | Continue after stop           | Resume execution                          |
| SIGTTIN  | Terminal input for bg process | Handle background/foreground transitions  |
| SIGTTOU  | Terminal output for bg process| Handle background/foreground transitions  |

## Implementation Patterns

### 1. Context-Based Signal Handling (Recommended)

Go's `context` package provides a clean way to propagate cancellation signals:

```go
// Create a cancellable context that is canceled when signals are received
ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
defer cancel()

// Use this context for all operations that should be cancellable
// When a signal is received, ctx.Done() will be closed
```

Key benefits:
- Automatic propagation of cancellation
- Clean integration with Go's concurrency patterns
- No explicit signal channels to manage

### 2. Channel-Based Signal Handling

For more direct control over signal handling:

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

// In a goroutine
go func() {
    sig := <-sigChan
    // Perform signal-specific handling
    switch sig {
    case syscall.SIGINT, syscall.SIGTERM:
        log.Printf("Received %s, shutting down...", sig)
        // Initiate graceful shutdown
    case syscall.SIGQUIT:
        log.Printf("Received %s, shutting down with stack trace...", sig)
        // Optional: dump stack trace before shutdown
        // Initiate graceful shutdown
    }
}()
```

### 3. Process Group Management

For tools that manage child processes, proper process group handling is important:

```go
// Create a new process group for child processes
cmd := exec.Command(serverCommand, serverArgs...)
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid: true, // Place the child in a new process group
}

// To forward signals to the entire process group
childPgid := cmd.Process.Pid // When Setpgid=true, the child's PID is also its PGID
syscall.Kill(-childPgid, sig) // Negative PID sends the signal to the entire process group
```

## mcpd Signal Handling Implementation

### 1. Main Daemon Process

```go
// Create a cancellable context for the entire daemon
ctx, cancel := signal.NotifyContext(context.Background(),
    syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
defer cancel()

// Pass this context to all long-running operations
d, err := daemon.New(cfg)
if err != nil {
    log.Fatal(err)
}

if err := d.Start(ctx); err != nil {
    if err != context.Canceled {
        log.Fatal(err)
    }
    log.Println("Daemon stopped due to signal")
}
```

### 2. Server Process Management

When starting server processes, mcpd should:

1. Properly manage server process groups
2. Forward appropriate signals to servers
3. Detect when servers terminate and respond accordingly

```go
// In the ServerManager
func (m *ServerManager) StartServer(ctx context.Context, id string) (*exec.Cmd, error) {
    cmd := exec.CommandContext(ctx, m.Command, m.Args...)
    
    // Put the server in its own process group
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,
    }
    
    // When ctx is canceled, CommandContext sends SIGKILL to the process
    // For more graceful shutdown, we want to send SIGTERM first
    go func() {
        <-ctx.Done()
        if cmd.Process != nil {
            // Try SIGTERM first
            log.Printf("Sending SIGTERM to server %s (PID %d)", id, cmd.Process.Pid)
            cmd.Process.Signal(syscall.SIGTERM)
            
            // Wait briefly for graceful shutdown
            terminationTimeout := time.NewTimer(3 * time.Second)
            terminationDone := make(chan struct{})
            
            go func() {
                cmd.Wait()
                close(terminationDone)
            }()
            
            select {
            case <-terminationDone:
                // Process exited after SIGTERM
                terminationTimeout.Stop()
            case <-terminationTimeout.C:
                // Timeout - force kill
                log.Printf("Server %s did not terminate gracefully, sending SIGKILL", id)
                cmd.Process.Kill()
            }
        }
    }()
    
    return cmd, nil
}
```

### 3. Interactive Mode Signal Handling

In interactive mode, additional signal handling is needed for terminal interaction:

```go
// Additional signals to handle in interactive mode
if cfg.Interactive {
    // Create a separate signal channel for terminal-related signals
    termSignals := make(chan os.Signal, 1)
    signal.Notify(termSignals, syscall.SIGWINCH, syscall.SIGTSTP, syscall.SIGCONT)
    
    go func() {
        for sig := range termSignals {
            switch sig {
            case syscall.SIGWINCH:
                // Handle terminal resize
                // This may be important for TUI-based prompts
                if promptHandler != nil {
                    promptHandler.HandleResize()
                }
            case syscall.SIGTSTP:
                // Handle terminal suspension
                // May need to temporarily disable raw mode
                if promptHandler != nil {
                    promptHandler.HandleSuspend()
                }
                // Forward to ourselves to actually suspend
                syscall.Kill(os.Getpid(), syscall.SIGTSTP)
            case syscall.SIGCONT:
                // Handle terminal resume
                // Restore terminal state if needed
                if promptHandler != nil {
                    promptHandler.HandleResume()
                }
            }
        }
    }()
}
```

## Terminal State Management

For interactive tools, proper terminal state management is crucial:

1. Save terminal state before changing modes
2. Restore terminal state on exit, especially after signals
3. Handle signals that may interrupt terminal interaction (SIGTSTP, SIGINT)

Example for terminal raw mode handling:

```go
// Save original terminal state
oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
if err != nil {
    return err
}

// Ensure terminal state is restored on exit
defer term.Restore(int(os.Stdin.Fd()), oldState)

// Also ensure it's restored on signals
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
go func() {
    <-sigCh
    term.Restore(int(os.Stdin.Fd()), oldState)
    os.Exit(1)
}()
```

## Socket File Cleanup

Unix domain sockets should be cleaned up on exit:

```go
// Register cleanup function for socket
socketPath := "/path/to/socket"
cleanup := func() {
    // Remove socket file if it exists
    if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
        log.Printf("Error removing socket file: %v", err)
    }
}

// Use defer for normal exit
defer cleanup()

// Also ensure cleanup on signals
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
go func() {
    <-sigCh
    cleanup()
    os.Exit(1)
}()
```

## PID File Management

PID files should be created securely and removed on exit:

```go
// Create PID file
pidFile := "/path/to/pidfile"
err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644)
if err != nil {
    log.Fatalf("Failed to write PID file: %v", err)
}

// Remove PID file on exit
cleanup := func() {
    if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
        log.Printf("Error removing PID file: %v", err)
    }
}

// Use defer for normal exit
defer cleanup()

// Also ensure cleanup on signals
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
go func() {
    <-sigCh
    cleanup()
    os.Exit(1)
}()
```

## Best Practices

1. **Use signal.NotifyContext**: Prefer using `signal.NotifyContext` for cancellation propagation
2. **Graceful Shutdown Sequence**: Implement a multi-stage shutdown sequence:
   - Stop accepting new connections
   - Complete in-progress operations (or set a deadline)
   - Close existing connections
   - Stop server processes
   - Clean up resources
3. **Resource Cleanup**: Always clean up all resources (sockets, files, processes)
4. **Timeout on Graceful Shutdown**: Set timeouts for graceful shutdown operations
5. **Process Group Management**: When managing multiple processes, use process groups
6. **Handle Terminal State**: For interactive tools, always restore terminal state
7. **Signal Propagation**: Correctly propagate signals to child processes
8. **Signal Documentation**: Document which signals the tool responds to and how