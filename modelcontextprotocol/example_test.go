package modelcontextprotocol_test

import (
	"encoding/json"
	"fmt"
	"log"

	mcp "github.com/tmc/mcp/modelcontextprotocol"
)

// ExampleClientCapabilities shows how to use client capabilities
func ExampleClientCapabilities() {
	// Create client capabilities with various options
	client := mcp.NewClientCapabilities(
		mcp.WithClientSampling(),
		mcp.WithClientRoots(
			mcp.WithRootsListChanged(true),
		),
		mcp.WithClientExperimental("myFeature", true),
	)

	// Check capabilities
	fmt.Println("Supports sampling:", client.SupportsSampling())
	fmt.Println("Supports root list changed:", client.SupportsRootListChanged())

	// Get experimental features
	if val, ok := client.GetClientExperimental("myFeature"); ok {
		fmt.Printf("My feature enabled: %v\n", val)
	}

	// JSON marshaling
	jsonData, err := json.Marshal(client)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("JSON representation length:", len(jsonData))

	// Output:
	// Supports sampling: true
	// Supports root list changed: true
	// My feature enabled: true
	// JSON representation length: 78
}

// ExampleServerCapabilities shows how to use server capabilities
func ExampleServerCapabilities() {
	// Create server capabilities with various options
	server := mcp.NewServerCapabilities(
		mcp.WithServerLogging(),
		mcp.WithServerCompletions(),
		mcp.WithServerTools(
			mcp.WithToolsListChanged(true),
		),
		mcp.WithServerResources(
			mcp.WithResourcesSubscription(true),
			mcp.WithResourcesListChanged(true),
		),
		mcp.WithServerExperimental("betaFeature", "enabled"),
	)

	// Check capabilities
	fmt.Println("Supports logging:", server.SupportsLogging())
	fmt.Println("Supports completions:", server.SupportsCompletions())
	fmt.Println("Supports tool list changed:", server.SupportsToolListChanged())
	fmt.Println("Supports resource subscription:", server.SupportsResourceSubscription())

	// Get experimental features
	if val, ok := server.GetServerExperimental("betaFeature"); ok {
		fmt.Printf("Beta feature value: %v\n", val)
	}

	// Output:
	// Supports logging: true
	// Supports completions: true
	// Supports tool list changed: true
	// Supports resource subscription: true
	// Beta feature value: enabled
}

// ExampleContent_unmarshaling shows content type unmarshaling
func ExampleContent_unmarshaling() {
	// Different content types JSON
	textJSON := `{"type": "text", "text": "Hello, world!"}`
	imageJSON := `{"type": "image", "data": "base64data", "mimeType": "image/png"}`
	audioJSON := `{"type": "audio", "data": "audiodata", "mimeType": "audio/mp3"}`

	// Unmarshal text content
	var textContent mcp.TextContent
	if err := json.Unmarshal([]byte(textJSON), &textContent); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Text content: %s\n", textContent.Text)

	// Unmarshal image content
	var imageContent mcp.ImageContent
	if err := json.Unmarshal([]byte(imageJSON), &imageContent); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Image mime type: %s\n", imageContent.MimeType)

	// Unmarshal audio content
	var audioContent mcp.AudioContent
	if err := json.Unmarshal([]byte(audioJSON), &audioContent); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Audio mime type: %s\n", audioContent.MimeType)

	// Output:
	// Text content: Hello, world!
	// Image mime type: image/png
	// Audio mime type: audio/mp3
}

// ExamplePrompt shows how to create and configure prompts
func ExamplePrompt() {
	// Create a prompt with arguments
	prompt := mcp.NewPrompt("generate_code",
		mcp.WithPromptDescription("Generate code based on requirements"),
		mcp.WithPromptArgument("language",
			mcp.WithPromptArgumentDescription("Programming language"),
			mcp.WithPromptArgumentRequired(true),
		),
		mcp.WithPromptArgument("requirements",
			mcp.WithPromptArgumentDescription("Code requirements"),
			mcp.WithPromptArgumentRequired(true),
		),
	)

	// Marshal to JSON
	_, err := json.Marshal(prompt)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Prompt name:", prompt.Name)
	fmt.Println("Number of arguments:", len(prompt.Arguments))
	fmt.Println("First argument required:", *prompt.Arguments[0].Required)

	// Output:
	// Prompt name: generate_code
	// Number of arguments: 2
	// First argument required: true
}

// ExampleResource shows how to create resources
func ExampleResource() {
	// Create a resource
	resource := mcp.NewResource("file:///docs/readme.md", "README Documentation",
		mcp.WithResourceDescription("Main documentation file"),
		mcp.WithResourceMimeType("text/markdown"),
		mcp.WithResourceSize(2048),
	)

	// Create text resource contents
	textContents := mcp.TextResourceContents{
		BaseResourceContents: mcp.BaseResourceContents{
			URI:      resource.URI,
			MimeType: resource.MimeType,
		},
		Text: "# README\n\nThis is the documentation.",
	}

	fmt.Println("Resource URI:", resource.URI)
	fmt.Println("Resource name:", resource.Name)
	fmt.Println("Content URI:", textContents.GetURI())

	// Output:
	// Resource URI: file:///docs/readme.md
	// Resource name: README Documentation
	// Content URI: file:///docs/readme.md
}

// ExampleTool shows how to create tools
func ExampleTool() {
	// Define tool input schema
	properties := map[string]json.RawMessage{
		"query": json.RawMessage(`{"type": "string", "description": "Search query"}`),
	}

	inputSchema := mcp.ToolSchema{
		Type:       "object",
		Properties: properties,
		Required:   []string{"query"},
	}

	// Create a tool
	tool := mcp.NewTool("search", inputSchema,
		mcp.WithToolDescription("Search for information"),
	)

	fmt.Println("Tool name:", tool.Name)
	if tool.Description != nil {
		fmt.Println("Tool description:", *tool.Description)
	}

	// Output:
	// Tool name: search
	// Tool description: Search for information
}

// ExampleSamplingMessage shows sampling message creation
func ExampleSamplingMessage() {
	// Create a sampling message directly
	textContent := mcp.NewTextContent("What's the weather like?")
	message := mcp.SamplingMessage{
		Role:    mcp.RoleUser,
		Content: textContent,
	}

	fmt.Println("Message role:", message.Role)
	if tc, ok := message.Content.(mcp.TextContent); ok {
		fmt.Println("Message text:", tc.Text)
	}

	// Output:
	// Message role: user
	// Message text: What's the weather like?
}

// ExampleCreateMessageResult shows message result creation
func ExampleCreateMessageResult() {
	// Create a message result directly
	responseContent := mcp.NewTextContent("The weather is sunny with a high of 75°F.")
	result := mcp.CreateMessageResult{
		Role:    mcp.RoleAssistant,
		Content: responseContent,
		Model:   "claude-3",
	}

	fmt.Println("Result model:", result.Model)
	if tc, ok := result.Content.(mcp.TextContent); ok {
		fmt.Println("Response text:", tc.Text)
	}

	// Output:
	// Result model: claude-3
	// Response text: The weather is sunny with a high of 75°F.
}
