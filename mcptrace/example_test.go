package mcptrace_test

import (
	"bytes"
	"fmt"

	"github.com/tmc/mcp/mcptrace"
)

func ExampleReader() {
	var trace bytes.Buffer
	writer := mcptrace.NewWriter(&trace)
	_ = writer.Write(mcptrace.Record{
		Direction: "send",
		Message:   []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`),
	})

	record, _ := mcptrace.NewReader(&trace).Read()
	fmt.Println(record.Direction, string(record.Message))
	// Output: send {"jsonrpc":"2.0","id":1,"method":"ping"}
}
