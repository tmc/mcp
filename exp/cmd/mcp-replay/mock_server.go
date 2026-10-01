package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// RequestContext represents a pending client request
type RequestContext struct {
	Line     string                 // Original line from client
	JsonData map[string]interface{} // Parsed JSON data
	ID       string                 // Request ID
	Method   string                 // Request method
}

// RecordOrderEntry represents an entry in the recorded message order
type RecordOrderEntry struct {
	IDOrMethod string // ID or method name
	IsMethod   bool   // Whether this is a method-based entry
	Content    string // JSON content
}

// runMockServer reads a recording file and acts as a server,
// responding to client requests with matching responses from the recording
func runMockServer(recordingFile string, out io.Writer, traceWriter io.Writer) {
	recording, err := ParseRecording(recordingFile)
	if err != nil {
		printError(recordingFile, 1, 1, "failed to parse recording: %v", err)
		log.Fatalf("Error parsing recording file: %v", err)
	}

	if *verbose {
		log.Printf("loaded %d request/response pairs", len(recording.RequestMap))
		log.Printf("loaded %d method-based notifications/responses", len(recording.MethodMap))
		log.Printf("found %d server-initiated notifications", len(recording.AutoNotifications))
		log.Printf("found %d operation progress notification groups", len(recording.ProgressNotifications))
		log.Printf("found %d total server responses", len(recording.Responses))
		if *useShadow {
			log.Printf("found %d shadow server responses", len(recording.ShadowResponses))
			log.Printf("using shadow responses for mock server")
		}
	}

	// For complete tracing, write both requests and responses to trace file immediately
	if traceWriter != nil {
		// Add header to trace file
		fmt.Fprintf(traceWriter, "# mcptrace: source=mock-server created=%d\n", time.Now().Unix())

		// First write all client requests (mcp-recv) to the trace file to ensure
		// the server trace has both directions of communication
		for _, req := range recording.Requests {
			// Generate timestamp for the traced message
			timestamp := time.Now().UnixNano() / 1000000 // Convert to milliseconds
			timestampStr := fmt.Sprintf("%d.%03d", timestamp/1000, timestamp%1000)

			// Format trace message with mcp-recv prefix and timestamp
			traceMsg := fmt.Sprintf("mcp-recv %s # %s", req.Content, timestampStr)
			fmt.Fprintln(traceWriter, traceMsg)

			// Flush trace file to ensure immediate write
			if flusher, ok := traceWriter.(interface{ Flush() error }); ok {
				flusher.Flush()
			}

			// Add small delay between messages to avoid identical timestamps
			time.Sleep(1 * time.Millisecond)
		}

		if *verbose {
			log.Printf("Added %d client requests (mcp-recv) to trace file", len(recording.Requests))
		}
	}

	// Auto-respond mode: send all server responses in sequence
	if *autoResponses && len(recording.Responses) > 0 {
		handleAutoResponses(recording, out, traceWriter)
		return
	}

	// For ordered message replay
	recordOrder := buildMessageOrder(recording)
	if *preserveOrder && len(recordOrder) > 0 {
		handleOrderedResponses(recording, recordOrder, out, traceWriter)
		return
	}

	// If we get here, use auto-respond mode as a fallback
	log.Printf("No ordered messages found or preserve-order is disabled, falling back to auto-respond mode")
	handleAutoResponses(recording, out, traceWriter)
}

