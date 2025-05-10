// Command mcpspy records MCP interactions.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

var (
	outFile         = flag.String("f", "", "output recording file")
	verbose         = flag.Bool("v", false, "verbose mode: print interactions to stderr")
	veryVerbose     = flag.Bool("vv", false, "very verbose mode: print raw stdin/stdout/stderr to stderr")
	useAppend       = flag.Bool("a", false, "append to existing file instead of overwriting")
	noCopyStderr    = flag.Bool("no-stderr", false, "do not copy stderr from command")
	prettyJSON      = flag.Bool("pretty", false, "pretty-print JSON output")
	pipeMode        = flag.Bool("pipe", false, "explicitly enable pipe mode (for testing)")
	indentLevel     = flag.Int("indent", 0, "indentation level for output (useful when piping multiple mcpspy commands)")
	indentChar      = flag.String("indent-char", "  ", "character(s) to use for indentation")
	parentPID       = flag.Int("parent-pid", 0, "parent mcpspy process ID (set automatically in pipelines)")
	passThrough     = flag.Bool("pass-through", false, "pass JSON through unmodified (no mcp- prefix/suffix)")
	autoIndent      = flag.Bool("auto-indent", false, "automatically determine indentation level based on pipeline depth")
	forceUnbuffered = flag.Bool("unbuffered", false, "force unbuffered output (useful for real-time piping)")
	quiet           = flag.Bool("q", false, "quiet mode: suppress log messages")
)

// safeWriter provides a thread-safe writer that reopens the file if it disappears
type safeWriter struct {
	mu       sync.Mutex
	filename string
	file     *os.File
	buf      *bufio.Writer
}

func newSafeWriter(filename string, appendMode bool) (*safeWriter, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return nil, err
	}

	return &safeWriter{
		filename: filename,
		file:     f,
		buf:      bufio.NewWriter(f),
	}, nil
}

func (w *safeWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(w.filename); os.IsNotExist(err) {
		// File disappeared, reopen it
		if err := w.reopen(); err != nil {
			return 0, err
		}
	}

	n, err = w.buf.Write(p)
	if err != nil {
		return n, err
	}

	// Flush after every write to disk
	if err := w.buf.Flush(); err != nil {
		return n, err
	}

	return n, nil
}

func (w *safeWriter) reopen() error {
	// Close existing file if open
	if w.file != nil {
		w.buf.Flush()
		w.file.Close()
	}

	// Reopen the file in append mode
	f, err := os.OpenFile(w.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	w.file = f
	w.buf = bufio.NewWriter(f)
	return nil
}

func (w *safeWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		w.buf.Flush()
		err := w.file.Close()
		w.file = nil
		w.buf = nil
		return err
	}
	return nil
}

// jsonScanner is a helper type for parsing and formatting JSON messages
type jsonScanner struct {
	buffer []byte
}

func newJSONScanner() *jsonScanner {
	return &jsonScanner{
		buffer: make([]byte, 0, 4096),
	}
}

func (s *jsonScanner) add(data []byte) []string {
	s.buffer = append(s.buffer, data...)

	var messages []string

	// Try to find complete JSON objects first
	foundJSON := false
	for {
		start := -1
		depth := 0
		inString := false
		escape := false

		// Find the start of a JSON object
		for i := 0; i < len(s.buffer); i++ {
			if s.buffer[i] == '{' {
				start = i
				depth = 1
				break
			}
		}

		if start == -1 {
			break // No JSON object found
		}

		// Find the end of the JSON object
		end := -1
		for i := start + 1; i < len(s.buffer); i++ {
			c := s.buffer[i]

			if escape {
				escape = false
				continue
			}

			if c == '\\' && inString {
				escape = true
				continue
			}

			if c == '"' {
				inString = !inString
				continue
			}

			if !inString {
				if c == '{' {
					depth++
				} else if c == '}' {
					depth--
					if depth == 0 {
						end = i + 1
						break
					}
				}
			}
		}

		if end == -1 {
			break // Incomplete JSON object
		}

		// Extract the complete JSON object
		jsonObj := string(s.buffer[start:end])

		// Verify it's valid JSON by trying to parse it
		var obj interface{}
		if err := json.Unmarshal([]byte(jsonObj), &obj); err == nil {
			// Valid JSON - add it to our messages
			messages = append(messages, jsonObj)
			foundJSON = true

			// Remove the processed JSON object from the buffer
			s.buffer = s.buffer[end:]
		} else {
			// Either this isn't JSON or it's malformed
			// Let's try to skip this opening brace and look for another JSON object
			if start < len(s.buffer) {
				s.buffer = s.buffer[start+1:]
			}
			// Don't break - continue looking for valid JSON
		}
	}

	// If no valid JSON objects were found, fall back to line-based processing
	if !foundJSON && len(messages) == 0 {
		// Split buffer on newlines
		lines := strings.Split(string(s.buffer), "\n")

		// If we have more than one line, process all but the last line
		// (the last line might be incomplete)
		if len(lines) > 1 {
			// Process and clear all complete lines except the last one
			for i := 0; i < len(lines)-1; i++ {
				if line := strings.TrimSpace(lines[i]); line != "" {
					messages = append(messages, line)
				}
			}

			// Update buffer to contain only the potentially incomplete last line
			s.buffer = []byte(lines[len(lines)-1])
		}
	}

	return messages
}

