package toolgraph

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// NodeType represents the type of node in the graph
type NodeType string

const (
	NodeTypeTest    NodeType = "test"
	NodeTypeTool    NodeType = "tool"
	NodeTypeCommand NodeType = "command"
	NodeTypeFile    NodeType = "file"
	NodeTypePackage NodeType = "package"
)

// Node represents a node in the tool graph
type Node struct {
	ID       string                 `json:"id"`
	Type     NodeType               `json:"type"`
	Label    string                 `json:"label"`
	Path     string                 `json:"path,omitempty"`
	Package  string                 `json:"package,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Edge represents an edge in the tool graph
type Edge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
	Label  string `json:"label,omitempty"`
}

// Graph represents the complete tool graph
type Graph struct {
	Nodes    map[string]*Node `json:"nodes"`
	Edges    []*Edge          `json:"edges"`
	Root     string           `json:"root"`
	MaxDepth int              `json:"maxDepth"`
}

// GraphOptions contains options for building the graph
type GraphOptions struct {
	MaxDepth   int
	IncludeStd bool
	Verbose    bool
}

// GraphBuilder builds tool dependency graphs
type GraphBuilder struct {
	options GraphOptions
	visited map[string]bool
	graph   *Graph
	fset    *token.FileSet
}

// NewGraphBuilder creates a new graph builder
func NewGraphBuilder(options GraphOptions) *GraphBuilder {
	return &GraphBuilder{
		options: options,
		visited: make(map[string]bool),
		graph: &Graph{
			Nodes: make(map[string]*Node),
			Edges: []*Edge{},
		},
		fset: token.NewFileSet(),
	}
}

// BuildFromTarget builds a graph starting from a target test or directory
func (b *GraphBuilder) BuildFromTarget(target string) (*Graph, error) {
	// Check if target is a file or directory
	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("failed to stat target: %w", err)
	}

	if info.IsDir() {
		// Process all test files in directory
		err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, "_test.go") {
				if err := b.processTestFile(path, 0); err != nil {
					b.log("Error processing %s: %v", path, err)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Process single file
		if err := b.processTestFile(target, 0); err != nil {
			return nil, err
		}
		b.graph.Root = target
	}

	b.graph.MaxDepth = b.options.MaxDepth
	return b.graph, nil
}

func (b *GraphBuilder) processTestFile(path string, depth int) error {
	if depth > b.options.MaxDepth {
		return nil
	}

	if b.visited[path] {
		return nil
	}
	b.visited[path] = true

	// Add test node
	testNode := &Node{
		ID:    path,
		Type:  NodeTypeTest,
		Label: filepath.Base(path),
		Path:  path,
		Metadata: map[string]interface{}{
			"depth": depth,
		},
	}
	b.graph.Nodes[testNode.ID] = testNode

	// Parse the test file
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// Parse Go AST
	astFile, err := parser.ParseFile(b.fset, path, content, parser.ParseComments)
	if err != nil {
		// If not a Go file, look for scripttest patterns
		return b.processScriptTest(path, string(content), depth)
	}

	// Process Go test file
	return b.processGoTest(path, astFile, depth)
}

func (b *GraphBuilder) processGoTest(path string, file *ast.File, depth int) error {
	// Extract package name
	pkgName := file.Name.Name
	pkgNode := &Node{
		ID:      "pkg:" + pkgName,
		Type:    NodeTypePackage,
		Label:   pkgName,
		Package: pkgName,
	}
	b.graph.Nodes[pkgNode.ID] = pkgNode

	// Connect test to package
	b.addEdge(path, pkgNode.ID, "belongs_to", "")

	// Find test functions and their dependencies
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if strings.HasPrefix(x.Name.Name, "Test") {
				b.processTestFunction(path, x, depth)
			}
		case *ast.CallExpr:
			b.processCallExpr(path, x, depth)
		case *ast.ImportSpec:
			b.processImport(path, x, depth)
		}
		return true
	})

	return nil
}

func (b *GraphBuilder) processTestFunction(testPath string, fn *ast.FuncDecl, depth int) {
	// Create node for test function
	fnNode := &Node{
		ID:    testPath + ":" + fn.Name.Name,
		Type:  NodeTypeTest,
		Label: fn.Name.Name,
		Path:  testPath,
		Metadata: map[string]interface{}{
			"function": true,
			"line":     b.fset.Position(fn.Pos()).Line,
		},
	}
	b.graph.Nodes[fnNode.ID] = fnNode

	// Connect test file to function
	b.addEdge(testPath, fnNode.ID, "contains", "")
}

func (b *GraphBuilder) processCallExpr(testPath string, call *ast.CallExpr, depth int) {
	// Look for tool/command invocations
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		// Check for exec.Command or similar patterns
		if ident, ok := sel.X.(*ast.Ident); ok {
			if ident.Name == "exec" && sel.Sel.Name == "Command" {
				b.processExecCommand(testPath, call, depth)
			}
		}
	}
}

func (b *GraphBuilder) processExecCommand(testPath string, call *ast.CallExpr, depth int) {
	if len(call.Args) > 0 {
		// Extract command name from first argument
		if lit, ok := call.Args[0].(*ast.BasicLit); ok {
			cmdName := strings.Trim(lit.Value, `"`)
			cmdNode := &Node{
				ID:    "cmd:" + cmdName,
				Type:  NodeTypeCommand,
				Label: cmdName,
				Metadata: map[string]interface{}{
					"line": b.fset.Position(call.Pos()).Line,
				},
			}
			b.graph.Nodes[cmdNode.ID] = cmdNode
			b.addEdge(testPath, cmdNode.ID, "executes", "")
		}
	}
}

