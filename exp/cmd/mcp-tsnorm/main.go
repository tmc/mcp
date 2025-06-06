// Command mcp-tsnorm normalizes timestamps in MCP trace files.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	outputFile     = flag.String("o", "", "output file (default: stdout)")
	startOffsetStr = flag.String("start", "0s", "start offset for relative timestamps (e.g., 0s, 1.5s, 500ms)")
	absoluteStart  = flag.Float64("absolute", -1, "rebase to this absolute unix.milli timestamp (overrides -start)")
	verbose        = flag.Bool("v", false, "verbose mode: print details about timestamp conversion")
	preserveHeader = flag.Bool("preserve-header", true, "preserve the mcptrace header if present")
)

var timestampRegex = regexp.MustCompile(` # (\d+)(?:\.(\d{1,3}))?$`)

func parseTimestampToMillis(secStr, milliStr string) (int64, error) {
	seconds, err := strconv.ParseInt(secStr, 10, 64)
	if err != nil {
		return 0, err
	}
	var millis int64
	if milliStr != "" {
		millis, err = strconv.ParseInt(milliStr, 10, 64)
		if err != nil {
			return 0, err
		}
		// Pad to 3 digits if needed (e.g. .1 -> 100ms, .12 -> 120ms)
		for i := len(milliStr); i < 3; i++ {
			millis *= 10
		}
	}
	return seconds*1000 + millis, nil
}

func formatMillisToTimestamp(totalMillis int64) string {
	seconds := totalMillis / 1000
	millis := totalMillis % 1000
	return fmt.Sprintf("%d.%03d", seconds, millis)
}

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
	match := timestampRegex.FindStringSubmatch(line)

	if len(match) < 2 { // Allow for optional milliseconds part
		fmt.Fprintln(out, line)
		return
	}

	currentTimestampMs, err := parseTimestampToMillis(match[1], match[2])
	if err != nil {
		log.Printf("warning: skipping line with invalid timestamp (%s): %v", line, err)
		fmt.Fprintln(out, line)
		return
	}

	if *firstOriginalTimestampMs == -1 {
		*firstOriginalTimestampMs = currentTimestampMs
		if *verbose {
			log.Printf("detected first timestamp: %d.%03d seconds",
				currentTimestampMs/1000, currentTimestampMs%1000)
		}
	}

	newTimestampMs := (currentTimestampMs - *firstOriginalTimestampMs) + baseTimestampMs

	dataEnd := strings.LastIndex(line, " # ")
	if dataEnd < 0 { // Should not happen if regex matched
		fmt.Fprintln(out, line)
		return
	}
	payloadPart := line[:dataEnd]

	fmt.Fprintf(out, "%s # %s\n", payloadPart, formatMillisToTimestamp(newTimestampMs))
}
