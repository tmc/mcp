// Package mcptestutil provides debug helpers for test troubleshooting and analysis.
package mcptestutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"
)

// DebugInfo contains comprehensive system and test state information.
type DebugInfo struct {
	Timestamp   time.Time              `json:"timestamp"`
	TestName    string                 `json:"testName"`
	Duration    time.Duration          `json:"duration"`
	Goroutines  int                    `json:"goroutines"`
	MemoryStats runtime.MemStats       `json:"memoryStats"`
	StackTrace  string                 `json:"stackTrace"`
	Environment map[string]string      `json:"environment"`
	BuildInfo   *debug.BuildInfo       `json:"buildInfo,omitempty"`
	CustomData  map[string]interface{} `json:"customData,omitempty"`
	Profiles    map[string]string      `json:"profiles,omitempty"`
}

// VerboseLogger provides conditional verbose logging that only outputs when tests fail.
type VerboseLogger struct {
	t         *testing.T
	buffer    *bytes.Buffer
	isEnabled bool
	mu        sync.RWMutex
	startTime time.Time
	entries   []logEntry
}

// logEntry represents a single log entry with metadata.
type logEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
}

// NewVerboseLogger creates a new verbose logger for the given test.
// Logs are buffered and only output if the test fails.
//
// Usage:
//
//	logger := NewVerboseLogger(t)
//	logger.Info("Starting test phase 1")
//	logger.Debug("Variable x = %v", x)
//	// Logs only appear if test fails
func NewVerboseLogger(t *testing.T) *VerboseLogger {
	t.Helper()

	logger := &VerboseLogger{
		t:         t,
		buffer:    &bytes.Buffer{},
		isEnabled: shouldEnableVerboseLogging(),
		startTime: time.Now(),
		entries:   make([]logEntry, 0),
	}

	// Register cleanup to output logs if test fails
	t.Cleanup(func() {
		if t.Failed() {
			logger.OutputLogs()
		}
	})

	return logger
}

// shouldEnableVerboseLogging determines if verbose logging should be enabled.
func shouldEnableVerboseLogging() bool {
	// Check environment variables
	if os.Getenv("VERBOSE_TESTS") != "" ||
		os.Getenv("DEBUG_TESTS") != "" ||
		os.Getenv("TEST_VERBOSE") != "" {
		return true
	}

	// Check if -v flag is present (testing.Verbose() is not reliable in all contexts)
	for _, arg := range os.Args {
		if arg == "-v" || arg == "-test.v" || strings.Contains(arg, "-test.v=true") {
			return true
		}
	}

	return false
}

// Info logs an informational message.
func (vl *VerboseLogger) Info(format string, args ...interface{}) {
	vl.log("INFO", format, args...)
}

// Debug logs a debug message.
func (vl *VerboseLogger) Debug(format string, args ...interface{}) {
	vl.log("DEBUG", format, args...)
}

// Warn logs a warning message.
func (vl *VerboseLogger) Warn(format string, args ...interface{}) {
	vl.log("WARN", format, args...)
}

// Error logs an error message.
func (vl *VerboseLogger) Error(format string, args ...interface{}) {
	vl.log("ERROR", format, args...)
}

// LogStruct logs a structured object as JSON.
func (vl *VerboseLogger) LogStruct(level, message string, data interface{}) {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	entry := logEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Data:      data,
	}

	vl.entries = append(vl.entries, entry)

	// Also write to buffer for immediate output if needed
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	vl.buffer.WriteString(fmt.Sprintf("[%s] %s %s: %s\n",
		entry.Timestamp.Format(time.RFC3339),
		level,
		message,
		string(jsonData)))
}

// log is the internal logging implementation.
func (vl *VerboseLogger) log(level, format string, args ...interface{}) {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	message := fmt.Sprintf(format, args...)

	entry := logEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	vl.entries = append(vl.entries, entry)

	// Write to buffer
	vl.buffer.WriteString(fmt.Sprintf("[%s] %s: %s\n",
		entry.Timestamp.Format(time.RFC3339),
		level,
		message))
}

