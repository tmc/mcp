package testing

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

var (
	logMu   sync.Mutex
	logFile *os.File
	once    sync.Once
)

// TestLogger wraps a testing.T to provide additional logging to a file
type TestLogger struct {
	T      *testing.T
	prefix string
}

// initLogFile ensures the log file is initialized exactly once
func initLogFile() {
	once.Do(func() {
		var err error
		logFile, err = os.OpenFile("t.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// Fall back to stderr if we can't open the log file
			fmt.Fprintf(os.Stderr, "Error opening t.log: %v\n", err)
			return
		}

		// Write a header to the log file
		timestamp := time.Now().Format(time.RFC3339)
		fmt.Fprintf(logFile, "\n\n--- TEST RUN STARTED AT %s ---\n\n", timestamp)
	})
}

// NewTestLogger creates a new TestLogger that wraps t
func NewTestLogger(t *testing.T) *TestLogger {
	initLogFile()
	return &TestLogger{
		T:      t,
		prefix: t.Name() + ": ",
	}
}

// Log logs to both the testing.T and the log file
func (tl *TestLogger) Log(args ...interface{}) {
	// First log to the testing.T
	tl.T.Log(args...)

	// Then log to the file
	if logFile != nil {
		logMu.Lock()
		defer logMu.Unlock()

		fmt.Fprintf(logFile, "%s %s", tl.prefix, fmt.Sprintln(args...))
	}
}

// Logf logs to both the testing.T and the log file
func (tl *TestLogger) Logf(format string, args ...interface{}) {
	// First log to the testing.T
	tl.T.Logf(format, args...)

	// Then log to the file
	if logFile != nil {
		logMu.Lock()
		defer logMu.Unlock()

		fmt.Fprintf(logFile, "%s %s\n", tl.prefix, fmt.Sprintf(format, args...))
	}
}

// Error logs an error to both the testing.T and the log file
func (tl *TestLogger) Error(args ...interface{}) {
	// First log to the testing.T
	tl.T.Error(args...)

	// Then log to the file
	if logFile != nil {
		logMu.Lock()
		defer logMu.Unlock()

		fmt.Fprintf(logFile, "%s ERROR: %s", tl.prefix, fmt.Sprintln(args...))
	}
}

// Errorf logs an error to both the testing.T and the log file
func (tl *TestLogger) Errorf(format string, args ...interface{}) {
	// First log to the testing.T
	tl.T.Errorf(format, args...)

	// Then log to the file
	if logFile != nil {
		logMu.Lock()
		defer logMu.Unlock()

		fmt.Fprintf(logFile, "%s ERROR: %s\n", tl.prefix, fmt.Sprintf(format, args...))
	}
}

// Fatal logs a fatal error to both the testing.T and the log file
func (tl *TestLogger) Fatal(args ...interface{}) {
	// Log to the file first since Fatal will exit
	if logFile != nil {
		logMu.Lock()
		fmt.Fprintf(logFile, "%s FATAL: %s", tl.prefix, fmt.Sprintln(args...))
		logMu.Unlock()
	}

	// Then log to the testing.T (which will exit)
	tl.T.Fatal(args...)
}

// Fatalf logs a fatal error to both the testing.T and the log file
func (tl *TestLogger) Fatalf(format string, args ...interface{}) {
	// Log to the file first since Fatal will exit
	if logFile != nil {
		logMu.Lock()
		fmt.Fprintf(logFile, "%s FATAL: %s\n", tl.prefix, fmt.Sprintf(format, args...))
		logMu.Unlock()
	}

	// Then log to the testing.T (which will exit)
	tl.T.Fatalf(format, args...)
}

// Skip logs a skip message to both the testing.T and the log file
func (tl *TestLogger) Skip(args ...interface{}) {
	// Log to the file first since Skip will exit this test
	if logFile != nil {
		logMu.Lock()
		fmt.Fprintf(logFile, "%s SKIP: %s", tl.prefix, fmt.Sprintln(args...))
		logMu.Unlock()
	}

	// Then log to the testing.T
	tl.T.Skip(args...)
}

// Skipf logs a skip message to both the testing.T and the log file
func (tl *TestLogger) Skipf(format string, args ...interface{}) {
	// Log to the file first since Skip will exit this test
	if logFile != nil {
		logMu.Lock()
		fmt.Fprintf(logFile, "%s SKIP: %s\n", tl.prefix, fmt.Sprintf(format, args...))
		logMu.Unlock()
	}

	// Then log to the testing.T
	tl.T.Skipf(format, args...)
}

// CapturePanicHandler runs f and captures any panics, logging them to both testing.T and the log file
func (tl *TestLogger) CapturePanicHandler(f func()) {
	defer func() {
		if r := recover(); r != nil {
			tl.Errorf("Panic captured: %v", r)
		}
	}()
	f()
}

// CloseLogFile closes the log file if it's open
func CloseLogFile() {
	logMu.Lock()
	defer logMu.Unlock()

	if logFile != nil {
		// Write a footer to the log file
		timestamp := time.Now().Format(time.RFC3339)
		fmt.Fprintf(logFile, "\n--- TEST RUN COMPLETED AT %s ---\n", timestamp)

		logFile.Close()
		logFile = nil
	}
}
