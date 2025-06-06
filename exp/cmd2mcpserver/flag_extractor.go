package cmd2mcpserver

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// FlagExtractor extracts flag definitions from Go source code
type FlagExtractor struct {
	sourceDir string
	flags     []FlagDef
	usesStdin bool
}

// NewFlagExtractor creates a new flag extractor
func NewFlagExtractor(sourceDir string) *FlagExtractor {
	return &FlagExtractor{
		sourceDir: sourceDir,
		flags:     []FlagDef{},
		usesStdin: false,
	}
}

// ExtractFlags analyzes Go source files to find flag definitions
func (fe *FlagExtractor) ExtractFlags() ([]FlagDef, error) {
	err := filepath.Walk(fe.sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "_test.go") {
			return nil
		}

		return fe.analyzeFile(path)
	})

	if err != nil {
		return nil, err
	}

	return fe.flags, nil
}

// UsesStdin returns whether the program uses stdin
func (fe *FlagExtractor) UsesStdin() bool {
	return fe.usesStdin
}

// analyzeFile analyzes a single Go file for flag definitions
func (fe *FlagExtractor) analyzeFile(filename string) error {
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.Inspect(node, func(n ast.Node) bool {
		// Check for stdin usage
		switch n := n.(type) {
		case *ast.SelectorExpr:
			// Check for os.Stdin
			if ident, ok := n.X.(*ast.Ident); ok && ident.Name == "os" && n.Sel.Name == "Stdin" {
				fe.usesStdin = true
			}
		case *ast.CallExpr:
			// Check for bufio.NewScanner(os.Stdin) or bufio.NewReader(os.Stdin) patterns
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "bufio" {
					if (sel.Sel.Name == "NewScanner" || sel.Sel.Name == "NewReader") && len(n.Args) > 0 {
						if arg, ok := n.Args[0].(*ast.SelectorExpr); ok {
							if ident, ok := arg.X.(*ast.Ident); ok && ident.Name == "os" && arg.Sel.Name == "Stdin" {
								fe.usesStdin = true
							}
						}
					}
				}
			}
		}

		// Look for flag.XXX() calls
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check if it's a flag package call
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Check if it's the flag package
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != "flag" {
			return true
		}

		// Extract flag definition based on the method name
		switch sel.Sel.Name {
		case "String", "StringVar":
			fe.extractStringFlag(call)
		case "Int", "IntVar":
			fe.extractIntFlag(call)
		case "Bool", "BoolVar":
			fe.extractBoolFlag(call)
		case "Float64", "Float64Var":
			fe.extractFloatFlag(call)
		}

		return true
	})

	return nil
}

// extractStringFlag extracts a string flag definition
func (fe *FlagExtractor) extractStringFlag(call *ast.CallExpr) {
	if len(call.Args) < 3 {
		return
	}

	flag := FlagDef{Type: "string"}

	// Extract flag name
	if name := extractStringLiteral(call.Args[0]); name != "" {
		flag.Name = name
	} else if len(call.Args) > 1 {
		if name := extractStringLiteral(call.Args[1]); name != "" {
			flag.Name = name
		}
	}

	// Extract default value (second argument for String, third for StringVar)
	defIndex := 1
	descIndex := 2
	if len(call.Args) > 3 { // StringVar
		defIndex = 2
		descIndex = 3
	}

	if defVal := extractStringLiteral(call.Args[defIndex]); defVal != "" {
		flag.Default = fmt.Sprintf("%q", defVal)
	}

	// Extract description
	if desc := extractStringLiteral(call.Args[descIndex]); desc != "" {
		flag.Description = desc
	}

	if flag.Name != "" {
		fe.flags = append(fe.flags, flag)
	}
}

// extractIntFlag extracts an integer flag definition
func (fe *FlagExtractor) extractIntFlag(call *ast.CallExpr) {
	if len(call.Args) < 3 {
		return
	}

	flag := FlagDef{Type: "integer"}

	// Extract flag name
	if name := extractStringLiteral(call.Args[0]); name != "" {
		flag.Name = name
	} else if len(call.Args) > 1 {
		if name := extractStringLiteral(call.Args[1]); name != "" {
			flag.Name = name
		}
	}

	// Extract default value
	if basicLit, ok := call.Args[1].(*ast.BasicLit); ok {
		flag.Default = basicLit.Value
	} else if len(call.Args) > 2 {
		if basicLit, ok := call.Args[2].(*ast.BasicLit); ok {
			flag.Default = basicLit.Value
		}
	}

	// Extract description
	if desc := extractStringLiteral(call.Args[2]); desc != "" {
		flag.Description = desc
	} else if len(call.Args) > 3 {
		if desc := extractStringLiteral(call.Args[3]); desc != "" {
			flag.Description = desc
		}
	}

	if flag.Name != "" {
		fe.flags = append(fe.flags, flag)
	}
}

// extractBoolFlag extracts a boolean flag definition
func (fe *FlagExtractor) extractBoolFlag(call *ast.CallExpr) {
	if len(call.Args) < 3 {
		return
	}

	flag := FlagDef{Type: "boolean"}

	// Extract flag name
	if name := extractStringLiteral(call.Args[0]); name != "" {
		flag.Name = name
	} else if len(call.Args) > 1 {
		if name := extractStringLiteral(call.Args[1]); name != "" {
			flag.Name = name
		}
	}

	// Extract default value (second argument for Bool, third for BoolVar)
	if basicLit, ok := call.Args[1].(*ast.BasicLit); ok && basicLit.Kind == token.IDENT {
		flag.Default = basicLit.Value
	} else if len(call.Args) > 2 {
		if basicLit, ok := call.Args[2].(*ast.BasicLit); ok && basicLit.Kind == token.IDENT {
			flag.Default = basicLit.Value
		}
	}

	// Extract description (third argument for Bool, fourth for BoolVar)
	if desc := extractStringLiteral(call.Args[2]); desc != "" {
		flag.Description = desc
	} else if len(call.Args) > 3 {
		if desc := extractStringLiteral(call.Args[3]); desc != "" {
			flag.Description = desc
		}
	}

	if flag.Name != "" {
		fe.flags = append(fe.flags, flag)
	}
}

// extractFloatFlag extracts a float64 flag definition
func (fe *FlagExtractor) extractFloatFlag(call *ast.CallExpr) {
	if len(call.Args) < 3 {
		return
	}

	flag := FlagDef{Type: "number"}

	// Extract flag name
	if name := extractStringLiteral(call.Args[0]); name != "" {
		flag.Name = name
	} else if len(call.Args) > 1 {
		if name := extractStringLiteral(call.Args[1]); name != "" {
			flag.Name = name
		}
	}

	// Extract default value
	if basicLit, ok := call.Args[1].(*ast.BasicLit); ok {
		flag.Default = basicLit.Value
	} else if len(call.Args) > 2 {
		if basicLit, ok := call.Args[2].(*ast.BasicLit); ok {
			flag.Default = basicLit.Value
		}
	}

	// Extract description
	if desc := extractStringLiteral(call.Args[2]); desc != "" {
		flag.Description = desc
	} else if len(call.Args) > 3 {
		if desc := extractStringLiteral(call.Args[3]); desc != "" {
			flag.Description = desc
		}
	}

	if flag.Name != "" {
		fe.flags = append(fe.flags, flag)
	}
}

// extractStringLiteral extracts a string literal from an expression
func extractStringLiteral(expr ast.Expr) string {
	switch lit := expr.(type) {
	case *ast.BasicLit:
		if lit.Kind == token.STRING {
			// Remove quotes
			return strings.Trim(lit.Value, `"`)
		}
	}
	return ""
}
