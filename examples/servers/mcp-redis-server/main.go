package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

const (
	SERVER_NAME    = "mcp-redis-server"
	SERVER_VERSION = "1.0.0"
)

type RedisServer struct {
	*mcp.Server
	host string
	port int
	db   int
}

func NewRedisServer() *RedisServer {
	host := getEnvOrDefault("REDIS_HOST", "localhost")
	port, _ := strconv.Atoi(getEnvOrDefault("REDIS_PORT", "6379"))
	db, _ := strconv.Atoi(getEnvOrDefault("REDIS_DB", "0"))

	s := &RedisServer{
		Server: mcp.NewServer(SERVER_NAME, SERVER_VERSION),
		host:   host,
		port:   port,
		db:     db,
	}

	s.SetupHandlers()
	return s
}

func (s *RedisServer) SetupHandlers() {
	// Initialize with tools
	s.OnInitialize(func(ctx context.Context, req *modelcontextprotocol.InitializeRequest) (*modelcontextprotocol.InitializeResult, error) {
		return &modelcontextprotocol.InitializeResult{
			ProtocolVersion: modelcontextprotocol.LatestProtocolVersion,
			Capabilities: &modelcontextprotocol.ServerCapabilities{
				Tools: &modelcontextprotocol.ToolsCapability{},
			},
			ServerInfo: &modelcontextprotocol.Implementation{
				Name:    SERVER_NAME,
				Version: SERVER_VERSION,
			},
		}, nil
	})

	// List available tools
	s.OnListTools(func(ctx context.Context, req *modelcontextprotocol.ListToolsRequest) (*modelcontextprotocol.ListToolsResult, error) {
		tools := []modelcontextprotocol.Tool{
			{
				Name:        "redis_get",
				Description: "Get value from Redis key",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key": map[string]interface{}{
							"type":        "string",
							"description": "Redis key to retrieve",
						},
					},
					"required": []string{"key"},
				},
			},
			{
				Name:        "redis_set",
				Description: "Set value in Redis",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key": map[string]interface{}{
							"type":        "string",
							"description": "Redis key to set",
						},
						"value": map[string]interface{}{
							"type":        "string",
							"description": "Value to store",
						},
						"ttl": map[string]interface{}{
							"type":        "integer",
							"description": "Time to live in seconds (optional)",
						},
					},
					"required": []string{"key", "value"},
				},
			},
			{
				Name:        "redis_delete",
				Description: "Delete Redis key",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key": map[string]interface{}{
							"type":        "string",
							"description": "Redis key to delete",
						},
					},
					"required": []string{"key"},
				},
			},
			{
				Name:        "redis_exists",
				Description: "Check if Redis key exists",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key": map[string]interface{}{
							"type":        "string",
							"description": "Redis key to check",
						},
					},
					"required": []string{"key"},
				},
			},
			{
				Name:        "redis_keys",
				Description: "List Redis keys matching pattern",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern": map[string]interface{}{
							"type":        "string",
							"description": "Pattern to match keys (default: *)",
							"default":     "*",
						},
					},
				},
			},
		}

		return &modelcontextprotocol.ListToolsResult{
			Tools: tools,
		}, nil
	})

	// Handle tool calls
	s.OnCallTool(func(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
		switch req.Params.Name {
		case "redis_get":
			return s.handleRedisGet(ctx, req)
		case "redis_set":
			return s.handleRedisSet(ctx, req)
		case "redis_delete":
			return s.handleRedisDelete(ctx, req)
		case "redis_exists":
			return s.handleRedisExists(ctx, req)
		case "redis_keys":
			return s.handleRedisKeys(ctx, req)
		default:
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: "Unknown tool: " + req.Params.Name,
					},
				},
				IsError: &[]bool{true}[0],
			}, nil
		}
	})
}

func (s *RedisServer) handleRedisGet(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Key string `json:"key"`
	}

	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: "Error parsing arguments: " + err.Error(),
				},
			},
			IsError: &[]bool{true}[0],
		}, nil
	}

	// Mock implementation - in real version would connect to Redis
	result := "value-for-" + args.Key

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: "Retrieved value: " + result,
			},
		},
	}, nil
}

func (s *RedisServer) handleRedisSet(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		TTL   *int   `json:"ttl,omitempty"`
	}

	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: "Error parsing arguments: " + err.Error(),
				},
			},
			IsError: &[]bool{true}[0],
		}, nil
	}

	// Mock implementation
	response := "Set " + args.Key + " = " + args.Value
	if args.TTL != nil {
		response += " (TTL: " + strconv.Itoa(*args.TTL) + "s)"
	}

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: response,
			},
		},
	}, nil
}

func (s *RedisServer) handleRedisDelete(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Key string `json:"key"`
	}

	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: "Error parsing arguments: " + err.Error(),
				},
			},
			IsError: &[]bool{true}[0],
		}, nil
	}

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: "Deleted key: " + args.Key,
			},
		},
	}, nil
}

func (s *RedisServer) handleRedisExists(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Key string `json:"key"`
	}

	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: "Error parsing arguments: " + err.Error(),
				},
			},
			IsError: &[]bool{true}[0],
		}, nil
	}

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: "Key " + args.Key + " exists: true",
			},
		},
	}, nil
}

func (s *RedisServer) handleRedisKeys(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Pattern string `json:"pattern"`
	}

	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: "Error parsing arguments: " + err.Error(),
				},
			},
			IsError: &[]bool{true}[0],
		}, nil
	}

	if args.Pattern == "" {
		args.Pattern = "*"
	}

	// Mock implementation
	keys := []string{"key1", "key2", "test:*"}
	result := "Keys matching " + args.Pattern + ": " + args.Pattern + " -> [" + "key1, key2, test:*" + "]"

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	server := NewRedisServer()

	log.Printf("Starting %s v%s", SERVER_NAME, SERVER_VERSION)
	log.Printf("Redis connection: %s:%d (db %d)", server.host, server.port, server.db)

	if err := server.Run(context.Background()); err != nil {
		log.Fatal("Server failed:", err)
	}
}
