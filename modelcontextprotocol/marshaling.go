package modelcontextprotocol

import (
	"encoding/json"
	"fmt"
)

// UnmarshalContent unmarshals content from raw JSON data
func UnmarshalContent(data []byte) (Content, error) {
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base content type: %w", err)
	}

	switch base.Type {
	case "text":
		var content TextContent
		if err := json.Unmarshal(data, &content); err != nil {
			return nil, fmt.Errorf("failed to unmarshal TextContent: %w", err)
		}
		return content, nil
	case "image":
		var content ImageContent
		if err := json.Unmarshal(data, &content); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ImageContent: %w", err)
		}
		return content, nil
	case "audio":
		var content AudioContent
		if err := json.Unmarshal(data, &content); err != nil {
			return nil, fmt.Errorf("failed to unmarshal AudioContent: %w", err)
		}
		return content, nil
	case "resource":
		var content EmbeddedResource
		if err := json.Unmarshal(data, &content); err != nil {
			return nil, fmt.Errorf("failed to unmarshal EmbeddedResource: %w", err)
		}
		return content, nil
	default:
		return nil, fmt.Errorf("unknown content type: '%s'", base.Type)
	}
}

// UnmarshalResourceContents unmarshals resource contents from raw JSON data
func UnmarshalResourceContents(data []byte) (ResourceContents, error) {
	var probe struct {
		URI      string  `json:"uri"`
		MimeType string  `json:"mimeType,omitempty"`
		Text     *string `json:"text,omitempty"` // Use pointer to distinguish missing from ""
		Blob     *string `json:"blob,omitempty"`
	}

	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("failed to probe resource contents: %w", err)
	}
	if probe.URI == "" { // URI is required for resource contents
		return nil, fmt.Errorf("resource contents missing URI field in: %s", string(data))
	}

	if probe.Text != nil {
		var tc TextResourceContents
		if err := json.Unmarshal(data, &tc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal TextResourceContents (uri: %s): %w", probe.URI, err)
		}
		if probe.Blob != nil {
			return nil, fmt.Errorf("resource contents (uri: %s) has both text and blob fields", probe.URI)
		}
		return &tc, nil
	}

	if probe.Blob != nil {
		var bc BlobResourceContents
		if err := json.Unmarshal(data, &bc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal BlobResourceContents (uri: %s): %w", probe.URI, err)
		}
		return &bc, nil
	}

	// If it's neither Text nor Blob, but has a URI, it's an unknown type or malformed.
	return nil, fmt.Errorf("unknown or ambiguous resource contents for URI '%s': no text or blob field", probe.URI)
}

// UnmarshalReference unmarshals reference from raw JSON data
func UnmarshalReference(data []byte) (Reference, error) {
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base reference type: %w", err)
	}

	switch base.Type {
	case "ref/resource":
		var ref ResourceReference
		if err := json.Unmarshal(data, &ref); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ResourceReference: %w", err)
		}
		return ref, nil
	case "ref/prompt":
		var ref PromptReference
		if err := json.Unmarshal(data, &ref); err != nil {
			return nil, fmt.Errorf("failed to unmarshal PromptReference: %w", err)
		}
		return ref, nil
	default:
		return nil, fmt.Errorf("unknown reference type: '%s'", base.Type)
	}
}