func formatJSON(jsonStr string, pretty bool) string {
	if !pretty {
		return jsonStr
	}

	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return jsonStr // Return original if we can't parse
	}

	prettyJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return jsonStr
	}

	return string(prettyJSON)
}

// teeReader reads from r and logs to log.
type teeReader struct {
	r               io.Reader
	log             io.Writer
	dir             string
	verbose         bool
	veryVerbose     bool
	scanner         *jsonScanner
	prettyJSON      bool
	indentLevel     int
	indentChar      string
	passThrough     bool // Whether to pass JSON through unmodified
	forceUnbuffered bool // Force unbuffered output
}

func (t *teeReader) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 && t.log != nil {
		now := time.Now()
		unixSec := now.Unix()
		unixMilli := now.UnixMilli() % 1000

		// Always use the advanced scanner that falls back to line-based parsing
		if t.scanner == nil {
			t.scanner = newJSONScanner()
		}

		messages := t.scanner.add(p[:n])
		if len(messages) > 0 {
			// Process parsed messages
			for _, msg := range messages {
				formattedJSON := formatJSON(msg, t.prettyJSON)
				var entry string
				if t.passThrough {
					// For pass-through mode, just output the JSON directly
					entry = formattedJSON + "\n"
				} else {
					// Normal mode with mcp- prefix and timestamp
					entry = fmt.Sprintf("mcp-%s %s # %d.%03d\n", t.dir, formattedJSON, unixSec, unixMilli)
				}

				if _, writeErr := t.log.Write([]byte(entry)); writeErr != nil {
					fmt.Fprintf(os.Stderr, "mcpspy: error writing to log: %v\n", writeErr)
				}

				if t.verbose {
					// Apply indentation based on direction for tree-like view:
					// recv messages are at base indentation level (represents client requests)
					baseIndent := strings.Repeat(t.indentChar, t.indentLevel)
					// add stdin fd to baseindent w a space after:
					fmt.Fprintf(os.Stderr, "\033[32m%s%s\033[0m", baseIndent, entry) // Green for recv with indentation
				}

				if t.veryVerbose {
					indent := strings.Repeat(t.indentChar, t.indentLevel)
					fmt.Fprintf(os.Stderr, "%s#> %s\n", indent, formattedJSON) // Raw input with indentation
				}
			}
		} else if n > 0 {
			// If scanner didn't parse anything but we received data,
			// just log the raw data as a fallback
			data := strings.TrimRight(string(p[:n]), " \t\r\n")
			entry := fmt.Sprintf("mcp-%s %s # %d.%03d\n", t.dir, data, unixSec, unixMilli)

			if _, writeErr := t.log.Write([]byte(entry)); writeErr != nil {
				fmt.Fprintf(os.Stderr, "mcpspy: error writing to log: %v\n", writeErr)
			}

			if t.verbose {
				// Apply base indentation for recv messages
				baseIndent := strings.Repeat(t.indentChar, t.indentLevel)
				// add stdin fd to baseindent w a space after:
				var stat syscall.Stat_t
				syscall.Fstat(0, &stat)
				log.Println(stat)
				fmt.Fprintf(os.Stderr, "\033[32m%s%s\033[0m", baseIndent, entry) // Green for recv with indentation
			}

			if t.veryVerbose {
				indent := strings.Repeat(t.indentChar, t.indentLevel)
				fmt.Fprintf(os.Stderr, "%s#> %s\n", indent, data) // Raw input with indentation
			}
		}
	}
	return
}

