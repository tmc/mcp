//go:build go1.18
// +build go1.18

package draft

import (
	"encoding/json"
	"testing"
)

func FuzzContentListUnmarshal(f *testing.F) {
	// Seed with valid examples
	seeds := []string{
		`[{"type": "text", "text": "Hello"}]`,
		`[{"type": "image", "data": "imgdata", "mimeType": "image/png"}]`,
		`[{"type": "audio", "data": "audiodata", "mimeType": "audio/mp3"}]`,
		`[{"type": "resource", "resource": {"uri": "/file.txt", "text": "content"}}]`,
		`[{"type": "text", "text": "a"}, {"type": "image", "data": "b", "mimeType": "image/jpeg"}]`,
		`[]`,
		`null`,
		`[{"type": "unknown"}]`,
		`[{"type": "text"}]`,                     // missing required field
		`[{"type": "resource", "resource": {}}]`, // invalid resource
		`[{}]`,                                   // no type field
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		var cl ContentList
		err := json.Unmarshal([]byte(data), &cl)
		_ = err

		// If unmarshal succeeds, try to marshal back
		if err == nil {
			_, marshalErr := json.Marshal(cl)
			_ = marshalErr
		}
	})
}

func FuzzCallToolResultDraftUnmarshal(f *testing.F) {
	seeds := []string{
		`{"content": [{"type": "text", "text": "result"}]}`,
		`{"structuredContent": {"key": "value"}}`,
		`{"content": [{"type": "text", "text": "text"}], "structuredContent": {"data": "structured"}}`,
		`{"isError": true}`,
		`{"isError": false, "content": []}`,
		`{}`,
		`{"_meta": {"key": "value"}}`,
		`{"content": null, "structuredContent": null}`,
		`{"structuredContent": {}, "content": [{"type": "text", "text": "compat"}]}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		var ctr CallToolResult
		err := json.Unmarshal([]byte(data), &ctr)
		_ = err

		if err == nil {
			// Try to marshal back
			marshaledData, marshalErr := json.Marshal(ctr)
			_ = marshalErr

			// If marshal succeeds, try to unmarshal again to test roundtrip
			if marshalErr == nil {
				var ctr2 CallToolResult
				unmarshalErr := json.Unmarshal(marshaledData, &ctr2)
				_ = unmarshalErr
			}
		}
	})
}

func FuzzCallToolResultDraftMarshal(f *testing.F) {
	// Test marshaling with different states
	f.Add(true, false, false) // isStructured, hasContent, isError
	f.Add(false, true, false)
	f.Add(true, true, false)
	f.Add(true, false, true)
	f.Add(false, false, true)

	f.Fuzz(func(t *testing.T, isStructured bool, hasContent bool, isError bool) {
		ctr := CallToolResult{
			isStructuredResult: isStructured,
		}

		if isError {
			e := true
			ctr.IsError = &e
		}

		if hasContent {
			cl := ContentList{
				TextContent{Type: "text", Text: "test"},
			}
			ctr.Content = &cl
		}

		if isStructured {
			ctr.StructuredContent = map[string]interface{}{
				"result": "structured",
			}
		}

		// Try to marshal
		data, err := json.Marshal(ctr)
		_ = err

		// If marshal succeeds, try to unmarshal back
		if err == nil {
			var ctr2 CallToolResult
			unmarshalErr := json.Unmarshal(data, &ctr2)
			_ = unmarshalErr
		}
	})
}
