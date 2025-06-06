// mcp-tunnel client - runs locally to tunnel MCP servers to the cloud
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// TunnelClient manages the local side of the tunnel
type TunnelClient struct {
	serverURL   string
	tunnelID    string
	token       string
	wsURL       string
	publicURL   string
	localCmd    string
	localArgs   []string
	transport   string
	conn        *websocket.Conn
	localServer *LocalServer
	mu          sync.Mutex
}

// LocalServer wraps the local MCP server
type LocalServer struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	transport string
	client    *http.Client
	baseURL   string
}

// Message types
type TunnelMessage struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type CreateTunnelRequest struct {
	Transport string `json:"transport"`
}

type CreateTunnelResponse struct {
	TunnelID string `json:"tunnel_id"`
	Token    string `json:"token"`
	URL      string `json:"url"`
	WsURL    string `json:"ws_url"`
}

func NewTunnelClient(serverURL, localCmd string, localArgs []string, transport string) *TunnelClient {
	return &TunnelClient{
		serverURL: serverURL,
		localCmd:  localCmd,
		localArgs: localArgs,
		transport: transport,
	}
}

// Connect creates a tunnel and establishes WebSocket connection
func (c *TunnelClient) Connect() error {
	// Create tunnel
	req := CreateTunnelRequest{Transport: c.transport}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(c.serverURL+"/tunnels", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error: %s - %s", resp.Status, string(body))
	}

	var tunnelResp CreateTunnelResponse
	if err := json.NewDecoder(resp.Body).Decode(&tunnelResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	c.tunnelID = tunnelResp.TunnelID
	c.token = tunnelResp.Token
	c.publicURL = tunnelResp.URL
	c.wsURL = tunnelResp.WsURL

	// Start local server
	if err := c.startLocalServer(); err != nil {
		return fmt.Errorf("failed to start local server: %w", err)
	}

	// Connect WebSocket
	u, err := url.Parse(c.wsURL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %w", err)
	}

	header := http.Header{}
	header.Add("Authorization", c.token)

	c.conn, _, err = websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	// Start message handlers
	go c.handleMessages()
	go c.pingLoop()

	return nil
}

// startLocalServer starts the local MCP server
func (c *TunnelClient) startLocalServer() error {
	switch c.transport {
	case "stdio":
		server := &LocalServer{
			cmd:       exec.Command(c.localCmd, c.localArgs...),
			transport: c.transport,
		}

		var err error
		server.stdin, err = server.cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdin pipe: %w", err)
		}

		server.stdout, err = server.cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		if err := server.cmd.Start(); err != nil {
			return fmt.Errorf("failed to start command: %w", err)
		}

		c.localServer = server
		log.Printf("Started local stdio server: %s %v", c.localCmd, c.localArgs)

	case "sse", "http":
		// For HTTP-based transports, assume the server is already running
		baseURL := c.localCmd
		if !strings.HasPrefix(baseURL, "http") {
			baseURL = "http://" + baseURL
		}

		c.localServer = &LocalServer{
			transport: c.transport,
			client:    &http.Client{Timeout: 30 * time.Second},
			baseURL:   baseURL,
		}
		log.Printf("Connecting to local %s server at %s", c.transport, baseURL)

	default:
		return fmt.Errorf("unsupported transport: %s", c.transport)
	}

	return nil
}

// handleMessages handles incoming messages from the tunnel server
func (c *TunnelClient) handleMessages() {
	for {
		var msg TunnelMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		switch msg.Type {
		case "ping":
			// Respond with pong
			pong := TunnelMessage{Type: "pong"}
			if err := c.conn.WriteJSON(pong); err != nil {
				log.Printf("Failed to send pong: %v", err)
			}

		case "request":
			// Handle regular MCP request
			go c.handleRequest(msg)

		case "sse_request":
			// Handle SSE request
			go c.handleSSERequest(msg)
		}
	}
}

