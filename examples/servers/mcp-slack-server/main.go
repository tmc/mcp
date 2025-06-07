package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

const (
	SERVER_NAME    = "mcp-slack-server"
	SERVER_VERSION = "1.0.0"
)

type SlackServer struct {
	*mcp.Server
	botToken string
}

func NewSlackServer() *SlackServer {
	s := &SlackServer{
		Server:   mcp.NewServer(SERVER_NAME, SERVER_VERSION),
		botToken: "mock-bot-token", // In real implementation, would read from env
	}

	s.SetupHandlers()
	return s
}

func (s *SlackServer) SetupHandlers() {
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
				Name:        "send_message",
				Description: "Send a message to a Slack channel",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"channel": map[string]interface{}{
							"type":        "string",
							"description": "Channel ID or name to send message to",
						},
						"text": map[string]interface{}{
							"type":        "string",
							"description": "Message text to send",
						},
						"thread_ts": map[string]interface{}{
							"type":        "string",
							"description": "Timestamp of thread to reply to (optional)",
						},
					},
					"required": []string{"channel", "text"},
				},
			},
			{
				Name:        "list_channels",
				Description: "List all channels in the workspace",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"types": map[string]interface{}{
							"type":        "string",
							"description": "Comma-separated list of channel types (public_channel, private_channel, mpim, im)",
							"default":     "public_channel,private_channel",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of channels to return",
							"default":     100,
						},
					},
				},
			},
			{
				Name:        "get_channel_history",
				Description: "Get message history from a channel",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"channel": map[string]interface{}{
							"type":        "string",
							"description": "Channel ID to get history from",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Number of messages to retrieve",
							"default":     10,
						},
						"oldest": map[string]interface{}{
							"type":        "string",
							"description": "Oldest timestamp of messages to include",
						},
					},
					"required": []string{"channel"},
				},
			},
			{
				Name:        "create_channel",
				Description: "Create a new channel",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the channel to create",
						},
						"is_private": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the channel should be private",
							"default":     false,
						},
					},
					"required": []string{"name"},
				},
			},
			{
				Name:        "invite_to_channel",
				Description: "Invite users to a channel",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"channel": map[string]interface{}{
							"type":        "string",
							"description": "Channel ID to invite users to",
						},
						"users": map[string]interface{}{
							"type":        "string",
							"description": "Comma-separated list of user IDs",
						},
					},
					"required": []string{"channel", "users"},
				},
			},
			{
				Name:        "get_user_info",
				Description: "Get information about a user",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"user": map[string]interface{}{
							"type":        "string",
							"description": "User ID to get information about",
						},
					},
					"required": []string{"user"},
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
		case "send_message":
			return s.handleSendMessage(ctx, req)
		case "list_channels":
			return s.handleListChannels(ctx, req)
		case "get_channel_history":
			return s.handleGetChannelHistory(ctx, req)
		case "create_channel":
			return s.handleCreateChannel(ctx, req)
		case "invite_to_channel":
			return s.handleInviteToChannel(ctx, req)
		case "get_user_info":
			return s.handleGetUserInfo(ctx, req)
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

func (s *SlackServer) handleSendMessage(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Channel  string `json:"channel"`
		Text     string `json:"text"`
		ThreadTS string `json:"thread_ts,omitempty"`
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

	// Mock response
	result := "Message sent to " + args.Channel + ": " + args.Text
	if args.ThreadTS != "" {
		result += " (in thread: " + args.ThreadTS + ")"
	}
	result += "\nMessage ID: msg_" + time.Now().Format("20060102150405")

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func (s *SlackServer) handleListChannels(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Types string `json:"types"`
		Limit int    `json:"limit"`
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

	if args.Types == "" {
		args.Types = "public_channel,private_channel"
	}
	if args.Limit == 0 {
		args.Limit = 100
	}

	// Mock response
	result := "Channels (" + args.Types + ", limit " + json.Number(args.Limit).String() + "):\n"
	result += "• #general (C123456789) - General discussion\n"
	result += "• #random (C234567890) - Random topics\n"
	result += "• #development (C345678901) - Development discussions\n"
	result += "• #marketing (C456789012) - Marketing team"

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func (s *SlackServer) handleGetChannelHistory(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Channel string `json:"channel"`
		Limit   int    `json:"limit"`
		Oldest  string `json:"oldest,omitempty"`
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

	if args.Limit == 0 {
		args.Limit = 10
	}

	// Mock response
	result := "Recent messages in " + args.Channel + " (limit " + json.Number(args.Limit).String() + "):\n\n"
	result += "[14:30] @alice: Hey everyone, how's the project going?\n"
	result += "[14:32] @bob: Making good progress on the API\n"
	result += "[14:35] @charlie: UI is almost done, just need to polish\n"
	result += "[14:37] @alice: Great! Let's sync up tomorrow morning"

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func (s *SlackServer) handleCreateChannel(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Name      string `json:"name"`
		IsPrivate bool   `json:"is_private"`
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

	// Mock response
	channelType := "public"
	if args.IsPrivate {
		channelType = "private"
	}

	result := "Created " + channelType + " channel: #" + args.Name + "\n"
	result += "Channel ID: C" + time.Now().Format("20060102150405") + "\n"
	result += "Members: 1 (you)"

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func (s *SlackServer) handleInviteToChannel(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Channel string `json:"channel"`
		Users   string `json:"users"`
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

	// Mock response
	result := "Invited users to " + args.Channel + ": " + args.Users + "\n"
	result += "Invitation sent successfully"

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func (s *SlackServer) handleGetUserInfo(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		User string `json:"user"`
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

	// Mock response
	result := "User info for " + args.User + ":\n"
	result += "Name: Alice Johnson\n"
	result += "Display Name: alice\n"
	result += "Email: alice@company.com\n"
	result += "Status: Active\n"
	result += "Timezone: America/New_York\n"
	result += "Role: Developer"

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func main() {
	server := NewSlackServer()

	log.Printf("Starting %s v%s", SERVER_NAME, SERVER_VERSION)

	if err := server.Run(context.Background()); err != nil {
		log.Fatal("Server failed:", err)
	}
}
