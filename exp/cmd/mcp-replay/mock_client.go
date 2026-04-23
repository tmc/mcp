package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MCPRequest represents a request from a recording file
type MCPRequest struct {
	Content   string  // JSON content
	Timestamp float64 // Timestamp in seconds with millisecond precision
	LineNum   int     // Line number in the recording file
	ID        string  // Request ID, if any
	Method    string  // Request method, if any
}

// MCPResponse represents a response from a recording file
type MCPResponse struct {
	Content   string  // JSON content
	Timestamp float64 // Timestamp in seconds with millisecond precision
	LineNum   int     // Line number in the recording file
	ID        string  // Response ID, if any
	Method    string  // Response method, if notification
}

// MCPRecording represents a parsed MCP recording file
type MCPRecording struct {
	Requests              []MCPRequest                       // All client requests in order
	Responses             []MCPResponse                      // All server responses in order
	ShadowResponses       []MCPResponse                      // All shadow server responses in order
	RequestMap            map[string][]string                // Map from request ID to expected responses
	ShadowRequestMap      map[string][]string                // Map from request ID to shadow responses
	MethodMap             map[string][]string                // Map from method name to responses
	ProgressNotifications map[string][]OperationNotification // Map from token to progress notifications
	AutoNotifications     []string                           // Server notifications
	FileName              string                             // Recording file name
}

// OperationNotificationInfo represents a notification related to an operation in client code
// This is a local type that uses the OperationNotification from main.go
type OperationNotificationInfo OperationNotification

