package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

type HelloWorldServer struct {
	greetings map[string]string
	fortunes  []string
}

func NewHelloWorldServer() *HelloWorldServer {
	greetings := map[string]string{
		"english":    "Hello",
		"spanish":    "Hola",
		"french":     "Bonjour",
		"german":     "Hallo",
		"italian":    "Ciao",
		"portuguese": "Olá",
		"russian":    "Привет",
		"japanese":   "こんにちは",
		"chinese":    "你好",
		"korean":     "안녕하세요",
		"arabic":     "مرحبا",
		"hindi":      "नमस्ते",
	}

	fortunes := []string{
		"The best time to plant a tree was 20 years ago. The second best time is now.",
		"Your future is created by what you do today, not tomorrow.",
		"The only impossible journey is the one you never begin.",
		"Success is not final, failure is not fatal: it is the courage to continue that counts.",
		"The way to get started is to quit talking and begin doing.",
		"Innovation distinguishes between a leader and a follower.",
		"Life is what happens when you're busy making other plans.",
		"The future belongs to those who believe in the beauty of their dreams.",
		"It is during our darkest moments that we must focus to see the light.",
		"Whether you think you can or you think you can't, you're right.",
	}

	return &HelloWorldServer{
		greetings: greetings,
		fortunes:  fortunes,
	}
}

func (s *HelloWorldServer) handleGreeting(args map[string]interface{}) (*modelcontextprotocol.CallToolResult, error) {
	name, hasName := args["name"].(string)
	if !hasName || name == "" {
		name = "World"
	}

	language, hasLang := args["language"].(string)
	if !hasLang {
		language = "english"
	}
	language = strings.ToLower(language)

	greeting, exists := s.greetings[language]
	if !exists {
		// Return available languages if unknown language requested
		var available []string
		for lang := range s.greetings {
			available = append(available, lang)
		}
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Unknown language '%s'. Available languages: %s", language, strings.Join(available, ", ")),
				},
			},
			IsError: true,
		}, nil
	}

	message := fmt.Sprintf("%s, %s!", greeting, name)

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			{
				Type: "text",
				Text: message,
			},
		},
	}, nil
}

func (s *HelloWorldServer) handleFortune(args map[string]interface{}) (*modelcontextprotocol.CallToolResult, error) {
	rand.Seed(time.Now().UnixNano())
	fortune := s.fortunes[rand.Intn(len(s.fortunes))]

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			{
				Type: "text",
				Text: fortune,
			},
		},
	}, nil
}

func (s *HelloWorldServer) handleListLanguages(args map[string]interface{}) (*modelcontextprotocol.CallToolResult, error) {
	var languages []string
	for lang := range s.greetings {
		languages = append(languages, lang)
	}

	languageList := strings.Join(languages, "\n- ")
	message := fmt.Sprintf("Supported languages:\n- %s", languageList)

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			{
				Type: "text",
				Text: message,
			},
		},
	}, nil
}

func (s *HelloWorldServer) handleGreetingWithFortune(args map[string]interface{}) (*modelcontextprotocol.CallToolResult, error) {
	// Get greeting
	greetingResult, err := s.handleGreeting(args)
	if err != nil {
		return nil, err
	}
	if greetingResult.IsError {
		return greetingResult, nil
	}

	// Get fortune
	fortuneResult, err := s.handleFortune(args)
	if err != nil {
		return nil, err
	}

	greeting := greetingResult.Content[0].Text
	fortune := fortuneResult.Content[0].Text

	combined := fmt.Sprintf("%s\n\nYour fortune: %s", greeting, fortune)

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			{
				Type: "text",
				Text: combined,
			},
		},
	}, nil
}

func main() {
	server := NewHelloWorldServer()
	mcpServer := mcp.NewServer("mcp-helloworld-server", "1.0.0")

	// Add tools
	mcpServer.AddTool(modelcontextprotocol.Tool{
		Name:        "greeting",
		Description: "Generate a greeting in various languages",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name to greet (optional, defaults to 'World')",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Language for the greeting (optional, defaults to 'english')",
				},
			},
		},
	})

	mcpServer.AddTool(modelcontextprotocol.Tool{
		Name:        "fortune",
		Description: "Get a random inspirational fortune",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	})

	mcpServer.AddTool(modelcontextprotocol.Tool{
		Name:        "list_languages",
		Description: "List all supported languages for greetings",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	})

	mcpServer.AddTool(modelcontextprotocol.Tool{
		Name:        "greeting_with_fortune",
		Description: "Generate a greeting with a bonus fortune",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name to greet (optional, defaults to 'World')",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Language for the greeting (optional, defaults to 'english')",
				},
			},
		},
	})

	// Add tool handlers
	mcpServer.OnToolCall("greeting", server.handleGreeting)
	mcpServer.OnToolCall("fortune", server.handleFortune)
	mcpServer.OnToolCall("list_languages", server.handleListLanguages)
	mcpServer.OnToolCall("greeting_with_fortune", server.handleGreetingWithFortune)

	if err := mcpServer.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
