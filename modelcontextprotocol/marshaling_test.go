package modelcontextprotocol

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Helper to wrap content for tests where a struct expects a Content interface
//
//nolint:unused // Test helper function that may be needed in future tests
func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Helper to create JSON content for testing
func makeContent(t *testing.T, contentType string, content any) json.RawMessage {
	t.Helper()
	data := map[string]any{
		"type": contentType,
	}
	switch contentType {
	case "text":
		data["text"] = content
	case "image":
		data["data"] = content
		data["mimeType"] = "image/png"
	case "audio":
		data["data"] = content
		data["mimeType"] = "audio/mp3"
	}
	return mustMarshal(t, data)
}

func TestUnmarshalContent(t *testing.T) {
	tests := []struct {
		name string
		json string
		want Content
	}{
		{
			name: "text content",
			json: `{"type": "text", "text": "hello world"}`,
			want: TextContent{Type: ContentTypeText, Text: "hello world"},
		},
		{
			name: "image content",
			json: `{"type": "image", "data": "base64data", "mimeType": "image/jpeg"}`,
			want: ImageContent{Type: ContentTypeImage, Data: "base64data", MimeType: "image/jpeg"},
		},
		{
			name: "audio content",
			json: `{"type": "audio", "data": "audiodata", "mimeType": "audio/wav"}`,
			want: AudioContent{Type: ContentTypeAudio, Data: "audiodata", MimeType: "audio/wav"},
		},
		{
			name: "embedded resource",
			json: `{"type": "resource", "resource": {"uri": "file://path", "text": "content"}}`,
			want: EmbeddedResource{
				Type: ContentTypeResource,
				Resource: TextResourceContents{
					BaseResourceContents: BaseResourceContents{URI: "file://path"},
					Text:                 "content",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshalContentInternal(json.RawMessage(tt.json))
			if err != nil {
				t.Fatalf("unmarshalContentInternal() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalResourceContents(t *testing.T) {
	tests := []struct {
		name string
		json string
		want ResourceContents
	}{
		{
			name: "text resource",
			json: `{"uri": "file://doc.md", "text": "content", "mimeType": "text/markdown"}`,
			want: TextResourceContents{
				BaseResourceContents: BaseResourceContents{
					URI:      "file://doc.md",
					MimeType: strPtr("text/markdown"),
				},
				Text: "content",
			},
		},
		{
			name: "blob resource",
			json: `{"uri": "file://image.png", "blob": "base64data", "mimeType": "image/png"}`,
			want: BlobResourceContents{
				BaseResourceContents: BaseResourceContents{
					URI:      "file://image.png",
					MimeType: strPtr("image/png"),
				},
				Blob: "base64data",
			},
		},
		{
			name: "text resource no mime",
			json: `{"uri": "file://doc.txt", "text": "plain text"}`,
			want: TextResourceContents{
				BaseResourceContents: BaseResourceContents{URI: "file://doc.txt"},
				Text:                 "plain text",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshalResourceContentsInternal(json.RawMessage(tt.json))
			if err != nil {
				t.Fatalf("unmarshalResourceContentsInternal() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalReference(t *testing.T) {
	tests := []struct {
		name string
		json string
		want Reference
	}{
		{
			name: "prompt reference",
			json: `{"type": "ref/prompt", "name": "test-prompt"}`,
			want: PromptReference{Type: ReferenceTypePrompt, Name: "test-prompt"},
		},
		{
			name: "resource reference",
			json: `{"type": "ref/resource", "uri": "file://resource.txt"}`,
			want: ResourceReference{Type: ReferenceTypeResource, URI: "file://resource.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshalReferenceInternal(json.RawMessage(tt.json))
			if err != nil {
				t.Fatalf("unmarshalReferenceInternal() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalComplexStructs(t *testing.T) {
	t.Run("PromptMessage", func(t *testing.T) {
		json := `{"role": "user", "content": {"type": "text", "text": "hello"}}`
		want := PromptMessage{
			Role:    RoleUser,
			Content: TextContent{Type: ContentTypeText, Text: "hello"},
		}
		var got PromptMessage
		err := got.UnmarshalJSON([]byte(json))
		if err != nil {
			t.Fatalf("UnmarshalJSON() error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("SamplingMessage", func(t *testing.T) {
		json := `{"role": "assistant", "content": {"type": "text", "text": "response"}}`
		want := SamplingMessage{
			Role:    RoleAssistant,
			Content: TextContent{Type: ContentTypeText, Text: "response"},
		}
		var got SamplingMessage
		err := got.UnmarshalJSON([]byte(json))
		if err != nil {
			t.Fatalf("UnmarshalJSON() error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("CreateMessageResult", func(t *testing.T) {
		json := `{"role": "assistant", "content": {"type": "text", "text": "response"}, "model": "test-model"}`
		want := CreateMessageResult{
			Role:    RoleAssistant,
			Content: TextContent{Type: ContentTypeText, Text: "response"},
			Model:   "test-model",
		}
		var got CreateMessageResult
		err := got.UnmarshalJSON([]byte(json))
		if err != nil {
			t.Fatalf("UnmarshalJSON() error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("CompleteRequestParams", func(t *testing.T) {
		json := `{"ref": {"type": "ref/prompt", "name": "test"}, "argument": {"name": "arg1", "value": "val1"}}`
		want := CompleteRequestParams{
			Ref: PromptReference{Type: ReferenceTypePrompt, Name: "test"},
			Argument: struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			}{Name: "arg1", Value: "val1"},
		}
		var got CompleteRequestParams
		err := got.UnmarshalJSON([]byte(json))
		if err != nil {
			t.Fatalf("UnmarshalJSON() error = %v", err)
		}
		// Compare fields individually due to anonymous struct
		if got.Ref != want.Ref || got.Argument.Name != want.Argument.Name || got.Argument.Value != want.Argument.Value {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("ReadResourceResult", func(t *testing.T) {
		json := `{"contents": [{"uri": "file://test.txt", "text": "content"}]}`
		want := ReadResourceResult{
			Contents: []ResourceContents{
				TextResourceContents{
					BaseResourceContents: BaseResourceContents{URI: "file://test.txt"},
					Text:                 "content",
				},
			},
		}
		var got ReadResourceResult
		err := got.UnmarshalJSON([]byte(json))
		if err != nil {
			t.Fatalf("UnmarshalJSON() error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("CallToolResult", func(t *testing.T) {
		json := `{"content": [{"type": "text", "text": "result"}], "isError": false}`
		want := CallToolResult{
			Content: []Content{
				TextContent{Type: ContentTypeText, Text: "result"},
			},
			IsError: boolPtr(false),
		}
		var got CallToolResult
		err := got.UnmarshalJSON([]byte(json))
		if err != nil {
			t.Fatalf("UnmarshalJSON() error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}

func TestMarshalUnmarshalRoundtrip(t *testing.T) {
	testCases := []struct {
		name        string
		input       any
		newInstance func() any
	}{
		{
			name: "Text Content with Annotations",
			input: TextContent{
				Type:        ContentTypeText,
				Text:        "Test text",
				Annotations: &Annotations{Audience: []Role{RoleUser}, Priority: floatPtr(0.5)},
			},
			newInstance: func() any { return new(TextContent) },
		},
		{
			name:        "Prompt Message with various content types",
			input:       PromptMessage{Role: RoleUser, Content: TextContent{Type: ContentTypeText, Text: "Hello"}},
			newInstance: func() any { return new(PromptMessage) },
		},
		{
			name:        "Create Message Result",
			input:       CreateMessageResult{Role: RoleAssistant, Content: ImageContent{Type: ContentTypeImage, Data: "base64", MimeType: "image/png"}, Model: "test", StopReason: strPtr("complete")},
			newInstance: func() any { return new(CreateMessageResult) },
		},
		{
			name:        "Call Tool Result with multiple contents",
			input:       CallToolResult{Content: []Content{TextContent{Type: ContentTypeText, Text: "Result 1"}, ImageContent{Type: ContentTypeImage, Data: "imgdata", MimeType: "image/jpeg"}}, IsError: boolPtr(false)},
			newInstance: func() any { return new(CallToolResult) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			targetInstance := tc.newInstance()
			err = json.Unmarshal(jsonData, targetInstance)
			if err != nil {
				t.Fatalf("json.Unmarshal() error = %v\nJSON: %s", err, string(jsonData))
			}

			// Use reflect.DeepEqual to compare the original and the unmarshaled instance.
			// Note: For interfaces, DeepEqual compares concrete types.
			if !reflect.DeepEqual(tc.input, reflect.Indirect(reflect.ValueOf(targetInstance)).Interface()) {
				t.Errorf("Marshal/Unmarshal roundtrip failed.\nOriginal: %+v\nUnmarshaled: %+v\nJSON: %s",
					tc.input, targetInstance, string(jsonData))
			}
		})
	}
}

// Helper functions for pointers
func floatPtr(f float64) *float64 { return &f }
func strPtr(s string) *string     { return &s }
func boolPtr(b bool) *bool        { return &b }

// Simplified test for unmarshal error handling
func TestUnmarshalErrors(t *testing.T) {
	// Test various invalid JSON inputs
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "invalid content json",
			fn: func() error {
				var c Content
				return json.Unmarshal([]byte(`{invalid}`), &c)
			},
		},
		{
			name: "invalid resource contents json",
			fn: func() error {
				var rc ResourceContents
				return json.Unmarshal([]byte(`{invalid}`), &rc)
			},
		},
		{
			name: "invalid reference json",
			fn: func() error {
				var r Reference
				return json.Unmarshal([]byte(`{invalid}`), &r)
			},
		},
		{
			name: "embedded resource error",
			fn: func() error {
				var er EmbeddedResource
				return er.UnmarshalJSON([]byte(`{invalid}`))
			},
		},
		{
			name: "prompt message error",
			fn: func() error {
				var pm PromptMessage
				return pm.UnmarshalJSON([]byte(`{invalid}`))
			},
		},
		{
			name: "sampling message error",
			fn: func() error {
				var sm SamplingMessage
				return sm.UnmarshalJSON([]byte(`{invalid}`))
			},
		},
		{
			name: "create message result error",
			fn: func() error {
				var cmr CreateMessageResult
				return cmr.UnmarshalJSON([]byte(`{invalid}`))
			},
		},
		{
			name: "complete request params error",
			fn: func() error {
				var crp CompleteRequestParams
				return crp.UnmarshalJSON([]byte(`{invalid}`))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Error("expected error but got none")
			}
		})
	}
}
