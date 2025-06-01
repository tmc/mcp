package testing

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewTestLogger(t *testing.T) {
	logger := NewTestLogger(t)
	if logger == nil {
		t.Fatal("NewTestLogger should not return nil")
	}
	if logger.T != t {
		t.Error("Logger.T should be the same as the passed testing.T")
	}
	if logger.prefix == "" {
		t.Error("Logger should have a non-empty prefix")
	}
	if !strings.Contains(logger.prefix, t.Name()) {
		t.Errorf("Logger prefix should contain test name, got: %s", logger.prefix)
	}
}

func TestTestLoggerLog(t *testing.T) {
	logger := NewTestLogger(t)

	// This should not panic
	logger.Log("Test log message")
	logger.Log("Multiple", "arguments", 123)
}

func TestTestLoggerLogf(t *testing.T) {
	logger := NewTestLogger(t)

	// This should not panic
	logger.Logf("Test log message with format: %s", "formatted")
	logger.Logf("Multiple arguments: %d, %s", 42, "test")
}

func TestTestLoggerError(t *testing.T) {
	// We can't test Error directly because it calls t.Error which would fail the test
	// But we can test that the methods exist and don't panic
	_ = NewTestLogger(t) // Just verify it can be created

	// Test with a sub-test so errors don't fail the main test
	t.Run("sub", func(subT *testing.T) {
		subLogger := NewTestLogger(subT)
		subLogger.Error("This is a test error")
		subLogger.Errorf("This is a formatted error: %s", "test")
	})
}

func TestTestLoggerCapturePanicHandler(t *testing.T) {
	logger := NewTestLogger(t)

	// Test function that doesn't panic
	logger.CapturePanicHandler(func() {
		// Do nothing
	})

	// Test function that does panic
	t.Run("sub", func(subT *testing.T) {
		subLogger := NewTestLogger(subT)
		subLogger.CapturePanicHandler(func() {
			panic("test panic")
		})
	})
}

func TestInitLogFile(t *testing.T) {
	// Call initLogFile multiple times to ensure it only initializes once
	initLogFile()
	initLogFile()
	initLogFile()

	// Check if log file was created
	if _, err := os.Stat("t.log"); os.IsNotExist(err) {
		t.Error("Log file should be created")
	}
}

func TestCloseLogFile(t *testing.T) {
	// Ensure log file is initialized
	initLogFile()

	// Close the log file
	CloseLogFile()

	// This should not panic even if called multiple times
	CloseLogFile()
	CloseLogFile()
}

func TestLogFileContent(t *testing.T) {
	// Clean up any existing log file
	os.Remove("t.log")

	// Force re-initialization for this test
	once.Do(func() {}) // Reset the once
	once = sync.Once{}

	logger := NewTestLogger(t)
	testMessage := "Test message for file content verification"
	logger.Log(testMessage)

	// Give some time for the write to complete
	time.Sleep(10 * time.Millisecond)

	// Check if the log file contains our message
	if content, err := os.ReadFile("t.log"); err == nil {
		if !strings.Contains(string(content), testMessage) {
			t.Errorf("Log file should contain test message: %s", testMessage)
		}
		if !strings.Contains(string(content), "TEST RUN STARTED") {
			t.Error("Log file should contain start header")
		}
	} else {
		t.Errorf("Failed to read log file: %v", err)
	}
}

func TestLoggerPrefix(t *testing.T) {
	logger := NewTestLogger(t)

	expectedPrefix := t.Name() + ": "
	if logger.prefix != expectedPrefix {
		t.Errorf("Logger prefix = %q, want %q", logger.prefix, expectedPrefix)
	}
}

func TestConcurrentLogging(t *testing.T) {
	logger := NewTestLogger(t)

	// Test concurrent logging to ensure thread safety
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Logf("Concurrent log from goroutine %d", id)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
