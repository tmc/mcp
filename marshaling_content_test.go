// Copyright 2025 The MCP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.

package mcp

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestImageContentRequiredFields verifies that ImageContent always emits the
// spec-required data and mimeType fields, even when empty.
func TestImageContentRequiredFields(t *testing.T) {
	data, err := json.Marshal(ImageContent{Type: "image"})
	if err != nil {
		t.Fatalf("marshal ImageContent: %v", err)
	}
	for _, field := range []string{`"data":`, `"mimeType":`} {
		if !bytes.Contains(data, []byte(field)) {
			t.Errorf("ImageContent JSON %s missing required field %s", data, field)
		}
	}

	data, err = json.Marshal(AudioContent{Type: "audio"})
	if err != nil {
		t.Fatalf("marshal AudioContent: %v", err)
	}
	for _, field := range []string{`"data":`, `"mimeType":`} {
		if !bytes.Contains(data, []byte(field)) {
			t.Errorf("AudioContent JSON %s missing required field %s", data, field)
		}
	}
}

// TestCreateMessageResultContentRoundTrip verifies that the polymorphic Content
// field decodes into concrete content types rather than map[string]any.
func TestCreateMessageResultContentRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want any
	}{
		{
			name: "text",
			in:   TextContent{Type: "text", Text: "hello"},
			want: TextContent{Type: "text", Text: "hello"},
		},
		{
			name: "image",
			in:   ImageContent{Type: "image", Data: []byte("img"), MimeType: "image/png"},
			want: ImageContent{Type: "image", Data: []byte("img"), MimeType: "image/png"},
		},
		{
			name: "audio",
			in:   AudioContent{Type: "audio", Data: []byte("snd"), MimeType: "audio/wav"},
			want: AudioContent{Type: "audio", Data: []byte("snd"), MimeType: "audio/wav"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(CreateMessageResult{Role: RoleAssistant, Model: "m", Content: tc.in})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got CreateMessageResult
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !jsonEqual(t, got.Content, tc.want) {
				t.Errorf("Content = %#v (%T), want %#v (%T)", got.Content, got.Content, tc.want, tc.want)
			}
		})
	}
}

// TestSamplingAndPromptMessageContentRoundTrip verifies typed content decoding
// for SamplingMessage and PromptMessage.
func TestSamplingAndPromptMessageContentRoundTrip(t *testing.T) {
	sm := SamplingMessage{Role: RoleUser, Content: TextContent{Type: "text", Text: "hi"}}
	data, err := json.Marshal(sm)
	if err != nil {
		t.Fatalf("marshal SamplingMessage: %v", err)
	}
	var gotSM SamplingMessage
	if err := json.Unmarshal(data, &gotSM); err != nil {
		t.Fatalf("unmarshal SamplingMessage: %v", err)
	}
	if _, ok := gotSM.Content.(TextContent); !ok {
		t.Errorf("SamplingMessage.Content = %T, want TextContent", gotSM.Content)
	}

	pm := PromptMessage{Role: RoleAssistant, Content: ImageContent{Type: "image", Data: []byte("x"), MimeType: "image/gif"}}
	data, err = json.Marshal(pm)
	if err != nil {
		t.Fatalf("marshal PromptMessage: %v", err)
	}
	var gotPM PromptMessage
	if err := json.Unmarshal(data, &gotPM); err != nil {
		t.Fatalf("unmarshal PromptMessage: %v", err)
	}
	if _, ok := gotPM.Content.(ImageContent); !ok {
		t.Errorf("PromptMessage.Content = %T, want ImageContent", gotPM.Content)
	}
}

// TestUnmarshalContentPreservesResource verifies resource and unknown content
// types are preserved as generic objects rather than dropped.
func TestUnmarshalContentPreservesResource(t *testing.T) {
	raw := json.RawMessage(`{"type":"resource","resource":{"uri":"file:///x","text":"data"}}`)
	got, err := unmarshalContent(raw)
	if err != nil {
		t.Fatalf("unmarshalContent: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("resource content = %T, want map[string]any", got)
	}
	if m["type"] != "resource" {
		t.Errorf("resource content type = %v, want resource", m["type"])
	}
}

// TestResultMetaRoundTrip verifies that the _meta field on result types is
// preserved through a marshal/unmarshal cycle rather than silently dropped.
func TestResultMetaRoundTrip(t *testing.T) {
	meta := map[string]any{"trace": "abc123"}

	t.Run("CallToolResult", func(t *testing.T) {
		in := CallToolResult{Content: []any{}, Meta: meta}
		data, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got CallToolResult
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.Meta["trace"] != "abc123" {
			t.Errorf("Meta = %#v, want trace=abc123", got.Meta)
		}
	})

	t.Run("ListToolsResult", func(t *testing.T) {
		in := ListToolsResult{Tools: []Tool{}, Meta: meta}
		data, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got ListToolsResult
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.Meta["trace"] != "abc123" {
			t.Errorf("Meta = %#v, want trace=abc123", got.Meta)
		}
	})

	t.Run("ReadResourceResult", func(t *testing.T) {
		in := ReadResourceResult{Contents: []ResourceContents{}, Meta: meta}
		data, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got ReadResourceResult
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.Meta["trace"] != "abc123" {
			t.Errorf("Meta = %#v, want trace=abc123", got.Meta)
		}
	})
}

func jsonEqual(t *testing.T, a, b any) bool {
	t.Helper()
	da, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal a: %v", err)
	}
	db, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal b: %v", err)
	}
	return bytes.Equal(da, db)
}
