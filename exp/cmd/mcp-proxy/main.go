// mcp-proxy is a transparent proxy for MCP that can monitor and log all transport types
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// JSONRPCMessage represents a JSON-RPC message (request or response)
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   interface{}     `json:"error,omitempty"`
}

// Logger interface for different output formats
type Logger interface {
	LogRequest(msg JSONRPCMessage, raw []byte)
	LogResponse(msg JSONRPCMessage, raw []byte)
	LogError(err error)
	LogInfo(info string)
}

// ConsoleLogger logs to console with pretty printing
type ConsoleLogger struct {
	verbose    bool
	timestamps bool
	mu         sync.Mutex
}

func (l *ConsoleLogger) LogRequest(msg JSONRPCMessage, raw []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.timestamps {
		fmt.Printf("[%s] ", time.Now().Format("15:04:05.000"))
	}

	fmt.Printf("→ REQUEST: ")
	if l.verbose {
		pretty, _ := json.MarshalIndent(msg, "", "  ")
		fmt.Println(string(pretty))
	} else {
		fmt.Printf("method=%s id=%v\n", msg.Method, msg.ID)
	}
}

func (l *ConsoleLogger) LogResponse(msg JSONRPCMessage, raw []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.timestamps {
		fmt.Printf("[%s] ", time.Now().Format("15:04:05.000"))
	}

	fmt.Printf("← RESPONSE: ")
	if l.verbose {
		pretty, _ := json.MarshalIndent(msg, "", "  ")
		fmt.Println(string(pretty))
	} else {
		if msg.Error != nil {
			fmt.Printf("id=%v error=%v\n", msg.ID, msg.Error)
		} else {
			fmt.Printf("id=%v success\n", msg.ID)
		}
	}
}

func (l *ConsoleLogger) LogError(err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.timestamps {
		fmt.Printf("[%s] ", time.Now().Format("15:04:05.000"))
	}
	fmt.Printf("✗ ERROR: %v\n", err)
}

func (l *ConsoleLogger) LogInfo(info string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.timestamps {
		fmt.Printf("[%s] ", time.Now().Format("15:04:05.000"))
	}
	fmt.Printf("ℹ INFO: %s\n", info)
}

// StdioProxy handles stdio transport proxying
type StdioProxy struct {
	cmd         *exec.Cmd
	logger      Logger
	wrapWithSpy bool
	spyFlags    []string
}

func NewStdioProxy(command string, args []string, logger Logger, wrapWithSpy bool, spyFlags []string) *StdioProxy {
	return &StdioProxy{
		cmd:         exec.Command(command, args...),
		logger:      logger,
		wrapWithSpy: wrapWithSpy,
		spyFlags:    spyFlags,
	}
}

func (p *StdioProxy) Start() error {
	var cmd *exec.Cmd

	if p.wrapWithSpy {
		// If wrapping with mcpspy, construct the command
		spyArgs := append(p.spyFlags, "--")
		spyArgs = append(spyArgs, p.cmd.Path)
		spyArgs = append(spyArgs, p.cmd.Args[1:]...)
		cmd = exec.Command("mcpspy", spyArgs...)
	} else {
		cmd = p.cmd
	}

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Log stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			p.logger.LogInfo(fmt.Sprintf("stderr: %s", scanner.Text()))
		}
	}()

	// Proxy stdin -> command
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()

			// Parse and log request
			var msg JSONRPCMessage
			if err := json.Unmarshal([]byte(line), &msg); err == nil {
				p.logger.LogRequest(msg, []byte(line))
			}

			// Forward to command
			fmt.Fprintln(stdin, line)
		}
	}()

	// Proxy command -> stdout
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse and log response
		var msg JSONRPCMessage
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			p.logger.LogResponse(msg, []byte(line))
		}

		// Forward to stdout
		fmt.Println(line)
	}

	return cmd.Wait()
}

// TCPProxy handles TCP socket proxying
type TCPProxy struct {
	listenAddr  string
	command     string
	args        []string
	logger      Logger
	wrapWithSpy bool
	spyFlags    []string
}

func NewTCPProxy(listenAddr, command string, args []string, logger Logger, wrapWithSpy bool, spyFlags []string) *TCPProxy {
	return &TCPProxy{
		listenAddr:  listenAddr,
		command:     command,
		args:        args,
		logger:      logger,
		wrapWithSpy: wrapWithSpy,
		spyFlags:    spyFlags,
	}
}

func (p *TCPProxy) Start() error {
	listener, err := net.Listen("tcp", p.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", p.listenAddr, err)
	}
	defer listener.Close()

	p.logger.LogInfo(fmt.Sprintf("TCP proxy listening on %s", p.listenAddr))

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		p.logger.LogInfo("Shutting down...")
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check if this is due to shutdown
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			p.logger.LogError(fmt.Errorf("accept error: %w", err))
			continue
		}

		go p.handleConnection(conn)
	}

	return nil
}

