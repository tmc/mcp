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
	"strings"

	"github.com/fatih/color"
)

func main() {
	var (
		format   = flag.String("format", "auto", "Log format: auto, json, jsonrpc, plain")
		theme    = flag.String("theme", "default", "Color theme: default, dark, light, none")
		input    = flag.String("input", "-", "Input file (- for stdin)")
		output   = flag.String("output", "-", "Output file (- for stdout)")
		filter   = flag.String("filter", "", "Filter pattern (regex)")
		noColor  = flag.Bool("no-color", false, "Disable color output")
		lineNums = flag.Bool("line-numbers", false, "Show line numbers")
		follow   = flag.Bool("follow", false, "Follow file (like tail -f)")
		verbose  = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	// Setup colorizer
	colorizer := &Colorizer{
		Format:   *format,
		Theme:    *theme,
		NoColor:  *noColor || os.Getenv("NO_COLOR") != "",
		LineNums: *lineNums,
		Verbose:  *verbose,
	}

	// Setup filter
	if *filter != "" {
		re, err := regexp.Compile(*filter)
		if err != nil {
			log.Fatalf("Invalid filter pattern: %v", err)
		}
		colorizer.Filter = re
	}

	// Setup input
	var reader io.Reader
	if *input == "-" {
		reader = os.Stdin
	} else {
		file, err := os.Open(*input)
		if err != nil {
			log.Fatalf("Failed to open input file: %v", err)
		}
		defer file.Close()
		reader = file

		if *follow {
			// TODO: Implement file following
			log.Fatal("Follow mode not yet implemented")
		}
	}

	// Setup output
	var writer io.Writer
	if *output == "-" {
		writer = os.Stdout
	} else {
		file, err := os.Create(*output)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer file.Close()
		writer = file
	}

	// Process input
	if err := colorizer.Process(reader, writer); err != nil {
		log.Fatalf("Processing failed: %v", err)
	}
}

// Colorizer colorizes log output
type Colorizer struct {
	Format   string
	Theme    string
	NoColor  bool
	LineNums bool
	Filter   *regexp.Regexp
	Verbose  bool

	lineNum int
	colors  map[string]*color.Color
}

// Process reads from input and writes colorized output
func (c *Colorizer) Process(reader io.Reader, writer io.Writer) error {
	c.setupColors()

	scanner := bufio.NewScanner(reader)
	c.lineNum = 0

	for scanner.Scan() {
		c.lineNum++
		line := scanner.Text()

		// Apply filter if set
		if c.Filter != nil && !c.Filter.MatchString(line) {
			continue
		}

		// Colorize line
		colorized := c.colorizeLine(line)

		// Add line number if requested
		if c.LineNums {
			colorized = fmt.Sprintf("%s%d:%s %s",
				c.colors["linenum"].Sprint(),
				c.lineNum,
				c.colors["reset"].Sprint(),
				colorized)
		}

		// Write output
		fmt.Fprintln(writer, colorized)
	}

	return scanner.Err()
}

func (c *Colorizer) setupColors() {
	c.colors = make(map[string]*color.Color)

	if c.NoColor {
		// Create no-op colors
		for _, name := range []string{
			"timestamp", "level", "message", "field", "value",
			"method", "id", "error", "warning", "info", "debug",
			"linenum", "reset",
		} {
			c.colors[name] = color.New()
		}
		return
	}

	// Setup theme colors
	switch c.Theme {
	case "dark":
		c.colors["timestamp"] = color.New(color.FgBlue)
		c.colors["level"] = color.New(color.FgYellow)
		c.colors["message"] = color.New(color.FgWhite)
		c.colors["field"] = color.New(color.FgCyan)
		c.colors["value"] = color.New(color.FgGreen)
		c.colors["method"] = color.New(color.FgMagenta)
		c.colors["id"] = color.New(color.FgYellow)
		c.colors["error"] = color.New(color.FgRed, color.Bold)
		c.colors["warning"] = color.New(color.FgYellow)
		c.colors["info"] = color.New(color.FgGreen)
		c.colors["debug"] = color.New(color.FgCyan)
		c.colors["linenum"] = color.New(color.FgHiBlack)

	case "light":
		c.colors["timestamp"] = color.New(color.FgBlue)
		c.colors["level"] = color.New(color.FgRed)
		c.colors["message"] = color.New(color.FgBlack)
		c.colors["field"] = color.New(color.FgBlue)
		c.colors["value"] = color.New(color.FgGreen)
		c.colors["method"] = color.New(color.FgMagenta)
		c.colors["id"] = color.New(color.FgRed)
		c.colors["error"] = color.New(color.FgRed, color.Bold)
		c.colors["warning"] = color.New(color.FgYellow)
		c.colors["info"] = color.New(color.FgGreen)
		c.colors["debug"] = color.New(color.FgBlue)
		c.colors["linenum"] = color.New(color.FgBlack)

	default: // default theme
		c.colors["timestamp"] = color.New(color.FgBlue)
		c.colors["level"] = color.New(color.FgYellow)
		c.colors["message"] = color.New(color.FgWhite)
		c.colors["field"] = color.New(color.FgCyan)
		c.colors["value"] = color.New(color.FgGreen)
		c.colors["method"] = color.New(color.FgMagenta)
		c.colors["id"] = color.New(color.FgYellow)
		c.colors["error"] = color.New(color.FgRed, color.Bold)
		c.colors["warning"] = color.New(color.FgYellow)
		c.colors["info"] = color.New(color.FgGreen)
		c.colors["debug"] = color.New(color.FgCyan)
		c.colors["linenum"] = color.New(color.FgBlack, color.Faint)
	}

	c.colors["reset"] = color.New(color.Reset)
}

func (c *Colorizer) colorizeLine(line string) string {
	// Auto-detect format if needed
	format := c.Format
	if format == "auto" {
		format = c.detectFormat(line)
	}

	switch format {
	case "json":
		return c.colorizeJSON(line)
	case "jsonrpc":
		return c.colorizeJSONRPC(line)
	case "plain":
		return c.colorizePlain(line)
	default:
		return line
	}
}

func (c *Colorizer) detectFormat(line string) string {
	trimmed := strings.TrimSpace(line)

	// Check if it's JSON
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {

		// Check for JSON-RPC
		if strings.Contains(trimmed, `"jsonrpc"`) || strings.Contains(trimmed, `"method"`) {
			return "jsonrpc"
		}
		return "json"
	}

	return "plain"
}

func (c *Colorizer) colorizeJSON(line string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return line // Return as-is if not valid JSON
	}

	// Rebuild with colors
	var parts []string

	// Common log fields
	if timestamp, ok := data["timestamp"].(string); ok {
		parts = append(parts, c.colors["timestamp"].Sprint(timestamp))
		delete(data, "timestamp")
	}
	if level, ok := data["level"].(string); ok {
		levelColor := c.getLevelColor(level)
		parts = append(parts, levelColor.Sprint(strings.ToUpper(level)))
		delete(data, "level")
	}
	if msg, ok := data["message"].(string); ok {
		parts = append(parts, c.colors["message"].Sprint(msg))
		delete(data, "message")
	}
	if msg, ok := data["msg"].(string); ok {
		parts = append(parts, c.colors["message"].Sprint(msg))
		delete(data, "msg")
	}

	// Remaining fields
	for k, v := range data {
		fieldStr := fmt.Sprintf("%s=%v",
			c.colors["field"].Sprint(k),
			c.colors["value"].Sprint(v))
		parts = append(parts, fieldStr)
	}

	return strings.Join(parts, " ")
}

