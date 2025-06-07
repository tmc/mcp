package draft

import base "github.com/tmc/mcp/modelcontextprotocol"

var NewTextContent = base.NewTextContent
var NewImageContent = base.NewImageContent
var NewAudioContent = base.NewAudioContent
var NewEmbeddedResource = base.NewEmbeddedResource
var NewTextResourceContents = base.NewTextResourceContents
var NewBlobResourceContents = base.NewBlobResourceContents
var NewPromptReference = base.NewPromptReference
var NewResourceReference = base.NewResourceReference

func (ctr *CallToolResult) GetStructuredContent() (map[string]any, bool) {
	return ctr.StructuredContent, ctr.isStructuredResult && ctr.StructuredContent != nil
}
func (ctr *CallToolResult) GetContent() *ContentList { return ctr.Content }
func (ctr *CallToolResult) IsToolError() bool        { return ctr.IsError != nil && *ctr.IsError }

// --- CallToolResult Creation Helpers ---
// NewCallToolResultStructured creates a CallToolResult for structured data
func NewCallToolResultStructured(structuredContent map[string]any, compatibilityContent ...base.Content) CallToolResult {
	var content *ContentList
	if len(compatibilityContent) > 0 {
		cl := ContentList(compatibilityContent)
		content = &cl
	}
	return CallToolResult{
		StructuredContent:  structuredContent,
		Content:            content,
		isStructuredResult: true,
	}
}

// NewCallToolResultUnstructured creates a CallToolResult for unstructured data
func NewCallToolResultUnstructured(content ...base.Content) CallToolResult {
	cl := ContentList(content)
	return CallToolResult{
		Content:            &cl,
		isStructuredResult: false,
	}
}

// NewCallToolResultError creates a CallToolResult for errors
func NewCallToolResultError(content ...base.Content) CallToolResult {
	isError := true
	var cl *ContentList
	if len(content) > 0 {
		contentList := ContentList(content)
		cl = &contentList
	}
	return CallToolResult{
		IsError:            &isError,
		Content:            cl,
		isStructuredResult: false,
	}
}
