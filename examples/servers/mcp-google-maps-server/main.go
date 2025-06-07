package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

const (
	SERVER_NAME    = "mcp-google-maps-server"
	SERVER_VERSION = "1.0.0"
)

type GoogleMapsServer struct {
	*mcp.Server
	apiKey string
}

func NewGoogleMapsServer() *GoogleMapsServer {
	s := &GoogleMapsServer{
		Server: mcp.NewServer(SERVER_NAME, SERVER_VERSION),
		apiKey: "mock-api-key", // In real implementation, would read from env
	}

	s.SetupHandlers()
	return s
}

func (s *GoogleMapsServer) SetupHandlers() {
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
				Name:        "search_places",
				Description: "Search for places using Google Maps",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search query for places",
						},
						"location": map[string]interface{}{
							"type":        "string",
							"description": "Location to search near (optional)",
						},
						"radius": map[string]interface{}{
							"type":        "integer",
							"description": "Search radius in meters (optional)",
						},
					},
					"required": []string{"query"},
				},
			},
			{
				Name:        "get_directions",
				Description: "Get directions between two locations",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"origin": map[string]interface{}{
							"type":        "string",
							"description": "Starting location",
						},
						"destination": map[string]interface{}{
							"type":        "string",
							"description": "Destination location",
						},
						"mode": map[string]interface{}{
							"type":        "string",
							"description": "Travel mode (driving, walking, transit, bicycling)",
							"default":     "driving",
						},
					},
					"required": []string{"origin", "destination"},
				},
			},
			{
				Name:        "geocode",
				Description: "Convert address to coordinates",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"address": map[string]interface{}{
							"type":        "string",
							"description": "Address to geocode",
						},
					},
					"required": []string{"address"},
				},
			},
			{
				Name:        "reverse_geocode",
				Description: "Convert coordinates to address",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"lat": map[string]interface{}{
							"type":        "number",
							"description": "Latitude coordinate",
						},
						"lng": map[string]interface{}{
							"type":        "number",
							"description": "Longitude coordinate",
						},
					},
					"required": []string{"lat", "lng"},
				},
			},
			{
				Name:        "get_place_details",
				Description: "Get detailed information about a place",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"place_id": map[string]interface{}{
							"type":        "string",
							"description": "Google Places ID",
						},
					},
					"required": []string{"place_id"},
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
		case "search_places":
			return s.handleSearchPlaces(ctx, req)
		case "get_directions":
			return s.handleGetDirections(ctx, req)
		case "geocode":
			return s.handleGeocode(ctx, req)
		case "reverse_geocode":
			return s.handleReverseGeocode(ctx, req)
		case "get_place_details":
			return s.handleGetPlaceDetails(ctx, req)
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

func (s *GoogleMapsServer) handleSearchPlaces(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Query    string `json:"query"`
		Location string `json:"location,omitempty"`
		Radius   int    `json:"radius,omitempty"`
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
	result := "Places found for '" + args.Query + "':\n"
	result += "1. Sample Restaurant - 123 Main St\n"
	result += "2. Example Cafe - 456 Oak Ave\n"
	result += "3. Test Store - 789 Pine Rd"

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func (s *GoogleMapsServer) handleGetDirections(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Origin      string `json:"origin"`
		Destination string `json:"destination"`
		Mode        string `json:"mode"`
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

	if args.Mode == "" {
		args.Mode = "driving"
	}

	// Mock response
	result := "Directions from " + args.Origin + " to " + args.Destination + " (" + args.Mode + "):\n"
	result += "1. Head north on Main St\n"
	result += "2. Turn right on Oak Ave\n"
	result += "3. Continue for 2.5 miles\n"
	result += "4. Arrive at destination\n"
	result += "Total distance: 3.2 miles\n"
	result += "Estimated time: 8 minutes"

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func (s *GoogleMapsServer) handleGeocode(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Address string `json:"address"`
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
	result := "Geocoded address: " + args.Address + "\n"
	result += "Latitude: 37.7749\n"
	result += "Longitude: -122.4194\n"
	result += "Formatted address: 123 Main St, San Francisco, CA 94102, USA"

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func (s *GoogleMapsServer) handleReverseGeocode(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
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
	result := "Reverse geocoded coordinates (" +
		json.Number(args.Lat).String() + ", " +
		json.Number(args.Lng).String() + "):\n"
	result += "Address: 123 Main St, San Francisco, CA 94102, USA\n"
	result += "Neighborhood: Financial District\n"
	result += "City: San Francisco\n"
	result += "State: California\n"
	result += "Country: United States"

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func (s *GoogleMapsServer) handleGetPlaceDetails(ctx context.Context, req *modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	var args struct {
		PlaceID string `json:"place_id"`
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
	result := "Place Details for ID: " + args.PlaceID + "\n"
	result += "Name: Sample Restaurant\n"
	result += "Address: 123 Main St, San Francisco, CA 94102\n"
	result += "Phone: (555) 123-4567\n"
	result += "Rating: 4.2/5 (123 reviews)\n"
	result += "Hours: Mon-Fri 8AM-9PM, Sat-Sun 9AM-10PM\n"
	result += "Price Level: $$\n"
	result += "Website: https://example.com"

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
	server := NewGoogleMapsServer()

	log.Printf("Starting %s v%s", SERVER_NAME, SERVER_VERSION)

	if err := server.Run(context.Background()); err != nil {
		log.Fatal("Server failed:", err)
	}
}