func (c *Colorizer) colorizeJSONRPC(line string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return line
	}

	var parts []string

	// JSON-RPC version
	if jsonrpc, ok := data["jsonrpc"].(string); ok {
		parts = append(parts, c.colors["field"].Sprintf("jsonrpc:%s", jsonrpc))
	}

	// Method (for requests)
	if method, ok := data["method"].(string); ok {
		parts = append(parts, c.colors["method"].Sprintf("method:%s", method))
	}

	// ID
	if id := data["id"]; id != nil {
		parts = append(parts, c.colors["id"].Sprintf("id:%v", id))
	}

	// Result (for responses)
	if result := data["result"]; result != nil {
		resultStr := c.formatValue(result)
		parts = append(parts, c.colors["value"].Sprintf("result:%s", resultStr))
	}

	// Error (for error responses)
	if errData := data["error"]; errData != nil {
		errStr := c.formatValue(errData)
		parts = append(parts, c.colors["error"].Sprintf("error:%s", errStr))
	}

	// Params
	if params := data["params"]; params != nil {
		paramsStr := c.formatValue(params)
		parts = append(parts, c.colors["field"].Sprintf("params:%s", paramsStr))
	}

	return strings.Join(parts, " ")
}

func (c *Colorizer) colorizePlain(line string) string {
	// Simple pattern matching for common log formats

	// Timestamp pattern
	timestampRe := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[\sT]\d{2}:\d{2}:\d{2}`)
	if match := timestampRe.FindString(line); match != "" {
		line = strings.Replace(line, match, c.colors["timestamp"].Sprint(match), 1)
	}

	// Log level patterns
	levels := map[string]*regexp.Regexp{
		"ERROR": regexp.MustCompile(`\b(ERROR|ERR|FATAL)\b`),
		"WARN":  regexp.MustCompile(`\b(WARN|WARNING)\b`),
		"INFO":  regexp.MustCompile(`\b(INFO|INFORMATION)\b`),
		"DEBUG": regexp.MustCompile(`\b(DEBUG|DBG|TRACE)\b`),
	}

	for level, re := range levels {
		if match := re.FindString(line); match != "" {
			levelColor := c.getLevelColor(strings.ToLower(level))
			line = strings.Replace(line, match, levelColor.Sprint(match), 1)
			break
		}
	}

	// Highlight quoted strings
	quotedRe := regexp.MustCompile(`"[^"]*"`)
	line = quotedRe.ReplaceAllStringFunc(line, func(match string) string {
		return c.colors["value"].Sprint(match)
	})

	// Highlight key=value pairs
	kvRe := regexp.MustCompile(`\b(\w+)=([^\s]+)`)
	line = kvRe.ReplaceAllStringFunc(line, func(match string) string {
		parts := strings.SplitN(match, "=", 2)
		if len(parts) == 2 {
			return fmt.Sprintf("%s=%s",
				c.colors["field"].Sprint(parts[0]),
				c.colors["value"].Sprint(parts[1]))
		}
		return match
	})

	return line
}

