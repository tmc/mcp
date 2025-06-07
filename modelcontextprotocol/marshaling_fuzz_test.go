//go:build go1.18
// +build go1.18

package modelcontextprotocol

import (
	"encoding/json"
	"testing"
)

func FuzzUnmarshalContent(f *testing.F) {
	// Seed corpus with valid examples
	seeds := []string{
		`{"type": "text", "text": "Hello, world!"}`,
		`{"type": "image", "data": "base64data", "mimeType": "image/png"}`,
		`{"type": "audio", "data": "audiodata", "mimeType": "audio/mp3"}`,
		`{"type": "resource", "resource": {"uri": "/file.txt", "text": "content"}}`,
		`{"type": "resource", "resource": {"uri": "/file.bin", "blob": "base64"}}`,
		`null`,
		`{}`,
		`{"type": "unknown"}`,
		`{"type": "text"}`,                     // missing required field
		`{"type": "image", "data": ""}`,        // empty data
		`{"type": "resource", "resource": {}}`, // missing uri
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		var raw json.RawMessage = json.RawMessage(data)
		content, err := unmarshalContentInternal(raw)

		// We don't check for errors because fuzz testing is meant to find crashes
		// Valid error returns are expected for invalid inputs
		_ = err
		_ = content

		// If it unmarshals successfully, try to marshal it back
		if err == nil && content != nil {
			_, marshalErr := json.Marshal(content)
			_ = marshalErr
		}
	})
}

func FuzzUnmarshalResourceContents(f *testing.F) {
	seeds := []string{
		`{"uri": "/file.txt", "text": "content"}`,
		`{"uri": "/file.bin", "blob": "base64data"}`,
		`{"uri": "/file.bin", "blob": "data", "mimeType": "application/octet-stream"}`,
		`{"uri": "/file.txt", "text": "content", "mimeType": "text/plain"}`,
		`{"uri": "/file.txt"}`, // missing both text and blob
		`{"text": "content"}`,  // missing uri
		`{"blob": "data"}`,     // missing uri
		`{"uri": "/file.txt", "text": "content", "blob": "data"}`, // both text and blob
		`null`,
		`{}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		var raw json.RawMessage = json.RawMessage(data)
		contents, err := unmarshalResourceContentsInternal(raw)
		_ = err
		_ = contents

		if err == nil && contents != nil {
			_, marshalErr := json.Marshal(contents)
			_ = marshalErr
		}
	})
}

func FuzzUnmarshalReference(f *testing.F) {
	seeds := []string{
		`{"type": "ref/resource", "uri": "/my/resource"}`,
		`{"type": "ref/prompt", "name": "myPrompt"}`,
		`{"type": "ref/unknown"}`,
		`{"type": "ref/resource"}`, // missing required field
		`{"type": "ref/prompt"}`,   // missing required field
		`{}`,
		`null`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		var raw json.RawMessage = json.RawMessage(data)
		ref, err := unmarshalReferenceInternal(raw)
		_ = err
		_ = ref

		if err == nil && ref != nil {
			_, marshalErr := json.Marshal(ref)
			_ = marshalErr
		}
	})
}

func FuzzEmbeddedResourceUnmarshal(f *testing.F) {
	seeds := []string{
		`{"type": "resource", "resource": {"uri": "/file.txt", "text": "hello"}}`,
		`{"type": "resource", "resource": {"uri": "/file.bin", "blob": "YmluYXJ5"}}`,
		`{"type": "resource"}`, // missing resource field
		`{"type": "resource", "resource": null}`,
		`{"type": "resource", "resource": {}}`,
		`{}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		var er EmbeddedResource
		err := json.Unmarshal([]byte(data), &er)
		_ = err

		if err == nil {
			// Try to marshal back
			_, marshalErr := json.Marshal(er)
			_ = marshalErr
		}
	})
}

func FuzzCallToolResultUnmarshal(f *testing.F) {
	seeds := []string{
		`{"content": [{"type": "text", "text": "result"}]}`,
		`{"content": [], "isError": true}`,
		`{"content": null}`,
		`{"isError": true}`,
		`{}`,
		`{"content": [{"type": "text", "text": "a"}, {"type": "image", "data": "b", "mimeType": "image/png"}]}`,
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
			_, marshalErr := json.Marshal(ctr)
			_ = marshalErr
		}
	})
}

func FuzzPromptMessageUnmarshal(f *testing.F) {
	seeds := []string{
		`{"role": "user", "content": {"type": "text", "text": "Hi"}}`,
		`{"role": "assistant", "content": {"type": "image", "data": "img", "mimeType": "image/png"}}`,
		`{"role": "user"}`, // missing content
		`{"content": {"type": "text", "text": "Hi"}}`, // missing role
		`{}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		var pm PromptMessage
		err := json.Unmarshal([]byte(data), &pm)
		_ = err

		if err == nil {
			// Try to marshal back
			_, marshalErr := json.Marshal(pm)
			_ = marshalErr
		}
	})
}

func FuzzReadResourceResultUnmarshal(f *testing.F) {
	seeds := []string{
		`{"contents": [{"uri": "/a.txt", "text": "a"}, {"uri": "/b.bin", "blob": "b"}]}`,
		`{"contents": []}`,
		`{"contents": null}`,
		`{}`,
		`{"contents": [{"invalid": "data"}]}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		var rrr ReadResourceResult
		err := json.Unmarshal([]byte(data), &rrr)
		_ = err

		if err == nil {
			// Try to marshal back
			_, marshalErr := json.Marshal(rrr)
			_ = marshalErr
		}
	})
}
