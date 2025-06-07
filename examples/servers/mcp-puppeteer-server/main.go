package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/chromedp/chromedp"
	"github.com/tmc/mcp"
)

type PuppeteerServer struct {
	ctx    context.Context
	cancel context.CancelFunc
	chrome context.Context
	mu     sync.RWMutex
	// Store screenshots
	screenshots map[string]string
	// Store console logs
	consoleLogs []string
}

func NewPuppeteerServer() (*PuppeteerServer, error) {
	ps := &PuppeteerServer{
		screenshots: make(map[string]string),
		consoleLogs: make([]string, 0),
	}

	// Initialize browser context
	ctx, cancel := chromedp.NewContext(context.Background())
	ps.ctx = ctx
	ps.cancel = cancel
	ps.chrome = ctx

	return ps, nil
}

func (s *PuppeteerServer) navigate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var navArgs struct {
		URL            string                 `json:"url"`
		LaunchOptions  map[string]interface{} `json:"launchOptions,omitempty"`
		AllowDangerous bool                   `json:"allowDangerous,omitempty"`
	}
	if err := json.Unmarshal(request.Arguments, &navArgs); err != nil {
		return nil, err
	}

	if err := chromedp.Run(s.chrome, chromedp.Navigate(navArgs.URL)); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("Navigated to %s", navArgs.URL),
			},
		},
	}, nil
}

func (s *PuppeteerServer) screenshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var ssArgs struct {
		Name     string `json:"name"`
		Selector string `json:"selector,omitempty"`
		Width    int    `json:"width,omitempty"`
		Height   int    `json:"height,omitempty"`
		Encoded  bool   `json:"encoded,omitempty"`
	}
	if err := json.Unmarshal(request.Arguments, &ssArgs); err != nil {
		return nil, err
	}

	// Default dimensions
	if ssArgs.Width == 0 {
		ssArgs.Width = 800
	}
	if ssArgs.Height == 0 {
		ssArgs.Height = 600
	}

	var buf []byte
	tasks := chromedp.Tasks{
		chromedp.EmulateViewport(int64(ssArgs.Width), int64(ssArgs.Height)),
	}

	if ssArgs.Selector != "" {
		tasks = append(tasks, chromedp.Screenshot(ssArgs.Selector, &buf, chromedp.NodeVisible))
	} else {
		tasks = append(tasks, chromedp.CaptureScreenshot(&buf))
	}

	if err := chromedp.Run(s.chrome, tasks); err != nil {
		return nil, err
	}

	encoded := base64.StdEncoding.EncodeToString(buf)

	s.mu.Lock()
	s.screenshots[ssArgs.Name] = encoded
	s.mu.Unlock()

	content := []interface{}{
		map[string]interface{}{
			"type": "text",
			"text": fmt.Sprintf("Screenshot '%s' taken at %dx%d", ssArgs.Name, ssArgs.Width, ssArgs.Height),
		},
	}

	if ssArgs.Encoded {
		content = append(content, map[string]interface{}{
			"type": "text",
			"text": fmt.Sprintf("data:image/png;base64,%s", encoded),
		})
	} else {
		content = append(content, map[string]interface{}{
			"type":     "image",
			"data":     encoded,
			"mimeType": "image/png",
		})
	}

	return &mcp.CallToolResult{
		Content: content,
	}, nil
}

func (s *PuppeteerServer) click(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var clickArgs struct {
		Selector string `json:"selector"`
	}
	if err := json.Unmarshal(request.Arguments, &clickArgs); err != nil {
		return nil, err
	}

	if err := chromedp.Run(s.chrome, chromedp.Click(clickArgs.Selector)); err != nil {
		return nil, fmt.Errorf("failed to click %s: %w", clickArgs.Selector, err)
	}

	return &mcp.CallToolResult{
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("Clicked: %s", clickArgs.Selector),
			},
		},
	}, nil
}

