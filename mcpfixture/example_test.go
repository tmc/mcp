package mcpfixture_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/tmc/mcp/mcpfixture"
)

func ExampleConvert() {
	var fixture bytes.Buffer
	err := mcpfixture.Convert(strings.NewReader(trace), &fixture, mcpfixture.Options{Record: 4})
	fmt.Println(err == nil, strings.Contains(fixture.String(), "exec mcpfixture check"))
	// Output: true true
}
func ExampleOptions() {
	options := mcpfixture.Options{Record: 4, Session: "demo"}
	fmt.Println(options.Record, options.Session)
	// Output: 4 demo
}
func ExampleFixture() {
	fixture := mcpfixture.Fixture{ProtocolVersion: "2025-11-25", SourceRecord: 4}
	fmt.Println(fixture.ProtocolVersion, fixture.SourceRecord)
	// Output: 2025-11-25 4
}
func ExampleCheck() {
	fmt.Println(mcpfixture.Check(context.Background(), nil, mcpfixture.Fixture{}))
	// Output: invalid fixture or session
}

func ExampleRead() {
	fixture, err := mcpfixture.Read(strings.NewReader(`{"protocolVersion":"2025-11-25","clientInfo":{"name":"demo","version":"1"},"params":{"name":"echo","arguments":{}},"expected":{"content":[]},"sourceRecord":4}`))
	fmt.Println(fixture.Params.Name, err)
	// Output: echo <nil>
}

func ExampleClientOptions() {
	fixture := mcpfixture.Fixture{}
	options, err := mcpfixture.ClientOptions(fixture)
	fmt.Println(options.Capabilities != nil, err)
	// Output: true <nil>
}
func ExamplePrerequisite() {
	prerequisite := mcpfixture.Prerequisite{Method: "tools/list"}
	fmt.Println(prerequisite.Method)
	// Output: tools/list
}
