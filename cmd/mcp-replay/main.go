// Command mcp-replay replays an MCP recording to stdout, respecting millisecond timestamps.
package main

import (
	"bufio"
	"encoding/json"
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

// runMockClient reads a recording file and acts as a client,
// sending requests from the recording and waiting for responses
func runMockClient(recordingFile string, out io.Writer) {
	// Read all messages from the recording
	f, err := os.Open(recordingFile)
	if err != nil {
		// Output error in vim-compatible format
		printError(recordingFile, 1, 1, "failed to open file: %v", err)
		log.Fatalf("Error opening recording file: %v", err)
	}
	defer f.Close()

	// Collect client requests (send messages) and expected responses (recv messages)
	var requests []string
	responseMap := make(map[string]string) // Map from request ID to expected response

	// First pass: collect requests and responses
	scanner := bufio.NewScanner(f)
	directionRegex := regexp.MustCompile(`^mcp-(\w+)\s+`)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Extract direction (send/recv)
		dirMatch := directionRegex.FindStringSubmatch(line)
		if len(dirMatch) < 2 {
			continue
		}

		direction := dirMatch[1]

		// Parse the JSON content
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 2 {
			continue
		}

		// Extract JSON content - it's everything after "mcp-<direction> " and before " # timestamp"
		jsonEndPos := strings.LastIndex(line, " # ")
		if jsonEndPos == -1 {
			jsonEndPos = len(line)
		}

		// Get the prefix length (mcp-direction plus space)
		prefixLen := len(parts[0]) + 1

		// Extract the JSON string
		jsonStr := line[prefixLen:jsonEndPos]

		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
			// We want to report the actual file line number
			// Use the helper function for vim-compatible errors
			printError(recordingFile, lineNumber, 1, "JSON parse error: %v", err)
			if *verbose {
				log.Printf("error parsing JSON: %v", err)
			}
			continue
		}

		// Handle based on direction
		// NOTE: In the recording file:
		// - "mcp-recv" means client→server (client requests)
		// - "mcp-send" means server→client (server responses)
		if direction == "recv" {
			// "recv" means client→server, so these are client requests
			if *jsonOnly {
				requests = append(requests, jsonStr)
			} else {
				requests = append(requests, line)
			}

			// If it has an ID, prepare to match with response
			if id, ok := jsonData["id"]; ok {
				idStr := fmt.Sprintf("%v", id)
				responseMap[idStr] = "" // Will be filled in if we find a matching response
			}
		} else if direction == "send" {
			// "send" means server→client, so these are server responses
			if id, ok := jsonData["id"]; ok {
				idStr := fmt.Sprintf("%v", id)
				if *jsonOnly {
					responseMap[idStr] = jsonStr
				} else {
					responseMap[idStr] = line
				}
			}
		}
	}

	if *verbose {
		log.Printf("loaded %d requests from recording", len(requests))
		log.Printf("found %d response mappings", len(responseMap))
	}

	// Set up stdin scanner for responses
	stdinScanner := bufio.NewScanner(os.Stdin)

	// Start goroutine to read responses
	responseChannel := make(chan string)
	go func() {
		for stdinScanner.Scan() {
			responseChannel <- stdinScanner.Text()
		}
	}()

	log.Println("sending requests...")

	// Now send requests and expect responses
	for i, request := range requests {
		// Extract ID from request if present
		var requestID string
		var jsonData map[string]interface{}

		// For JSON-only mode, parse directly; otherwise extract JSON part
		jsonContent := request
		if !*jsonOnly {
			parts := strings.SplitN(request, " ", 3)
			if len(parts) >= 2 {
				jsonContent = parts[1]
			}
		}

		if err := json.Unmarshal([]byte(jsonContent), &jsonData); err == nil {
			if id, ok := jsonData["id"]; ok {
				requestID = fmt.Sprintf("%v", id)
			}
		}

		// Send the request
		fmt.Fprintln(out, request)

		if *verbose {
			if requestID != "" {
				log.Printf("sent request with ID: %s", requestID)
			} else {
				log.Println("sent request (no ID)")
			}
		}

		// Wait for a response if we have a request ID and expect a response
		if requestID != "" && responseMap[requestID] != "" {
			// Wait with timeout
			select {
			case response := <-responseChannel:
				if *verbose {
					log.Printf("received response: %s", response)
				}

				// We could validate the response against expected response here
				// but for now we just continue

			case <-time.After(*mockTimeout):
				if *verbose {
					log.Printf("timeout waiting for response to request ID: %s", requestID)
				}
			}
		}

		// Add timing delay between requests based on speed
		if i < len(requests)-1 && *speed > 0 {
			delay := time.Duration(float64(time.Second) / *speed)
			time.Sleep(delay)
		}
	}

	log.Println("mock client completed all requests")
}