// teeWriter writes to w and logs to log.
type teeWriter struct {
	w               io.Writer
	log             io.Writer
	dir             string
	verbose         bool
	veryVerbose     bool
	scanner         *jsonScanner
	prettyJSON      bool
	indentLevel     int
	indentChar      string
	passThrough     bool // Whether to pass JSON through unmodified
	forceUnbuffered bool // Force unbuffered output
}

func (t *teeWriter) Write(p []byte) (n int, err error) {
	var dataToWrite []byte

	// Handle indentation for passthrough mode
	if t.passThrough && t.indentLevel > 0 {
		// Create indentation string
		indent := strings.Repeat(t.indentChar, t.indentLevel)

		// Add indentation to the beginning of the content
		content := string(p)
		if strings.HasSuffix(content, "\n") {
			content = content[:len(content)-1]
			dataToWrite = []byte(indent + content + "\n")
		} else {
			dataToWrite = []byte(indent + content)
		}
	} else {
		dataToWrite = p
	}

	n, err = t.w.Write(dataToWrite)

	// Force flush to ensure unbuffered output
	if t.forceUnbuffered {
		if f, ok := t.w.(*os.File); ok {
			f.Sync()
		}
	}
	if n > 0 && t.log != nil {
		now := time.Now()
		unixSec := now.Unix()
		unixMilli := now.UnixMilli() % 1000

		// Always use the advanced scanner that falls back to line-based parsing
		if t.scanner == nil {
			t.scanner = newJSONScanner()
		}

		messages := t.scanner.add(p[:n])
		if len(messages) > 0 {
			// Process parsed messages
			for _, msg := range messages {
				formattedJSON := formatJSON(msg, t.prettyJSON)
				var entry string
				if t.passThrough {
					// For pass-through mode, just output the JSON directly
					entry = formattedJSON + "\n"
				} else {
					// Normal mode with mcp- prefix and timestamp
					entry = fmt.Sprintf("mcp-%s %s # %d.%03d\n", t.dir, formattedJSON, unixSec, unixMilli)
				}

				if _, writeErr := t.log.Write([]byte(entry)); writeErr != nil {
					fmt.Fprintf(os.Stderr, "mcpspy: error writing to log: %v\n", writeErr)
				}

				if t.verbose {
					// No indentation for send messages
					fmt.Fprintf(os.Stderr, "\033[34m%s\033[0m", entry) // Blue for send with deeper indentation
				}

				if t.veryVerbose {
					indent := strings.Repeat(t.indentChar, t.indentLevel)
					fmt.Fprintf(os.Stderr, "%s#< %s\n", indent, formattedJSON) // Raw output with indentation
				}
			}
		} else if n > 0 {
			// If scanner didn't parse anything but we received data,
			// just log the raw data as a fallback
			data := strings.TrimRight(string(p[:n]), " \t\r\n")
			entry := fmt.Sprintf("mcp-%s %s # %d.%03d\n", t.dir, data, unixSec, unixMilli)

			if _, writeErr := t.log.Write([]byte(entry)); writeErr != nil {
				fmt.Fprintf(os.Stderr, "mcpspy: error writing to log: %v\n", writeErr)
			}

			if t.verbose {
				// No indentation for send messages
				fmt.Fprintf(os.Stderr, "\033[34m%s\033[0m", entry) // Blue for send messages
			}

			if t.veryVerbose {
				indent := strings.Repeat(t.indentChar, t.indentLevel)
				fmt.Fprintf(os.Stderr, "%s#< %s\n", indent, data) // Raw output with indentation
			}
		}
	}
	return
}

// generateFilename creates a filename based on the command and args,
// ensuring it does not already exist by attempting to create it exclusively.
func generateFilename(args []string) string {
	if len(args) == 0 {
		return "mcpspy.0.mcp"
	}

	// Start with the command name
	parts := []string{filepath.Base(args[0])}

	// Add up to 3 arguments, replacing special chars
	for i := 1; i < len(args) && i <= 3; i++ {
		// Clean up the argument for use in a filename
		arg := filepath.Base(args[i])
		arg = strings.ReplaceAll(arg, "/", "-")
		arg = strings.ReplaceAll(arg, "\\", "-")
		arg = strings.ReplaceAll(arg, ":", "-")
		parts = append(parts, arg)
	}

	// Join with hyphens and add a sequence number
	base := strings.Join(parts, "-")

	// Find an available filename by trying to create it exclusively
	for i := 0; ; i++ { // Loop indefinitely until we find a free filename
		filename := fmt.Sprintf("%s.%d.mcp", base, i)
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if os.IsNotExist(err) {
			// Should not happen with O_EXCL, but just in case
			continue
		}
		if err == nil {
			// Successfully created the file, it's available
			f.Close()
			os.Remove(filename) // Clean up, we just needed to check existence
			return filename
		}
		// If err is not os.IsExist, it's a real error
		if !os.IsExist(err) {
			log.Printf("warning: error checking file %s: %v, falling back to timestamp", filename, err)
			break // Fallback to timestamp
		}
		// File exists, try next sequence number
	}

	// Fallback with timestamp if we can't find an available name or if there's an error
	return fmt.Sprintf("%s.%d.mcp", base, time.Now().Unix())
}

