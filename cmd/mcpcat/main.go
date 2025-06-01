// Command mcpcat colorizes MCP recordings with detailed syntax highlighting.
// Shadow lines (mcp-send-shadow) are displayed in grey to distinguish them from primary traffic.
// Respects the NO_COLOR environment variable to disable coloring.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/term"
)

var (
	colorMode = flag.String("color", "auto", "color mode: auto, always, never")
	// Keep the old flag for backward compatibility
	oldColorFlag = flag.Bool("c", true, "colorize output (deprecated, use -color)")
)

// ANSI color codes
const (
	reset      = "\033[0m"
	bold       = "\033[1m"
	green      = "\033[32m" // recv: client→server
	brightCyan = "\033[96m" // send: server→client (more readable than blue)
	cyan       = "\033[36m"
	yellow     = "\033[33m"
	magenta    = "\033[35m"
	red        = "\033[31m"
	gray       = "\033[90m" // shadow responses
)

// Color palette for cycling through IDs
var colorPalette = []string{
	"\033[96m", // bright cyan
	"\033[93m", // bright yellow
	"\033[95m", // bright magenta
	"\033[91m", // bright red
	"\033[92m", // bright green
	"\033[94m", // bright blue (might be more readable for IDs)
	"\033[97m", // bright white
	cyan,       // regular cyan
}

// Regular expression to match the timestamp portion with milliseconds
var timestampRegex = regexp.MustCompile(` # (\d+)(?:\.(\d+))?$`)

func main() {
	flag.Parse()

	// Determine if we should use color
	useColor := shouldUseColor()

	// Determine input source
	var input io.Reader = os.Stdin
	if flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening %s: %v\n", flag.Arg(0), err)
			os.Exit(1)
		}
		defer f.Close()
		input = f
	}

	// Process the input line by line
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "mcp-") {
			// Non-MCP lines pass through unchanged
			fmt.Fprintln(os.Stderr, line)
			continue
		}

		// Parse the MCP line
		if useColor {
			colorize(line)
		} else {
			fmt.Fprintln(os.Stderr, line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}
}

// shouldUseColor determines whether to use color based on flags, environment, and terminal
func shouldUseColor() bool {
	// Check NO_COLOR environment variable first
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Handle the old -c flag if it was explicitly set
	if flag.Lookup("c").Value.String() == "false" {
		return false
	}

	// Handle the new -color flag
	switch *colorMode {
	case "never":
		return false
	case "always":
		return true
	case "auto":
		// Check if stderr is a terminal
		return term.IsTerminal(int(os.Stderr.Fd()))
	default:
		// Default to auto behavior for invalid values
		return term.IsTerminal(int(os.Stderr.Fd()))
	}
}

// colorize applies color formatting to an MCP line
func colorize(line string) {
	// Split into prefix and content
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		fmt.Fprintln(os.Stderr, line)
		return
	}

	prefix := parts[0]
	content := parts[1]

	// Determine direction
	var dirColor string
	var isRecv bool
	var isShadow bool
	if strings.HasSuffix(prefix, "recv") {
		dirColor = green
		isRecv = true
	} else if strings.HasSuffix(prefix, "send-shadow") {
		dirColor = gray
		isRecv = false
		isShadow = true
	} else {
		dirColor = brightCyan
		isRecv = false
	}

	// Format the prefix
	formattedPrefix := dirColor + prefix + reset

	// Find and format the timestamp
	timestampMatch := timestampRegex.FindStringIndex(content)
	if timestampMatch == nil {
		// No timestamp found
		if isShadow {
			fmt.Fprintf(os.Stderr, "%s %s\n", formattedPrefix, gray+content+reset)
		} else {
			fmt.Fprintf(os.Stderr, "%s %s\n", formattedPrefix, highlightJSON(content, isRecv))
		}
		return
	}

	jsonContent := content[:timestampMatch[0]]
	timestamp := content[timestampMatch[0]:]

	// Format JSON and timestamp
	var formattedJSON string
	if isShadow {
		formattedJSON = gray + jsonContent + reset
	} else {
		formattedJSON = highlightJSON(jsonContent, isRecv)
	}
	formattedTimestamp := gray + timestamp + reset

	fmt.Fprintf(os.Stderr, "%s %s %s\n", formattedPrefix, formattedJSON, formattedTimestamp)
}

// highlightJSON applies syntax highlighting to JSON content
func highlightJSON(jsonStr string, isRecv bool) string {
	// Try to parse as JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		// If not valid JSON, return as-is
		return jsonStr
	}

	// Highlight method and params for both directions
	if method, ok := data["method"].(string); ok {
		jsonStr = strings.Replace(jsonStr, "\"method\"", "\""+bold+"method"+reset+"\"", 1)
		jsonStr = strings.Replace(jsonStr, "\""+method+"\"", "\""+bold+method+reset+"\"", 1)
	}

	// Highlight ID with cycling colors
	if id, ok := data["id"]; ok {
		idStr := fmt.Sprintf("%v", id)
		idNum, err := strconv.Atoi(idStr)
		if err == nil {
			// Use cycling colors based on ID
			idColor := colorPalette[idNum%len(colorPalette)]
			jsonStr = strings.Replace(jsonStr, "\"id\"", "\""+bold+"id"+reset+"\"", 1)
			jsonStr = strings.Replace(jsonStr, "\"id\": "+idStr, "\"id\": "+idColor+bold+idStr+reset, 1)
			jsonStr = strings.Replace(jsonStr, "\"id\":"+idStr, "\"id\":"+idColor+bold+idStr+reset, 1)
		}
	}

	if isRecv {
		// For recv messages, highlight params and content
		if _, ok := data["params"]; ok {
			jsonStr = strings.Replace(jsonStr, "\"params\"", "\""+bold+"params"+reset+"\"", 1)
		}

		if _, ok := data["content"]; ok {
			jsonStr = strings.Replace(jsonStr, "\"content\"", "\""+bold+"content"+reset+"\"", 1)
		}
	} else {
		// For send messages, highlight result
		if _, ok := data["result"]; ok {
			jsonStr = strings.Replace(jsonStr, "\"result\"", "\""+bold+"result"+reset+"\"", 1)

			// Also highlight content in result if present
			if result, ok := data["result"].(map[string]interface{}); ok {
				if _, ok := result["content"]; ok {
					contentStr := "\"content\""
					jsonStr = strings.Replace(jsonStr, contentStr, "\""+bold+"content"+reset+"\"", 1)
				}
			}
		}
	}

	return jsonStr
}
