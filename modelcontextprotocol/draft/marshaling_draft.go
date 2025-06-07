package draft

import (
	"encoding/json"
	"fmt"

	base "github.com/tmc/mcp/modelcontextprotocol"
)

func (cl *ContentList) UnmarshalJSON(data []byte) error {
	// Handle null value - for array types, null should result in empty array
	if string(data) == "null" {
		*cl = ContentList{}
		return nil
	}
	var rawMessages []json.RawMessage
	if err := json.Unmarshal(data, &rawMessages); err != nil {
		return fmt.Errorf("draft.ContentList base: %w", err)
	}
	*cl = make(ContentList, len(rawMessages))
	for i, raw := range rawMessages {
		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			return fmt.Errorf("draft.ContentList item %d probe: %w", i, err)
		}
		var item base.Content
		switch probe.Type {
		case base.ContentTypeText:
			var tc base.TextContent
			err := json.Unmarshal(raw, &tc)
			if err != nil {
				return fmt.Errorf("draft.ContentList: TextContent item %d: %w", i, err)
			}
			item = tc
		case base.ContentTypeImage:
			var ic base.ImageContent
			err := json.Unmarshal(raw, &ic)
			if err != nil {
				return fmt.Errorf("draft.ContentList: ImageContent item %d: %w", i, err)
			}
			item = ic
		case base.ContentTypeAudio:
			var ac base.AudioContent
			err := json.Unmarshal(raw, &ac)
			if err != nil {
				return fmt.Errorf("draft.ContentList: AudioContent item %d: %w", i, err)
			}
			item = ac
		case base.ContentTypeResource:
			// EmbeddedResource needs its own UnmarshalJSON method in the base package to handle its internal ResourceContents
			var er base.EmbeddedResource
			if err := json.Unmarshal(raw, &er); err != nil {
				return fmt.Errorf("draft.ContentList: EmbeddedResource item %d: %w", i, err)
			}
			item = er
		default:
			return fmt.Errorf("draft.ContentList item %d unknown type: '%s' from data %s", i, probe.Type, string(raw))
		}
		(*cl)[i] = item
	}
	return nil
}

func (ctr *CallToolResult) UnmarshalJSON(data []byte) error {
	// Handle null value
	if string(data) == "null" {
		return nil
	}
	ctr.StructuredContent = nil
	ctr.Content = nil
	ctr.isStructuredResult = false
	var temp struct {
		Meta              map[string]any  `json:"_meta,omitempty"`
		IsError           *bool           `json:"isError,omitempty"`
		Content           json.RawMessage `json:"content,omitempty"`
		StructuredContent json.RawMessage `json:"structuredContent,omitempty"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("draft.CallToolResult base: %w", err)
	}
	ctr.Meta = temp.Meta
	ctr.IsError = temp.IsError
	hasS := temp.StructuredContent != nil && string(temp.StructuredContent) != "null" && string(temp.StructuredContent) != "\"\""
	hasC := temp.Content != nil && string(temp.Content) != "null" && string(temp.Content) != "\"\""

	if hasS {
		ctr.isStructuredResult = true
		if err := json.Unmarshal(temp.StructuredContent, &ctr.StructuredContent); err != nil {
			return fmt.Errorf("draft.CallToolResult structured: %w", err)
		}
		if hasC {
			var cl ContentList
			if err := json.Unmarshal(temp.Content, &cl); err != nil {
				return fmt.Errorf("draft.CallToolResult compat content: %w", err)
			}
			ctr.Content = &cl
		}
		return nil
	}
	if hasC {
		ctr.isStructuredResult = false
		var cl ContentList
		if err := json.Unmarshal(temp.Content, &cl); err != nil {
			return fmt.Errorf("draft.CallToolResult unstructured: %w", err)
		}
		ctr.Content = &cl
		return nil
	}
	if ctr.IsToolError() {
		return nil
	}
	return nil
}

func (ctr CallToolResult) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)

	if ctr.Meta != nil {
		m["_meta"] = ctr.Meta
	}
	if ctr.IsError != nil {
		m["isError"] = *ctr.IsError
	}

	if ctr.isStructuredResult {
		// Per spec: If the Tool defines an outputSchema, `structuredContent` MUST be present
		if ctr.StructuredContent != nil {
			m["structuredContent"] = ctr.StructuredContent
		} else if !(ctr.IsError != nil && *ctr.IsError) {
			// If it's a structured result, not an error, and SC is nil,
			// marshal it as an empty object `{}` to indicate presence
			m["structuredContent"] = map[string]any{}
		}
		// else if it's an error and StructuredContent is nil, it's fine for it to be absent

		// Compatibility content
		if ctr.Content != nil && len(*ctr.Content) > 0 {
			m["content"] = ctr.Content
		}
	} else {
		// Unstructured result
		// Per spec: Content MUST be present (unless error or truly no output)
		// StructuredContent MUST NOT be present
		if ctr.Content != nil {
			m["content"] = ctr.Content
		}
		// If it's an error, content can be absent
	}
	return json.Marshal(m)
}
