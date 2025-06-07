package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

type BraveSearchServer struct {
	apiKey     string
	httpClient *http.Client
}

type SearchResponse struct {
	Web    WebResults    `json:"web"`
	News   NewsResults   `json:"news"`
	Videos VideosResults `json:"videos"`
	Images ImagesResults `json:"images"`
	Query  QueryInfo     `json:"query"`
	Type   string        `json:"type"`
}

type WebResults struct {
	Type    string      `json:"type"`
	Results []WebResult `json:"results"`
}

type WebResult struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Age         string    `json:"age,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

type NewsResults struct {
	Type    string       `json:"type"`
	Results []NewsResult `json:"results"`
}

type NewsResult struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Age         string    `json:"age"`
	PublishedAt time.Time `json:"published_at"`
}

type VideosResults struct {
	Type    string        `json:"type"`
	Results []VideoResult `json:"results"`
}

type VideoResult struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Age         string    `json:"age"`
	PublishedAt time.Time `json:"published_at"`
}

type ImagesResults struct {
	Type    string        `json:"type"`
	Results []ImageResult `json:"results"`
}

type ImageResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Src   string `json:"src"`
}

type QueryInfo struct {
	Original           string `json:"original"`
	ShowStrictWarning  bool   `json:"show_strict_warning"`
	IsNavigational     bool   `json:"is_navigational"`
	SpellingCorrection string `json:"spellcheck_off"`
}

func NewBraveSearchServer(apiKey string) (*BraveSearchServer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Brave Search API key is required")
	}

	return &BraveSearchServer{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (bss *BraveSearchServer) search(query string, searchType string, count int, country string, safesearch string) (string, error) {
	if count <= 0 {
		count = 10
	}
	if count > 20 {
		count = 20 // Brave API limit
	}

	baseURL := "https://api.search.brave.com/res/v1/web/search"

	params := url.Values{}
	params.Set("q", query)
	params.Set("count", strconv.Itoa(count))

	if country != "" {
		params.Set("country", country)
	}
	if safesearch != "" {
		params.Set("safesearch", safesearch)
	}

	// Add result types based on searchType
	resultFilter := []string{}
	switch searchType {
	case "web":
		resultFilter = append(resultFilter, "web")
	case "news":
		resultFilter = append(resultFilter, "news")
	case "images":
		resultFilter = append(resultFilter, "images")
	case "videos":
		resultFilter = append(resultFilter, "videos")
	case "all":
		resultFilter = append(resultFilter, "web", "news")
	default:
		resultFilter = append(resultFilter, "web")
	}

	if len(resultFilter) > 0 {
		params.Set("result_filter", strings.Join(resultFilter, ","))
	}

	requestURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", bss.apiKey)

	resp, err := bss.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return bss.formatSearchResults(searchResp, searchType), nil
}

func (bss *BraveSearchServer) formatSearchResults(resp SearchResponse, searchType string) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("Search results for: \"%s\"\n", resp.Query.Original))
	result.WriteString(fmt.Sprintf("Search type: %s\n\n", searchType))

	// Format web results
	if len(resp.Web.Results) > 0 {
		result.WriteString("Web Results:\n")
		result.WriteString("============\n")
		for i, webResult := range resp.Web.Results {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, webResult.Title))
			result.WriteString(fmt.Sprintf("   URL: %s\n", webResult.URL))
			if webResult.Description != "" {
				desc := webResult.Description
				if len(desc) > 200 {
					desc = desc[:197] + "..."
				}
				result.WriteString(fmt.Sprintf("   Description: %s\n", desc))
			}
			if webResult.Age != "" {
				result.WriteString(fmt.Sprintf("   Age: %s\n", webResult.Age))
			}
			result.WriteString("\n")
		}
	}

	// Format news results
	if len(resp.News.Results) > 0 {
		result.WriteString("News Results:\n")
		result.WriteString("=============\n")
		for i, newsResult := range resp.News.Results {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, newsResult.Title))
			result.WriteString(fmt.Sprintf("   URL: %s\n", newsResult.URL))
			if newsResult.Description != "" {
				desc := newsResult.Description
				if len(desc) > 200 {
					desc = desc[:197] + "..."
				}
				result.WriteString(fmt.Sprintf("   Description: %s\n", desc))
			}
			if newsResult.Age != "" {
				result.WriteString(fmt.Sprintf("   Published: %s\n", newsResult.Age))
			}
			result.WriteString("\n")
		}
	}

	// Format video results
	if len(resp.Videos.Results) > 0 {
		result.WriteString("Video Results:\n")
		result.WriteString("==============\n")
		for i, videoResult := range resp.Videos.Results {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, videoResult.Title))
			result.WriteString(fmt.Sprintf("   URL: %s\n", videoResult.URL))
			if videoResult.Description != "" {
				desc := videoResult.Description
				if len(desc) > 200 {
					desc = desc[:197] + "..."
				}
				result.WriteString(fmt.Sprintf("   Description: %s\n", desc))
			}
			if videoResult.Age != "" {
				result.WriteString(fmt.Sprintf("   Published: %s\n", videoResult.Age))
			}
			result.WriteString("\n")
		}
	}

	// Format image results
	if len(resp.Images.Results) > 0 {
		result.WriteString("Image Results:\n")
		result.WriteString("==============\n")
		for i, imageResult := range resp.Images.Results {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, imageResult.Title))
			result.WriteString(fmt.Sprintf("   Source URL: %s\n", imageResult.URL))
			result.WriteString(fmt.Sprintf("   Image URL: %s\n", imageResult.Src))
			result.WriteString("\n")
		}
	}

	if len(resp.Web.Results) == 0 && len(resp.News.Results) == 0 &&
		len(resp.Videos.Results) == 0 && len(resp.Images.Results) == 0 {
		result.WriteString("No results found.\n")
	}

	return result.String()
}

func main() {
	apiKey := os.Getenv("BRAVE_API_KEY")
	if apiKey == "" {
		log.Fatal("BRAVE_API_KEY environment variable is required")
	}

	// Initialize Brave Search server
	bss, err := NewBraveSearchServer(apiKey)
	if err != nil {
		log.Fatalf("Failed to initialize Brave Search server: %v", err)
	}

	// Create server with name and version
	srv := mcp.NewServer("bravesearch-server", "1.0.0")

	// Register web_search tool
	srv.RegisterTool("web_search", "Search the web using Brave Search API", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var queryRaw json.RawMessage
		var exists bool
		if queryRaw, exists = args["query"]; !exists {
			return nil, fmt.Errorf("missing required argument: query")
		}

		var query string
		if err := json.Unmarshal(queryRaw, &query); err != nil {
			return nil, fmt.Errorf("invalid query argument: %w", err)
		}

		var searchType string = "web"
		if typeRaw, exists := args["type"]; exists {
			if err := json.Unmarshal(typeRaw, &searchType); err != nil {
				return nil, fmt.Errorf("invalid type argument: %w", err)
			}
		}

		var count int = 10
		if countRaw, exists := args["count"]; exists {
			if err := json.Unmarshal(countRaw, &count); err != nil {
				return nil, fmt.Errorf("invalid count argument: %w", err)
			}
		}

		var country string
		if countryRaw, exists := args["country"]; exists {
			if err := json.Unmarshal(countryRaw, &country); err != nil {
				return nil, fmt.Errorf("invalid country argument: %w", err)
			}
		}

		var safesearch string = "moderate"
		if safesearchRaw, exists := args["safesearch"]; exists {
			if err := json.Unmarshal(safesearchRaw, &safesearch); err != nil {
				return nil, fmt.Errorf("invalid safesearch argument: %w", err)
			}
		}

		result, err := bss.search(query, searchType, count, country, safesearch)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error performing search: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Performed %s search for: %s", searchType, query)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register news_search tool
	srv.RegisterTool("news_search", "Search for news using Brave Search API", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var queryRaw json.RawMessage
		var exists bool
		if queryRaw, exists = args["query"]; !exists {
			return nil, fmt.Errorf("missing required argument: query")
		}

		var query string
		if err := json.Unmarshal(queryRaw, &query); err != nil {
			return nil, fmt.Errorf("invalid query argument: %w", err)
		}

		var count int = 10
		if countRaw, exists := args["count"]; exists {
			if err := json.Unmarshal(countRaw, &count); err != nil {
				return nil, fmt.Errorf("invalid count argument: %w", err)
			}
		}

		var country string
		if countryRaw, exists := args["country"]; exists {
			if err := json.Unmarshal(countryRaw, &country); err != nil {
				return nil, fmt.Errorf("invalid country argument: %w", err)
			}
		}

		result, err := bss.search(query, "news", count, country, "")
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error performing news search: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Performed news search for: %s", query)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register image_search tool
	srv.RegisterTool("image_search", "Search for images using Brave Search API", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var queryRaw json.RawMessage
		var exists bool
		if queryRaw, exists = args["query"]; !exists {
			return nil, fmt.Errorf("missing required argument: query")
		}

		var query string
		if err := json.Unmarshal(queryRaw, &query); err != nil {
			return nil, fmt.Errorf("invalid query argument: %w", err)
		}

		var count int = 10
		if countRaw, exists := args["count"]; exists {
			if err := json.Unmarshal(countRaw, &count); err != nil {
				return nil, fmt.Errorf("invalid count argument: %w", err)
			}
		}

		var safesearch string = "moderate"
		if safesearchRaw, exists := args["safesearch"]; exists {
			if err := json.Unmarshal(safesearchRaw, &safesearch); err != nil {
				return nil, fmt.Errorf("invalid safesearch argument: %w", err)
			}
		}

		result, err := bss.search(query, "images", count, "", safesearch)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error performing image search: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Performed image search for: %s", query)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register video_search tool
	srv.RegisterTool("video_search", "Search for videos using Brave Search API", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var queryRaw json.RawMessage
		var exists bool
		if queryRaw, exists = args["query"]; !exists {
			return nil, fmt.Errorf("missing required argument: query")
		}

		var query string
		if err := json.Unmarshal(queryRaw, &query); err != nil {
			return nil, fmt.Errorf("invalid query argument: %w", err)
		}

		var count int = 10
		if countRaw, exists := args["count"]; exists {
			if err := json.Unmarshal(countRaw, &count); err != nil {
				return nil, fmt.Errorf("invalid count argument: %w", err)
			}
		}

		var safesearch string = "moderate"
		if safesearchRaw, exists := args["safesearch"]; exists {
			if err := json.Unmarshal(safesearchRaw, &safesearch); err != nil {
				return nil, fmt.Errorf("invalid safesearch argument: %w", err)
			}
		}

		result, err := bss.search(query, "videos", count, "", safesearch)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error performing video search: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Performed video search for: %s", query)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Start server with stdio transport
	transport := mcp.StdioTransport{}
	log.Printf("Brave Search server running on stdio")

	if err := srv.Serve(context.Background(), transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
