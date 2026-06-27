package mcp

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/tmc/mcp/modelcontextprotocol"
)

// TestSamplingMessageTypes tests the sampling message types
func TestSamplingMessageTypes(t *testing.T) {
	// Test SamplingMessage creation
	textContent := modelcontextprotocol.TextContent{
		Type: "text",
		Text: "Hello, how are you?",
	}

	samplingMsg := modelcontextprotocol.SamplingMessage{
		Role:    modelcontextprotocol.RoleUser,
		Content: textContent,
	}

	if samplingMsg.Role != modelcontextprotocol.RoleUser {
		t.Errorf("Expected role %s, got %s", modelcontextprotocol.RoleUser, samplingMsg.Role)
	}

	// Test JSON marshaling
	data, err := json.Marshal(samplingMsg)
	if err != nil {
		t.Fatalf("Failed to marshal SamplingMessage: %v", err)
	}

	var unmarshaled modelcontextprotocol.SamplingMessage
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SamplingMessage: %v", err)
	}

	if unmarshaled.Role != samplingMsg.Role {
		t.Errorf("Role mismatch after round-trip: %s != %s", unmarshaled.Role, samplingMsg.Role)
	}
}

// TestCreateMessageRequestParams tests CreateMessage request parameters
func TestCreateMessageRequestParams(t *testing.T) {
	messages := []modelcontextprotocol.SamplingMessage{
		{
			Role: modelcontextprotocol.RoleUser,
			Content: modelcontextprotocol.TextContent{
				Type: "text",
				Text: "What is the weather like today?",
			},
		},
		{
			Role: modelcontextprotocol.RoleAssistant,
			Content: modelcontextprotocol.TextContent{
				Type: "text",
				Text: "I'd be happy to help you with the weather information.",
			},
		},
	}

	modelPrefs := &modelcontextprotocol.ModelPreferences{
		Hints: []modelcontextprotocol.ModelHint{
			{
				Name: &[]string{"prefer-speed"}[0],
			},
		},
	}

	systemPrompt := "You are a helpful weather assistant."
	temperature := 0.7
	maxTokens := 150
	stopSequences := []string{"\n\n", "END"}

	params := modelcontextprotocol.CreateMessageRequestParams{
		Messages:         messages,
		ModelPreferences: modelPrefs,
		SystemPrompt:     &systemPrompt,
		Temperature:      &temperature,
		MaxTokens:        maxTokens,
		StopSequences:    stopSequences,
		Metadata: map[string]any{
			"session_id": "12345",
			"user_id":    "user789",
		},
	}

	// Test field values
	if len(params.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(params.Messages))
	}

	if params.MaxTokens != maxTokens {
		t.Errorf("Expected maxTokens %d, got %d", maxTokens, params.MaxTokens)
	}

	if params.Temperature == nil || *params.Temperature != temperature {
		t.Errorf("Expected temperature %f, got %v", temperature, params.Temperature)
	}

	if len(params.StopSequences) != 2 {
		t.Errorf("Expected 2 stop sequences, got %d", len(params.StopSequences))
	}

	// Test JSON marshaling
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal CreateMessageRequestParams: %v", err)
	}

	var unmarshaled modelcontextprotocol.CreateMessageRequestParams
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal CreateMessageRequestParams: %v", err)
	}

	if len(unmarshaled.Messages) != len(params.Messages) {
		t.Errorf("Messages count mismatch after round-trip")
	}

	if unmarshaled.MaxTokens != params.MaxTokens {
		t.Errorf("MaxTokens mismatch after round-trip")
	}
}

// TestCreateMessageResult tests CreateMessage result
func TestCreateMessageResult(t *testing.T) {
	result := modelcontextprotocol.CreateMessageResult{
		Role: modelcontextprotocol.RoleAssistant,
		Content: modelcontextprotocol.TextContent{
			Type: "text",
			Text: "Based on current data, it's sunny with a temperature of 72°F.",
		},
		Model:      "gpt-3.5-turbo",
		StopReason: &[]string{"stop_sequence"}[0],
		Meta: map[string]any{
			"tokens_used":   42,
			"response_time": "150ms",
		},
	}

	// Test field values
	if result.Role != modelcontextprotocol.RoleAssistant {
		t.Errorf("Expected assistant role, got %s", result.Role)
	}

	if result.StopReason == nil || *result.StopReason != "stop_sequence" {
		t.Errorf("Expected stop_sequence, got %v", result.StopReason)
	}

	if result.Model != "gpt-3.5-turbo" {
		t.Errorf("Expected model gpt-3.5-turbo, got %s", result.Model)
	}

	// Test JSON marshaling
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal CreateMessageResult: %v", err)
	}

	var unmarshaled modelcontextprotocol.CreateMessageResult
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal CreateMessageResult: %v", err)
	}

	if unmarshaled.Role != result.Role {
		t.Errorf("Role mismatch after round-trip")
	}
}

