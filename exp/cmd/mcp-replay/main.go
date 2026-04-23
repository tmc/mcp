// Command mcp-replay replays an MCP recording to stdout, respecting millisecond timestamps.
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
	inFile            = flag.String("f", "", "recording file to replay")
	speed             = flag.Float64("speed", 1.0, "replay speed multiplier (1.0 = original speed)")
	verbose           = flag.Bool("v", false, "verbose output")
	quiet             = flag.Bool("q", false, "quiet mode: suppress log messages")
	output            = flag.String("o", "", "output file (default: stdout)")
	useNowTime        = flag.Bool("now", false, "use current time for timestamps")
	stripTime         = flag.Bool("strip", false, "strip timestamps entirely")
	relTime           = flag.Bool("rel", false, "use relative timestamps (seconds from start)")
	sendsOnly         = flag.Bool("sends", false, "only replay send messages")
	recvsOnly         = flag.Bool("recvs", false, "only replay receive messages")
	jsonOnly          = flag.Bool("json", false, "output only the JSON content (no mcp- prefix or timestamp)")
	mockServer        = flag.Bool("mock-server", false, "act as a mock server, reading requests from stdin and replying with matching responses")
	mockClient        = flag.Bool("mock-client", false, "act as a mock client, sending requests and expecting responses on stdin")
	mockTimeout       = flag.Duration("timeout", 5*time.Second, "timeout for waiting for response in server/client mode")
	autoNotifications = flag.Bool("auto-notify", true, "automatically send server notifications from the recording without waiting for requests")
	autoResponses     = flag.Bool("auto-respond", false, "automatically send all server responses from the recording in sequence without waiting for matching requests")
	preserveOrder     = flag.Bool("preserve-order", true, "preserve message order for complex interactions like sampling")
	useShadow         = flag.Bool("use-shadow", false, "use shadow server responses in mock server mode")
)

// Regular expression to match the timestamp portion with milliseconds
// Used for timing calculations, not for error reporting
var timestampRegex = regexp.MustCompile(` # (\d+)(?:\.(\d+))?$`)

// OperationNotification represents a notification related to an operation
type OperationNotification struct {
	JSON          string
	ProgressValue int
	DelayMs       int
}

