package mcptest

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/tmc/mcp"
)

type TestServer struct {
	t          *testing.T
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	client     *mcp.Client
	debugLog   io.Writer
	mu         sync.Mutex
	msgCounter int
	done       chan struct{}
}

type ServerOption func(*TestServer)

func WithDebugLog(w io.Writer) ServerOption {
	return func(s *TestServer) {
		s.debugLog = w
	}
}

func WithArgs(args ...string) ServerOption {
	return func(s *TestServer) {
		s.cmd.Args = append(s.cmd.Args, args...)
	}
}

func NewTestServer(t *testing.T, serverPath string, opts ...ServerOption) *TestServer {
	t.Helper()

	s := &TestServer{
		t:        t,
		debugLog: io.Discard,
		done:     make(chan struct{}),
	}

	// Create command but don't start it yet
	s.cmd = exec.Command(serverPath)

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Set up pipes
	var err error
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}
	s.stderr, err = s.cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	// Log command being started
	fmt.Fprintf(s.debugLog, "Starting server: %v\n", s.cmd.Args)

	// Start process
	if err := s.cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Start stderr logger
	go s.logStderr()

	// Create debug transport
	transport := newDebugTransport(
		rwc{s.stdin, s.stdout},
		s.debugLog,
	)

	// Create client
	s.client = mcp.NewClient(transport)

	// Monitor process
	go func() {
		err := s.cmd.Wait()
		fmt.Fprintf(s.debugLog, "Server process exited: %v\n", err)
		close(s.done)
	}()

	return s
}

func (s *TestServer) Initialize(ctx context.Context) (*mcp.InitializeReply, error) {
	// Send initialize request
	reply, err := s.client.Initialize(ctx, mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	})
	if err != nil {
		// Check if process exited
		select {
		case <-s.done:
			return nil, fmt.Errorf("server process exited before initialization completed")
		default:
			return nil, err
		}
	}

	// Send initialized notification
	s.client.SendNotification(ctx, "initialized", nil)

	return reply, nil
}

func (s *TestServer) Call(method string, params interface{}) (json.RawMessage, error) {
	// Check if process is still running
	select {
	case <-s.done:
		return nil, fmt.Errorf("server process is not running")
	default:
	}

	s.mu.Lock()
	id := s.msgCounter
	s.msgCounter++
	s.mu.Unlock()

	req := struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      int         `json:"id"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params,omitempty"`
	}{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	fmt.Fprintf(s.debugLog, "-> %s\n", reqBytes)

	if _, err := fmt.Fprintf(s.stdin, "%s\n", reqBytes); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Read response with timeout
	type readResult struct {
		resp json.RawMessage
		err  error
	}
	ch := make(chan readResult, 1)

	go func() {
		var resp struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      int             `json:"id"`
			Result  json.RawMessage `json:"result,omitempty"`
			Error   *struct {
				Code    int             `json:"code"`
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data,omitempty"`
			} `json:"error,omitempty"`
		}

		decoder := json.NewDecoder(s.stdout)
		err := decoder.Decode(&resp)
		if err != nil {
			ch <- readResult{err: fmt.Errorf("decode response: %w", err)}
			return
		}

		respBytes, _ := json.Marshal(resp)
		fmt.Fprintf(s.debugLog, "<- %s\n", respBytes)

		if resp.Error != nil {
			ch <- readResult{err: fmt.Errorf("server error %d: %s", resp.Error.Code, resp.Error.Message)}
			return
		}

		ch <- readResult{resp: resp.Result}
	}()

	select {
	case result := <-ch:
		return result.resp, result.err
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	case <-s.done:
		return nil, fmt.Errorf("server process exited while waiting for response")
	}
}

func (s *TestServer) Close() error {
	if s.client != nil {
		s.client.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		<-s.done // Wait for process to exit
	}
	return nil
}

func (s *TestServer) logStderr() {
	scanner := bufio.NewScanner(s.stderr)
	for scanner.Scan() {
		fmt.Fprintf(s.debugLog, "ERR: %s\n", scanner.Text())
	}
}