// TestSamplingWithImageContent tests sampling with image content
func TestSamplingWithImageContent(t *testing.T) {
	imageContent := modelcontextprotocol.ImageContent{
		Type:     "image",
		Data:     "base64encodedimagedata...",
		MimeType: "image/jpeg",
	}

	samplingMsg := modelcontextprotocol.SamplingMessage{
		Role:    modelcontextprotocol.RoleUser,
		Content: imageContent,
	}

	// Test JSON marshaling with image content
	data, err := json.Marshal(samplingMsg)
	if err != nil {
		t.Fatalf("Failed to marshal SamplingMessage with image: %v", err)
	}

	var unmarshaled modelcontextprotocol.SamplingMessage
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SamplingMessage with image: %v", err)
	}

	if unmarshaled.Role != samplingMsg.Role {
		t.Errorf("Role mismatch with image content")
	}
}

// TestSamplingWithAudioContent tests sampling with audio content
func TestSamplingWithAudioContent(t *testing.T) {
	audioContent := modelcontextprotocol.AudioContent{
		Type:     "audio",
		Data:     "base64encodedaudiodata...",
		MimeType: "audio/wav",
	}

	samplingMsg := modelcontextprotocol.SamplingMessage{
		Role:    modelcontextprotocol.RoleUser,
		Content: audioContent,
	}

	// Test JSON marshaling with audio content
	data, err := json.Marshal(samplingMsg)
	if err != nil {
		t.Fatalf("Failed to marshal SamplingMessage with audio: %v", err)
	}

	var unmarshaled modelcontextprotocol.SamplingMessage
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SamplingMessage with audio: %v", err)
	}

	if unmarshaled.Role != samplingMsg.Role {
		t.Errorf("Role mismatch with audio content")
	}
}

// TestModelPreferencesAndHints tests model preferences and hints
func TestModelPreferencesAndHints(t *testing.T) {
	hints := []modelcontextprotocol.ModelHint{
		{Name: &[]string{"prefer-speed"}[0]},
		{Name: &[]string{"prefer-quality"}[0]},
		{Name: &[]string{"prefer-cost-efficiency"}[0]},
	}

	prefs := modelcontextprotocol.ModelPreferences{
		Hints: hints,
	}

	// Test JSON marshaling
	data, err := json.Marshal(prefs)
	if err != nil {
		t.Fatalf("Failed to marshal ModelPreferences: %v", err)
	}

	var unmarshaled modelcontextprotocol.ModelPreferences
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ModelPreferences: %v", err)
	}

	if len(unmarshaled.Hints) != len(prefs.Hints) {
		t.Errorf("Hints count mismatch: expected %d, got %d", len(prefs.Hints), len(unmarshaled.Hints))
	}

	for i, hint := range unmarshaled.Hints {
		if hint.Name == nil || prefs.Hints[i].Name == nil || *hint.Name != *prefs.Hints[i].Name {
			t.Errorf("Hint %d name mismatch: expected %v, got %v", i, prefs.Hints[i].Name, hint.Name)
		}
	}
}

// TestSamplingCapabilityNegotiation tests sampling capability negotiation
func TestSamplingCapabilityNegotiation(t *testing.T) {
	// Round-trip the root ClientCapabilities type that the Server actually gates
	// server-initiated requests on (see Server.CreateMessage/Elicit/ListRoots),
	// covering sampling, elicitation, and roots together.
	clientCaps := ClientCapabilities{
		Sampling:    &struct{}{},
		Elicitation: &ElicitationCapabilities{Form: &struct{}{}, URL: &struct{}{}},
		Roots: &struct {
			ListChanged bool `json:"listChanged,omitempty"`
		}{ListChanged: true},
	}

	data, err := json.Marshal(clientCaps)
	if err != nil {
		t.Fatalf("Failed to marshal client capabilities: %v", err)
	}

	var unmarshaled ClientCapabilities
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal client capabilities: %v", err)
	}

	if unmarshaled.Sampling == nil {
		t.Error("Sampling capability should be non-nil after round-trip")
	}
	if unmarshaled.Elicitation == nil || unmarshaled.Elicitation.Form == nil || unmarshaled.Elicitation.URL == nil {
		t.Errorf("Elicitation capability not preserved: %+v", unmarshaled.Elicitation)
	}
	if unmarshaled.Roots == nil || !unmarshaled.Roots.ListChanged {
		t.Errorf("Roots capability not preserved: %+v", unmarshaled.Roots)
	}
}

