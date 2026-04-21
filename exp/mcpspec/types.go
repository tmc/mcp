package mcpspec

import (
	"encoding/json"
)

// MCPSpec represents a full MCP server specification
type MCPSpec struct {
	SpecVersion string               `json:"specVersion" yaml:"specVersion"`
	Server      ServerMetadata       `json:"server" yaml:"server"`
	Tools       []ToolDefinition     `json:"tools" yaml:"tools"`
	Resources   []ResourceDefinition `json:"resources" yaml:"resources"`
	Prompts     []PromptDefinition   `json:"prompts" yaml:"prompts"`
}

// ServerMetadata represents metadata about the server
type ServerMetadata struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`
	License     string `json:"license,omitempty" yaml:"license,omitempty"`
}

// ToolDefinition represents an MCP tool description (formerly MCPToolDescription)
type ToolDefinition struct {
	Name        string          `json:"name" yaml:"name"`
	Description string          `json:"description" yaml:"description"`
	InputSchema json.RawMessage `json:"inputSchema" yaml:"inputSchema"`
	ReturnType  json.RawMessage `json:"returnType" yaml:"returnType"`
}

// ResourceDefinition represents an MCP resource definition
type ResourceDefinition struct {
	URI         string                 `json:"uri" yaml:"uri"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	MimeType    string                 `json:"mimeType" yaml:"mimeType"`
	Annotations map[string]interface{} `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// PromptDefinition represents an MCP prompt definition
type PromptDefinition struct {
	Name        string           `json:"name" yaml:"name"`
	Description string           `json:"description" yaml:"description"`
	Arguments   []PromptArgument `json:"arguments" yaml:"arguments"`
}

// PromptArgument represents an argument for a prompt
type PromptArgument struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Required    bool   `json:"required" yaml:"required"`
}

// JSONSchema represents a JSON schema
type JSONSchema struct {
	Type                 string                 `json:"type" yaml:"type"`
	Description          string                 `json:"description" yaml:"description"`
	Properties           map[string]*JSONSchema `json:"properties" yaml:"properties"`
	Items                *JSONSchema            `json:"items" yaml:"items"`
	Required             []string               `json:"required" yaml:"required"`
	Enum                 []interface{}          `json:"enum" yaml:"enum"`
	Default              interface{}            `json:"default" yaml:"default"`
	Format               string                 `json:"format" yaml:"format"`
	Minimum              *float64               `json:"minimum" yaml:"minimum"`
	Maximum              *float64               `json:"maximum" yaml:"maximum"`
	MinLength            *int                   `json:"minLength" yaml:"minLength"`
	MaxLength            *int                   `json:"maxLength" yaml:"maxLength"`
	Pattern              string                 `json:"pattern" yaml:"pattern"`
	Ref                  string                 `json:"$ref" yaml:"$ref"`
	Definitions          map[string]*JSONSchema `json:"definitions" yaml:"definitions"`
	AllOf                []*JSONSchema          `json:"allOf" yaml:"allOf"`
	AnyOf                []*JSONSchema          `json:"anyOf" yaml:"anyOf"`
	OneOf                []*JSONSchema          `json:"oneOf" yaml:"oneOf"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
}
