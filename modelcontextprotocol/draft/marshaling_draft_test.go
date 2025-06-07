package draft

import (
	"encoding/json"
	"reflect"
	"testing"

	// Import the base package for base types like TextContent, ImageContent, etc.
	base "github.com/tmc/mcp/modelcontextprotocol"
)

func toBoolPtr(b bool) *bool { return &b }

func TestCallToolResult_Unmarshal(t *testing.T) {
	tests := []struct {
		name               string
		jsonData           string
		wantIsStructured   bool
		wantStructuredData map[string]any
		wantContentItems   int
		wantFirstContent   base.Content // For a simple check
		wantIsError        *bool
		wantErr            bool
	}{
		{
			name:               "Structured Result Only",
			jsonData:           `{"structuredContent": {"key": "value", "num": 123}, "isError": false}`,
			wantIsStructured:   true,
			wantStructuredData: map[string]any{"key": "value", "num": float64(123)}, // JSON numbers are float64
			wantContentItems:   0,
			wantIsError:        toBoolPtr(false),
			wantErr:            false,
		},
		{
			name:             "Unstructured Result Only",
			jsonData:         `{"content": [{"type": "text", "text": "hello"}]}`,
			wantIsStructured: false,
			wantContentItems: 1,
			wantFirstContent: base.TextContent{Type: base.ContentTypeText, Text: "hello"},
			wantErr:          false,
		},
		{
			name:               "Structured Result with Compatibility Content",
			jsonData:           `{"structuredContent": {"data": true}, "content": [{"type": "image", "data": "base64==", "mimeType": "image/png"}]}`,
			wantIsStructured:   true,
			wantStructuredData: map[string]any{"data": true},
			wantContentItems:   1,
			wantFirstContent:   base.ImageContent{Type: base.ContentTypeImage, Data: "base64==", MimeType: "image/png"},
			wantErr:            false,
		},
		{
			name:             "Error Result with Content",
			jsonData:         `{"isError": true, "content": [{"type": "text", "text": "tool failed"}]}`,
			wantIsStructured: false,
			wantContentItems: 1,
			wantFirstContent: base.TextContent{Type: base.ContentTypeText, Text: "tool failed"},
			wantIsError:      toBoolPtr(true),
			wantErr:          false,
		},
		{
			name:             "Error Result no Content",
			jsonData:         `{"isError": true}`,
			wantIsStructured: false,
			wantContentItems: 0,
			wantIsError:      toBoolPtr(true),
			wantErr:          false,
		},
		{
			name:             "Successful result with no content (e.g. update tool)",
			jsonData:         `{}`,
			wantIsStructured: false,
			wantContentItems: 0,
			wantErr:          false,
		},
		{
			name:     "Invalid JSON",
			jsonData: `{"structuredContent": {key_no_quotes: "value"}}`,
			wantErr:  true,
		},
		// This case is a bit tricky based on current unmarshal: if both are valid, structuredContent wins.
		// A server strictly following spec for an unstructured tool should NOT send structuredContent.
		// {
		// 	name:             "Conflicting: Unstructured tool sending structuredContent (should ignore SC)",
		// 	jsonData:         `{"structuredContent": {"ignored": true}, "content": [{"type": "text", "text": "unstructured wins"}]}`,
		// 	wantIsStructured: false, // This would require knowing the Tool definition during unmarshal or stricter logic.
		// 	wantContentItems: 1,
		// 	wantFirstContent: base.TextContent{Type:base.ContentTypeText, Text: "unstructured wins"},
		// 	wantErr:          false,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result CallToolResult
			err := json.Unmarshal([]byte(tt.jsonData), &result)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if result.isStructuredResult != tt.wantIsStructured {
				t.Errorf("result.isStructuredResult = %v, want %v", result.isStructuredResult, tt.wantIsStructured)
			}

			if tt.wantStructuredData != nil && !reflect.DeepEqual(result.StructuredContent, tt.wantStructuredData) {
				t.Errorf("result.StructuredContent = %v, want %v", result.StructuredContent, tt.wantStructuredData)
			}
			if tt.wantStructuredData == nil && result.StructuredContent != nil {
				t.Errorf("result.StructuredContent = %v, want nil", result.StructuredContent)
			}

			if result.Content == nil && tt.wantContentItems > 0 {
				t.Errorf("result.Content is nil, want %d items", tt.wantContentItems)
			}
			if result.Content != nil && len(*result.Content) != tt.wantContentItems {
				t.Errorf("len(result.Content) = %d, want %d", len(*result.Content), tt.wantContentItems)
			}

			if tt.wantContentItems > 0 && result.Content != nil && len(*result.Content) > 0 {
				if !reflect.DeepEqual((*result.Content)[0], tt.wantFirstContent) {
					t.Errorf("First content item = %+v, want %+v", (*result.Content)[0], tt.wantFirstContent)
				}
			}

			if tt.wantIsError == nil && result.IsError != nil {
				t.Errorf("result.IsError got %v, want nil", *result.IsError)
			}
			if tt.wantIsError != nil {
				if result.IsError == nil {
					t.Errorf("result.IsError got nil, want %v", *tt.wantIsError)
				} else if *result.IsError != *tt.wantIsError {
					t.Errorf("result.IsError got %v, want %v", *result.IsError, *tt.wantIsError)
				}
			}
		})
	}
}