func (p *TCPProxy) handleConnection(conn net.Conn) {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	p.logger.LogInfo(fmt.Sprintf("New connection from %s", clientAddr))

	// Create the command
	var cmd *exec.Cmd

	if p.wrapWithSpy {
		// If wrapping with mcpspy, construct the command
		spyArgs := append(p.spyFlags, "--")
		spyArgs = append(spyArgs, p.command)
		spyArgs = append(spyArgs, p.args...)
		cmd = exec.Command("mcpspy", spyArgs...)
	} else {
		cmd = exec.Command(p.command, p.args...)
	}

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		p.logger.LogError(fmt.Errorf("failed to create stdin pipe: %w", err))
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		p.logger.LogError(fmt.Errorf("failed to create stdout pipe: %w", err))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		p.logger.LogError(fmt.Errorf("failed to create stderr pipe: %w", err))
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		p.logger.LogError(fmt.Errorf("failed to start command: %w", err))
		return
	}

	// Handler for stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			p.logger.LogInfo(fmt.Sprintf("[%s] stderr: %s", clientAddr, scanner.Text()))
		}
	}()

	// Handler for conn -> stdin
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()

			// Parse and log request
			var msg JSONRPCMessage
			if err := json.Unmarshal([]byte(line), &msg); err == nil {
				p.logger.LogRequest(msg, []byte(line))
			}

			// Forward to command
			fmt.Fprintln(stdin, line)
		}
		stdin.Close()
	}()

	// Handler for stdout -> conn
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse and log response
		var msg JSONRPCMessage
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			p.logger.LogResponse(msg, []byte(line))
		}

		// Forward to connection
		fmt.Fprintln(conn, line)
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		p.logger.LogError(fmt.Errorf("command exited with error: %w", err))
	}

	p.logger.LogInfo(fmt.Sprintf("Connection from %s closed", clientAddr))
}

// HTTPProxy handles HTTP/SSE transport proxying
type HTTPProxy struct {
	listenAddr string
	targetURL  string
	logger     Logger
	proxy      *httputil.ReverseProxy
}

func NewHTTPProxy(listenAddr, targetURL string, logger Logger) (*HTTPProxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize the proxy to log requests/responses
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Log request
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(body))

			var msg JSONRPCMessage
			if err := json.Unmarshal(body, &msg); err == nil {
				logger.LogRequest(msg, body)
			}
		}
	}

	// Customize response handling
	proxy.ModifyResponse = func(resp *http.Response) error {
		// For SSE responses, we need special handling
		if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
			// TODO: Implement SSE proxying with logging
			logger.LogInfo("SSE stream established")
		} else {
			// Regular response
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			resp.Body = io.NopCloser(bytes.NewReader(body))

			var msg JSONRPCMessage
			if err := json.Unmarshal(body, &msg); err == nil {
				logger.LogResponse(msg, body)
			}
		}
		return nil
	}

	return &HTTPProxy{
		listenAddr: listenAddr,
		targetURL:  targetURL,
		logger:     logger,
		proxy:      proxy,
	}, nil
}

func (p *HTTPProxy) Start() error {
	p.logger.LogInfo(fmt.Sprintf("HTTP proxy listening on %s, forwarding to %s", p.listenAddr, p.targetURL))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.proxy.ServeHTTP(w, r)
	})

	return http.ListenAndServe(p.listenAddr, handler)
}

func main() {
	var (
		transport   = flag.String("transport", "stdio", "Transport type: stdio, tcp, http")
		verbose     = flag.Bool("v", false, "Verbose output")
		timestamps  = flag.Bool("t", false, "Include timestamps")
		listen      = flag.String("listen", ":8080", "Listen address for TCP/HTTP proxy")
		target      = flag.String("target", "", "Target URL for HTTP proxy")
		useSpyFlag  = flag.Bool("spy", false, "Wrap server command with mcpspy")
		spyVerbose  = flag.Bool("spy-v", false, "Pass -v flag to mcpspy")
		spyVeryVerb = flag.Bool("spy-vv", false, "Pass -vv flag to mcpspy")
		spyPretty   = flag.Bool("spy-pretty", false, "Pass -pretty flag to mcpspy")
		spyFile     = flag.String("spy-f", "", "Pass -f flag to mcpspy for recording")
	)
	flag.Parse()

	// Create logger
	logger := &ConsoleLogger{
		verbose:    *verbose,
		timestamps: *timestamps,
	}

	// Build mcpspy flags
	var spyFlags []string
	if *spyVerbose {
		spyFlags = append(spyFlags, "-v")
	}
	if *spyVeryVerb {
		spyFlags = append(spyFlags, "-vv")
	}
	if *spyPretty {
		spyFlags = append(spyFlags, "-pretty")
	}
	if *spyFile != "" {
		spyFlags = append(spyFlags, "-f", *spyFile)
	}

	switch *transport {
	case "stdio":
		if flag.NArg() == 0 {
			log.Fatal("No command specified for stdio proxy")
		}

		command := flag.Arg(0)
		args := flag.Args()[1:]

		logger.LogInfo(fmt.Sprintf("Starting stdio proxy for: %s %s", command, strings.Join(args, " ")))

		proxy := NewStdioProxy(command, args, logger, *useSpyFlag, spyFlags)
		if err := proxy.Start(); err != nil {
			log.Fatalf("Proxy error: %v", err)
		}

	case "tcp":
		if flag.NArg() == 0 {
			log.Fatal("No command specified for TCP proxy")
		}

		command := flag.Arg(0)
		args := flag.Args()[1:]

		logger.LogInfo(fmt.Sprintf("Starting TCP proxy on %s for: %s %s", *listen, command, strings.Join(args, " ")))

		proxy := NewTCPProxy(*listen, command, args, logger, *useSpyFlag, spyFlags)
		if err := proxy.Start(); err != nil {
			log.Fatalf("Proxy error: %v", err)
		}

	case "http":
		if *target == "" {
			*target = "http://localhost:3001"
		}

		proxy, err := NewHTTPProxy(*listen, *target, logger)
		if err != nil {
			log.Fatalf("Failed to create HTTP proxy: %v", err)
		}

		if err := proxy.Start(); err != nil {
			log.Fatalf("Proxy error: %v", err)
		}

	default:
		log.Fatalf("Unknown transport: %s", *transport)
	}
}