// OutputLogs outputs all buffered logs to the test output.
// This is automatically called if the test fails.
func (vl *VerboseLogger) OutputLogs() {
	vl.mu.RLock()
	defer vl.mu.RUnlock()

	if vl.buffer.Len() == 0 {
		return
	}

	vl.t.Logf("=== Verbose Test Logs ===\n%s=== End Verbose Logs ===", vl.buffer.String())
}

// ForceOutput forces output of logs regardless of test status.
func (vl *VerboseLogger) ForceOutput() {
	vl.OutputLogs()
}

// GetEntries returns all log entries for programmatic access.
func (vl *VerboseLogger) GetEntries() []logEntry {
	vl.mu.RLock()
	defer vl.mu.RUnlock()

	// Return a copy to prevent modification
	entries := make([]logEntry, len(vl.entries))
	copy(entries, vl.entries)
	return entries
}

// DebugOnFailure runs a debug function only if the test fails.
// This is useful for expensive debug operations that should only run on failure.
//
// Usage:
//
//	DebugOnFailure(t, func() {
//	    // Expensive debug operations
//	    dumpMemoryProfile()
//	    dumpGoroutineStacks()
//	})
func DebugOnFailure(t *testing.T, debugFunc func()) {
	t.Helper()

	t.Cleanup(func() {
		if t.Failed() {
			t.Log("Test failed - running debug function")
			debugFunc()
		}
	})
}

// CaptureDebugInfo collects comprehensive debug information about the current system state.
// This includes memory stats, goroutine info, environment variables, and custom data.
func CaptureDebugInfo(t *testing.T, customData map[string]interface{}) *DebugInfo {
	t.Helper()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get stack trace of all goroutines
	buf := make([]byte, 1024*1024) // 1MB buffer
	n := runtime.Stack(buf, true)
	stackTrace := string(buf[:n])

	// Get environment variables
	env := make(map[string]string)
	for _, envVar := range os.Environ() {
		if parts := strings.SplitN(envVar, "=", 2); len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	// Get build info
	buildInfo, _ := debug.ReadBuildInfo()

	info := &DebugInfo{
		Timestamp:   time.Now(),
		TestName:    t.Name(),
		Goroutines:  runtime.NumGoroutine(),
		MemoryStats: memStats,
		StackTrace:  stackTrace,
		Environment: env,
		BuildInfo:   buildInfo,
		CustomData:  customData,
		Profiles:    make(map[string]string),
	}

	return info
}

// CaptureProfiles captures CPU and memory profiles for debug analysis.
// Returns the captured profiles as strings for inclusion in debug info.
func CaptureProfiles(t *testing.T, duration time.Duration) map[string]string {
	t.Helper()

	profiles := make(map[string]string)

	// Capture heap profile
	if heapProfile := captureHeapProfile(); heapProfile != "" {
		profiles["heap"] = heapProfile
	}

	// Capture goroutine profile
	if goroutineProfile := captureGoroutineProfile(); goroutineProfile != "" {
		profiles["goroutine"] = goroutineProfile
	}

	// Capture CPU profile if duration is specified
	if duration > 0 {
		if cpuProfile := captureCPUProfile(duration); cpuProfile != "" {
			profiles["cpu"] = cpuProfile
		}
	}

	return profiles
}

// captureHeapProfile captures a heap memory profile.
func captureHeapProfile() string {
	var buf bytes.Buffer
	if err := pprof.WriteHeapProfile(&buf); err != nil {
		return fmt.Sprintf("Error capturing heap profile: %v", err)
	}
	return buf.String()
}

// captureGoroutineProfile captures a goroutine profile.
func captureGoroutineProfile() string {
	var buf bytes.Buffer
	if err := pprof.Lookup("goroutine").WriteTo(&buf, 1); err != nil {
		return fmt.Sprintf("Error capturing goroutine profile: %v", err)
	}
	return buf.String()
}

// captureCPUProfile captures a CPU profile for the specified duration.
func captureCPUProfile(duration time.Duration) string {
	var buf bytes.Buffer

	if err := pprof.StartCPUProfile(&buf); err != nil {
		return fmt.Sprintf("Error starting CPU profile: %v", err)
	}

	time.Sleep(duration)
	pprof.StopCPUProfile()

	return buf.String()
}

// DumpDebugInfo dumps debug information to the test log and optionally to a file.
func DumpDebugInfo(t *testing.T, info *DebugInfo, filename string) {
	t.Helper()

	// Format as JSON for readability
	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		t.Logf("Error marshaling debug info: %v", err)
		return
	}

	// Log to test output
	t.Logf("=== Debug Information ===\n%s\n=== End Debug Info ===", string(jsonData))

	// Write to file if specified
	if filename != "" {
		if err := os.WriteFile(filename, jsonData, 0644); err != nil {
			t.Logf("Error writing debug info to file %s: %v", filename, err)
		} else {
			t.Logf("Debug info written to %s", filename)
		}
	}
}