func (s *PuppeteerServer) fill(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var fillArgs struct {
		Selector string `json:"selector"`
		Value    string `json:"value"`
	}
	if err := json.Unmarshal(request.Arguments, &fillArgs); err != nil {
		return nil, err
	}

	tasks := chromedp.Tasks{
		chromedp.WaitVisible(fillArgs.Selector),
		chromedp.SendKeys(fillArgs.Selector, fillArgs.Value),
	}

	if err := chromedp.Run(s.chrome, tasks); err != nil {
		return nil, fmt.Errorf("failed to fill %s: %w", fillArgs.Selector, err)
	}

	return &mcp.CallToolResult{
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("Filled %s with: %s", fillArgs.Selector, fillArgs.Value),
			},
		},
	}, nil
}

func (s *PuppeteerServer) selectOption(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var selectArgs struct {
		Selector string `json:"selector"`
		Value    string `json:"value"`
	}
	if err := json.Unmarshal(request.Arguments, &selectArgs); err != nil {
		return nil, err
	}

	tasks := chromedp.Tasks{
		chromedp.WaitVisible(selectArgs.Selector),
		chromedp.SetValue(selectArgs.Selector, selectArgs.Value),
	}

	if err := chromedp.Run(s.chrome, tasks); err != nil {
		return nil, fmt.Errorf("failed to select %s: %w", selectArgs.Selector, err)
	}

	return &mcp.CallToolResult{
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("Selected %s with: %s", selectArgs.Selector, selectArgs.Value),
			},
		},
	}, nil
}

func (s *PuppeteerServer) hover(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var hoverArgs struct {
		Selector string `json:"selector"`
	}
	if err := json.Unmarshal(request.Arguments, &hoverArgs); err != nil {
		return nil, err
	}

	tasks := chromedp.Tasks{
		chromedp.WaitVisible(hoverArgs.Selector),
		chromedp.MouseClickXY(0, 0), // Move mouse to element
	}

	if err := chromedp.Run(s.chrome, tasks); err != nil {
		return nil, fmt.Errorf("failed to hover %s: %w", hoverArgs.Selector, err)
	}

	return &mcp.CallToolResult{
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("Hovered %s", hoverArgs.Selector),
			},
		},
	}, nil
}

func (s *PuppeteerServer) evaluate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var evalArgs struct {
		Script string `json:"script"`
	}
	if err := json.Unmarshal(request.Arguments, &evalArgs); err != nil {
		return nil, err
	}

	var result interface{}
	if err := chromedp.Run(s.chrome, chromedp.Evaluate(evalArgs.Script, &result)); err != nil {
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("Execution result:\n%s", string(resultJSON)),
			},
		},
	}, nil
}

// Tool to get console logs
func (s *PuppeteerServer) getConsoleLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.mu.RLock()
	logs := make([]string, len(s.consoleLogs))
	copy(logs, s.consoleLogs)
	s.mu.RUnlock()

	return &mcp.CallToolResult{
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("Console logs:\n%s", strings.Join(logs, "\n")),
			},
		},
	}, nil
}

// Tool to list stored screenshots
func (s *PuppeteerServer) listScreenshots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.mu.RLock()
	names := make([]string, 0, len(s.screenshots))
	for name := range s.screenshots {
		names = append(names, name)
	}
	s.mu.RUnlock()

	return &mcp.CallToolResult{
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("Stored screenshots: %s", strings.Join(names, ", ")),
			},
		},
	}, nil
}

// Tool to get a specific screenshot
func (s *PuppeteerServer) getScreenshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Name    string `json:"name"`
		Encoded bool   `json:"encoded,omitempty"`
	}
	if err := json.Unmarshal(request.Arguments, &args); err != nil {
		return nil, err
	}

	s.mu.RLock()
	screenshot, exists := s.screenshots[args.Name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("screenshot '%s' not found", args.Name)
	}

	content := []interface{}{}
	if args.Encoded {
		content = append(content, map[string]interface{}{
			"type": "text",
			"text": fmt.Sprintf("data:image/png;base64,%s", screenshot),
		})
	} else {
		content = append(content, map[string]interface{}{
			"type":     "image",
			"data":     screenshot,
			"mimeType": "image/png",
		})
	}

	return &mcp.CallToolResult{
		Content: content,
	}, nil
}

func (s *PuppeteerServer) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

