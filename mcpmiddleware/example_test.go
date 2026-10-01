package mcpmiddleware_test

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/tmc/mcp/mcpmiddleware"
)

func ExampleLog() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	middleware := mcpmiddleware.Log(logger)
	fmt.Println(middleware != nil)
	// Output: true
}