// detectPipeline determines if we're in a pipeline with other mcpspy instances
// and calculates the appropriate indentation level automatically
func detectPipeline() (int, bool) {
	// Get the current process info
	_ = os.Getpid() // For future use

	// Check if stdin is not a terminal (indicating a pipe)
	stdinIsPipe := !term.IsTerminal(int(os.Stdin.Fd()))

	// Check if stdout is not a terminal (indicating a pipe)
	stdoutIsPipe := !term.IsTerminal(int(os.Stdout.Fd()))

	// Check if we're in a pipeline
	inPipeline := stdinIsPipe || stdoutIsPipe

	// Calculate depth based on parent processes
	pipeDepth := 0

	// Check if explicitly set parent PID
	if *parentPID > 0 {
		// We know we're in a pipeline with at least one other mcpspy
		pipeDepth = 1

		if *verbose {
			fmt.Fprintf(os.Stderr, "Detected pipeline: parent mcpspy PID %d\n", *parentPID)
		}
	} else if inPipeline {
		// We're in a pipeline but don't know if there are other mcpspy instances
		// Count stdin/stdout pipes
		if stdinIsPipe {
			pipeDepth++
		}
		if stdoutIsPipe {
			pipeDepth++
		}

		if *verbose {
			fmt.Fprintf(os.Stderr, "Detected pipeline: stdin pipe=%v, stdout pipe=%v\n",
				stdinIsPipe, stdoutIsPipe)
		}
	}

	return pipeDepth, inPipeline
}

// modifyArgsForChild modifies command arguments to add parent PID information
func modifyArgsForChild(args []string) []string {
	currentPID := os.Getpid()

	// Look for mcpspy in the command
	var newArgs []string
	foundMCPSpy := false

	for i, arg := range args {
		newArgs = append(newArgs, arg)

		// If this is a mcpspy command in the pipeline
		if strings.Contains(arg, "mcpspy") && !foundMCPSpy {
			foundMCPSpy = true

			// Add the parent-pid flag right after the mcpspy command
			parentPIDFlag := fmt.Sprintf("-parent-pid=%d", currentPID)

			// Check if the next arg is a flag; if not, insert our flag
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				newArgs = append(newArgs, parentPIDFlag)
			} else {
				// Otherwise, find a good place to insert it
				newArgs = append(newArgs[:i+1], append([]string{parentPIDFlag}, newArgs[i+1:]...)...)
			}

			// If auto-indent is enabled, also pass that flag
			if *autoIndent {
				autoIndentFlag := "-auto-indent"
				newArgs = append(newArgs[:i+2], append([]string{autoIndentFlag}, newArgs[i+2:]...)...)
			}
		}
	}

	return newArgs
}

