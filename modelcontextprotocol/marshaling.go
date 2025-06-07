package modelcontextprotocol

import (
	"encoding/json"
	"fmt"
)

func unmarshalContentInternal(data json.RawMessage) (Content, error) {
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
	case ContentTypeText:
		var tc TextContent
		err := json.Unmarshal(data, &tc)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling TextContent: %w", err)
		}
		return tc, nil
	case ContentTypeImage:
		var ic ImageContent
		err := json.Unmarshal(data, &ic)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling ImageContent: %w", err)
		}
		return ic, nil
	case ContentTypeAudio:
		var ac AudioContent
		err := json.Unmarshal(data, &ac)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling AudioContent: %w", err)
		}
		return ac, nil
	case ContentTypeResource:
		var er EmbeddedResource
		err := json.Unmarshal(data, &er)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling EmbeddedResource: %w", err)
		}
		return er, nil
	default:
		return nil, fmt.Errorf("unknown content type: '%s' from data: %s", probe.Type, string(data))
	}
}
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
func unmarshalReferenceInternal(data json.RawMessage) (Reference, error) {
	if string(data) == "null" || len(data) == 0 {
		return nil, nil
	}
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("probing reference type: %w (data: %s)", err, string(data))
	}

	switch probe.Type {
	case ReferenceTypeResource:
		var rr ResourceReference
		err := json.Unmarshal(data, &rr)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling ResourceReference: %w", err)
		}
		return rr, nil
	case ReferenceTypePrompt:
		var pr PromptReference
		err := json.Unmarshal(data, &pr)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling PromptReference: %w", err)
		}
		return pr, nil
	default:
		return nil, fmt.Errorf("unknown reference type: '%s' from data: %s", probe.Type, string(data))
	}
}

func (er *EmbeddedResource) UnmarshalJSON(data []byte) error {
	// Handle null value
	if string(data) == "null" {
		return nil
	}
	type Alias EmbeddedResource
	aux := &struct {
		ResourceRaw json.RawMessage `json:"resource"`
		*Alias
	}{Alias: (*Alias)(er)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("EmbeddedResource base: %w", err)
	}
	if len(aux.ResourceRaw) == 0 {
		return fmt.Errorf("EmbeddedResource missing required 'resource' field")
	}
	rc, err := unmarshalResourceContentsInternal(aux.ResourceRaw)
	if err != nil {
		return fmt.Errorf("EmbeddedResource.Resource: %w", err)
	}
	er.Resource = rc
	return nil
}
func (pm *PromptMessage) UnmarshalJSON(data []byte) error {
	// Handle null value
	if string(data) == "null" {
		return nil
	}
	type Alias PromptMessage
	aux := &struct {
		ContentRaw json.RawMessage `json:"content"`
		*Alias
	}{Alias: (*Alias)(pm)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("PromptMessage base: %w", err)
	}
	c, err := unmarshalContentInternal(aux.ContentRaw)
	if err != nil {
		return fmt.Errorf("PromptMessage.Content: %w", err)
	}
	pm.Content = c
	return nil
}
func (sm *SamplingMessage) UnmarshalJSON(data []byte) error {
	// Handle null value
	if string(data) == "null" {
		return nil
	}
	type Alias SamplingMessage
	aux := &struct {
		ContentRaw json.RawMessage `json:"content"`
		*Alias
	}{Alias: (*Alias)(sm)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("SamplingMessage base: %w", err)
	}
	c, err := unmarshalContentInternal(aux.ContentRaw)
	if err != nil {
		return fmt.Errorf("SamplingMessage.Content: %w", err)
	}
	sm.Content = c
	return nil
}
func (cmr *CreateMessageResult) UnmarshalJSON(data []byte) error {
	// Handle null value
	if string(data) == "null" {
		return nil
	}
	type Alias CreateMessageResult
	aux := &struct {
		ContentRaw json.RawMessage `json:"content"`
		*Alias
	}{Alias: (*Alias)(cmr)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("CreateMessageResult base: %w", err)
	}
	c, err := unmarshalContentInternal(aux.ContentRaw)
	if err != nil {
		return fmt.Errorf("CreateMessageResult.Content: %w", err)
	}
	cmr.Content = c
	return nil
}
func (crp *CompleteRequestParams) UnmarshalJSON(data []byte) error {
	// Handle null value
	if string(data) == "null" {
		return nil
	}
	type Alias CompleteRequestParams
	aux := &struct {
		RefRaw json.RawMessage `json:"ref"`
		*Alias
	}{Alias: (*Alias)(crp)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("CompleteRequestParams base: %w", err)
	}
	r, err := unmarshalReferenceInternal(aux.RefRaw)
	if err != nil {
		return fmt.Errorf("CompleteRequestParams.Ref: %w", err)
	}
	crp.Ref = r
	return nil
}
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

type contentListForCallToolResult []Content

func (cl *contentListForCallToolResult) UnmarshalJSON(data []byte) error {
	// Handle null value - for array types, null should result in empty array
	if string(data) == "null" {
		*cl = contentListForCallToolResult{}
		return nil
	}
	var rawMessages []json.RawMessage
	if err := json.Unmarshal(data, &rawMessages); err != nil {
		return err
	}
	*cl = make(contentListForCallToolResult, len(rawMessages))
	for i, raw := range rawMessages {
		c, err := unmarshalContentInternal(raw)
		if err != nil {
			return fmt.Errorf("contentListForCallToolResult item %d: %w", i, err)
		}
		(*cl)[i] = c
	}
	return nil
}
func (ctr *CallToolResult) UnmarshalJSON(data []byte) error {
	// Handle null value
	if string(data) == "null" {
		return nil
	}
	type Alias CallToolResult
	aux := &struct {
		ContentRaw json.RawMessage `json:"content,omitempty"`
		*Alias
	}{Alias: (*Alias)(ctr)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("CallToolResult base: %w", err)
	}
	if aux.ContentRaw != nil && string(aux.ContentRaw) != "null" {
		var cl contentListForCallToolResult
		if err := json.Unmarshal(aux.ContentRaw, &cl); err != nil {
			return fmt.Errorf("CallToolResult.Content: %w", err)
		}
		ctr.Content = cl
	}
	return nil
}
