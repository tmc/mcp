// Command mcp-tsnorm normalizes timestamps in MCP trace files.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tmc/mcp/mcptrace"
)

var (
	outputFile     = flag.String("o", "", "output file (default: stdout)")
	startOffsetStr = flag.String("start", "0s", "start offset for relative timestamps (e.g., 0s, 1.5s, 500ms)")
	absoluteStart  = flag.Float64("absolute", -1, "rebase to this absolute unix.milli timestamp (overrides -start)")
	verbose        = flag.Bool("v", false, "verbose mode: print details about timestamp conversion")
	preserveHeader = flag.Bool("preserve-header", true, "preserve the mcptrace header if present")
)

func main() {
	log.SetPrefix("mcp-tsnorm: ")
	flag.Parse()

	var in io.Reader = os.Stdin
	if flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatalf("opening input file: %v", err)
		}
		defer f.Close()
		in = f
	}

	var out io.Writer = os.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("creating output file: %v", err)
		}
		defer f.Close()
		out = f
	}

	scanner := bufio.NewScanner(in)
	var firstOriginalTimestampMs int64 = -1
	var baseTimestampMs int64
	var lineNumber int

	if *absoluteStart != -1 {
		baseTimestampMs = int64(*absoluteStart * 1000)
		if *verbose {
			log.Printf("using absolute timestamp base: %d.%03d seconds", baseTimestampMs/1000, baseTimestampMs%1000)
		}
	} else {
		offsetDuration, err := time.ParseDuration(*startOffsetStr)
		if err != nil {
			log.Fatalf("invalid -start offset: %v", err)
		}
		baseTimestampMs = offsetDuration.Milliseconds()
		if *verbose {
			log.Printf("using offset timestamp base: %d.%03d seconds", baseTimestampMs/1000, baseTimestampMs%1000)
		}
	}

	// Check first line for header
	if scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// If it's a header and we're preserving headers, write it and move on
		if strings.HasPrefix(line, "# mcptrace:") {
			if *verbose {
				log.Printf("found header: %s", line)
			}
			if *preserveHeader {
				fmt.Fprintln(out, line)
			} else if *verbose {
				log.Printf("skipping header (preserve-header=false)")
			}
		} else {
			// Not a header, process as a normal line
			processLine(line, out, &firstOriginalTimestampMs, baseTimestampMs)
		}
	}

	// Process all remaining lines
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		processLine(line, out, &firstOriginalTimestampMs, baseTimestampMs)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("reading input: %v", err)
	}

	if *verbose && firstOriginalTimestampMs != -1 {
		log.Printf("processed %d lines, first timestamp: %d.%03d",
			lineNumber, firstOriginalTimestampMs/1000, firstOriginalTimestampMs%1000)
	}
}

func processLine(line string, out io.Writer, firstOriginalTimestampMs *int64, baseTimestampMs int64) {
	record, err := mcptrace.ParseRecord(line)
	if err != nil || record.Time.IsZero() {
		fmt.Fprintln(out, line)
		return
	}
	currentTimestampMs := record.Time.UnixMilli()

	if *firstOriginalTimestampMs == -1 {
		*firstOriginalTimestampMs = currentTimestampMs
		if *verbose {
			log.Printf("detected first timestamp: %d.%03d seconds",
				currentTimestampMs/1000, currentTimestampMs%1000)
		}
	}

	newTimestampMs := (currentTimestampMs - *firstOriginalTimestampMs) + baseTimestampMs

	record.Time = time.UnixMilli(newTimestampMs)
	if err := mcptrace.NewWriter(out).Write(record); err != nil {
		log.Printf("writing normalized record: %v", err)
	}
}
