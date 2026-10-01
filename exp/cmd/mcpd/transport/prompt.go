package transport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

// InteractivePromptHandler handles interactive prompts from the server
type InteractivePromptHandler struct {
	Enabled     bool
	StdinFd     int
	StderrFd    int
	TraceLogger *TraceLogger
	termWidth   int
	termHeight  int
	savedState  *term.State // Saved terminal state for restoration
	inRawMode   bool        // Whether terminal is currently in raw mode
	inPrompt    bool        // Whether we're currently handling a prompt
	promptMutex sync.Mutex  // Mutex to protect terminal state changes
}

// PromptParameters contains parameters for a prompt request
type PromptParameters struct {
	PromptMessage string `json:"prompt_message"`
	InputID       string `json:"input_id"`
	InputType     string `json:"input_type,omitempty"`
	DefaultValue  string `json:"default_value,omitempty"`
	TimeoutSecs   int    `json:"timeout_seconds,omitempty"`
}

// PromptResponse contains the response to a prompt request
type PromptResponse struct {
	InputID      string `json:"original_input_id"`
	Status       string `json:"status"`
	Value        string `json:"value,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

const (
	// ANSI escape codes
	bell      = "\a"
	clearLine = "\r\033[K"

	// Default prompt timeout in seconds
	defaultPromptTimeout = 60
)

// NewInteractivePromptHandler creates a new interactive prompt handler
func NewInteractivePromptHandler(enabled bool) *InteractivePromptHandler {
	handler := &InteractivePromptHandler{
		Enabled:  enabled,
		StdinFd:  int(os.Stdin.Fd()),
		StderrFd: int(os.Stderr.Fd()),
	}

	// Initialize terminal dimensions if interactive
	if enabled && term.IsTerminal(handler.StderrFd) {
		handler.HandleResize()
	}

	return handler
}

// WithTraceLogger sets the trace logger
func (h *InteractivePromptHandler) WithTraceLogger(logger *TraceLogger) *InteractivePromptHandler {
	h.TraceLogger = logger
	return h
}

// IsInteractive returns true if the process has an interactive TTY
func (h *InteractivePromptHandler) IsInteractive() bool {
	return h.Enabled && term.IsTerminal(h.StdinFd) && term.IsTerminal(h.StderrFd)
}

// HandleResize updates the terminal dimensions
func (h *InteractivePromptHandler) HandleResize() {
	if h.IsInteractive() {
		width, height, err := term.GetSize(h.StderrFd)
		if err != nil {
			return
		}
		h.termWidth = width
		h.termHeight = height
	}
}

// HandleSuspend prepares for terminal suspension
func (h *InteractivePromptHandler) HandleSuspend() {
	h.promptMutex.Lock()
	defer h.promptMutex.Unlock()

	// Restore terminal state if we're currently in raw mode
	if h.inRawMode && h.savedState != nil {
		slog.Debug("Restoring terminal state before suspension")
		_ = term.Restore(h.StdinFd, h.savedState)
		h.inRawMode = false
	}
}

// HandleResume restores terminal state after resuming from suspension
func (h *InteractivePromptHandler) HandleResume() {
	h.promptMutex.Lock()
	defer h.promptMutex.Unlock()

	// We don't re-enter raw mode here since the appropriate prompt
	// function will do that when it continues. Just update terminal info.
	h.HandleResize()

	if h.inPrompt {
		// If we were in a prompt when suspended, indicate this to the user
		fmt.Fprintln(os.Stderr, "\nResumed prompt session...")
	}
}

// HandlePrompt handles an interactive/promptUser request from the server
func (h *InteractivePromptHandler) HandlePrompt(message []byte) (PromptResponse, error) {
	// Default response for non-interactive mode
	response := PromptResponse{
		Status:       "error_no_tty",
		ErrorMessage: "No interactive TTY available for prompt",
	}

	// Parse the message
	var jsonMsg map[string]interface{}
	if err := json.Unmarshal(message, &jsonMsg); err != nil {
		return response, fmt.Errorf("failed to parse prompt message: %w", err)
	}

	// Check if this is an interactive/promptUser request
	method, _ := jsonMsg["method"].(string)
	if method != "interactive/promptUser" {
		return response, fmt.Errorf("not an interactive/promptUser request: %s", method)
	}

	// Extract parameters
	paramsJson, ok := jsonMsg["params"].(map[string]interface{})
	if !ok {
		return response, fmt.Errorf("invalid params in promptUser request")
	}

	// Convert to PromptParameters
	params := PromptParameters{
		PromptMessage: getStringParam(paramsJson, "prompt_message", "Enter input:"),
		InputID:       getStringParam(paramsJson, "input_id", fmt.Sprintf("prompt_%d", time.Now().Unix())),
		InputType:     getStringParam(paramsJson, "input_type", "text"),
		DefaultValue:  getStringParam(paramsJson, "default_value", ""),
	}

	// Get timeout
	if timeoutJson, ok := paramsJson["timeout_seconds"]; ok {
		if timeout, ok := timeoutJson.(float64); ok {
			params.TimeoutSecs = int(timeout)
		}
	}
	if params.TimeoutSecs <= 0 {
		params.TimeoutSecs = defaultPromptTimeout
	}

	// Set input ID in response
	response.InputID = params.InputID

	// Check if we can prompt interactively
	if !h.IsInteractive() {
		slog.Warn("Cannot prompt interactively: no TTY available",
			"input_id", params.InputID)
		return response, nil
	}

	// Prompt interactively
	return h.promptInteractive(params)
}

// promptInteractive handles the actual interactive prompt
func (h *InteractivePromptHandler) promptInteractive(params PromptParameters) (PromptResponse, error) {
	response := PromptResponse{
		InputID: params.InputID,
		Status:  "ok",
	}

	// Alert user
	fmt.Fprint(os.Stderr, bell)

	// Format prompt with input type and default value indicators
	prompt := params.PromptMessage
	if params.InputType == "password" {
		prompt += " (password)"
	}
	if params.DefaultValue != "" {
		if params.InputType != "password" {
			prompt += fmt.Sprintf(" [%s]", params.DefaultValue)
		} else {
			prompt += " [*****]"
		}
	}
	prompt += ": "

	// Display the prompt
	fmt.Fprint(os.Stderr, clearLine+prompt)

	// Set up timeout
	timeoutCh := time.After(time.Duration(params.TimeoutSecs) * time.Second)

	// Handle different input types
	var input string

	// Create a channel for input completion
	doneCh := make(chan struct{})
	errCh := make(chan error, 1)
	inputCh := make(chan string, 1)

	// Start input reading goroutine
	go func() {
		// Set some common state for all input types
		h.promptMutex.Lock()
		wasAlreadyInPrompt := h.inPrompt
		h.inPrompt = true
		h.promptMutex.Unlock()

		// Make sure we mark completion when done
		defer func() {
			close(doneCh)

			// Only reset prompt state if we set it (and not if being handled by another goroutine)
			if !wasAlreadyInPrompt {
				h.promptMutex.Lock()
				h.inPrompt = false
				h.promptMutex.Unlock()
			}
		}()

		switch params.InputType {
		case "password":
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

			// Read the password (term.ReadPassword puts terminal in raw mode)
			var passwordBytes []byte
			passwordBytes, err = term.ReadPassword(h.StdinFd)

			// ReadPassword returns terminal to cooked mode, update our tracking
			h.promptMutex.Lock()
			h.inRawMode = false
			h.promptMutex.Unlock()

			// Check for errors
			if err != nil {
				errCh <- fmt.Errorf("failed to read password: %w", err)
				return
			}

			// Print newline after password
			fmt.Fprintln(os.Stderr)

			// Send password
			inputCh <- string(passwordBytes)

		case "confirm_yn":
			// Read y/n confirmation with improved signal handling

			// Setup signal handling
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			// Process signals in background
			stopSigCh := make(chan struct{})
			go func() {
				defer signal.Stop(sigCh)

				select {
				case <-stopSigCh:
					return
				case sig := <-sigCh:
					// Interrupt
					errCh <- fmt.Errorf("confirmation input interrupted by %v", sig)
				}
			}()

			// Ensure we stop the signal handler when done
			defer close(stopSigCh)

			// Read confirmation
			reader := bufio.NewReader(os.Stdin)
			confirm, readErr := reader.ReadString('\n')
			if readErr != nil {
				errCh <- fmt.Errorf("failed to read confirmation: %w", readErr)
				return
			}

			// Process result
			confirm = strings.ToLower(strings.TrimSpace(confirm))
			if confirm == "y" || confirm == "yes" {
				inputCh <- "yes"
			} else if confirm == "n" || confirm == "no" {
				inputCh <- "no"
			} else if confirm == "" && params.DefaultValue != "" {
				inputCh <- params.DefaultValue
			} else {
				inputCh <- "no" // Default to no
			}

		default: // "text" and others
			// Setup signal handling
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			// Process signals in background
			stopSigCh := make(chan struct{})
			go func() {
				defer signal.Stop(sigCh)

				select {
				case <-stopSigCh:
					return
				case sig := <-sigCh:
					// Interrupt
					errCh <- fmt.Errorf("text input interrupted by %v", sig)
				}
			}()

			// Ensure we stop the signal handler when done
			defer close(stopSigCh)

			// Read line
			reader := bufio.NewReader(os.Stdin)
			inputStr, readErr := reader.ReadString('\n')
			if readErr != nil {
				errCh <- fmt.Errorf("failed to read input: %w", readErr)
				return
			}
			inputStr = strings.TrimSpace(inputStr)

			// Use default value if input is empty
			if inputStr == "" && params.DefaultValue != "" {
				inputStr = params.DefaultValue
			}

			// Send input
			inputCh <- inputStr
		}
	}()

	// Wait for input or timeout
	select {
	case <-doneCh:
		// Input completed successfully

	case err := <-errCh:
		// Error reading input
		fmt.Fprintln(os.Stderr, clearLine+"Error reading input:", err)
		response.Status = "error"
		response.ErrorMessage = err.Error()
		return response, err

	case input = <-inputCh:
		// Got input

	case <-timeoutCh:
		// Timeout
		fmt.Fprintln(os.Stderr, clearLine+"Prompt timed out")
		response.Status = "timeout"
		response.ErrorMessage = fmt.Sprintf("Prompt timed out after %d seconds", params.TimeoutSecs)
		return response, nil
	}

	response.Value = input
	return response, nil
}

// HandleAndGenerateResponse handles a prompt request and generates a JSON-RPC response
func (h *InteractivePromptHandler) HandleAndGenerateResponse(message []byte) ([]byte, error) {
	// Handle the prompt
	promptResp, err := h.HandlePrompt(message)
	if err != nil {
		slog.Error("Error handling prompt", "error", err)
	}

	// Generate a JSON-RPC request to send back to the server
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      fmt.Sprintf("prompt_response_%s", promptResp.InputID),
		"method":  "interactive/userInput",
		"params":  promptResp,
	}

	// Marshal to JSON
	response, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return response, nil
}

// Helper function to get string parameters
func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if val, ok := params[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}
