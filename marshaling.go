// Copyright 2025 The MCP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.

package mcp

import (
	"encoding/json"
	"fmt"
)

// UnmarshalJSON implements custom unmarshaling for ReadResourceResult
func (rrr *ReadResourceResult) UnmarshalJSON(data []byte) error {
	// Handle null value
	if string(data) == "null" {
		return nil
	}
	type Alias ReadResourceResult
	aux := &struct {
		ContentsRaw []json.RawMessage `json:"contents"`
		*Alias
	}{Alias: (*Alias)(rrr)}

	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("ReadResourceResult base: %w", err)
	}

	rrr.Contents = make([]ResourceContents, len(aux.ContentsRaw))
	for i, raw := range aux.ContentsRaw {
		rc, err := unmarshalResourceContentsInternal(raw)
		if err != nil {
			return fmt.Errorf("ReadResourceResult.Contents item %d: %w", i, err)
		}
		rrr.Contents[i] = rc
	}
	return nil
}

// unmarshalContent decodes a single polymorphic content block into a concrete
// Content value based on its "type" discriminator. Resource content blocks are
// returned as their raw JSON wrapped in a generic map so callers retain the data
// without a dedicated embedded-resource type in this package.
func unmarshalContent(data json.RawMessage) (any, error) {
	if string(data) == "null" || len(data) == 0 {
		return nil, nil
	}
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("probing content type: %w (data: %s)", err, string(data))
	}

	switch probe.Type {
	case "text":
		var tc TextContent
		if err := json.Unmarshal(data, &tc); err != nil {
			return nil, fmt.Errorf("unmarshaling TextContent: %w", err)
		}
		return tc, nil
	case "image":
		var ic ImageContent
		if err := json.Unmarshal(data, &ic); err != nil {
			return nil, fmt.Errorf("unmarshaling ImageContent: %w", err)
		}
		return ic, nil
	case "audio":
		var ac AudioContent
		if err := json.Unmarshal(data, &ac); err != nil {
			return nil, fmt.Errorf("unmarshaling AudioContent: %w", err)
		}
		return ac, nil
	default:
		// Resource content and unknown future types are preserved as a generic
		// object so no data is lost.
		var generic map[string]any
		if err := json.Unmarshal(data, &generic); err != nil {
			return nil, fmt.Errorf("unmarshaling content (type %q): %w", probe.Type, err)
		}
		return generic, nil
	}
}

// UnmarshalJSON decodes a CreateMessageResult, resolving the polymorphic
// Content field to a concrete content type.
func (r *CreateMessageResult) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	type Alias CreateMessageResult
	aux := &struct {
		Content json.RawMessage `json:"content"`
		*Alias
	}{Alias: (*Alias)(r)}
	if err := json.Unmarshal(data, aux); err != nil {
		return fmt.Errorf("CreateMessageResult base: %w", err)
	}
	content, err := unmarshalContent(aux.Content)
	if err != nil {
		return fmt.Errorf("CreateMessageResult.Content: %w", err)
	}
	r.Content = content
	return nil
}

// UnmarshalJSON decodes a SamplingMessage, resolving the polymorphic Content
// field to a concrete content type.
func (m *SamplingMessage) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	type Alias SamplingMessage
	aux := &struct {
		Content json.RawMessage `json:"content"`
		*Alias
	}{Alias: (*Alias)(m)}
	if err := json.Unmarshal(data, aux); err != nil {
		return fmt.Errorf("SamplingMessage base: %w", err)
	}
	content, err := unmarshalContent(aux.Content)
	if err != nil {
		return fmt.Errorf("SamplingMessage.Content: %w", err)
	}
	m.Content = content
	return nil
}

// UnmarshalJSON decodes a PromptMessage, resolving the polymorphic Content
// field to a concrete content type.
func (m *PromptMessage) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	type Alias PromptMessage
	aux := &struct {
		Content json.RawMessage `json:"content"`
		*Alias
	}{Alias: (*Alias)(m)}
	if err := json.Unmarshal(data, aux); err != nil {
		return fmt.Errorf("PromptMessage base: %w", err)
	}
	content, err := unmarshalContent(aux.Content)
	if err != nil {
		return fmt.Errorf("PromptMessage.Content: %w", err)
	}
	m.Content = content
	return nil
}

// unmarshalResourceContentsInternal handles unmarshaling of ResourceContents interface
func unmarshalResourceContentsInternal(data json.RawMessage) (ResourceContents, error) {
	if string(data) == "null" || len(data) == 0 {
		return nil, nil
	}

	var probe struct {
		URI  string  `json:"uri"`
		Text *string `json:"text,omitempty"`
		Blob *string `json:"blob,omitempty"`
	}

	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("probing resource contents fields: %w (data: %s)", err, string(data))
	}

	if probe.URI == "" {
		return nil, fmt.Errorf("resource contents missing URI field from data: %s", string(data))
	}

	if probe.Text != nil && probe.Blob != nil {
		return nil, fmt.Errorf("resource contents (uri: %s) has both text and blob fields", probe.URI)
	}

	if probe.Text != nil {
		var trc TextResourceContents
		err := json.Unmarshal(data, &trc)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling TextResourceContents (uri: %s): %w", probe.URI, err)
		}
		return trc, nil
	}

	if probe.Blob != nil {
		var brc BlobResourceContents
		err := json.Unmarshal(data, &brc)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling BlobResourceContents (uri: %s): %w", probe.URI, err)
		}
		return brc, nil
	}

	return nil, fmt.Errorf("resource contents (uri: %s) has neither text nor blob field", probe.URI)
}
