package main

import (
	"io"
	"os"

	"github.com/tmc/mcp/internal/mcpspy"
)

// handlePipeMode sets up mcpspy in pipe logger mode.
func handlePipeMode(recorder *mcpspy.Recorder, stdout io.Writer) error {
	_, err := io.Copy(stdout, recorder.Reader("recv", os.Stdin))
	return err
}