// DebugContext provides a context for collecting debug information throughout a test.
type DebugContext struct {
	t           *testing.T
	logger      *VerboseLogger
	checkpoints []DebugCheckpoint
	startTime   time.Time
	mu          sync.RWMutex
}

// DebugCheckpoint represents a point-in-time snapshot of debug information.
type DebugCheckpoint struct {
	Name        string                 `json:"name"`
	Timestamp   time.Time              `json:"timestamp"`
	Goroutines  int                    `json:"goroutines"`
	MemoryAlloc uint64                 `json:"memoryAlloc"`
	CustomData  map[string]interface{} `json:"customData,omitempty"`
}

// NewDebugContext creates a new debug context for comprehensive test debugging.
//
// Usage:
//
//	debugCtx := NewDebugContext(t)
//	debugCtx.Checkpoint("start", nil)
//	// ... test operations ...
//	debugCtx.Checkpoint("middle", map[string]interface{}{"connections": 5})
//	// ... more operations ...
//	debugCtx.Checkpoint("end", nil)
func NewDebugContext(t *testing.T) *DebugContext {
	t.Helper()

	ctx := &DebugContext{
		t:           t,
		logger:      NewVerboseLogger(t),
		checkpoints: make([]DebugCheckpoint, 0),
		startTime:   time.Now(),
	}

	// Register cleanup to dump debug info on failure
	t.Cleanup(func() {
		if t.Failed() {
			ctx.DumpCheckpoints()
		}
	})

	return ctx
}

// Logger returns the verbose logger for this debug context.
func (dc *DebugContext) Logger() *VerboseLogger {
	return dc.logger
}

// Checkpoint creates a debug checkpoint with the given name and custom data.
func (dc *DebugContext) Checkpoint(name string, customData map[string]interface{}) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	checkpoint := DebugCheckpoint{
		Name:        name,
		Timestamp:   time.Now(),
		Goroutines:  runtime.NumGoroutine(),
		MemoryAlloc: memStats.Alloc,
		CustomData:  customData,
	}

	dc.checkpoints = append(dc.checkpoints, checkpoint)
	dc.logger.LogStruct("CHECKPOINT", fmt.Sprintf("Checkpoint: %s", name), checkpoint)
}

// DumpCheckpoints outputs all checkpoints to the test log.
func (dc *DebugContext) DumpCheckpoints() {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	if len(dc.checkpoints) == 0 {
		dc.t.Log("No debug checkpoints recorded")
		return
	}

	dc.t.Log("=== Debug Checkpoints ===")
	for i, checkpoint := range dc.checkpoints {
		duration := checkpoint.Timestamp.Sub(dc.startTime)
		dc.t.Logf("Checkpoint %d: %s (at %v)", i+1, checkpoint.Name, duration)
		dc.t.Logf("  Goroutines: %d", checkpoint.Goroutines)
		dc.t.Logf("  Memory Alloc: %d bytes", checkpoint.MemoryAlloc)

		if checkpoint.CustomData != nil {
			if jsonData, err := json.MarshalIndent(checkpoint.CustomData, "  ", "  "); err == nil {
				dc.t.Logf("  Custom Data:\n  %s", string(jsonData))
			}
		}
	}
	dc.t.Log("=== End Checkpoints ===")
}

