package transport

import (
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
)

// HandleConnection handles bidirectional proxying between connection and server pipes
// with support for interactive prompting
func (t *Listener) HandleConnection(ctx context.Context, conn net.Conn, serverIn io.Writer, serverOut io.Reader) {
	// Ensure connection is closed when done
	defer func() {
		conn.Close()
		t.RemoveConnection(conn)
		slog.Info("Connection closed", "remote", conn.RemoteAddr().String())
	}()

	// Create cancellable context for this connection
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Set up a WaitGroup for the two copy goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Copy from client to server (client -> server)
	go func() {
		defer wg.Done()
		defer cancel() // Cancel context on exit

		buffer := make([]byte, 4096)

		for {
			// Check if we're shutting down
			select {
			case <-ctx.Done():
				slog.Info("Shutting down via context, stopping client to server copy")
				return
			case <-t.shutdownCh:
				slog.Info("Shutting down, stopping client to server copy")
				return
			default:
				// Continue
			}

			// Read from client
			n, err := conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					slog.Error("Error reading from client", "error", err)
				}
				return
			}

			// Log to trace file
			if t.TraceLogger != nil {
				if err := t.TraceLogger.LogClientToServer(buffer[:n]); err != nil {
					slog.Error("Error logging client message", "error", err)
				}
			}

			// Write to server
			_, err = serverIn.Write(buffer[:n])
			if err != nil {
				slog.Error("Error writing to server", "error", err)
				return
			}
		}
	}()

	// Copy from server to client (server -> client)
	go func() {
		defer wg.Done()
		defer cancel() // Cancel context on exit

		buffer := make([]byte, 4096)

		for {
			// Check if we're shutting down
			select {
			case <-ctx.Done():
				slog.Info("Shutting down via context, stopping client to server copy")
				return
			case <-t.shutdownCh:
				slog.Info("Shutting down, stopping client to server copy")
				return
			default:
				// Continue
			}

			// Read from server
			n, err := serverOut.Read(buffer)
			if err != nil {
				if err != io.EOF {
					slog.Error("Error reading from server", "error", err)
				}
				return
			}

			// Extract the message
			message := buffer[:n]

			// Log to trace file
			if t.TraceLogger != nil {
				if err := t.TraceLogger.LogServerToClient(message); err != nil {
					slog.Error("Error logging server message", "error", err)
				}
			}

			// Check if this is an interactive prompt request
			isPrompt := false
			if t.PromptHandler != nil && t.PromptHandler.IsInteractive() {
				// Look for interactive/promptUser method indicator
				if strings.Contains(string(message), `"method":"interactive/promptUser"`) {
					isPrompt = true

					// Handle the prompt
					response, err := t.PromptHandler.HandleAndGenerateResponse(message)
					if err != nil {
						slog.Error("Error handling prompt", "error", err)
					} else {
						// Log the generated response
						if t.TraceLogger != nil {
							if err := t.TraceLogger.LogClientToServer(response); err != nil {
								slog.Error("Error logging prompt response", "error", err)
							}
						}

						// Send the response to the server
						_, err = serverIn.Write(response)
						if err != nil {
							slog.Error("Error sending prompt response to server", "error", err)
						}
					}

					// Don't forward the prompt request to client
					continue
				}
			}

			// If not a prompt (or prompt handling failed), write to client normally
			if !isPrompt {
				_, err = conn.Write(message)
				if err != nil {
					slog.Error("Error writing to client", "error", err)
					return
				}
			}
		}
	}()

	// Wait for both goroutines to complete
	wg.Wait()
}