func (b *GraphBuilder) processImport(testPath string, imp *ast.ImportSpec, depth int) {
	if imp.Path == nil {
		return
	}

	importPath := strings.Trim(imp.Path.Value, `"`)

	// Skip standard library unless requested
	if !b.options.IncludeStd && !strings.Contains(importPath, ".") {
		return
	}

	// Create import node
	impNode := &Node{
		ID:      "import:" + importPath,
		Type:    NodeTypePackage,
		Label:   filepath.Base(importPath),
		Package: importPath,
	}
	b.graph.Nodes[impNode.ID] = impNode
	b.addEdge(testPath, impNode.ID, "imports", "")
}

func (b *GraphBuilder) processScriptTest(path string, content string, depth int) error {
	// Use the dedicated scripttest analyzer
	analyzer := NewScriptTestAnalyzer(b)
	return analyzer.AnalyzeScriptTest(path, content, depth)
}

func (b *GraphBuilder) extractToolFromJSON(path string, jsonStr string, line int, depth int) {
	// Simple pattern matching for tool names in JSON
	patterns := []string{
		`"method":\s*"([^"]+)"`,
		`"tool":\s*"([^"]+)"`,
		`"name":\s*"([^"]+)"`,
	}

	for _, pattern := range patterns {
		if tool := extractPattern(jsonStr, pattern); tool != "" {
			toolNode := &Node{
				ID:    fmt.Sprintf("tool:%s:%d", tool, line),
				Type:  NodeTypeTool,
				Label: tool,
				Metadata: map[string]interface{}{
					"line": line + 1,
					"json": true,
				},
			}
			b.graph.Nodes[toolNode.ID] = toolNode
			b.addEdge(path, toolNode.ID, "uses", fmt.Sprintf("line %d", line+1))
		}
	}
}

func (b *GraphBuilder) addEdge(source, target, edgeType, label string) {
	edge := &Edge{
		ID:     fmt.Sprintf("%s->%s", source, target),
		Source: source,
		Target: target,
		Type:   edgeType,
		Label:  label,
	}
	b.graph.Edges = append(b.graph.Edges, edge)
}

func (b *GraphBuilder) log(format string, args ...interface{}) {
	if b.options.Verbose {
		fmt.Printf("[GraphBuilder] "+format+"\n", args...)
	}
}

// Helper function to extract pattern from string
func extractPattern(s, pattern string) string {
	// Simple pattern extraction without regex for now
	// This is a placeholder - in production, use regexp
	return ""
}
