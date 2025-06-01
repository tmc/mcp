# Signal Handling Improvements

This document summarizes the signal handling improvements implemented in mcpd.

## Main Improvements

1. **Context-Based Signal Handling**:
   - Using `signal.NotifyContext` for propagating cancellation throughout the daemon
   - Handling SIGINT, SIGTERM, SIGQUIT, and SIGHUP for proper daemon shutdown

2. **Process Group Management**:
   - Placing server processes in their own process groups with `Setpgid: true`
   - Sending signals to entire process groups for proper cleanup

3. **Graceful Shutdown Sequence**:
   - Trying SIGTERM first for graceful shutdown
   - Waiting with timeout for processes to exit
   - Using SIGKILL as a last resort

4. **Terminal Signal Handling**:
   - Handling SIGWINCH for terminal resize events in interactive mode
   - Support for SIGTSTP/SIGCONT for terminal suspension and resumption
   - Handling SIGTTIN/SIGTTOU for background process terminal I/O

5. **Terminal State Management**:
   - Saving terminal state before entering raw mode
   - Setting up signal handlers to restore terminal state on interruption
   - Properly handling terminal state during suspension/resumption
   - Using mutex protection for terminal state changes
   - Using defer blocks to ensure state is restored properly

6. **Comprehensive Prompt Handler**:
   - Thread-safe terminal state tracking (raw mode, suspension)
   - Support for interruption during any input type
   - Proper cleanup of signal handlers when input is complete
   - Correctly handling password input with terminal states
   - Managing multiple concurrent prompts

## Implementation Details

### Main Process Signal Handling

```go
// Create a cancellable context for graceful shutdown on signals
ctx, cancel := signal.NotifyContext(context.Background(),
    syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
defer cancel()

// Create a shared struct for storing prompt handler reference
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
                if promptHandler != nil {
                    promptHandler.HandleResize()
                }

            case syscall.SIGTSTP:
                // Terminal suspend (Ctrl+Z)
                if promptHandler != nil {
                    promptHandler.HandleSuspend()
                }

                // Re-send the signal to actually suspend
                signal.Stop(termSignals)
                syscall.Kill(os.Getpid(), syscall.SIGTSTP)
                signal.Notify(termSignals, syscall.SIGTSTP)

            case syscall.SIGCONT:
                // Continue after suspension
                if promptHandler != nil {
                    promptHandler.HandleResume()
                }
            }
        }
    }()

    // Clean up when done
    go func() {
        <-ctx.Done()
        signal.Stop(termSignals)
        close(termSignals)
    }()
}
```

### Server Process Management

```go
// Place the server in its own process group
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid: true,
}

// Signal handling for server shutdown
pgid := cmd.Process.Pid // When Setpgid is true, child's PID is also its PGID

// First try SIGTERM to process group for graceful shutdown
if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
    // Handle error and try alternatives
}

// Wait with timeout for process exit
// ...

// Force termination if needed
if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
    // Handle error and try alternatives
}
```

### Password Input Terminal State Management

The improved password input handling includes thread-safe state management and comprehensive signal handling:

```go
// Read password with robust signal handling
// Save terminal state before entering raw mode
h.promptMutex.Lock()
oldState, err := term.GetState(h.StdinFd)
if err != nil {
    h.promptMutex.Unlock()
    errCh <- fmt.Errorf("failed to get terminal state: %w", err)
    return
}

// Save state for signal handling
h.savedState = oldState
h.promptMutex.Unlock()

// Set up signal handling for interrupts
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)

// Process signals in background
stopSigCh := make(chan struct{})
go func() {
    defer signal.Stop(sigCh)

    for {
        select {
        case <-stopSigCh:
            return
        case sig := <-sigCh:
            switch sig {
            case syscall.SIGINT, syscall.SIGTERM:
                // Interrupt - restore terminal and exit
                h.promptMutex.Lock()
                if h.inRawMode && h.savedState != nil {
                    _ = term.Restore(h.StdinFd, h.savedState)
                    h.inRawMode = false
                }
                h.promptMutex.Unlock()

                errCh <- fmt.Errorf("password input interrupted")
                return

            case syscall.SIGWINCH:
                // Terminal resize
                h.HandleResize()
            }
        }
    }
}()

// Ensure we stop the signal handler when done
defer close(stopSigCh)

// Enter raw mode and track it
h.promptMutex.Lock()
h.inRawMode = true
h.promptMutex.Unlock()
```

### Callback Registration for Signal Handlers

```go
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
```

## Benefits

These improvements provide the following benefits:

1. **Graceful Shutdown**: Proper cleanup of resources on termination
2. **Robustness**: Better handling of edge cases and error conditions
3. **User Experience**: Proper terminal state management for interactive mode
4. **Child Process Management**: Cleaner handling of child processes

## Further Improvements

Potential future improvements:

1. More sophisticated terminal UI for interactive mode
2. Better handling of terminal state in other input types
3. Support for additional signals (SIGTSTP, SIGCONT, etc.)
4. Additional configuration options for signal handling behavior