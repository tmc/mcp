package mcpcatalog_test

import (
	"fmt"
	"strings"

	"github.com/tmc/mcp/mcpcatalog"
)

func Example() {
	s, err := mcpcatalog.Read(strings.NewReader(`{"format":1,"context":{"scope":"project","principal":"local","client":"default"},"initialize":{"protocolVersion":"2026-07-28","serverInfo":{"name":"example"},"capabilities":{}},"tools":[]}`))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(len(mcpcatalog.Compare(s, s)))
	// Output: 0
}

func ExampleRead() {
	_, err := mcpcatalog.Read(strings.NewReader(`{"format":2}`))
	fmt.Println(err)
	// Output: invalid catalog header
}
func ExampleWrite() {
	s, _ := mcpcatalog.Read(strings.NewReader(`{"format":1,"context":{"scope":"p","principal":"local","client":"default"},"initialize":{"protocolVersion":"2026-07-28","serverInfo":{"name":"s"},"capabilities":{}},"tools":[]}`))
	var out strings.Builder
	fmt.Println(mcpcatalog.Write(&out, s))
	fmt.Println(strings.Contains(out.String(), `"format": 1`))
	// Output:
	// <nil>
	// true
}
func ExampleCompare() {
	fmt.Println(mcpcatalog.Compare(nil, nil)[0].Kind)
	// Output: unknown
}
func ExampleContext() {
	visibility := mcpcatalog.Context{Scope: "project", Principal: "local", Client: "default"}
	fmt.Println(visibility.Scope, visibility.Principal)
	// Output: project local
}
func ExampleSnapshot() {
	s, _ := mcpcatalog.Read(strings.NewReader(`{"format":1,"initialize":{"protocolVersion":"2026-07-28","serverInfo":{"name":"s"},"capabilities":{}},"tools":[]}`))
	fmt.Println(s.Format, len(s.Tools))
	// Output: 1 0
}
func ExampleChange() {
	change := mcpcatalog.Change{Kind: mcpcatalog.Incompatible, Tool: "search", Path: "tool", Message: "tool removed"}
	fmt.Println(change.Kind, change.Tool, change.Message)
	// Output: incompatible search tool removed
}
func ExampleKind() {
	fmt.Println(mcpcatalog.Incompatible, mcpcatalog.Unknown, mcpcatalog.Changed)
	// Output: incompatible unknown changed
}