func main() {
	// Check for double dash to separate flags from command
	doubleDashIndex := -1
	for i, arg := range os.Args {
		if arg == "--" {
			doubleDashIndex = i
			break
		}
	}

	// Setup the logger
	log.SetPrefix("mcpspy: ")
	log.SetFlags(0) // No date/time prefix

	// Split args for flag parsing vs command
	flagArgs := os.Args[1:] // Default: all args except program name
	var cmdArgs []string

	if doubleDashIndex > 0 {
		flagArgs = os.Args[1:doubleDashIndex]
		cmdArgs = os.Args[doubleDashIndex+1:]
		// Reset os.Args temporarily for flag parsing
		originalArgs := os.Args
		os.Args = append([]string{os.Args[0]}, flagArgs...)
		defer func() { os.Args = originalArgs }()
	}

	flag.Parse()

	// Configure logger based on quiet flag
	if *quiet {
		log.SetOutput(io.Discard) // Discard all logs in quiet mode
	}

	// Default output file based on commands or automatic naming
	if *outFile == "" {
		var cmdForFilename []string
		if len(cmdArgs) > 0 {
			cmdForFilename = cmdArgs
		} else if len(flag.Args()) > 0 {
			cmdForFilename = flag.Args()
		}
		filename := generateFilename(cmdForFilename)
		*outFile = filename
	}

	writer, err := newSafeWriter(*outFile, *useAppend)
	if err != nil {
		log.Fatal(err)
	}
	defer writer.Close()

	if *verbose || *veryVerbose {
		mode := "overwriting"
		if *useAppend {
			mode = "appending to"
		}
		fmt.Fprintf(os.Stderr, "mcpspy: %s %s\n", mode, *outFile)
	}

	// Determine which command to run (from -- or regular args)
	var command []string
	if len(cmdArgs) > 0 {
		command = cmdArgs
	} else if len(flag.Args()) > 0 {
		command = flag.Args()
	}

	// If no command specified or pipe mode is explicitly enabled, act as a bidirectional pipe logger
	if len(command) == 0 || *pipeMode {
		// Use the pipe logger handler
		handlePipeMode(writer, *verbose, *veryVerbose, *prettyJSON, *indentLevel, *indentChar, *passThrough, *forceUnbuffered)
		return
	}

	// Calculate auto-indent level if enabled
	pipeDepth := 0
	inPipeline := false

	if *autoIndent {
		pipeDepth, inPipeline = detectPipeline()
		// If we're in a pipeline and auto-indent is enabled, override indentLevel
		if inPipeline {
			// For tree-like visualization of message flows:
			// - No indentation for the 'recv' direction (incoming requests)
			// - Add indentation for the 'send' direction (responses)
			if *verbose {
				fmt.Fprintf(os.Stderr, "Auto-indent: using tree-like visualization\n")
			}

			// Set base indentation level
			*indentLevel = pipeDepth
		}
	}

	// Modify command args for child mcpspy processes
	if len(command) > 0 && *autoIndent {
		command = modifyArgsForChild(command)
	}

	// Set up command
	cmd := exec.Command(command[0], command[1:]...)

	// Create pipes for stdin/stdout
	cmdStdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	// Handle stderr if very verbose mode is enabled
	if *veryVerbose {
		cmdStderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}
		// Copy stderr to os.Stderr with prefix
		go func() {
			scanner := bufio.NewScanner(cmdStderr)
			for scanner.Scan() {
				fmt.Fprintf(os.Stderr, "#! %s\n", scanner.Text())
			}
		}()
	} else if !*noCopyStderr {
		cmd.Stderr = os.Stderr
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// Set up signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-signalChan
		if sig == syscall.SIGQUIT {
			// Forward SIGQUIT to the child process
			if *verbose || *veryVerbose {
				fmt.Fprintf(os.Stderr, "mcpspy: forwarding SIGQUIT to child process (PID %d)\n", cmd.Process.Pid)
			}
			cmd.Process.Signal(syscall.SIGQUIT)
		} else {
			// Forward other signals as well
			if *verbose || *veryVerbose {
				fmt.Fprintf(os.Stderr, "mcpspy: forwarding signal %s to child process (PID %d)\n", sig, cmd.Process.Pid)
			}
			cmd.Process.Signal(sig)
		}
	}()

	// Create a WaitGroup to ensure all goroutines finish
	var wg sync.WaitGroup
	wg.Add(2)

	// Handle input: copy from stdin to command stdin, logging as we go
	go func() {
		defer wg.Done()
		r := &teeReader{
			r:           os.Stdin,
			log:         writer,
			dir:         "recv",
			verbose:     *verbose,
			veryVerbose: *veryVerbose,
			prettyJSON:  *prettyJSON,
			indentLevel: *indentLevel,
			indentChar:  *indentChar,
			passThrough: *passThrough,
		}
		io.Copy(cmdStdin, r)
		cmdStdin.Close()
	}()

	// Handle output: copy from command stdout to stdout, logging as we go
	go func() {
		defer wg.Done()
		w := &teeWriter{
			w:           os.Stdout,
			log:         writer,
			dir:         "send",
			verbose:     *verbose,
			veryVerbose: *veryVerbose,
			prettyJSON:  *prettyJSON,
			indentLevel: *indentLevel,
			indentChar:  *indentChar,
			passThrough: *passThrough,
		}
		io.Copy(w, cmdStdout)
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		log.Fatal(err)
	}
}