// TestSamplingConversationFlow tests a complete conversation flow
func TestSamplingConversationFlow(t *testing.T) {
	// Create a conversation with multiple turns
	conversation := []modelcontextprotocol.SamplingMessage{
		{
			Role: modelcontextprotocol.RoleUser,
			Content: modelcontextprotocol.TextContent{
				Type: "text",
				Text: "Hello, I need help with a math problem.",
			},
		},
		{
			Role: modelcontextprotocol.RoleAssistant,
			Content: modelcontextprotocol.TextContent{
				Type: "text",
				Text: "I'd be happy to help you with your math problem! What specific problem are you working on?",
			},
		},
		{
			Role: modelcontextprotocol.RoleUser,
			Content: modelcontextprotocol.TextContent{
				Type: "text",
				Text: "I need to solve: 2x + 5 = 15",
			},
		},
	}

	// Create sampling request with the conversation
	params := modelcontextprotocol.CreateMessageRequestParams{
		Messages:     conversation,
		MaxTokens:    100,
		Temperature:  &[]float64{0.3}[0], // Low temperature for math
		SystemPrompt: &[]string{"You are a helpful math tutor. Explain your reasoning step by step."}[0],
	}

	// Test that the conversation structure is preserved
	if len(params.Messages) != 3 {
		t.Errorf("Expected 3 messages in conversation, got %d", len(params.Messages))
	}

	// Test alternating roles
	expectedRoles := []modelcontextprotocol.Role{
		modelcontextprotocol.RoleUser,
		modelcontextprotocol.RoleAssistant,
		modelcontextprotocol.RoleUser,
	}

	for i, msg := range params.Messages {
		if msg.Role != expectedRoles[i] {
			t.Errorf("Message %d: expected role %s, got %s", i, expectedRoles[i], msg.Role)
		}
	}

	// Test JSON serialization of the full conversation
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal conversation: %v", err)
	}

	var unmarshaled modelcontextprotocol.CreateMessageRequestParams
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal conversation: %v", err)
	}

	if len(unmarshaled.Messages) != len(params.Messages) {
		t.Errorf("Message count mismatch after serialization")
	}
}

// TestSamplingParameterValidation tests parameter validation
func TestSamplingParameterValidation(t *testing.T) {
	testCases := []struct {
		name        string
		maxTokens   int
		temperature *float64
		expectValid bool
	}{
		{
			name:        "valid parameters",
			maxTokens:   100,
			temperature: &[]float64{0.7}[0],
			expectValid: true,
		},
		{
			name:        "zero max tokens",
			maxTokens:   0,
			temperature: &[]float64{0.7}[0],
			expectValid: false, // Typically invalid
		},
		{
			name:        "negative max tokens",
			maxTokens:   -1,
			temperature: &[]float64{0.7}[0],
			expectValid: false,
		},
		{
			name:        "high temperature",
			maxTokens:   100,
			temperature: &[]float64{2.0}[0],
			expectValid: true, // May be valid depending on model
		},
		{
			name:        "negative temperature",
			maxTokens:   100,
			temperature: &[]float64{-0.1}[0],
			expectValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := modelcontextprotocol.CreateMessageRequestParams{
				Messages: []modelcontextprotocol.SamplingMessage{
					{
						Role: modelcontextprotocol.RoleUser,
						Content: modelcontextprotocol.TextContent{
							Type: "text",
							Text: "Test message",
						},
					},
				},
				MaxTokens:   tc.maxTokens,
				Temperature: tc.temperature,
			}

			// Test that we can serialize the parameters
			// (In a real implementation, you'd validate these parameters)
			_, err := json.Marshal(params)
			if err != nil {
				t.Errorf("Failed to marshal parameters: %v", err)
			}

			// Log the test case for demonstration
			t.Logf("Parameters: maxTokens=%d, temperature=%v, expected_valid=%v",
				tc.maxTokens, tc.temperature, tc.expectValid)
		})
	}
}

// BenchmarkSamplingMessageSerialization benchmarks sampling message serialization
func BenchmarkSamplingMessageSerialization(b *testing.B) {
	samplingMsg := modelcontextprotocol.SamplingMessage{
		Role: modelcontextprotocol.RoleUser,
		Content: modelcontextprotocol.TextContent{
			Type: "text",
			Text: "This is a test message for benchmarking serialization performance.",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(samplingMsg)
		if err != nil {
			b.Fatal(err)
		}

		var unmarshaled modelcontextprotocol.SamplingMessage
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCreateMessageRequest benchmarks CreateMessage request serialization
func BenchmarkCreateMessageRequest(b *testing.B) {
	messages := make([]modelcontextprotocol.SamplingMessage, 10)
	for i := 0; i < 10; i++ {
		role := modelcontextprotocol.RoleUser
		if i%2 == 1 {
			role = modelcontextprotocol.RoleAssistant
		}

		messages[i] = modelcontextprotocol.SamplingMessage{
			Role: role,
			Content: modelcontextprotocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Message %d in the conversation", i),
			},
		}
	}

	params := modelcontextprotocol.CreateMessageRequestParams{
		Messages:    messages,
		MaxTokens:   200,
		Temperature: &[]float64{0.7}[0],
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(params)
		if err != nil {
			b.Fatal(err)
		}

		var unmarshaled modelcontextprotocol.CreateMessageRequestParams
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			b.Fatal(err)
		}
	}
}
