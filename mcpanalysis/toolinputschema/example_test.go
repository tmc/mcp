package toolinputschema_test

import (
	"fmt"

	"github.com/tmc/mcp/mcpanalysis/toolinputschema"
)

func Example() {
	fmt.Println(toolinputschema.Analyzer.Name)
	// Output: toolinputschema
}

func ExampleAnalyzer() {
	fmt.Println(toolinputschema.Analyzer.Name)
	// Output: toolinputschema
}
