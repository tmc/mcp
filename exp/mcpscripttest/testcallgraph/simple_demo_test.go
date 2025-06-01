package testcallgraph

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSimpleDemo(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run demo
	Demo()

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check output
	if !strings.Contains(output, "mcpdiff") {
		t.Errorf("Expected output to contain 'mcpdiff', got: %s", output)
	}

	if !strings.Contains(output, "testcallgraph creates these missing edges") {
		t.Error("Expected explanation of missing edges")
	}
}

func TestStitcherAnalysis(t *testing.T) {
	stitcher := &SimpleStitcher{}
	
	testContent := `
# Test script
exec mcpdiff --help
exec echo "test"
exec mcpdiff trace1.json trace2.json
`

	connections := stitcher.AnalyzeAndStitch("test.txt", testContent)
	
	if len(connections) != 3 {
		t.Errorf("expected 3 connections, got %d", len(connections))
	}
	
	// Count mcpdiff connections
	mcpdiffCount := 0
	for _, conn := range connections {
		if conn.Program == "mcpdiff" {
			mcpdiffCount++
			t.Logf("Connection: %s:%d -> %s", conn.TestFile, conn.TestLine, conn.Program)
		}
	}
	
	if mcpdiffCount != 2 {
		t.Errorf("expected 2 mcpdiff connections, got %d", mcpdiffCount)
	}
}

func ExampleSimpleStitcher() {
	stitcher := &SimpleStitcher{}
	
	testContent := `exec mcpdiff --help`
	connections := stitcher.AnalyzeAndStitch("example.txt", testContent)
	
	for _, conn := range connections {
		fmt.Printf("%s:%d -> %s\n", conn.TestFile, conn.TestLine, conn.Program)
	}
	// Output: example.txt:1 -> mcpdiff
}