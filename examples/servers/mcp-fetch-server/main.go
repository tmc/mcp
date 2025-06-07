package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

type FetchServer struct {
	client          *http.Client
	allowedDomains  []string
	blockedDomains  []string
	userAgent       string
	maxResponseSize int64
}

func NewFetchServer() *FetchServer {
	return &FetchServer{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent:       "MCP-Fetch-Server/1.0",
		maxResponseSize: 10 * 1024 * 1024, // 10MB limit
	}
}

func (fs *FetchServer) SetAllowedDomains(domains []string) {
	fs.allowedDomains = domains
}

func (fs *FetchServer) SetBlockedDomains(domains []string) {
	fs.blockedDomains = domains
}

func (fs *FetchServer) validateURL(urlStr string) error {
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
	for _, blocked := range fs.blockedDomains {
		if strings.Contains(hostname, blocked) {
			return fmt.Errorf("domain is blocked: %s", hostname)
		}
	}

	// If allowed domains are specified, check them
	if len(fs.allowedDomains) > 0 {
		allowed := false
		for _, allowedDomain := range fs.allowedDomains {
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

func (fs *FetchServer) fetchURL(urlStr string) (string, string, error) {
	if err := fs.validateURL(urlStr); err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", fs.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := fs.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, fs.maxResponseSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")

	return string(body), contentType, nil
}

func (fs *FetchServer) htmlToText(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	var extractText func(*html.Node) string
	extractText = func(n *html.Node) string {
		if n.Type == html.TextNode {
			return n.Data
		}

		// Skip script and style tags
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return ""
		}

		var result strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			result.WriteString(extractText(c))
		}

		// Add spacing for block elements
		if n.Type == html.ElementNode {
			switch n.Data {
			case "div", "p", "br", "h1", "h2", "h3", "h4", "h5", "h6", "li":
				result.WriteString("\n")
			}
		}

		return result.String()
	}

	text := extractText(doc)

	// Clean up whitespace
	text = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(text, "\n\n")
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text, nil
}

func main() {
	// Create server with name and version
	srv := mcp.NewServer("fetch-server", "1.0.0")

	// Initialize fetch server
	fs := NewFetchServer()

	// Register fetch tool
	srv.RegisterTool("fetch", "Fetch content from a URL", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var urlRaw json.RawMessage
		var exists bool
		if urlRaw, exists = args["url"]; !exists {
			return nil, fmt.Errorf("missing required argument: url")
		}

		var urlStr string
		if err := json.Unmarshal(urlRaw, &urlStr); err != nil {
			return nil, fmt.Errorf("invalid url argument: %w", err)
		}

		content, contentType, err := fs.fetchURL(urlStr)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error fetching URL: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		result := map[string]interface{}{
			"url":         urlStr,
			"content":     content,
			"contentType": contentType,
			"size":        len(content),
		}

		responseJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		log.Printf("Fetched URL: %s (%d bytes, %s)", urlStr, len(content), contentType)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}, nil
	})

	// Register fetch_text tool
	srv.RegisterTool("fetch_text", "Fetch content from a URL and convert HTML to plain text", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var urlRaw json.RawMessage
		var exists bool
		if urlRaw, exists = args["url"]; !exists {
			return nil, fmt.Errorf("missing required argument: url")
		}

		var urlStr string
		if err := json.Unmarshal(urlRaw, &urlStr); err != nil {
			return nil, fmt.Errorf("invalid url argument: %w", err)
		}

		content, contentType, err := fs.fetchURL(urlStr)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error fetching URL: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		var textContent string
		if strings.Contains(contentType, "text/html") {
			textContent, err = fs.htmlToText(content)
			if err != nil {
				return &modelcontextprotocol.CallToolResult{
					Content: []modelcontextprotocol.Content{
						modelcontextprotocol.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Error converting HTML to text: %s", err.Error()),
						},
					},
					IsError: boolPtr(true),
				}, nil
			}
		} else {
			textContent = content
		}

		result := map[string]interface{}{
			"url":          urlStr,
			"text":         textContent,
			"contentType":  contentType,
			"originalSize": len(content),
			"textSize":     len(textContent),
		}

		responseJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		log.Printf("Fetched and converted URL: %s (%d -> %d bytes)", urlStr, len(content), len(textContent))

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}, nil
	})

	// Register get_headers tool
	srv.RegisterTool("get_headers", "Get HTTP headers for a URL without fetching the full content", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var urlRaw json.RawMessage
		var exists bool
		if urlRaw, exists = args["url"]; !exists {
			return nil, fmt.Errorf("missing required argument: url")
		}

		var urlStr string
		if err := json.Unmarshal(urlRaw, &urlStr); err != nil {
			return nil, fmt.Errorf("invalid url argument: %w", err)
		}

		if err := fs.validateURL(urlStr); err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error validating URL: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		req, err := http.NewRequest("HEAD", urlStr, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", fs.userAgent)

		resp, err := fs.client.Do(req)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error getting headers: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		defer resp.Body.Close()

		headers := make(map[string][]string)
		for name, values := range resp.Header {
			headers[name] = values
		}

		result := map[string]interface{}{
			"url":        urlStr,
			"statusCode": resp.StatusCode,
			"status":     resp.Status,
			"headers":    headers,
		}

		responseJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		log.Printf("Got headers for URL: %s (status: %d)", urlStr, resp.StatusCode)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}, nil
	})

	// Start server with stdio transport
	transport := mcp.StdioTransport{}
	log.Printf("Fetch server running on stdio")

	if err := srv.Serve(context.Background(), transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