// handleAutoResponses sends all responses in sequence without waiting for requests
func handleAutoResponses(recording *MCPRecording, out io.Writer, traceWriter io.Writer) {
	responses := recording.Responses
	if *useShadow && len(recording.ShadowResponses) > 0 {
		responses = recording.ShadowResponses
	}
	log.Printf("auto-respond mode: sending all %d server responses sequentially...", len(responses))

	// Start a goroutine to read client requests even if we're not waiting for them
	// This ensures we properly trace incoming client requests
	if traceWriter != nil {
		go func() {
			// Set up input scanner
			stdinScanner := bufio.NewScanner(os.Stdin)
			for stdinScanner.Scan() {
				line := stdinScanner.Text()

				// Generate timestamp for the request
				timestamp := time.Now().UnixNano() / 1000000 // milliseconds
				timestampStr := fmt.Sprintf("%d.%03d", timestamp/1000, timestamp%1000)

				// Always write client requests to trace file with mcp-recv prefix
				traceMsg := fmt.Sprintf("mcp-recv %s # %s", line, timestampStr)
				fmt.Fprintln(traceWriter, traceMsg)

				// Flush trace file to ensure immediate write
				if flusher, ok := traceWriter.(interface{ Flush() error }); ok {
					flusher.Flush()
				}

				if *verbose {
					log.Printf("traced client request: %s", line)
				}
			}
		}()
	}

	for i, resp := range responses {
		// Add timing delay between responses based on speed
		if i > 0 && *speed > 0 {
			// Calculate a reasonable delay based on speed
			delay := time.Duration(float64(500*time.Millisecond) / *speed)
			time.Sleep(delay)
		}

		// Generate current timestamp for the message
		timestamp := time.Now().UnixNano() / 1000000 // milliseconds
		timestampStr := fmt.Sprintf("%d.%03d", timestamp/1000, timestamp%1000)

		// Format output based on mode
		var response string
		if *jsonOnly {
			response = resp.Content
		} else {
			direction := "mcp-send"
			if *useShadow {
				direction = "mcp-send-shadow"
			}
			response = fmt.Sprintf("%s %s # %s", direction, resp.Content, timestampStr)
		}

		// Output the response
		fmt.Fprintln(out, response)

		// Write to trace file if specified
		if traceWriter != nil {
			// Always write in trace format regardless of jsonOnly setting
			direction := "mcp-send"
			if *useShadow {
				direction = "mcp-send-shadow"
			}
			traceMsg := fmt.Sprintf("%s %s # %s", direction, resp.Content, timestampStr)
			fmt.Fprintln(traceWriter, traceMsg)

			// Flush trace file to ensure immediate write
			if flusher, ok := traceWriter.(interface{ Flush() error }); ok {
				flusher.Flush()
			}
		}

		if *verbose {
			log.Printf("auto-sent response %d/%d", i+1, len(responses))
		}
	}

	log.Println("auto-respond mode: all responses sent")
}

// buildMessageOrder constructs an ordered list of messages from the recording
func buildMessageOrder(recording *MCPRecording) []RecordOrderEntry {
	var recordOrder []RecordOrderEntry

	// Select which responses to use based on shadow mode
	responses := recording.Responses
	if *useShadow && len(recording.ShadowResponses) > 0 {
		responses = recording.ShadowResponses
	}

	// Only include responses - we don't want to echo back requests
	// Preserve the original order of responses including notifications
	for i, resp := range responses {
		if *verbose {
			log.Printf("Processing response %d: ID=%s, Method=%s", i, resp.ID, resp.Method)
		}

		if resp.ID != "" {
			recordOrder = append(recordOrder, RecordOrderEntry{
				IDOrMethod: resp.ID,
				IsMethod:   false,
				Content:    resp.Content,
			})
		} else if resp.Method != "" {
			recordOrder = append(recordOrder, RecordOrderEntry{
				IDOrMethod: resp.Method,
				IsMethod:   true,
				Content:    resp.Content,
			})
		}
	}

	return recordOrder
}

// identifyServerNotifications finds indexes of server notifications in the record order
func identifyServerNotifications(recordOrder []RecordOrderEntry) []int {
	notifications := make([]int, 0)

	for i, record := range recordOrder {
		// Skip records with IDs since they are likely responses to requests
		if !record.IsMethod {
			continue
		}

		// Try to parse the JSON to verify it's a notification
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(record.Content), &jsonData); err == nil {
			_, hasID := jsonData["id"]
			_, hasMethod := jsonData["method"]

			// It's a notification if it has a method but no ID
			if !hasID && hasMethod {
				notifications = append(notifications, i)
			}
		}
	}

	return notifications
}

