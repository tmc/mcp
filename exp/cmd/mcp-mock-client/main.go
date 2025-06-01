package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	dryRun    = flag.Bool("n", false, "print requests instead of sending them")
	scenario  = flag.Bool("scenario", false, "run a scenario file")
	validate  = flag.Bool("validate", false, "validate responses against expectations")
	timeout   = flag.Duration("timeout", 30*time.Second, "timeout for the entire scenario")
	stepDelay = flag.Duration("step-delay", 0, "delay between scenario steps")
	verbose   = flag.Bool("v", false, "verbose output")
)

func main() {
	log.SetPrefix("mcp-mock-client: ")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: mcp-mock-client [flags] recording|scenario\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	inputFile := flag.Arg(0)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	var err error
	if *scenario {
		err = runScenario(ctx, os.Stdout, inputFile)
	} else {
		err = runRecording(os.Stdout, inputFile)
	}

	if err != nil {
		log.Fatal(err)
	}
}

// runRecording processes a recording file (original behavior)
func runRecording(out io.Writer, recordingFile string) error {
	f, err := os.Open(recordingFile)
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Bytes()
		if !bytes.HasPrefix(line, []byte("mcp-in ")) {
			continue
		}
		req := line[7:] // skip "mcp-in "
		if *dryRun {
			fmt.Printf("would send: %s\n", req)
			continue
		}
		if _, err := fmt.Fprintf(out, "%s\n", req); err != nil {
			return err
		}
	}
	return s.Err()
}

// detectFileType determines if the file is a scenario or recording
func detectFileType(filename string) (string, error) {
	// Check file extension first
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".json" {
		return "scenario", nil
	}

	// If no clear extension, peek at file contents
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Read the first few bytes
	peek := make([]byte, 100)
	n, err := f.Read(peek)
	if err != nil && err != io.EOF {
		return "", err
	}

	// Check if it looks like JSON
	peek = peek[:n]
	if bytes.HasPrefix(bytes.TrimSpace(peek), []byte{'{'}) {
		// Try to parse as JSON
		var obj map[string]interface{}
		if json.Unmarshal(peek, &obj) == nil {
			return "scenario", nil
		}
	}

	// Default to recording format
	return "recording", nil
}

// run is a high-level function that automatically detects file type
func run(out io.Writer, inputFile string) error {
	fileType, err := detectFileType(inputFile)
	if err != nil {
		return fmt.Errorf("detecting file type: %w", err)
	}

	ctx := context.Background()
	if fileType == "scenario" {
		return runScenario(ctx, out, inputFile)
	}
	return runRecording(out, inputFile)
}
