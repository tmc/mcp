package main

import (
	"os"
	"time"

	"github.com/tmc/mcp/mcptrace"
)

func main() {
	writer := mcptrace.NewWriter(os.Stdout)
	_ = writer.Write(mcptrace.Record{
		Direction: "send",
		Message:   []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`),
		Time:      time.Now(),
	})
}