func (c *Colorizer) getLevelColor(level string) *color.Color {
	switch strings.ToLower(level) {
	case "error", "err", "fatal":
		return c.colors["error"]
	case "warn", "warning":
		return c.colors["warning"]
	case "info", "information":
		return c.colors["info"]
	case "debug", "dbg", "trace":
		return c.colors["debug"]
	default:
		return c.colors["message"]
	}
}

func (c *Colorizer) formatValue(value interface{}) string {
	// Format complex values for display
	switch v := value.(type) {
	case string:
		return v
	case map[string]interface{}, []interface{}:
		// Compact JSON for complex objects
		data, _ := json.Marshal(v)
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `logcolor - Colorize log output

Usage:
  logcolor [options] < input.log

Examples:
  tail -f app.log | logcolor                    # Colorize streaming logs
  logcolor -format json < server.log            # Colorize JSON logs
  logcolor -filter ERROR -line-numbers < debug.log  # Filter and number lines
  logcolor -theme dark -o colored.log < plain.log   # Save colorized output

Formats:
  auto    - Auto-detect format (default)
  json    - JSON structured logs
  jsonrpc - JSON-RPC format
  plain   - Plain text logs

Themes:
  default - Default color scheme
  dark    - Dark terminal theme
  light   - Light terminal theme
  none    - No colors

Options:
`)
		flag.PrintDefaults()
	}
}
