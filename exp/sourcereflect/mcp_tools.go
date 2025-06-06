package sourcereflect

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
)

// MCPToolDescription represents an MCP tool description
type MCPToolDescription struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	InputSchema *Schema       `json:"inputSchema"`
	Hints       *MCPToolHints `json:"hints,omitempty"`
}

// ToJSON converts the tool description to JSON
func (t *MCPToolDescription) ToJSON() (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToPrettyJSON converts the tool description to pretty-printed JSON
func (t *MCPToolDescription) ToPrettyJSON() (string, error) {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MCPToolHints represents behavioral hints for an MCP tool
type MCPToolHints struct {
	ReadOnlyHint    *bool `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool `json:"openWorldHint,omitempty"`
}

// ToMCPTool converts a Go function to an MCP tool description
func ToMCPTool(funcName string, funcType reflect.Type) (*MCPToolDescription, error) {
	if funcType.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected function type, got %v", funcType.Kind())
	}

	// Create input schema from function parameters
	inputSchema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Required:   []string{},
	}

	// Process function parameters
	for i := 0; i < funcType.NumIn(); i++ {
		paramType := funcType.In(i)
		paramName := fmt.Sprintf("arg%d", i)

		paramSchema, err := TypeToSchema(paramType)
		if err != nil {
			return nil, fmt.Errorf("error processing parameter %d: %w", i, err)
		}

		inputSchema.Properties[paramName] = paramSchema
		inputSchema.Required = append(inputSchema.Required, paramName)
	}

	// Create tool description
	tool := &MCPToolDescription{
		Name:        funcName,
		Description: fmt.Sprintf("Function %s", funcName),
		InputSchema: inputSchema,
	}

	return tool, nil
}

// AnalyzeSourceHints analyzes source code to determine MCP tool hints
func AnalyzeSourceHints(filename string, funcName string) (*MCPToolHints, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("error parsing file: %w", err)
	}

	// Find the target function
	var targetFunc *ast.FuncDecl
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Name.Name == funcName {
				targetFunc = fn
				return false
			}
		}
		return true
	})

	if targetFunc == nil {
		return nil, fmt.Errorf("function %q not found", funcName)
	}

	// Analyze the function body
	analyzer := &sourceAnalyzer{
		fset:     fset,
		node:     node,
		funcDecl: targetFunc,
	}

	return analyzer.analyze()
}

// sourceAnalyzer performs static analysis on Go source code
type sourceAnalyzer struct {
	fset     *token.FileSet
	node     *ast.File
	funcDecl *ast.FuncDecl

	// Analysis results
	hasDiskOps     bool
	hasNetworkOps  bool
	hasStateChange bool
	isIdempotent   bool
}

func (a *sourceAnalyzer) analyze() (*MCPToolHints, error) {
	// Walk the function body
	ast.Inspect(a.funcDecl.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			a.analyzeCall(x)
		case *ast.AssignStmt:
			// Check for state changes
			a.hasStateChange = true
		}
		return true
	})

	// Determine hints based on analysis
	hints := &MCPToolHints{}

	// ReadOnlyHint: true if no disk ops and no state changes
	readOnly := !a.hasDiskOps && !a.hasStateChange
	hints.ReadOnlyHint = &readOnly

	// DestructiveHint: true if has disk operations
	if !readOnly {
		destructive := a.hasDiskOps
		hints.DestructiveHint = &destructive
	}

	// IdempotentHint: harder to determine, default to false
	if !readOnly {
		idempotent := a.isIdempotent
		hints.IdempotentHint = &idempotent
	}

	// OpenWorldHint: true if has network operations
	openWorld := a.hasNetworkOps
	hints.OpenWorldHint = &openWorld

	return hints, nil
}

func (a *sourceAnalyzer) analyzeCall(call *ast.CallExpr) {
	// Get the function name being called
	funcName := a.getFunctionName(call)
	if funcName == "" {
		return
	}

	// Check for disk operations
	if strings.Contains(funcName, "os.") || strings.Contains(funcName, "ioutil.") {
		if strings.Contains(funcName, "Write") || strings.Contains(funcName, "Create") ||
			strings.Contains(funcName, "Remove") || strings.Contains(funcName, "Rename") ||
			strings.Contains(funcName, "Mkdir") {
			a.hasDiskOps = true
		}
	}

	// Check for network operations
	if strings.Contains(funcName, "net.") || strings.Contains(funcName, "http.") {
		a.hasNetworkOps = true
	}

	// Check for idempotent operations
	if strings.Contains(funcName, "Get") || strings.Contains(funcName, "Read") ||
		strings.Contains(funcName, "List") || strings.Contains(funcName, "Describe") {
		// These are typically idempotent
		if !a.hasDiskOps && !a.hasStateChange {
			a.isIdempotent = true
		}
	}
}

func (a *sourceAnalyzer) getFunctionName(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		if x, ok := fun.X.(*ast.Ident); ok {
			return x.Name + "." + fun.Sel.Name
		}
		return fun.Sel.Name
	}
	return ""
}