// handleOrderedResponses handles requests and responses in the order they appear in the recording
func handleOrderedResponses(recording *MCPRecording, recordOrder []RecordOrderEntry, out io.Writer, traceWriter io.Writer) {
	log.Printf("using preserved message order mode with %d recorded messages", len(recordOrder))

	// Identify server notifications in the record order
	serverNotifications := identifyServerNotifications(recordOrder)
	if *verbose {
		log.Printf("identified %d server notifications in the record order", len(serverNotifications))
	}

	// Read client requests
	pendingRequests := make(map[string]RequestContext)
	seenRequests := make(map[string]bool)
	currentRequestIndex := 0

	// Set up input scanner
	stdinScanner := bufio.NewScanner(os.Stdin)
	stdinLineNumber := 0

	for stdinScanner.Scan() {
		stdinLineNumber++
		line := stdinScanner.Text()
		log.Println("Received line:", line, "Line number:", stdinLineNumber, "Current index:", currentRequestIndex, "Pending requests:", len(pendingRequests), "Seen requests:", len(seenRequests))

		// Parse JSON from input
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
			printError("stdin", stdinLineNumber, 1, "JSON parse error: %v", err)
			if *verbose {
				log.Printf("error parsing input JSON: %v", err)
			}
			continue
		}

		// Write request to trace file if specified
		if traceWriter != nil {
			timestamp := time.Now().UnixNano() / 1000000 // milliseconds
			timestampStr := fmt.Sprintf("%d.%03d", timestamp/1000, timestamp%1000)
			traceMsg := fmt.Sprintf("mcp-recv %s # %s", line, timestampStr)
			fmt.Fprintln(traceWriter, traceMsg)

			// Flush trace file to ensure immediate write
			if flusher, ok := traceWriter.(interface{ Flush() error }); ok {
				flusher.Flush()
			}
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
		ctx := RequestContext{
			Line:     line,
			JsonData: jsonData,
			ID:       idStr,
			Method:   methodStr,
		}

		// Handle operation-specific progress notifications
		if methodStr != "" && idStr != "" {
			if handleOperationNotifications(recording, idStr, out, traceWriter) {
				// If we handled the operation notifications, skip further processing
				log.Printf("handled operation notifications for method %s with ID %s", methodStr, idStr)
				continue
			}
		}

		log.Printf("received request without an id or method: %s", line)
		// If the request has an ID, remember it
		if idStr != "" {
			pendingRequests[idStr] = ctx
			seenRequests[idStr] = true
		} else if methodStr != "" {
			pendingRequests[methodStr] = ctx
		}

		// Process responses according to the original order
		processNextResponses := true
		for processNextResponses && currentRequestIndex < len(recordOrder) {
			record := recordOrder[currentRequestIndex]

			// Handle server notifications in sequence just like any other message
			// Don't skip them - we need to send them at the correct time
			if isServerNotification(record, serverNotifications, currentRequestIndex) {
				sendResponse(record.Content, out, traceWriter)
				currentRequestIndex++
				if *verbose {
					log.Printf("sent notification for method %s in original sequence", record.IDOrMethod)
				}
				continue
			}

			// Check if this is a response we need to send now
			if record.IsMethod {
				// Method-based response - only send if we've seen a matching method
				if _, ok := pendingRequests[record.IDOrMethod]; ok {
					delete(pendingRequests, record.IDOrMethod)
					sendResponse(record.Content, out, traceWriter)
					currentRequestIndex++
				} else {
					// Haven't seen this method request yet
					processNextResponses = false
				}
			} else {
				// ID-based response - only send if we've seen this ID
				if _, ok := pendingRequests[record.IDOrMethod]; ok {
					delete(pendingRequests, record.IDOrMethod)
					sendResponse(record.Content, out, traceWriter)
					currentRequestIndex++
				} else if _, ok := seenRequests[record.IDOrMethod]; ok {
					// We've seen this ID before but don't have a pending request
					// This might be a notification related to an earlier request
					sendResponse(record.Content, out, traceWriter)
					currentRequestIndex++
				} else {
					// Haven't seen this ID yet
					processNextResponses = false
				}
			}
		}
	}

	if err := stdinScanner.Err(); err != nil {
		log.Fatalf("Error reading from stdin: %v", err)
	}
}

// isServerNotification checks if the current record index is a server notification
func isServerNotification(record RecordOrderEntry, serverNotifications []int, currentIndex int) bool {
	for _, notifIdx := range serverNotifications {
		if currentIndex == notifIdx {
			return true
		}
	}
	return false
}