func main() {
	// Redirect logs to stderr to keep stdout clean for the protocol
	log.SetOutput(os.Stderr)
	// Set to WARN level to suppress INFO logs like "Build info"
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))

	log.Println("Starting MCP Puppeteer Server...")

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	// Create puppeteer instance
	ps, err := NewPuppeteerServer()
	if err != nil {
		log.Fatalf("Failed to create puppeteer server: %v", err)
	}
	defer ps.Close()

	// Create MCP server
	server := mcp.NewServer(
		"mcp-puppeteer-server",
		"0.1.0",
		mcp.WithServerInstructions("A browser automation server using Chrome DevTools Protocol"),
	)

	// Register tools
	tools := []struct {
		tool    mcp.Tool
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{
			tool: mcp.Tool{
				Name:        "puppeteer_navigate",
				Description: "Navigate to a URL",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"url": {"type": "string", "description": "URL to navigate to"},
						"launchOptions": {"type": "object", "description": "Chrome launch options"},
						"allowDangerous": {"type": "boolean", "description": "Allow dangerous options"}
					},
					"required": ["url"]
				}`),
			},
			handler: ps.navigate,
		},
		{
			tool: mcp.Tool{
				Name:        "puppeteer_screenshot",
				Description: "Take a screenshot of the current page or a specific element",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"name": {"type": "string", "description": "Name for the screenshot"},
						"selector": {"type": "string", "description": "CSS selector for element to screenshot"},
						"width": {"type": "number", "description": "Width in pixels (default: 800)"},
						"height": {"type": "number", "description": "Height in pixels (default: 600)"},
						"encoded": {"type": "boolean", "description": "Return as base64-encoded data URI"}
					},
					"required": ["name"]
				}`),
			},
			handler: ps.screenshot,
		},
		{
			tool: mcp.Tool{
				Name:        "puppeteer_click",
				Description: "Click an element on the page",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"selector": {"type": "string", "description": "CSS selector for element to click"}
					},
					"required": ["selector"]
				}`),
			},
			handler: ps.click,
		},
		{
			tool: mcp.Tool{
				Name:        "puppeteer_fill",
				Description: "Fill out an input field",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"selector": {"type": "string", "description": "CSS selector for input field"},
						"value": {"type": "string", "description": "Value to fill"}
					},
					"required": ["selector", "value"]
				}`),
			},
			handler: ps.fill,
		},
		{
			tool: mcp.Tool{
				Name:        "puppeteer_select",
				Description: "Select an element on the page with Select tag",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"selector": {"type": "string", "description": "CSS selector for element to select"},
						"value": {"type": "string", "description": "Value to select"}
					},
					"required": ["selector", "value"]
				}`),
			},
			handler: ps.selectOption,
		},
		{
			tool: mcp.Tool{
				Name:        "puppeteer_hover",
				Description: "Hover an element on the page",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"selector": {"type": "string", "description": "CSS selector for element to hover"}
					},
					"required": ["selector"]
				}`),
			},
			handler: ps.hover,
		},
		{
			tool: mcp.Tool{
				Name:        "puppeteer_evaluate",
				Description: "Execute JavaScript in the browser console",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"script": {"type": "string", "description": "JavaScript code to execute"}
					},
					"required": ["script"]
				}`),
			},
			handler: ps.evaluate,
		},
		{
			tool: mcp.Tool{
				Name:        "puppeteer_console_logs",
				Description: "Get browser console logs",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {}
				}`),
			},
			handler: ps.getConsoleLogs,
		},
		{
			tool: mcp.Tool{
				Name:        "puppeteer_list_screenshots",
				Description: "List all stored screenshots",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {}
				}`),
			},
			handler: ps.listScreenshots,
		},
		{
			tool: mcp.Tool{
				Name:        "puppeteer_get_screenshot",
				Description: "Get a specific stored screenshot",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"name": {"type": "string", "description": "Name of the screenshot to retrieve"},
						"encoded": {"type": "boolean", "description": "Return as base64-encoded data URI"}
					},
					"required": ["name"]
				}`),
			},
			handler: ps.getScreenshot,
		},
	}

	for _, t := range tools {
		if err := server.RegisterTool(t.tool, t.handler); err != nil {
			log.Fatalf("Failed to register tool %s: %v", t.tool.Name, err)
		}
	}

	// Run the server
	log.Println("Ready to accept connections")
	if err := server.Serve(ctx, nil); err != nil {
		log.Printf("Server error: %v", err)
	}
}
