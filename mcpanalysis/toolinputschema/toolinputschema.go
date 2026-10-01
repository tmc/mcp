package toolinputschema

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

const sdkPath = "github.com/modelcontextprotocol/go-sdk/mcp"

// Analyzer reports missing or explicitly nil InputSchema fields in Tool
// literals passed directly to the SDK's low-level Server.AddTool method.
// It does not track variables or inspect generic mcp.AddTool calls.
var Analyzer = &analysis.Analyzer{
	Name: "toolinputschema",
	Doc:  "check low-level MCP tool registrations for missing input schemas",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
			if !ok {
				return true
			}
			selection := pass.TypesInfo.Selections[sel]
			if selection == nil {
				return true
			}
			method := selection.Obj()
			if method.Pkg() == nil || method.Pkg().Path() != sdkPath || method.Name() != "AddTool" {
				return true
			}
			sig, ok := method.Type().(*types.Signature)
			if !ok || sig.Recv() == nil || !isSDKType(sig.Recv().Type(), "Server") {
				return true
			}
			arg := 0
			if selection.Kind() == types.MethodExpr {
				arg = 1
			}
			if arg >= len(call.Args) {
				return true
			}
			address, ok := ast.Unparen(call.Args[arg]).(*ast.UnaryExpr)
			if !ok || address.Op != token.AND {
				return true
			}
			lit, ok := ast.Unparen(address.X).(*ast.CompositeLit)
			if !ok || !isSDKType(pass.TypesInfo.TypeOf(lit), "Tool") {
				return true
			}
			for _, elt := range lit.Elts {
				field, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					// Do not interpret positional literals.
					return true
				}
				name, ok := field.Key.(*ast.Ident)
				if !ok {
					return true
				}
				if name.Name != "InputSchema" {
					continue
				}
				value, ok := ast.Unparen(field.Value).(*ast.Ident)
				if ok && pass.TypesInfo.Uses[value] == types.Universe.Lookup("nil") {
					pass.Reportf(field.Value.Pos(), "Server.AddTool panics when InputSchema is nil; provide an object schema or use generic mcp.AddTool")
				}
				return true
			}
			pass.Reportf(lit.Pos(), "Server.AddTool panics when InputSchema is missing; provide an object schema or use generic mcp.AddTool")
			return true
		})
	}
	return nil, nil
}

func isSDKType(t types.Type, name string) bool {
	t = types.Unalias(t)
	if pointer, ok := t.(*types.Pointer); ok {
		t = types.Unalias(pointer.Elem())
	}
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == sdkPath && named.Obj().Name() == name
}