// ParseRecording reads and parses an MCP recording file
func ParseRecording(recordingFile string) (*MCPRecording, error) {
	// Read all messages from the recording
	f, err := os.Open(recordingFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	recording := &MCPRecording{
		Requests:              make([]MCPRequest, 0),
		Responses:             make([]MCPResponse, 0),
		ShadowResponses:       make([]MCPResponse, 0),
		RequestMap:            make(map[string][]string),
		ShadowRequestMap:      make(map[string][]string),
		MethodMap:             make(map[string][]string),
		ProgressNotifications: make(map[string][]OperationNotification),
		AutoNotifications:     make([]string, 0),
		FileName:              recordingFile,
	}

	// First pass: build maps of requests and responses
	scanner := bufio.NewScanner(f)
	lineNumber := 0
	directionRegex := regexp.MustCompile(`^mcp-([\w-]+)\s+`)

	// Store the previous message timestamp for calculating delays
	var prevTimestampMs float64
	var firstLine = true

	// Check for the header line and skip it if present
	if scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// If this is a header line, process it and move on
		if strings.HasPrefix(line, "# mcptrace:") {
			if *verbose {
				log.Printf("Found header: %s", line)
			}
			// Header found, continue to next line
		} else {
			// Not a header, process as a normal line
			// Extract direction (send/recv)
			dirMatch := directionRegex.FindStringSubmatch(line)
			if len(dirMatch) >= 2 {
				direction := dirMatch[1]

				// Extract JSON content
				parts := strings.SplitN(line, " ", 3)
				if len(parts) >= 2 {
					// Extract JSON content - everything after "mcp-<direction> " and before " # timestamp"
					jsonEndPos := strings.LastIndex(line, " # ")
					if jsonEndPos == -1 {
						jsonEndPos = len(line)
					}

					// Get the prefix length (mcp-direction plus space)
					prefixLen := len(parts[0]) + 1

					// Extract the JSON string
					if prefixLen < jsonEndPos {
						jsonStr := line[prefixLen:jsonEndPos]

						// Extract timestamp for timing
						timestamp := 0.0
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
								timestamp = float64(seconds) + millis
							}
						}

						// Calculate delta between messages for timing
						if firstLine {
							firstLine = false
							prevTimestampMs = timestamp
						}

						// Process the line based on direction
						processLineContent(jsonStr, timestamp, lineNumber, direction, recording)
					}
				}
			}
		}
	}

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Extract direction (send/recv)
		dirMatch := directionRegex.FindStringSubmatch(line)
		if len(dirMatch) < 2 {
			continue
		}

		direction := dirMatch[1]

		// Extract JSON content
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 2 {
			continue
		}

		// Extract JSON content - everything after "mcp-<direction> " and before " # timestamp"
		jsonEndPos := strings.LastIndex(line, " # ")
		if jsonEndPos == -1 {
			jsonEndPos = len(line)
		}

		// Get the prefix length (mcp-direction plus space)
		prefixLen := len(parts[0]) + 1

		// Extract the JSON string
		if prefixLen >= jsonEndPos {
			// Invalid format, skip this line
			continue
		}
		jsonStr := line[prefixLen:jsonEndPos]

		// Extract timestamp for timing
		timestamp := 0.0
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
				timestamp = float64(seconds) + millis
			}
		}

		// Calculate delta between messages for timing
		var recordedDelta float64
		if firstLine {
			firstLine = false
			prevTimestampMs = timestamp
		} else {
			recordedDelta = timestamp - prevTimestampMs
			if recordedDelta < 0 {
				recordedDelta = 0 // Handle out-of-order timestamps
			}
			prevTimestampMs = timestamp
		}

		// Parse JSON data
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
			if *verbose {
				log.Printf("error parsing JSON: %v", err)
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

		// Handle based on direction
		// NOTE: In the recording file:
		// - "mcp-recv" means client→server (client requests)
		// - "mcp-send" means server→client (server responses)
		// - "mcp-send-shadow" means shadow server→client (shadow responses)
		if direction == "recv" {
			// This is a client request
			req := MCPRequest{
				Content:   jsonStr,
				Timestamp: timestamp,
				LineNum:   lineNumber,
				ID:        idStr,
				Method:    methodStr,
			}
			recording.Requests = append(recording.Requests, req)

			// Initialize the map entry for this ID
			if idStr != "" {
				if _, exists := recording.RequestMap[idStr]; !exists {
					recording.RequestMap[idStr] = make([]string, 0)
				}
				// Also initialize shadow map entry
				if _, exists := recording.ShadowRequestMap[idStr]; !exists {
					recording.ShadowRequestMap[idStr] = make([]string, 0)
				}
			}
		} else if direction == "send" {
			// This is a server response
			resp := MCPResponse{
				Content:   jsonStr,
				Timestamp: timestamp,
				LineNum:   lineNumber,
				ID:        idStr,
				Method:    methodStr,
			}
			recording.Responses = append(recording.Responses, resp)

			if idStr != "" {
				// Response with ID - add to map
				recording.RequestMap[idStr] = append(recording.RequestMap[idStr], jsonStr)
			} else if methodStr != "" {
				// Server notification
				recording.MethodMap[methodStr] = append(recording.MethodMap[methodStr], jsonStr)
				recording.AutoNotifications = append(recording.AutoNotifications, jsonStr)

				// Check if this is a progress notification
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
							recording.ProgressNotifications[tokenStr] = append(
								recording.ProgressNotifications[tokenStr], notification)
						}
					}
				}
			}
		} else if direction == "send-shadow" {
			// This is a shadow server response
			resp := MCPResponse{
				Content:   jsonStr,
				Timestamp: timestamp,
				LineNum:   lineNumber,
				ID:        idStr,
				Method:    methodStr,
			}
			recording.ShadowResponses = append(recording.ShadowResponses, resp)

			if idStr != "" {
				// Shadow response with ID - add to shadow map
				recording.ShadowRequestMap[idStr] = append(recording.ShadowRequestMap[idStr], jsonStr)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	return recording, nil
}

// processLineContent handles the processing of a parsed line based on its direction
func processLineContent(jsonStr string, timestamp float64, lineNumber int, direction string, recording *MCPRecording) {
	// Parse JSON data
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		if *verbose {
			log.Printf("error parsing JSON: %v", err)
		}
		return
	}

	// Extract ID and method
	var idStr, methodStr string
	if id, ok := jsonData["id"]; ok {
		idStr = fmt.Sprintf("%v", id)
	}
	if method, ok := jsonData["method"]; ok {
		methodStr = fmt.Sprintf("%v", method)
	}

	// Handle based on direction
	// NOTE: In the recording file:
	// - "mcp-recv" means client→server (client requests)
	// - "mcp-send" means server→client (server responses)
	if direction == "recv" {
		// This is a client request
		req := MCPRequest{
			Content:   jsonStr,
			Timestamp: timestamp,
			LineNum:   lineNumber,
			ID:        idStr,
			Method:    methodStr,
		}
		recording.Requests = append(recording.Requests, req)

		// Initialize the map entry for this ID
		if idStr != "" {
			if _, exists := recording.RequestMap[idStr]; !exists {
				recording.RequestMap[idStr] = make([]string, 0)
			}
		}
	} else if direction == "send" {
		// This is a server response
		resp := MCPResponse{
			Content:   jsonStr,
			Timestamp: timestamp,
			LineNum:   lineNumber,
			ID:        idStr,
			Method:    methodStr,
		}
		recording.Responses = append(recording.Responses, resp)

		if idStr != "" {
			// Response with ID - add to map
			recording.RequestMap[idStr] = append(recording.RequestMap[idStr], jsonStr)
		} else if methodStr != "" {
			// Server notification
			recording.MethodMap[methodStr] = append(recording.MethodMap[methodStr], jsonStr)
			recording.AutoNotifications = append(recording.AutoNotifications, jsonStr)

			// Check if this is a progress notification
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
							DelayMs:       0, // We don't have delay information here
						}

						// Add to progress notifications map by token
						recording.ProgressNotifications[tokenStr] = append(
							recording.ProgressNotifications[tokenStr], notification)
					}
				}
			}
		}
	}
}