// handleRequest processes a regular MCP request
func (c *TunnelClient) handleRequest(msg TunnelMessage) {
	var response TunnelMessage
	response.Type = "response"
	response.ID = msg.ID

	switch c.localServer.transport {
	case "stdio":
		// Forward to stdio server
		if _, err := c.localServer.stdin.Write(msg.Payload); err != nil {
			response.Error = fmt.Sprintf("Failed to write to server: %v", err)
		} else {
			// Read response with proper JSON message boundary detection
			scanner := bufio.NewScanner(c.localServer.stdout)
			scanner.Buffer(make([]byte, 64*1024), 64*1024)

			var responseData []byte
			bracketCount := 0
			inString := false
			escape := false

			for scanner.Scan() {
				line := scanner.Bytes()

				// Simple JSON object detection
				for _, b := range line {
					if escape {
						escape = false
						continue
					}

					switch b {
					case '\\':
						if inString {
							escape = true
						}
					case '"':
						if !escape {
							inString = !inString
						}
					case '{':
						if !inString {
							bracketCount++
						}
					case '}':
						if !inString {
							bracketCount--
						}
					}
				}

				responseData = append(responseData, line...)
				responseData = append(responseData, '\n')

				// Check if we have a complete JSON object
				if bracketCount == 0 && len(responseData) > 0 {
					// Try to parse as valid JSON
					var msg json.RawMessage
					if err := json.Unmarshal(responseData, &msg); err == nil {
						response.Payload = responseData
						break
					}
				}
			}

			if err := scanner.Err(); err != nil {
				response.Error = fmt.Sprintf("Failed to read response: %v", err)
			} else if len(response.Payload) == 0 {
				response.Error = "No response received"
			}
		}

	case "http":
		// Forward to HTTP server
		endpoint := c.localServer.baseURL + "/mcp"
		resp, err := c.localServer.client.Post(endpoint, "application/json", bytes.NewBuffer(msg.Payload))
		if err != nil {
			response.Error = fmt.Sprintf("Failed to forward request: %v", err)
		} else {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			response.Payload = body
		}
	}

	// Send response back to tunnel
	if err := c.conn.WriteJSON(response); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

// handleSSERequest processes an SSE request
func (c *TunnelClient) handleSSERequest(msg TunnelMessage) {
	if c.localServer.transport != "sse" {
		response := TunnelMessage{
			Type:  "sse_response",
			ID:    msg.ID,
			Error: "Server not configured for SSE",
		}
		c.conn.WriteJSON(response)
		return
	}

	// Forward to SSE endpoint
	endpoint := c.localServer.baseURL + "/message"
	resp, err := c.localServer.client.Post(endpoint, "application/json", bytes.NewBuffer(msg.Payload))
	if err != nil {
		log.Printf("Failed to forward SSE request: %v", err)
		return
	}
	defer resp.Body.Close()

	// Forward the response through the tunnel
	body, _ := io.ReadAll(resp.Body)
	response := TunnelMessage{
		Type:    "sse_response",
		ID:      msg.ID,
		Payload: body,
	}
	c.conn.WriteJSON(response)
}

// pingLoop sends periodic pings to keep connection alive
func (c *TunnelClient) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		if c.conn == nil {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()

		if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Printf("Failed to send ping: %v", err)
			return
		}
	}
}

// Close closes the tunnel connection
func (c *TunnelClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}

	if c.localServer != nil && c.localServer.cmd != nil {
		c.localServer.cmd.Process.Kill()
	}

	return nil
}

func main() {
	var (
		serverURL = flag.String("server", "https://mcp-tunnel.run", "Tunnel server URL")
		transport = flag.String("transport", "stdio", "Transport type: stdio, sse, http")
		verbose   = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Usage: mcp-tunnel [options] -- command [args...]")
	}

	cmd := flag.Arg(0)
	args := flag.Args()[1:]

	// Create tunnel client
	client := NewTunnelClient(*serverURL, cmd, args, *transport)

	// Connect to tunnel server
	log.Println("Creating tunnel...")
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to create tunnel: %v", err)
	}

	log.Printf("Tunnel established!")
	log.Printf("Public URL: %s", client.publicURL)
	log.Printf("Local command: %s %v", cmd, args)
	log.Printf("Transport: %s", *transport)

	if *verbose {
		log.Printf("Tunnel ID: %s", client.tunnelID)
		log.Printf("WebSocket URL: %s", client.wsURL)
	}

	// Wait for interruption
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	client.Close()
}
