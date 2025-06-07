package draft

// Constructors for draft.CallToolResult
func NewStructuredToolResult(structuredData map[string]any, compatContent *ContentList, isError *bool, meta map[string]any) CallToolResult {
	return CallToolResult{Meta: meta, IsError: isError, StructuredContent: structuredData, Content: compatContent, isStructuredResult: true}
}
func NewUnstructuredToolResult(content ContentList, isError *bool, meta map[string]any) CallToolResult {
	var cList *ContentList
	if content != nil {
		cl := content
		cList = &cl
	}
	return CallToolResult{Meta: meta, IsError: isError, Content: cList, StructuredContent: nil, isStructuredResult: false}
}
