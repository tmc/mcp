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
