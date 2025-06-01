package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tmc/mcp"
)

const (
	ServerName    = "mcp-http-server"
	ServerVersion = "0.1.0"
)

type HTTPResponse struct {
	URL         string              `json:"url"`
	Method      string              `json:"method"`
	StatusCode  int                 `json:"status_code"`
	Status      string              `json:"status"`
	Headers     map[string][]string `json:"headers"`
	Body        string              `json:"body"`
	Size        int                 `json:"size"`
	Duration    string              `json:"duration"`
}

type HTTPClient struct {
	client          *http.Client
	allowedDomains  []string
	blockedDomains  []string
	maxResponseSize int64
	timeout         time.Duration
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxResponseSize: 10 * 1024 * 1024, // 10MB limit
		timeout:         30 * time.Second,
	}
}

func (hc *HTTPClient) SetAllowedDomains(domains []string) {
	hc.allowedDomains = domains
}

func (hc *HTTPClient) SetBlockedDomains(domains []string) {
	hc.blockedDomains = domains
}

func (hc *HTTPClient) validateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", parsedURL.Scheme)
	}

	hostname := parsedURL.Hostname()

	// Check blocked domains first
	for _, blocked := range hc.blockedDomains {
		if strings.Contains(hostname, blocked) {
			return fmt.Errorf("domain is blocked: %s", hostname)
		}
	}

	// If allowed domains are specified, check them
	if len(hc.allowedDomains) > 0 {
		allowed := false
		for _, allowedDomain := range hc.allowedDomains {
			if strings.Contains(hostname, allowedDomain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("domain not in allowed list: %s", hostname)
		}
	}

	// Block private IP ranges and localhost
	if isPrivateIP(hostname) {
		return fmt.Errorf("private IP addresses are not allowed: %s", hostname)
	}

	return nil
}

func isPrivateIP(hostname string) bool {
	// Simple check for common private IP patterns and localhost
	privatePatterns := []string{
		"localhost",
		"127.",
		"10.",
		"192.168.",
		"172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.",
		"172.24.", "172.25.", "172.26.", "172.27.",
		"172.28.", "172.29.", "172.30.", "172.31.",
	}

	for _, pattern := range privatePatterns {
		if strings.HasPrefix(hostname, pattern) {
			return true
		}
	}
	return false
}

func (hc *HTTPClient) makeRequest(method, urlStr string, headers map[string]string, body string) (HTTPResponse, error) {
	if err := hc.validateURL(urlStr); err != nil {
		return HTTPResponse{}, err
	}

	startTime := time.Now()

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, urlStr, reqBody)
	if err != nil {
		return HTTPResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set default User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "MCP-HTTP-Server/1.0")
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		return HTTPResponse{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, hc.maxResponseSize)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return HTTPResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	return HTTPResponse{
		URL:        urlStr,
		Method:     method,
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header,
		Body:       string(responseBody),
		Size:       len(responseBody),
		Duration:   duration.String(),
	}, nil
}