func main() {
	log.SetPrefix("mcp-replay: ")
	log.SetFlags(0) // No date/time prefix
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] recording\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Configure logger based on quiet flag
	if *quiet {
		log.SetOutput(io.Discard) // Discard all logs in quiet mode
	}

	// Check for recording filename from positional argument
	var recordingFile string
	if *inFile != "" {
		recordingFile = *inFile
	} else if flag.NArg() > 0 {
		recordingFile = flag.Arg(0)
	} else {
		flag.Usage()
		os.Exit(1)
	}

	// Set log prefix based on mode
	if *mockServer {
		log.SetPrefix("mcp-replay [mock-server] ")
	} else if *mockClient {
		log.SetPrefix("mcp-replay [mock-client] ")
	}

	// Set up output
	var out io.Writer = os.Stdout
	if *output != "" {
		outFile, err := os.Create(*output)
		if err != nil {
			printError(*output, 1, 1, "failed to create output file: %v", err)
			log.Fatalf("Error creating output file: %v", err)
		}
		defer outFile.Close()
		out = outFile
	}

	// Check for mock server/client modes
	if *mockServer {
		// In mock server mode, we want to use JSON mode by default for inputs and outputs
		// Unless explicitly overridden by the user
		jsonFlag := flag.Lookup("json")
		wasExplicitlySet := false

		// Check if flag was explicitly set by looking at args
		for _, arg := range os.Args {
			if arg == "-json" || arg == "-json=true" || arg == "-json=false" {
				wasExplicitlySet = true
				break
			}
		}

		if !wasExplicitlySet && jsonFlag != nil {
			*jsonOnly = true
			if *verbose {
				log.Println("mock server mode: enabling JSON mode by default")
			}
		}
		runMockServer(recordingFile, out, nil)
		return
	}

	if *mockClient {
		// In mock client mode, we also want to use JSON mode by default
		// Unless explicitly overridden by the user
		jsonFlag := flag.Lookup("json")
		wasExplicitlySet := false

		// Check if flag was explicitly set by looking at args
		for _, arg := range os.Args {
			if arg == "-json" || arg == "-json=true" || arg == "-json=false" {
				wasExplicitlySet = true
				break
			}
		}

		if !wasExplicitlySet && jsonFlag != nil {
			*jsonOnly = true
			if *verbose {
				log.Println("enabling json mode by default")
			}
		}

		runMockClient(recordingFile, out, nil)
		return
	}

	// Normal replay mode
	f, err := os.Open(recordingFile)
	if err != nil {
		// Output error in vim-compatible format
		printError(recordingFile, 1, 1, "failed to open file: %v", err)
		log.Fatalf("Error opening recording file: %v", err)
	}
	defer f.Close()

	if *verbose {
		log.Printf("replaying %s at %.1fx speed", recordingFile, *speed)
	}

	scanner := bufio.NewScanner(f)
	var firstTimestampMs float64
	var prevTimestampMs float64
	var firstLine = true
	var lastReplayTime time.Time

	// Regular expression to extract the direction (send/recv)
	directionRegex := regexp.MustCompile(`^mcp-(\w+)\s+`)

	for scanner.Scan() {
		line := scanner.Text()

		// Check direction and filter if needed
		dirMatch := directionRegex.FindStringSubmatch(line)
		if len(dirMatch) > 1 {
			direction := dirMatch[1]
			if (*sendsOnly && direction != "send") || (*recvsOnly && direction != "recv") {
				if *verbose {
					log.Printf("skipping %s message", direction)
				}
				continue
			}
		}

		// Extract timestamp using regex
		match := timestampRegex.FindStringSubmatch(line)
		if len(match) < 2 {
			if *verbose {
				log.Printf("warning: no timestamp found in line: %s", line)
			}
			// Write line without modification
			fmt.Fprintln(out, line)
			continue
		}

		// Parse timestamp with milliseconds
		seconds, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			log.Printf("warning: invalid timestamp: %v", err)
			fmt.Fprintln(out, line)
			continue
		}

		// Parse milliseconds if present
		var millis float64
		if len(match) > 2 && match[2] != "" {
			millisStr := match[2]
			// Ensure we have the right precision
			for len(millisStr) < 3 {
				millisStr += "0"
			}
			millisInt, err := strconv.ParseInt(millisStr[:3], 10, 64)
			if err == nil {
				millis = float64(millisInt) / 1000.0
			}
		}

		// Convert to total seconds with millisecond precision
		timestampMs := float64(seconds) + millis

		// Handle timing
		if firstLine {
			// Initialize for the first line
			firstTimestampMs = timestampMs
			prevTimestampMs = timestampMs
			firstLine = false
			lastReplayTime = time.Now() // Reset time when we find the first valid timestamp
		} else {
			// For subsequent lines, we calculate based on the time difference
			// between this record and the previous record (more accurate timing)
			recordedDelta := timestampMs - prevTimestampMs
			if recordedDelta < 0 {
				// Handle out-of-order timestamps
				if *verbose {
					log.Printf("warning: timestamps out of order (%.3f before %.3f)",
						timestampMs, prevTimestampMs)
				}
				recordedDelta = 0
			}

			// Wait for the appropriate amount of time based on speed setting
			waitTime := time.Duration(float64(recordedDelta) * float64(time.Second) / *speed)

			if waitTime > 0 {
				if *verbose {
					log.Printf("waiting %v (delta: %.3fs)...",
						waitTime, recordedDelta)
				}

				// Calculate actual time to wait by considering time spent processing
				elapsed := time.Since(lastReplayTime)
				if elapsed < waitTime {
					time.Sleep(waitTime - elapsed)
				}
			}

			// Update previous timestamp for next iteration
			prevTimestampMs = timestampMs
			lastReplayTime = time.Now()
		}

		// Extract data (everything before the timestamp)
		dataEnd := strings.LastIndex(line, " # ")
		if dataEnd < 0 {
			dataEnd = len(line)
		}
		data := line[:dataEnd]

		if *verbose {
			log.Printf("> %s", data)
		}

		// Extract JSON content if needed
		var jsonContent string
		if dirMatch := directionRegex.FindStringSubmatch(data); len(dirMatch) > 1 {
			// Format: "mcp-<direction> <json content>"
			parts := strings.SplitN(data, " ", 2)
			if len(parts) == 2 {
				jsonContent = parts[1]
			}
		}

		// Determine output format and content
		var outputLine string
		if *jsonOnly && jsonContent != "" {
			// Output only the raw JSON
			outputLine = jsonContent
		} else if *stripTime {
			// No timestamp
			outputLine = data
		} else if *useNowTime {
			// Current time with milliseconds
			now := time.Now()
			outputLine = fmt.Sprintf("%s # %.3f", data, float64(now.UnixNano())/1e9)
		} else if *relTime {
			// Relative time (seconds from start with millisecond precision)
			relTimeSec := timestampMs - firstTimestampMs
			outputLine = fmt.Sprintf("%s # %.3f", data, relTimeSec)
		} else {
			// Original timestamp (DEFAULT)
			if len(match) > 2 && match[2] != "" {
				outputLine = fmt.Sprintf("%s # %d.%s", data, seconds, match[2])
			} else {
				outputLine = fmt.Sprintf("%s # %d", data, seconds)
			}
		}

		// Write the line to output
		fmt.Fprintln(out, outputLine)
	}

	if err := scanner.Err(); err != nil {
		printError(recordingFile, 0, 1, "error reading file: %v", err)
		log.Fatalf("Error reading file: %v", err)
	}
}

// printError outputs an error in vim-compatible format
// This allows error messages to be processed by vim's quickfix mode
func printError(filename string, line int, col int, msg string, details ...interface{}) {
	// Format: filename:line:column: type: message
	// This follows Vim's errorformat spec with: %f:%l:%c: %t: %m
	// Where:
	// %f = filename
	// %l = line number
	// %c = column number
	// %t = error type (single character: e=error, w=warning, i=info, n=note)
	// %m = error message
	if len(details) > 0 {
		msg = fmt.Sprintf(msg, details...)
	}
	fmt.Printf("%s:%d:%d: error: %s\n", filename, line, col, msg)
}