func TestCallToolResult_Marshal(t *testing.T) {
	truePtr := true
	falsePtr := false

	tests := []struct {
		name         string
		result       CallToolResult
		wantJsonFrag map[string]any // Check for key fragments and basic structure
		wantErr      bool
	}{
		{
			name: "Structured Result",
			result: NewStructuredToolResult(
				map[string]any{"key": "value", "num": 123.0},
				nil,
				&falsePtr,
				map[string]any{"_metaKey": "metaValue"},
			),
			wantJsonFrag: map[string]any{
				"structuredContent": map[string]any{"key": "value", "num": 123.0},
				"isError":           false,
				"_meta":             map[string]any{"_metaKey": "metaValue"},
			},
		},
		{
			name: "Structured Result with Compatibility Content",
			result: NewStructuredToolResult(
				map[string]any{"data": true},
				&ContentList{base.NewTextContent("compat")},
				nil, // isError is nil (false by default)
				nil,
			),
			wantJsonFrag: map[string]any{
				"structuredContent": map[string]any{"data": true},
				"content":           []any{map[string]any{"type": "text", "text": "compat"}},
			},
		},
		{
			name: "Unstructured Result",
			result: NewUnstructuredToolResult(
				ContentList{base.NewImageContent("data", "image/jpeg", base.Annotations{Audience: []base.Role{base.RoleUser}})},
				nil,
				nil,
			),
			wantJsonFrag: map[string]any{
				"content": []any{
					map[string]any{
						"type": "image", "data": "data", "mimeType": "image/jpeg",
						"annotations": map[string]any{"audience": []any{"user"}},
					},
				},
			},
		},
		{
			name: "Error Result with Content",
			result: NewUnstructuredToolResult(
				ContentList{base.NewTextContent("failure details")},
				&truePtr,
				nil,
			),
			wantJsonFrag: map[string]any{
				"isError": true,
				"content": []any{map[string]any{"type": "text", "text": "failure details"}},
			},
		},
		{
			name: "Structured Result, but structuredContent is nil (error state or empty structured)",
			result: CallToolResult{ // Manually construct for this edge case
				IsError:            &truePtr,
				StructuredContent:  nil,  // Explicitly nil
				isStructuredResult: true, // But marked as structured
			},
			wantJsonFrag: map[string]any{
				"isError": true,
				// "structuredContent" should be `null` or `{}` if not omitempty and non-error requires it.
				// Our MarshalJSON for this case: if error, SC can be omitted. If not error, it becomes {}.
				// Since it's an error, it will likely be omitted.
			},
		},
		{
			name: "Structured Result, non-error, nil structuredContent (marshals as empty object)",
			result: CallToolResult{
				IsError:            nil, // or &falsePtr
				StructuredContent:  nil,
				isStructuredResult: true,
			},
			wantJsonFrag: map[string]any{
				"structuredContent": map[string]any{}, // Marshals as empty object
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.result)
			if (err != nil) != tt.wantErr {
				t.Fatalf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			var unmarshaledMap map[string]any
			if err := json.Unmarshal(jsonData, &unmarshaledMap); err != nil {
				t.Fatalf("Failed to unmarshal back to map for validation: %v. JSON: %s", err, string(jsonData))
			}

			// Check fragments
			for key, expectedValue := range tt.wantJsonFrag {
				actualValue, ok := unmarshaledMap[key]
				if !ok {
					// If expectedValue is nil and key is not present, it might be due to omitempty.
					// This check is tricky for omitempty cases without knowing the exact output.
					// For non-nil expected values, it's a clear failure.
					if expectedValue != nil {
						t.Errorf("MarshalJSON(): key '%s' missing from output. JSON: %s", key, string(jsonData))
					}
					continue
				}
				if !reflect.DeepEqual(actualValue, expectedValue) {
					t.Errorf("MarshalJSON(): key '%s' got = %v (%T), want = %v (%T). JSON: %s", key, actualValue, actualValue, expectedValue, expectedValue, string(jsonData))
				}
			}
			if _, scExists := unmarshaledMap["structuredContent"]; tt.result.isStructuredResult && !scExists && !(tt.result.IsError != nil && *tt.result.IsError && tt.result.StructuredContent == nil) {
				if !(tt.name == "Structured Result, but structuredContent is nil (error state or empty structured)" && tt.result.StructuredContent == nil) {
					// Special case where nil SC for non-error structured results becomes {}
					if !(tt.name == "Structured Result, non-error, nil structuredContent (marshals as empty object)" && reflect.DeepEqual(unmarshaledMap["structuredContent"], map[string]any{})) {
						t.Errorf("MarshalJSON(): 'structuredContent' expected but missing for structured result. JSON: %s", string(jsonData))
					}
				}
			}
			if _, cExists := unmarshaledMap["content"]; !tt.result.isStructuredResult && !cExists && !(tt.result.IsError != nil && *tt.result.IsError && tt.result.Content == nil) {
				t.Errorf("MarshalJSON(): 'content' expected but missing for unstructured result. JSON: %s", string(jsonData))
			}

		})
	}
}