func main() {
	log.SetOutput(os.Stderr)
	log.Println("Starting MCP HTTP Server...")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	httpClient := NewHTTPClient()

	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("An HTTP client server for making REST API calls and web requests"),
	)

	registerHTTPTools(server, httpClient)

	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerHTTPTools(server *mcp.Server, hc *HTTPClient) {
	// HTTP GET tool
	getTool := mcp.Tool{
		Name:        "http_get",
		Description: "Make an HTTP GET request",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "The URL to send the GET request to"
				},
				"headers": {
					"type": "object",
					"description": "Optional HTTP headers as key-value pairs",
					"additionalProperties": {
						"type": "string"
					}
				}
			},
			"required": ["url"]
		}`),
	}

	server.RegisterTool(getTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		url, ok := params["url"].(string)
		if !ok || url == "" {
			return nil, fmt.Errorf("url is required and must be a string")
		}

		headers := make(map[string]string)
		if h, ok := params["headers"].(map[string]interface{}); ok {
			for key, value := range h {
				if strValue, ok := value.(string); ok {
					headers[key] = strValue
				}
			}
		}

		response, err := hc.makeRequest("GET", url, headers, "")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error making GET request: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(response, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	// HTTP POST tool
	postTool := mcp.Tool{
		Name:        "http_post",
		Description: "Make an HTTP POST request",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "The URL to send the POST request to"
				},
				"body": {
					"type": "string",
					"description": "The request body (JSON, form data, or plain text)"
				},
				"headers": {
					"type": "object",
					"description": "Optional HTTP headers as key-value pairs",
					"additionalProperties": {
						"type": "string"
					}
				}
			},
			"required": ["url"]
		}`),
	}

	server.RegisterTool(postTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		url, ok := params["url"].(string)
		if !ok || url == "" {
			return nil, fmt.Errorf("url is required and must be a string")
		}

		body := ""
		if b, ok := params["body"].(string); ok {
			body = b
		}

		headers := make(map[string]string)
		if h, ok := params["headers"].(map[string]interface{}); ok {
			for key, value := range h {
				if strValue, ok := value.(string); ok {
					headers[key] = strValue
				}
			}
		}

		// Set default Content-Type if not provided and body is not empty
		if body != "" && headers["Content-Type"] == "" {
			headers["Content-Type"] = "application/json"
		}

		response, err := hc.makeRequest("POST", url, headers, body)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error making POST request: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(response, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	// HTTP PUT tool
	putTool := mcp.Tool{
		Name:        "http_put",
		Description: "Make an HTTP PUT request",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "The URL to send the PUT request to"
				},
				"body": {
					"type": "string",
					"description": "The request body (JSON, form data, or plain text)"
				},
				"headers": {
					"type": "object",
					"description": "Optional HTTP headers as key-value pairs",
					"additionalProperties": {
						"type": "string"
					}
				}
			},
			"required": ["url"]
		}`),
	}

	server.RegisterTool(putTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		url, ok := params["url"].(string)
		if !ok || url == "" {
			return nil, fmt.Errorf("url is required and must be a string")
		}

		body := ""
		if b, ok := params["body"].(string); ok {
			body = b
		}

		headers := make(map[string]string)
		if h, ok := params["headers"].(map[string]interface{}); ok {
			for key, value := range h {
				if strValue, ok := value.(string); ok {
					headers[key] = strValue
				}
			}
		}

		if body != "" && headers["Content-Type"] == "" {
			headers["Content-Type"] = "application/json"
		}

		response, err := hc.makeRequest("PUT", url, headers, body)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error making PUT request: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(response, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	// HTTP DELETE tool
	deleteTool := mcp.Tool{
		Name:        "http_delete",
		Description: "Make an HTTP DELETE request",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "The URL to send the DELETE request to"
				},
				"headers": {
					"type": "object",
					"description": "Optional HTTP headers as key-value pairs",
					"additionalProperties": {
						"type": "string"
					}
				}
			},
			"required": ["url"]
		}`),
	}

	server.RegisterTool(deleteTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		url, ok := params["url"].(string)
		if !ok || url == "" {
			return nil, fmt.Errorf("url is required and must be a string")
		}

		headers := make(map[string]string)
		if h, ok := params["headers"].(map[string]interface{}); ok {
			for key, value := range h {
				if strValue, ok := value.(string); ok {
					headers[key] = strValue
				}
			}
		}

		response, err := hc.makeRequest("DELETE", url, headers, "")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error making DELETE request: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(response, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	// Generic HTTP request tool
	requestTool := mcp.Tool{
		Name:        "http_request",
		Description: "Make a custom HTTP request with any method",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"method": {
					"type": "string",
					"description": "HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, etc.)"
				},
				"url": {
					"type": "string",
					"description": "The URL to send the request to"
				},
				"body": {
					"type": "string",
					"description": "The request body (for methods that support it)"
				},
				"headers": {
					"type": "object",
					"description": "Optional HTTP headers as key-value pairs",
					"additionalProperties": {
						"type": "string"
					}
				}
			},
			"required": ["method", "url"]
		}`),
	}

	server.RegisterTool(requestTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		method, ok := params["method"].(string)
		if !ok || method == "" {
			return nil, fmt.Errorf("method is required and must be a string")
		}

		url, ok := params["url"].(string)
		if !ok || url == "" {
			return nil, fmt.Errorf("url is required and must be a string")
		}

		body := ""
		if b, ok := params["body"].(string); ok {
			body = b
		}

		headers := make(map[string]string)
		if h, ok := params["headers"].(map[string]interface{}); ok {
			for key, value := range h {
				if strValue, ok := value.(string); ok {
					headers[key] = strValue
				}
			}
		}

		response, err := hc.makeRequest(strings.ToUpper(method), url, headers, body)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error making %s request: %v", method, err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(response, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	log.Println("Registered HTTP tools: http_get, http_post, http_put, http_delete, http_request")
}