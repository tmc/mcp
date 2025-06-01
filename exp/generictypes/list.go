package generictypes

import (
	"github.com/tmc/mcp/modelcontextprotocol"
)

// ListResult is a generic paginated list result that can replace multiple specific list types.
// It can replace ListResourcesResult, ListResourceTemplatesResult, ListPromptsResult, etc.
type ListResult[T any] struct {
	Meta       map[string]any                 `json:"_meta,omitempty"`
	Items      []T                            `json:"items"`
	NextCursor *modelcontextprotocol.Cursor   `json:"nextCursor,omitempty"`
}

// ListRequest is a generic paginated list request.
type ListRequest struct {
	Meta   *modelcontextprotocol.RequestMeta `json:"_meta,omitempty"`
	Cursor *modelcontextprotocol.Cursor      `json:"cursor,omitempty"`
}

// WithCursor returns a new ListRequest with the specified cursor.
func (lr ListRequest) WithCursor(cursor modelcontextprotocol.Cursor) ListRequest {
	lr.Cursor = &cursor
	return lr
}

// --- Examples of how these could be used ---

// ResourceList would replace ListResourcesResult
type ResourceList = ListResult[modelcontextprotocol.Resource]

// ResourceTemplateList would replace ListResourceTemplatesResult  
type ResourceTemplateList = ListResult[modelcontextprotocol.ResourceTemplate]

// PromptList would replace ListPromptsResult
type PromptList = ListResult[modelcontextprotocol.Prompt]

// ToolList would replace ListToolsResult
type ToolList = ListResult[modelcontextprotocol.Tool]

// RootList would replace ListRootsResult
type RootList = ListResult[modelcontextprotocol.Root]

// Helper functions for list operations

// MapList transforms a list of items using a mapping function.
func MapList[T, U any](list ListResult[T], fn func(T) U) ListResult[U] {
	result := ListResult[U]{
		Meta:       list.Meta,
		NextCursor: list.NextCursor,
		Items:      make([]U, len(list.Items)),
	}
	for i, item := range list.Items {
		result.Items[i] = fn(item)
	}
	return result
}

// FilterList filters a list based on a predicate function.
func FilterList[T any](list ListResult[T], predicate func(T) bool) ListResult[T] {
	result := ListResult[T]{
		Meta:       list.Meta,
		NextCursor: list.NextCursor,
		Items:      make([]T, 0),
	}
	for _, item := range list.Items {
		if predicate(item) {
			result.Items = append(result.Items, item)
		}
	}
	return result
}

// CombineLists combines multiple lists into a single list.
func CombineLists[T any](lists ...ListResult[T]) ListResult[T] {
	if len(lists) == 0 {
		return ListResult[T]{}
	}
	
	result := ListResult[T]{
		Meta:  lists[0].Meta,
		Items: make([]T, 0),
	}
	
	for _, list := range lists {
		result.Items = append(result.Items, list.Items...)
	}
	
	// Use the last cursor from the last list
	if len(lists) > 0 {
		result.NextCursor = lists[len(lists)-1].NextCursor
	}
	
	return result
}