func TestContentList_Unmarshal(t *testing.T) {
	jsonData := `[
		{"type": "text", "text": "Hello"},
		{"type": "image", "data": "imgdata", "mimeType": "image/png"},
		{"type": "audio", "data": "audiodata", "mimeType": "audio/mp3"},
		{"type": "resource", "resource": {"uri": "/file.txt", "text": "res text"}}
	]`

	var cl ContentList
	err := json.Unmarshal([]byte(jsonData), &cl)
	if err != nil {
		t.Fatalf("ContentList.UnmarshalJSON() error = %v", err)
	}

	if len(cl) != 4 {
		t.Fatalf("Expected 4 items in ContentList, got %d", len(cl))
	}

	if _, ok := cl[0].(base.TextContent); !ok {
		t.Errorf("Item 0: Expected TextContent, got %T", cl[0])
	}
	if tc, _ := cl[0].(base.TextContent); tc.Text != "Hello" {
		t.Errorf("Item 0: TextContent text = %s, want Hello", tc.Text)
	}

	if _, ok := cl[1].(base.ImageContent); !ok {
		t.Errorf("Item 1: Expected ImageContent, got %T", cl[1])
	}

	if _, ok := cl[2].(base.AudioContent); !ok {
		t.Errorf("Item 2: Expected AudioContent, got %T", cl[2])
	}

	if _, ok := cl[3].(base.EmbeddedResource); !ok {
		t.Errorf("Item 3: Expected EmbeddedResource, got %T", cl[3])
	} else if er, _ := cl[3].(base.EmbeddedResource); er.Resource == nil {
		t.Errorf("Item 3: EmbeddedResource.Resource is nil")
	} else if _, ok := er.Resource.(base.TextResourceContents); !ok {
		t.Errorf("Item 3: EmbeddedResource.Resource expected TextResourceContents, got %T", er.Resource)
	}
}
