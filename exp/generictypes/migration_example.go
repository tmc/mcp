package generictypes

import (
	"encoding/json"

	"github.com/tmc/mcp/modelcontextprotocol"
)

// This file demonstrates how the current modelcontextprotocol types
// could be simplified using generics.

// --- Before: Multiple similar list types ---
// type ListResourcesResult struct {
//     Meta       map[string]any `json:"_meta,omitempty"`
//     Resources  []Resource     `json:"resources"`
//     NextCursor *Cursor        `json:"nextCursor,omitempty"`
// }
// type ListPromptsResult struct {
//     Meta       map[string]any `json:"_meta,omitempty"`
//     Prompts    []Prompt       `json:"prompts"`
//     NextCursor *Cursor        `json:"nextCursor,omitempty"`
// }
// ... and many more

// --- After: Single generic type ---
// type ListResult[T any] struct {
//     Meta       map[string]any `json:"_meta,omitempty"`
//     Items      []T            `json:"items"`
//     NextCursor *Cursor        `json:"nextCursor,omitempty"`
// }

// --- Before: Repetitive unmarshaling code ---
// func unmarshalContentInternal(data json.RawMessage) (Content, error) {
//     var probe struct { Type string `json:"type"` }
//     if err := json.Unmarshal(data, &probe); err != nil { ... }
//     switch probe.Type {
//     case ContentTypeText:
//         var tc TextContent
//         err := json.Unmarshal(data, &tc)
//         ...
//     case ContentTypeImage:
//         var ic ImageContent
//         err := json.Unmarshal(data, &ic)
//         ...
//     }
// }

// --- After: Generic unmarshaling ---
func SimplifiedContentUnmarshaler() *TypedUnion[modelcontextprotocol.Content] {
	return NewTypedUnion[modelcontextprotocol.Content]("type").
		Register("text", unmarshalAs[modelcontextprotocol.TextContent]).
		Register("image", unmarshalAs[modelcontextprotocol.ImageContent]).
		Register("audio", unmarshalAs[modelcontextprotocol.AudioContent]).
		Register("resource", unmarshalAs[modelcontextprotocol.EmbeddedResource])
}

// Generic unmarshal helper
func unmarshalAs[T any](data json.RawMessage) (modelcontextprotocol.Content, error) {
	var t T
	err := json.Unmarshal(data, &t)
	if content, ok := any(t).(modelcontextprotocol.Content); ok {
		return content, err
	}
	return nil, err
}

// --- Before: Pointer fields everywhere ---
// type Resource struct {
//     URI         string       `json:"uri"`
//     Name        string       `json:"name"`
//     Description *string      `json:"description,omitempty"`
//     MimeType    *string      `json:"mimeType,omitempty"`
//     Size        *int64       `json:"size,omitempty"`
// }

// --- After: Optional fields ---
type ResourceWithGenerics struct {
	URI         string                                     `json:"uri"`
	Name        string                                     `json:"name"`
	Description Optional[string]                           `json:"description,omitempty"`
	MimeType    Optional[string]                           `json:"mimeType,omitempty"`
	Size        Optional[int64]                            `json:"size,omitempty"`
	Annotations Optional[modelcontextprotocol.Annotations] `json:"annotations,omitempty"`
}

// --- Before: Manual helper functions ---
// func NewTextContent(text string, annotations ...Annotations) TextContent {
//     var ann *Annotations
//     if len(annotations) > 0 {
//         ann = &annotations[0]
//     }
//     return TextContent{Type: ContentTypeText, Text: text, Annotations: ann}
// }

// --- After: Generic builder ---
func NewTextContentGeneric(text string) *Builder[modelcontextprotocol.TextContent] {
	return NewBuilderFrom(modelcontextprotocol.TextContent{
		Type: "text",
		Text: text,
	})
}

// Example: Converting existing code to use generics
func ConvertResourceList(old modelcontextprotocol.ListResourcesResult) ListResult[modelcontextprotocol.Resource] {
	return ListResult[modelcontextprotocol.Resource]{
		Meta:       old.Meta,
		Items:      old.Resources,
		NextCursor: old.NextCursor,
	}
}

// Example: Simplified request handling
type GenericRequestHandler[T any, U any] func(Request[T]) (Result[U], error)

func HandleListResources(req Request[modelcontextprotocol.ListResourcesRequestParams]) (
	Result[ListResult[modelcontextprotocol.Resource]], error) {

	// Implementation would go here
	return Result[ListResult[modelcontextprotocol.Resource]]{
		Data: ListResult[modelcontextprotocol.Resource]{
			Items: []modelcontextprotocol.Resource{
				{URI: "example", Name: "Example Resource"},
			},
		},
	}, nil
}

// Example: Type-safe event handling
type EventHandler[T any] func(event T) error

type EventManager[T any] struct {
	handlers []EventHandler[T]
}

func (em *EventManager[T]) Subscribe(handler EventHandler[T]) {
	em.handlers = append(em.handlers, handler)
}

func (em *EventManager[T]) Publish(event T) error {
	for _, handler := range em.handlers {
		if err := handler(event); err != nil {
			return err
		}
	}
	return nil
}

// Usage example
func ExampleEventManager() {
	resourceEvents := &EventManager[modelcontextprotocol.ResourceUpdatedNotificationParams]{}

	resourceEvents.Subscribe(func(event modelcontextprotocol.ResourceUpdatedNotificationParams) error {
		// Handle resource update
		return nil
	})
}