// runMockClient reads a recording file and acts as a client,
// sending requests from the recording and waiting for responses
func runMockClient(recordingFile string, out io.Writer, traceWriter io.Writer) {
	recording, err := ParseRecording(recordingFile)
	if err != nil {
		printError(recordingFile, 1, 1, "failed to parse recording: %v", err)
		log.Fatalf("Error parsing recording file: %v", err)
	}

	if *verbose {
		log.Printf("loaded %d requests from recording", len(recording.Requests))
		log.Printf("found %d responses from recording", len(recording.Responses))
		log.Printf("found %d response mappings", len(recording.RequestMap))
	}

	// For complete tracing, write both requests and responses to trace file immediately
	if traceWriter != nil {
		// Add header to trace file
		fmt.Fprintf(traceWriter, "# mcptrace: source=mock-client created=%d\n", time.Now().Unix())

		// First write all server responses (mcp-send) to the trace file to ensure
		// the client trace has both directions of communication
		for _, resp := range recording.Responses {
			// Generate timestamp for the traced message
			timestamp := time.Now().UnixNano() / 1000000 // Convert to milliseconds
			timestampStr := fmt.Sprintf("%d.%03d", timestamp/1000, timestamp%1000)

			// Format trace message with mcp-send prefix and timestamp
			traceMsg := fmt.Sprintf("mcp-send %s # %s", resp.Content, timestampStr)
			fmt.Fprintln(traceWriter, traceMsg)

			// Flush trace file to ensure immediate write
			if flusher, ok := traceWriter.(interface{ Flush() error }); ok {
				flusher.Flush()
			}

			// Add small delay between messages to avoid identical timestamps
			time.Sleep(1 * time.Millisecond)
		}

		if *verbose {
			log.Printf("Added %d server responses (mcp-send) to trace file", len(recording.Responses))
		}
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

	// Now send requests without waiting for responses (non-blocking)
	var lastReplayTime = time.Now()
	var prevRequestTime float64

	// Use a map to track which request IDs we're waiting for responses on
	pendingResponses := make(map[string]bool)

	// Start a goroutine to collect responses
	responseDone := make(chan struct{})
	go processMockClientResponses(responseChannel, out, pendingResponses, traceWriter, responseDone)

	// Send all requests according to their timing in the recording
	sendMockClientRequests(recording, out, lastReplayTime, prevRequestTime, pendingResponses, traceWriter)

	// Wait for any remaining responses to arrive, with a reasonable timeout
	finalTimeoutDuration := *mockTimeout
	select {
	case <-responseDone:
		// All responses received
	case <-time.After(finalTimeoutDuration):
		if *verbose && len(pendingResponses) > 0 {
			log.Printf("final timeout reached with %d pending responses", len(pendingResponses))
		}
	}

	log.Println("mock client completed all requests")
}

// processMockClientResponses processes responses received from the server
func processMockClientResponses(responseChannel <-chan string, out io.Writer, pendingResponses map[string]bool, traceWriter io.Writer, done chan<- struct{}) {
	defer close(done)
	for {
		select {
		case response, ok := <-responseChannel:
			if !ok {
				log.Println("response channel closed")
				return // Channel was closed, exit goroutine
			}

			if *verbose {
				log.Printf("received response: %s", response)
			}

			// Try to extract information from the response
			var jsonData map[string]interface{}
			var shouldForward = true // Default to forwarding all responses
			var isNotification = false
			if err := json.Unmarshal([]byte(response), &jsonData); err == nil {
				if id, ok := jsonData["id"]; ok {
					responseID := fmt.Sprintf("%v", id)
					// Mark the response as received
					delete(pendingResponses, responseID)

					if *verbose {
						log.Printf("matched response for request ID: %s", responseID)
					}
				} else if method, ok := jsonData["method"]; ok {
					// This is a server notification (no ID but has method)
					isNotification = true
					if *verbose {
						log.Printf("received server notification with method: %v", method)
					}
				}
			} else {
				if *verbose {
					log.Printf("error parsing response JSON: %v", err)
				}
				shouldForward = false
			}

			// Forward valid responses and notifications to out writer for capture by mcpspy
			if shouldForward {
				// Generate timestamp for the message
				timestamp := time.Now().UnixNano() / 1000000 // Convert to milliseconds
				timestampStr := fmt.Sprintf("%d.%03d", timestamp/1000, timestamp%1000)

				// Format the response correctly based on the content
				var formattedResponse string
				if *jsonOnly {
					formattedResponse = response
				} else {
					// If it's a notification or response, format it with mcp-send prefix
					// Since these are messages coming from the server to the client
					formattedResponse = fmt.Sprintf("mcp-send %s # %s", response, timestampStr)

					// Log extra information for notifications to help with debugging
					if isNotification && *verbose {
						log.Printf("forwarding server notification: %s", formattedResponse)
					}
				}
				fmt.Fprintln(out, formattedResponse)

				// Write to trace file if specified
				if traceWriter != nil {
					// Always write in trace format (mcp-send prefix with timestamp)
					// regardless of jsonOnly setting
					traceMsg := fmt.Sprintf("mcp-send %s # %s", response, timestampStr)
					fmt.Fprintln(traceWriter, traceMsg)

					// Flush trace file to ensure immediate write
					if flusher, ok := traceWriter.(interface{ Flush() error }); ok {
						flusher.Flush()
					}
				}

				// Ensure we flush the output to avoid buffering issues, especially for notifications
				if flusher, ok := out.(interface{ Flush() error }); ok {
					flusher.Flush()
				}
			}

		case <-time.After(*mockTimeout):
			log.Println("timeout waiting for response")
			// If no response received in timeout period and we're not waiting for any more responses, exit
			if len(pendingResponses) == 0 {
				return
			}
		}
	}
}

// sendMockClientRequests sends requests from the recording file to the server
func sendMockClientRequests(recording *MCPRecording, out io.Writer, lastReplayTime time.Time, prevRequestTime float64, pendingResponses map[string]bool, traceWriter io.Writer) {
	for i, req := range recording.Requests {
		request := req.Content
		timestamp := req.Timestamp

		// Calculate timing delay based on recorded timestamps
		if i > 0 {
			// Calculate the recorded time delta between this request and the previous one
			recordedDelta := timestamp - prevRequestTime
			if recordedDelta < 0 {
				// Handle out-of-order timestamps
				if *verbose {
					log.Printf("warning: timestamps out of order (%.3f before %.3f)",
						timestamp, prevRequestTime)
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
		}

		// Update previous timestamp for next iteration
		prevRequestTime = timestamp
		lastReplayTime = time.Now()

		// Generate current timestamp for message
		now := time.Now().UnixNano() / 1000000 // milliseconds
		currentTimestamp := fmt.Sprintf("%d.%03d", now/1000, now%1000)

		// Format the request depending on mode
		requestContent := request
		if !*jsonOnly {
			requestContent = fmt.Sprintf("mcp-recv %s # %s", request, currentTimestamp)
		}

		// Check if we expect a response for this request
		if req.ID != "" && recording.RequestMap[req.ID] != nil && len(recording.RequestMap[req.ID]) > 0 {
			pendingResponses[req.ID] = true
		}

		// Send the request
		fmt.Fprintln(out, requestContent)

		// Write to trace file if specified
		if traceWriter != nil {
			// Always write in trace format regardless of jsonOnly setting
			traceMsg := fmt.Sprintf("mcp-recv %s # %s", request, currentTimestamp)
			fmt.Fprintln(traceWriter, traceMsg)

			// Flush trace file to ensure immediate write
			if flusher, ok := traceWriter.(interface{ Flush() error }); ok {
				flusher.Flush()
			}
		}

		if *verbose {
			if req.ID != "" {
				log.Printf("sent request with ID: %s", req.ID)
			} else {
				log.Println("sent request (no ID)")
			}
		}
	}
}
