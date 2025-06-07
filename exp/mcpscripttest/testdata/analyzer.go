package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

func main() {
	fmt.Println("Static analysis sees function calls...")
	src := `package main
import "os/exec"
func runTest() {
    cmd := exec.Command("mcpdiff", "--help")
    cmd.Run()
}`

	// Parse the source
	fset := token.NewFileSet()
	node, _ := parser.ParseFile(fset, "", src, 0)

	// Find exec.Command calls
	ast.Inspect(node, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "Command" {
					fmt.Println("Found exec.Command call")
					// But we can't determine at compile time what it will execute!
					fmt.Println("Cannot determine target binary statically")
				}
			}
		}
		return true
	})
}
