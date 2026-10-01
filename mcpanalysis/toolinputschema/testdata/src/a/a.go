package a

import sdk "github.com/modelcontextprotocol/go-sdk/mcp"

type Alias = sdk.Tool
type ServerAlias = sdk.Server
type Other struct{}

func (*Other) AddTool(*sdk.Tool, any) {}

func registrations(s *sdk.Server, alias *ServerAlias, other *Other, dynamic any) {
	s.AddTool(&sdk.Tool{Name: "missing"}, nil)                   // want "Server.AddTool panics when InputSchema is missing"
	s.AddTool(&sdk.Tool{Name: "nil", InputSchema: nil}, nil)     // want "Server.AddTool panics when InputSchema is nil"
	alias.AddTool(&Alias{Name: "alias"}, nil)                    // want "Server.AddTool panics when InputSchema is missing"
	(*sdk.Server).AddTool(s, &sdk.Tool{Name: "expression"}, nil) // want "Server.AddTool panics when InputSchema is missing"
	s.AddTool((&sdk.Tool{InputSchema: (nil)}), nil)              // want "Server.AddTool panics when InputSchema is nil"
	s.AddTool(&sdk.Tool{InputSchema: map[string]any{"type": "object"}}, nil)
	s.AddTool(&sdk.Tool{InputSchema: dynamic}, nil)
	s.AddTool(&sdk.Tool{"positional", dynamic}, nil)
	tool := &sdk.Tool{Name: "later"}
	tool.InputSchema = map[string]any{"type": "object"}
	s.AddTool(tool, nil)
	other.AddTool(&sdk.Tool{Name: "other"}, nil)
	sdk.AddTool(s, &sdk.Tool{Name: "generic"}, func(in struct{}) struct{} { return in })
	method := s.AddTool
	method(&sdk.Tool{Name: "bound"}, nil) // Indirect calls are outside this analyzer's scope.
}

func shadowNil(s *sdk.Server, schema any) {
	nil := schema
	s.AddTool(&sdk.Tool{InputSchema: nil}, nil)
}
