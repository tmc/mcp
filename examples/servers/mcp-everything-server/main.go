package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tmc/mcp"
)

const (
	ServerName    = "example-servers/everything"
	ServerVersion = "1.0.0"
)

func main() {
	// Command line flags
	timeout := flag.Duration("timeout", 0, "Auto-terminate after specified duration (for testing)")
	quiet := flag.Bool("quiet", false, "Disable verbose logging")
	verbose := flag.Bool("verbose", true, "Enable verbose logging (default: true)")
	flag.Parse()

	// Redirect logs to stderr to keep stdout clean for the protocol
	log.SetOutput(os.Stderr)

	// Enable or disable verbose logging
	if *quiet || !*verbose {
		log.SetOutput(io.Discard)
	}

	log.Println("Starting MCP Everything Server...")

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	// Create a server
	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A comprehensive example MCP server with various capabilities"),
	)

	// Register tools
	registerTools(server)

	// Auto-terminate for testing if timeout is set
	if *timeout > 0 {
		go func() {
			log.Printf("Auto-termination enabled, will exit after %v", *timeout)
			time.Sleep(*timeout)
			log.Println("Auto-terminating due to timeout...")
			cancel()
		}()
	}

	// Serve via stdio
	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerTools(server *mcp.Server) {
	// Register time tool
	timeTool := mcp.Tool{
		Name:        "current_time",
		Description: "Get the current time",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"timezone": {"type": "string", "description": "Timezone (e.g., 'UTC', 'America/New_York')"}
			}
		}`),
	}
	server.RegisterTool(timeTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		timezone := "UTC"
		if tz, ok := params["timezone"].(string); ok && tz != "" {
			timezone = tz
		}

		loc, err := time.LoadLocation(timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone: %v", err)
		}

		now := time.Now().In(loc)
		result := map[string]interface{}{
			"time":     now.Format(time.RFC3339),
			"timezone": timezone,
			"unix":     now.Unix(),
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type":   "text",
					"format": "json",
					"text":   string(resultJSON),
				},
			},
		}, nil
	})

	// Register echo tool
	echoTool := mcp.Tool{
		Name:        "echo",
		Description: "Echo the input",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {"type": "string", "description": "Text to echo back"}
			},
			"required": ["message"]
		}`),
	}
	server.RegisterTool(echoTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		message, ok := params["message"].(string)
		if !ok {
			return nil, fmt.Errorf("message parameter is required")
		}

		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Echo: " + message,
				},
			},
		}, nil
	})

	// Register random number tool
	randomTool := mcp.Tool{
		Name:        "random",
		Description: "Generate a random number",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"min": {"type": "number", "description": "Minimum value (inclusive)"},
				"max": {"type": "number", "description": "Maximum value (inclusive)"}
			},
			"required": ["min", "max"]
		}`),
	}
	server.RegisterTool(randomTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		min, minOk := params["min"].(float64)
		max, maxOk := params["max"].(float64)

		if !minOk || !maxOk {
			return nil, fmt.Errorf("min and max parameters are required")
		}

		if min > max {
			return nil, fmt.Errorf("min cannot be greater than max")
		}

		// Generate random number
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		result := min + r.Float64()*(max-min)

		// Create response
		jsonResult, _ := json.Marshal(map[string]interface{}{
			"min":    min,
			"max":    max,
			"random": result,
		})

		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type":   "text",
					"format": "json",
					"text":   string(jsonResult),
				},
			},
		}, nil
	})

	// Register add tool
	addTool := mcp.Tool{
		Name:        "add",
		Description: "Add two numbers",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"a": {"type": "number", "description": "First number"},
				"b": {"type": "number", "description": "Second number"}
			},
			"required": ["a", "b"]
		}`),
	}
	server.RegisterTool(addTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		a, aOk := params["a"].(float64)
		b, bOk := params["b"].(float64)

		if !aOk || !bOk {
			return nil, fmt.Errorf("a and b parameters are required and must be numbers")
		}

		sum := a + b
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": fmt.Sprintf("The sum of %v and %v is %v.", a, b, sum),
				},
			},
		}, nil
	})

	log.Println("Registered all tools")
}