// GetCheckpoints returns all recorded checkpoints.
func (dc *DebugContext) GetCheckpoints() []DebugCheckpoint {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	// Return a copy to prevent modification
	checkpoints := make([]DebugCheckpoint, len(dc.checkpoints))
	copy(checkpoints, dc.checkpoints)
	return checkpoints
}

// WatchResource starts monitoring a resource and logs changes.
// This is useful for tracking resource usage over time.
func (dc *DebugContext) WatchResource(name string, getter func() interface{}, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				value := getter()
				dc.logger.LogStruct("RESOURCE", fmt.Sprintf("Resource: %s", name), map[string]interface{}{
					"name":  name,
					"value": value,
					"time":  time.Now(),
				})
			}
		}
	}()

	return cancel
}

// ErrorCollector collects and categorizes errors that occur during testing.
type ErrorCollector struct {
	errors []CategorizedError
	mu     sync.RWMutex
}

// CategorizedError represents an error with additional context.
type CategorizedError struct {
	Error      error                  `json:"error"`
	Category   string                 `json:"category"`
	Timestamp  time.Time              `json:"timestamp"`
	StackTrace string                 `json:"stackTrace"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// NewErrorCollector creates a new error collector.
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]CategorizedError, 0),
	}
}

// CollectError adds an error to the collection with categorization.
func (ec *ErrorCollector) CollectError(err error, category string, context map[string]interface{}) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	// Capture stack trace
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	stackTrace := string(buf[:n])

	categorizedErr := CategorizedError{
		Error:      err,
		Category:   category,
		Timestamp:  time.Now(),
		StackTrace: stackTrace,
		Context:    context,
	}

	ec.errors = append(ec.errors, categorizedErr)
}

// GetErrors returns all collected errors.
func (ec *ErrorCollector) GetErrors() []CategorizedError {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// Return a copy
	errors := make([]CategorizedError, len(ec.errors))
	copy(errors, ec.errors)
	return errors
}

// GetErrorsByCategory returns errors filtered by category.
func (ec *ErrorCollector) GetErrorsByCategory(category string) []CategorizedError {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var filtered []CategorizedError
	for _, err := range ec.errors {
		if err.Category == category {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// HasErrors returns true if any errors have been collected.
func (ec *ErrorCollector) HasErrors() bool {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	return len(ec.errors) > 0
}

// DumpErrors outputs all collected errors to the provided writer.
func (ec *ErrorCollector) DumpErrors(w io.Writer) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	if len(ec.errors) == 0 {
		fmt.Fprintln(w, "No errors collected")
		return
	}

	fmt.Fprintf(w, "=== Collected Errors (%d total) ===\n", len(ec.errors))
	for i, err := range ec.errors {
		fmt.Fprintf(w, "Error %d [%s]:\n", i+1, err.Category)
		fmt.Fprintf(w, "  Time: %s\n", err.Timestamp.Format(time.RFC3339))
		fmt.Fprintf(w, "  Error: %v\n", err.Error)

		if err.Context != nil {
			if jsonData, jerr := json.MarshalIndent(err.Context, "  ", "  "); jerr == nil {
				fmt.Fprintf(w, "  Context:\n  %s\n", string(jsonData))
			}
		}

		if err.StackTrace != "" {
			fmt.Fprintf(w, "  Stack trace:\n%s\n", err.StackTrace)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "=== End Collected Errors ===")
}

// SafeLogger provides a logger that won't panic during test cleanup.
// It's useful when you need logging during potentially unstable cleanup phases.
type SafeLogger struct {
	logger *log.Logger
	prefix string
}

// NewSafeLogger creates a new safe logger with the given prefix.
func NewSafeLogger(prefix string) *SafeLogger {
	return &SafeLogger{
		logger: log.New(os.Stderr, prefix, log.LstdFlags|log.Lshortfile),
		prefix: prefix,
	}
}

// Log safely logs a message, recovering from any panics.
func (sl *SafeLogger) Log(format string, args ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			// If regular logging fails, try direct output
			fmt.Fprintf(os.Stderr, "[SAFE] %s: %s\n", sl.prefix, fmt.Sprintf(format, args...))
		}
	}()

	sl.logger.Printf(format, args...)
}