// handleOperationNotifications handles progress notifications for long-running operations
func handleOperationNotifications(recording *MCPRecording, operationID string, out io.Writer, traceWriter io.Writer) bool {
	if *verbose {
		log.Printf("handling operation notifications for ID %s", operationID)
	}
	// Check if we have specific progress notifications for this operation ID
	notifications, ok := recording.ProgressNotifications[operationID]
	if !ok || len(notifications) == 0 {
		return false
	}

	if *verbose {
		log.Printf("found %d progress notifications for operation ID %s",
			len(notifications), operationID)
	}

	// Sort notifications by progress value if applicable (simple implementation)
	sortNotificationsByProgress(notifications)

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

		// Generate timestamp for the current time
		timestamp := time.Now().UnixNano() / 1000000
		timestampStr := fmt.Sprintf("%d.%03d", timestamp/1000, timestamp%1000)

		// Format notification with timestamp
		formattedNotification := notification.JSON
		if !*jsonOnly {
			formattedNotification = fmt.Sprintf("mcp-send %s # %s", notification.JSON, timestampStr)
		}

		// Output to main writer
		fmt.Fprintln(out, formattedNotification)

		// Write to trace file if specified
		if traceWriter != nil {
			traceMsg := fmt.Sprintf("mcp-send %s # %s", notification.JSON, timestampStr)
			fmt.Fprintln(traceWriter, traceMsg)

			// Flush trace file to ensure immediate write
			if flusher, ok := traceWriter.(interface{ Flush() error }); ok {
				flusher.Flush()
			}
		}

		if *verbose {
			if notification.ProgressValue > 0 {
				log.Printf("sent progress notification %d/%d (progress: %d) for operation ID %s",
					i+1, len(notifications), notification.ProgressValue, operationID)
			} else {
				log.Printf("sent notification %d/%d for operation ID %s",
					i+1, len(notifications), operationID)
			}
		}
	}

	// After all notifications, send the final result
	// Find the response for this operation from the appropriate request map
	var responses []string
	if *useShadow {
		responses, ok = recording.ShadowRequestMap[operationID]
	} else {
		responses, ok = recording.RequestMap[operationID]
	}
	if ok && len(responses) > 0 {
		// Add a small delay before the final result
		time.Sleep(time.Duration(float64(200*time.Millisecond) / *speed))

		// Generate timestamp for the response
		timestamp := time.Now().UnixNano() / 1000000
		timestampStr := fmt.Sprintf("%d.%03d", timestamp/1000, timestamp%1000)

		// Format response with timestamp if not in JSON-only mode
		response := responses[0]
		if !*jsonOnly {
			response = fmt.Sprintf("mcp-send %s # %s", responses[0], timestampStr)
		}

		// Send response to main output
		fmt.Fprintln(out, response)

		// Write to trace file if specified
		if traceWriter != nil {
			traceMsg := fmt.Sprintf("mcp-send %s # %s", responses[0], timestampStr)
			fmt.Fprintln(traceWriter, traceMsg)

			// Flush trace file to ensure immediate write
			if flusher, ok := traceWriter.(interface{ Flush() error }); ok {
				flusher.Flush()
			}
		}

		if *verbose {
			log.Printf("sent final result for operation ID %s", operationID)
		}
	} else {
		// If we don't have a stored result, construct a generic one
		defaultResult := fmt.Sprintf(`{"result":{"content":[{"type":"text","text":"Operation completed successfully."}]},"jsonrpc":"2.0","id":%s}`, operationID)

		// Generate timestamp for the response
		timestamp := time.Now().UnixNano() / 1000000
		timestampStr := fmt.Sprintf("%d.%03d", timestamp/1000, timestamp%1000)

		// Format response with timestamp if not in JSON-only mode
		response := defaultResult
		if !*jsonOnly {
			response = fmt.Sprintf("mcp-send %s # %s", defaultResult, timestampStr)
		}

		// Send response to main output
		fmt.Fprintln(out, response)

		// Write to trace file if specified
		if traceWriter != nil {
			traceMsg := fmt.Sprintf("mcp-send %s # %s", defaultResult, timestampStr)
			fmt.Fprintln(traceWriter, traceMsg)

			// Flush trace file to ensure immediate write
			if flusher, ok := traceWriter.(interface{ Flush() error }); ok {
				flusher.Flush()
			}
		}

		if *verbose {
			log.Printf("sent default result for operation ID %s (no stored result found)", operationID)
		}
	}

	return true
}

// sortNotificationsByProgress sorts notifications by their progress value
func sortNotificationsByProgress(notifications []OperationNotification) {
	if len(notifications) <= 1 {
		return
	}

	// Simple bubble sort
	for i := 0; i < len(notifications)-1; i++ {
		for j := 0; j < len(notifications)-i-1; j++ {
			if notifications[j].ProgressValue > notifications[j+1].ProgressValue {
				notifications[j], notifications[j+1] = notifications[j+1], notifications[j]
			}
		}
	}
}

// sendResponse sends a formatted response to the output
func sendResponse(response string, out io.Writer, traceWriter io.Writer) {
	// Generate timestamp for response
	timestamp := time.Now().UnixNano() / 1000000 // Convert to milliseconds
	timestampStr := fmt.Sprintf("%d.%03d", timestamp/1000, timestamp%1000)

	if *jsonOnly {
		fmt.Fprintln(out, response)
	} else {
		// Determine the direction based on shadow mode
		direction := "mcp-send"
		if *useShadow {
			direction = "mcp-send-shadow"
		}
		// If it's not JSON only mode, format with appropriate prefix and timestamp
		fmt.Fprintf(out, "%s %s # %s\n", direction, response, timestampStr)
	}

	// Write to trace file if specified
	if traceWriter != nil {
		// Determine the direction based on shadow mode
		direction := "mcp-send"
		if *useShadow {
			direction = "mcp-send-shadow"
		}
		// Always write in trace format regardless of jsonOnly setting
		traceMsg := fmt.Sprintf("%s %s # %s", direction, response, timestampStr)
		fmt.Fprintln(traceWriter, traceMsg)

		// Flush trace file to ensure immediate write
		if flusher, ok := traceWriter.(interface{ Flush() error }); ok {
			flusher.Flush()
		}
	}

	// Ensure we flush the output to avoid buffering issues
	if flusher, ok := out.(interface{ Flush() error }); ok {
		flusher.Flush()
	}
}