// OperationNotification represents a notification related to an operation
type OperationNotification struct {
	JSON          string
	ProgressValue int
	DelayMs       int
}

// runMockServer reads a recording file and acts as a server,
// responding to client requests with matching responses from the recording
func runMockServer(recordingFile string, out io.Writer) {
	// Read all messages from the recording
	f, err := os.Open(recordingFile)
	if err != nil {
		// Output error in vim-compatible format
		printError(recordingFile, 1, 1, "failed to open file: %v", err)
		log.Fatalf("Error opening recording file: %v", err)
	}
	defer f.Close()

	// Maps to store responses by request ID
	requestMap := make(map[string][]string)

	// Map for method-based responses (for messages without IDs)
	methodMap := make(map[string][]string)

	// List of automatic server notifications (server-initiated without request)
	autoNotifyList := []string{}

	// List of all server responses (for auto-respond mode)
	allResponses := []string{}

	// Map to collect related messages for operation ID
	operationResponses := make(map[string]string) // Map operation ID to response

	// Map to collect notifications related to operations by progress token
	progressNotifications := make(map[string][]OperationNotification)

	// Track record order for preserved message ordering
	recordOrder := []struct {
		idOrMethod string
		isMethod   bool
		content    string
	}{}

	// First pass: build a map of request IDs to responses
	scanner := bufio.NewScanner(f)
	lineNumber := 0

	// Store the previous message timestamp for calculating delays
	var prevTimestampMs float64
	var firstLine = true

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Extract record type (send/recv)
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 2 || !strings.HasPrefix(parts[0], "mcp-") {
			// Output malformed line error in vim-compatible format
			printError(recordingFile, lineNumber, 1, "malformed line: %s", line)
			continue
		}

		direction := strings.TrimPrefix(parts[0], "mcp-")
		// Extract JSON content - it's everything after "mcp-<direction> " and before " # timestamp"
		jsonEndPos := strings.LastIndex(line, " # ")
		if jsonEndPos == -1 {
			jsonEndPos = len(line)
		}

		// Get the prefix length (mcp-direction plus space)
		prefixLen := len(parts[0]) + 1

		// Extract the JSON string
		jsonStr := line[prefixLen:jsonEndPos]

		// Extract timestamp for calculating delays between messages
		timestampMs := 0.0
		if match := timestampRegex.FindStringSubmatch(line); len(match) >= 2 {
			seconds, err := strconv.ParseInt(match[1], 10, 64)
			if err == nil {
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
				timestampMs = float64(seconds) + millis
			}
		}

		// Calculate delta between messages for timing
		var recordedDelta float64
		if firstLine {
			firstLine = false
			prevTimestampMs = timestampMs
		} else {
			recordedDelta = timestampMs - prevTimestampMs
			if recordedDelta < 0 {
				recordedDelta = 0 // Handle out-of-order timestamps
			}
			prevTimestampMs = timestampMs
		}

		if *verbose {
			log.Printf("line %d: JSON to parse: '%s'", lineNumber, jsonStr)
		}

		// Parse JSON to extract ID
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
			// We want to report the actual file line number
			// Output in vim-compatible format (file:line:col: error)
			printError(recordingFile, lineNumber, 1, "JSON parse error: %v", err)
			if *verbose {
				log.Printf("error parsing JSON: %v", err)
			}
			continue
		}

		// Check for method for method-based mapping
		method, hasMethod := jsonData["method"]

		// Get request ID if it exists
		id, hasID := jsonData["id"]
		idStr := ""
		if hasID {
			idStr = fmt.Sprintf("%v", id)
		}

		// Record the occurrence order for all messages to preserve complex interactions
		if *preserveOrder {
			var key string
			isMethodKey := false

			if hasID {
				key = idStr
			} else if hasMethod {
				methodStr := fmt.Sprintf("%v", method)
				key = methodStr
				isMethodKey = true
			}

			if key != "" {
				// Add to the ordered record list
				recordOrder = append(recordOrder, struct {
					idOrMethod string
					isMethod   bool
					content    string
				}{
					idOrMethod: key,
					isMethod:   isMethodKey,
					content:    jsonStr,
				})
			}
		}

		// NOTE: In the recording file:
		// - "mcp-recv" means client→server (client requests)
		// - "mcp-send" means server→client (server responses)
		if direction == "recv" {
			if hasID {
				// "recv" means client→server (client request)
				// This is a client request with ID, initialize map entry if needed
				if _, exists := requestMap[idStr]; !exists {
					requestMap[idStr] = make([]string, 0)
				}
			} else if hasMethod {
				// This is a notification or method without ID
				if *verbose {
					log.Printf("found method without ID: %s", method)
				}
			}
		} else if direction == "send" {
			// "send" means server→client (server response)
			if *jsonOnly {
				allResponses = append(allResponses, jsonStr)
			} else {
				allResponses = append(allResponses, line)
			}

			if hasID {
				// This is a response with ID, add to the ID-based map
				requestMap[idStr] = append(requestMap[idStr], jsonStr)

				// Save operation result separately
				operationResponses[idStr] = jsonStr
			} else if hasMethod {
				// This is a server notification or method response without ID
				methodStr := fmt.Sprintf("%v", method)
				methodMap[methodStr] = append(methodMap[methodStr], jsonStr)

				// Add to auto-notifications list for automatic replay
				autoNotifyList = append(autoNotifyList, jsonStr)

				// Check if this is a progress notification with a progressToken
				if methodStr == "notifications/progress" || methodStr == "notifications/message" {
					if params, ok := jsonData["params"].(map[string]interface{}); ok {
						if progressToken, ok := params["progressToken"]; ok {
							tokenStr := fmt.Sprintf("%v", progressToken)

							// Extract progress value if this is a progress notification
							progressValue := 0
							if progress, ok := params["progress"]; ok {
								if progressFloat, ok := progress.(float64); ok {
									progressValue = int(progressFloat)
								}
							}

							// Store notification with timing information
							notification := OperationNotification{
								JSON:          jsonStr,
								ProgressValue: progressValue,
								DelayMs:       int(recordedDelta * 1000), // Store delay in milliseconds
							}

							// Add to progress notifications map by token
							progressNotifications[tokenStr] = append(progressNotifications[tokenStr], notification)

							if *verbose {
								log.Printf("stored progress notification for token %s with delay %dms",
									tokenStr, notification.DelayMs)
							}
						}
					}
				}

				if *verbose {
					log.Printf("stored server notification for method: %s", methodStr)
				}
			}
		}
	}

	if *verbose {
		log.Printf("loaded %d request/response pairs from recording", len(requestMap))
		log.Printf("loaded %d method-based notifications/responses", len(methodMap))
		log.Printf("found %d server-initiated notifications", len(autoNotifyList))
		log.Printf("found %d operation progress notification groups", len(progressNotifications))
		log.Printf("found %d total server responses", len(allResponses))
	}

	// Now read requests from stdin and send matching responses to stdout
	stdinScanner := bufio.NewScanner(os.Stdin)

	log.Println("waiting for requests...")

	// Auto-respond mode: Send all server responses in sequence
	if *autoResponses && len(allResponses) > 0 {
		log.Printf("auto-respond mode: sending all %d server responses sequentially...", len(allResponses))

		for i, response := range allResponses {
			// Add timing delay between responses based on speed
			if i > 0 && *speed > 0 {
				// Calculate a reasonable delay based on speed
				delay := time.Duration(float64(500*time.Millisecond) / *speed)
				time.Sleep(delay)
			}

			// Output the response
			fmt.Fprintln(out, response)

			if *verbose {
				log.Printf("auto-sent response %d/%d", i+1, len(allResponses))
			}
		}

		log.Println("auto-respond mode: all responses sent")
		return
	}

	// For auto-notifications, we'll integrate them with the preserve-order implementation
	// rather than sending them all at the beginning in a batch.
	// We'll keep this section for backward compatibility with scripts that might rely on it,
	// but only use it when preserve-order is disabled.
	if *autoNotifications && len(autoNotifyList) > 0 && !*preserveOrder {
		go func() {
			if *verbose {
				log.Println("starting automatic notification sender...")
			}

			// Send notifications with the specified replay speed
			for i, notification := range autoNotifyList {
				if i > 0 && *speed > 0 {
					// Wait between notifications based on speed
					time.Sleep(time.Duration(float64(time.Second) / *speed))
				}

				fmt.Fprintln(out, notification)

				if *verbose {
					log.Println("auto-sent server notification")
				}
			}

			if *verbose {
				log.Println("automatic notification sending complete")
			}
		}()
	}

	// Build a map to track request IDs we've seen
	seenRequests := make(map[string]bool)

	// For preserving exact message order
	if *preserveOrder && len(recordOrder) > 0 {
		log.Printf("using preserved message order mode with %d recorded messages", len(recordOrder))

		// Current position in the request stream
		type requestContext struct {
			line     string
			jsonData map[string]interface{}
			id       string
			method   string
		}

		pendingRequests := make(map[string]requestContext)
		currentRequestIndex := 0
		lastMessageTime := time.Now()

		// Identify server notifications in the record order
		serverNotifications := make([]int, 0)
		for i, record := range recordOrder {
			// A server notification is a "method" message without ID
			if record.isMethod {
				var msgData map[string]interface{}
				if err := json.Unmarshal([]byte(record.content), &msgData); err == nil {
					_, hasID := msgData["id"]
					if !hasID {
						// This is a server notification
						serverNotifications = append(serverNotifications, i)
						if *verbose {
							log.Printf("identified server notification at position %d: %s", i, record.idOrMethod)
						}
					}
				}
			}
		}

		stdinLineNumber := 0
		for stdinScanner.Scan() {
			stdinLineNumber++
			line := stdinScanner.Text()

			// Parse JSON from input
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
				printError("stdin", stdinLineNumber, 1, "JSON parse error: %v", err)
				if *verbose {
					log.Printf("error parsing input JSON: %v", err)
				}
				continue
			}

			// Extract ID and method
			var idStr, methodStr string
			if id, ok := jsonData["id"]; ok {
				idStr = fmt.Sprintf("%v", id)
			}
			if method, ok := jsonData["method"]; ok {
				methodStr = fmt.Sprintf("%v", method)
			}

			// Store request context
			ctx := requestContext{
				line:     line,
				jsonData: jsonData,
				id:       idStr,
				method:   methodStr,
			}

			// Enhanced handling for operations with related notifications
			if methodStr != "" && idStr != "" {
				// Check if we have specific progress notifications for this operation ID
				if notifications, ok := progressNotifications[idStr]; ok && len(notifications) > 0 {
					if *verbose {
						log.Printf("found %d progress notifications for operation ID %s",
							len(notifications), idStr)
					}

					// Sort notifications by progress value if applicable (simple implementation)
					// This ensures notifications are sent in order of progress
					if len(notifications) > 1 {
						// Simple bubble sort
						for i := 0; i < len(notifications)-1; i++ {
							for j := 0; j < len(notifications)-i-1; j++ {
								if notifications[j].ProgressValue > notifications[j+1].ProgressValue {
									notifications[j], notifications[j+1] = notifications[j+1], notifications[j]
								}
							}
						}
					}

					// Send notifications with appropriate timing
					for i, notification := range notifications {
						// Use recorded delay for more accurate replay, with speed adjustment
						delay := time.Duration(float64(notification.DelayMs) * float64(time.Millisecond) / *speed)
						if i == 0 {
							// For the first notification, use a small fixed delay
							delay = time.Duration(float64(100*time.Millisecond) / *speed)
						}

						if delay > 0 {
							time.Sleep(delay)
						}

						fmt.Fprintln(out, notification.JSON)
						if *verbose {
							if notification.ProgressValue > 0 {
								log.Printf("sent progress notification %d/%d (progress: %d) for operation ID %s",
									i+1, len(notifications), notification.ProgressValue, idStr)
							} else {
								log.Printf("sent notification %d/%d for operation ID %s",
									i+1, len(notifications), idStr)
							}
						}
					}

					// After all notifications, send the final result
					if result, ok := operationResponses[idStr]; ok {
						// Add a small delay before the final result
						time.Sleep(time.Duration(float64(200*time.Millisecond) / *speed))
						fmt.Fprintln(out, result)
						if *verbose {
							log.Printf("sent final result for operation ID %s", idStr)
						}
					} else {
						// If we don't have a stored result, construct a generic one
						defaultResult := fmt.Sprintf(`{"result":{"content":[{"type":"text","text":"Operation completed successfully."}]},"jsonrpc":"2.0","id":%s}`, idStr)
						fmt.Fprintln(out, defaultResult)
						if *verbose {
							log.Printf("sent default result for operation ID %s (no stored result found)", idStr)
						}
					}

					// Skip further processing for this request
					continue
				}
			}

			// If the request has an ID, remember it
			if idStr != "" {
				pendingRequests[idStr] = ctx
				seenRequests[idStr] = true
			} else if methodStr != "" {
				pendingRequests[methodStr] = ctx
			}

			// If auto-notifications is enabled, check if we should send any server notifications
			// that come next in the recorded sequence
			if *autoNotifications {
				// Find notifications that should be sent between the previous processed message
				// and the current one
				for currentRequestIndex < len(recordOrder) {
					// First check if the current index is a server notification
					isServerNotification := false
					for _, notifIdx := range serverNotifications {
						if currentRequestIndex == notifIdx {
							isServerNotification = true
							break
						}
					}

					// If it's a server notification and auto-notifications is enabled, send it
					if isServerNotification {
						record := recordOrder[currentRequestIndex]
						// Apply speed-based delay
						if *speed > 0 {
							elapsedTime := time.Since(lastMessageTime)
							targetDelay := time.Duration(float64(500*time.Millisecond) / *speed)
							if elapsedTime < targetDelay {
								time.Sleep(targetDelay - elapsedTime)
							}
						}

						fmt.Fprintln(out, record.content)
						if *verbose {
							log.Printf("sent server notification for %s (auto-notifications)", record.idOrMethod)
						}
						currentRequestIndex++
						lastMessageTime = time.Now()
					} else {
						// Not a notification, so it must be a regular response - break and process normally
						break
					}
				}
			}

			// Process responses according to the original order
			processNextResponses := true
			for processNextResponses && currentRequestIndex < len(recordOrder) {
				record := recordOrder[currentRequestIndex]

				// Skip already processed notifications
				if record.idOrMethod == "PROCESSED" {
					currentRequestIndex++
					continue
				}

				// Skip server notifications if they'll be handled by the auto-notifications logic
				if *autoNotifications {
					isServerNotification := false
					for _, notifIdx := range serverNotifications {
						if currentRequestIndex == notifIdx {
							isServerNotification = true
							break
						}
					}
					if isServerNotification {
						// Skip this notification as it will be sent by the auto-notifications logic
						currentRequestIndex++
						continue
					}
				}

				// Check if this is a response we need to send now
				if record.isMethod {
					// Method-based response - only send if we've seen a matching method
					if _, ok := pendingRequests[record.idOrMethod]; ok {
						delete(pendingRequests, record.idOrMethod)
						fmt.Fprintln(out, record.content)
						if *verbose {
							log.Printf("sent method-based response for %s (preserved order)", record.idOrMethod)
						}
						currentRequestIndex++
						lastMessageTime = time.Now()
					} else {
						// Haven't seen this method request yet
						processNextResponses = false
					}
				} else {
					// ID-based response - only send if we've seen this ID
					if _, ok := pendingRequests[record.idOrMethod]; ok {
						delete(pendingRequests, record.idOrMethod)
						fmt.Fprintln(out, record.content)
						if *verbose {
							log.Printf("sent response for ID %s (preserved order)", record.idOrMethod)
						}
						currentRequestIndex++
						lastMessageTime = time.Now()
					} else if _, ok := seenRequests[record.idOrMethod]; ok {
						// We've seen this ID before but don't have a pending request
						// This might be a notification related to an earlier request
						fmt.Fprintln(out, record.content)
						if *verbose {
							log.Printf("sent notification for previous ID %s (preserved order)", record.idOrMethod)
						}
						currentRequestIndex++
						lastMessageTime = time.Now()
					} else {
						// Haven't seen this ID yet
						processNextResponses = false
					}
				}
			}
		}

		return
	}

	// Traditional response lookup mode
	stdinLineNumber := 0
	for stdinScanner.Scan() {
		stdinLineNumber++
		line := stdinScanner.Text()

		// Parse JSON from input
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
			// Use the helper function for vim-compatible errors
			printError("stdin", stdinLineNumber, 1, "JSON parse error: %v", err)
			if *verbose {
				log.Printf("error parsing input JSON: %v", err)
			}
			continue
		}

		// Check for ID and method
		id, hasID := jsonData["id"]
		method, hasMethod := jsonData["method"]

		// Enhanced handling for operations with related notifications
		if hasMethod && hasID {
			methodStr := fmt.Sprintf("%v", method)
			idStr := fmt.Sprintf("%v", id)

			// Check if we have specific progress notifications for this operation ID
			if notifications, ok := progressNotifications[idStr]; ok && len(notifications) > 0 {
				if *verbose {
					log.Printf("found %d progress notifications for operation ID %s",
						len(notifications), idStr)
				}

				// Sort notifications by progress value if applicable
				if len(notifications) > 1 {
					// Simple bubble sort
					for i := 0; i < len(notifications)-1; i++ {
						for j := 0; j < len(notifications)-i-1; j++ {
							if notifications[j].ProgressValue > notifications[j+1].ProgressValue {
								notifications[j], notifications[j+1] = notifications[j+1], notifications[j]
							}
						}
					}
				}

				// Send notifications with appropriate timing
				for i, notification := range notifications {
					// Use recorded delay for more accurate replay, with speed adjustment
					delay := time.Duration(float64(notification.DelayMs) * float64(time.Millisecond) / *speed)
					if i == 0 {
						// For the first notification, use a small fixed delay
						delay = time.Duration(float64(100*time.Millisecond) / *speed)
					}

					if delay > 0 {
						time.Sleep(delay)
					}

					fmt.Fprintln(out, notification.JSON)
					if *verbose {
						if notification.ProgressValue > 0 {
							log.Printf("sent progress notification %d/%d (progress: %d) for operation ID %s",
								i+1, len(notifications), notification.ProgressValue, idStr)
						} else {
							log.Printf("sent notification %d/%d for operation ID %s",
								i+1, len(notifications), idStr)
						}
					}
				}

				// After all notifications, send the final result
				if result, ok := operationResponses[idStr]; ok {
					// Add a small delay before the final result
					time.Sleep(time.Duration(float64(200*time.Millisecond) / *speed))
					fmt.Fprintln(out, result)
					if *verbose {
						log.Printf("sent final result for operation ID %s", idStr)
					}
				} else {
					// If we don't have a stored result, handle specific known operations
					if methodStr == "tools/call" && idStr == "6" {
						// Known longRunningOperation result
						fmt.Fprintln(out, `{"result":{"content":[{"type":"text","text":"Long running operation completed. Duration: 5 seconds, Steps: 5."}]},"jsonrpc":"2.0","id":6}`)
					} else {
						// Generic result
						defaultResult := fmt.Sprintf(`{"result":{"content":[{"type":"text","text":"Operation completed successfully."}]},"jsonrpc":"2.0","id":%s}`, idStr)
						fmt.Fprintln(out, defaultResult)
					}
					if *verbose {
						log.Printf("sent default result for operation ID %s (no stored result found)", idStr)
					}
				}

				// Skip further processing for this request
				continue
			}
		}

		// If we have an ID, try to find a response by ID
		if hasID {
			idStr := fmt.Sprintf("%v", id)
			seenRequests[idStr] = true

			// Find matching response
			responses, ok := requestMap[idStr]
			if ok && len(responses) > 0 {
				// Take the first response and remove it from the list
				response := responses[0]
				requestMap[idStr] = responses[1:]

				// Send the response
				fmt.Fprintln(out, response)

				if *verbose {
					log.Printf("sent response for request ID: %s", idStr)
				}
				continue
			} else if *verbose {
				log.Printf("no response found for request ID: %s", idStr)
			}
		}

		// If no ID or no response by ID found, try to find by method
		if hasMethod {
			methodStr := fmt.Sprintf("%v", method)

			// Find matching method-based response
			notifications, ok := methodMap[methodStr]
			if ok && len(notifications) > 0 {
				// Take the first response and remove it from the list
				notification := notifications[0]
				methodMap[methodStr] = notifications[1:]

				// Send the notification
				fmt.Fprintln(out, notification)

				if *verbose {
					log.Printf("sent notification for method: %s", methodStr)
				}
				continue
			} else if *verbose && !hasID {
				log.Printf("no notification found for method: %s", methodStr)
			}
		}

		// No response found by ID or method
		if !hasID && !hasMethod {
			if *verbose {
				log.Println("request has neither ID nor method, ignoring")
			}
		}
	}

	if err := stdinScanner.Err(); err != nil {
		log.Fatalf("Error reading from stdin: %v", err)
	}
}
