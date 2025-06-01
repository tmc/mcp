package io

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReader(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "mcpspec-io-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test content to the file
	testContent := "line1\nline2\nline3"
	if _, err := tempFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	t.Run("read from file", func(t *testing.T) {
		reader, err := NewFileReader(tempFile.Name())
		if err != nil {
			t.Fatalf("NewFileReader() error = %v", err)
		}
		defer reader.Close()

		content, err := reader.Read()
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if string(content) != testContent {
			t.Errorf("Read() = %v, want %v", string(content), testContent)
		}
	})

	t.Run("read line from file", func(t *testing.T) {
		reader, err := NewFileReader(tempFile.Name())
		if err != nil {
			t.Fatalf("NewFileReader() error = %v", err)
		}
		defer reader.Close()

		line, err := reader.ReadLine()
		if err != nil {
			t.Fatalf("ReadLine() error = %v", err)
		}

		if line != "line1" {
			t.Errorf("ReadLine() = %v, want %v", line, "line1")
		}
	})

	t.Run("read from stdin", func(t *testing.T) {
		// Save original stdin
		oldStdin := os.Stdin
		// Create a pipe
		r, w, _ := os.Pipe()
		// Set stdin to the read end
		os.Stdin = r
		// Write test input to the write end
		go func() {
			w.Write([]byte(testContent))
			w.Close()
		}()
		// Defer restoring stdin
		defer func() { os.Stdin = oldStdin }()

		reader, err := NewFileReader("")
		if err != nil {
			t.Fatalf("NewFileReader() error = %v", err)
		}
		defer reader.Close()

		content, err := reader.Read()
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if string(content) != testContent {
			t.Errorf("Read() = %v, want %v", string(content), testContent)
		}
	})

	t.Run("read from nonexistent file", func(t *testing.T) {
		_, err := NewFileReader("/this/file/does/not/exist.txt")
		if err == nil {
			t.Errorf("NewFileReader() error = nil, want error")
		}
	})
}

func TestFileWriter(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "mcpspec-io-test-dir")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testContent := "test content"
	testJson := map[string]string{"key": "value"}

	t.Run("write to file", func(t *testing.T) {
		tempFilePath := filepath.Join(tempDir, "test-write.txt")
		writer, err := NewFileWriter(tempFilePath)
		if err != nil {
			t.Fatalf("NewFileWriter() error = %v", err)
		}
		defer writer.Close()

		n, err := writer.Write([]byte(testContent))
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if n != len(testContent) {
			t.Errorf("Write() = %v, want %v", n, len(testContent))
		}

		// Verify file content
		content, err := os.ReadFile(tempFilePath)
		if err != nil {
			t.Fatalf("Failed to read temp file: %v", err)
		}
		if string(content) != testContent {
			t.Errorf("File content = %v, want %v", string(content), testContent)
		}
	})

	t.Run("write line to file", func(t *testing.T) {
		tempFilePath := filepath.Join(tempDir, "test-writeline.txt")
		writer, err := NewFileWriter(tempFilePath)
		if err != nil {
			t.Fatalf("NewFileWriter() error = %v", err)
		}
		defer writer.Close()

		n, err := writer.WriteLine(testContent)
		if err != nil {
			t.Fatalf("WriteLine() error = %v", err)
		}
		if n != len(testContent)+1 { // +1 for newline
			t.Errorf("WriteLine() = %v, want %v", n, len(testContent)+1)
		}

		// Verify file content
		content, err := os.ReadFile(tempFilePath)
		if err != nil {
			t.Fatalf("Failed to read temp file: %v", err)
		}
		if string(content) != testContent+"\n" {
			t.Errorf("File content = %v, want %v", string(content), testContent+"\n")
		}
	})

	t.Run("write JSON to file (no indent)", func(t *testing.T) {
		tempFilePath := filepath.Join(tempDir, "test-json.json")
		writer, err := NewFileWriter(tempFilePath)
		if err != nil {
			t.Fatalf("NewFileWriter() error = %v", err)
		}
		defer writer.Close()

		err = writer.WriteJSON(testJson, false)
		if err != nil {
			t.Fatalf("WriteJSON() error = %v", err)
		}

		// Verify file content
		content, err := os.ReadFile(tempFilePath)
		if err != nil {
			t.Fatalf("Failed to read temp file: %v", err)
		}
		expectedJSON := `{"key":"value"}` + "\n"
		if string(content) != expectedJSON {
			t.Errorf("File content = %v, want %v", string(content), expectedJSON)
		}
	})

	t.Run("write JSON to file (with indent)", func(t *testing.T) {
		tempFilePath := filepath.Join(tempDir, "test-json-indent.json")
		writer, err := NewFileWriter(tempFilePath)
		if err != nil {
			t.Fatalf("NewFileWriter() error = %v", err)
		}
		defer writer.Close()

		err = writer.WriteJSON(testJson, true)
		if err != nil {
			t.Fatalf("WriteJSON() error = %v", err)
		}

		// Verify file content
		content, err := os.ReadFile(tempFilePath)
		if err != nil {
			t.Fatalf("Failed to read temp file: %v", err)
		}
		expectedJSON := "{\n  \"key\": \"value\"\n}\n"
		if string(content) != expectedJSON {
			t.Errorf("File content = %v, want %v", string(content), expectedJSON)
		}
	})

	t.Run("write to stdout", func(t *testing.T) {
		// Save original stdout
		oldStdout := os.Stdout
		// Create a pipe
		r, w, _ := os.Pipe()
		// Set stdout to the write end
		os.Stdout = w

		writer, err := NewFileWriter("")
		if err != nil {
			t.Fatalf("NewFileWriter() error = %v", err)
		}

		_, err = writer.Write([]byte(testContent))
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Close the write end of the pipe to flush
		w.Close()
		// Restore stdout
		os.Stdout = oldStdout

		// Read from the pipe
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if buf.String() != testContent {
			t.Errorf("Stdout = %v, want %v", buf.String(), testContent)
		}
	})

	t.Run("create nested directories", func(t *testing.T) {
		nestedPath := filepath.Join(tempDir, "nested/dirs/test.txt")
		writer, err := NewFileWriter(nestedPath)
		if err != nil {
			t.Fatalf("NewFileWriter() error = %v", err)
		}
		defer writer.Close()

		// Verify directories were created
		dirPath := filepath.Join(tempDir, "nested/dirs")
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dirPath)
		}
	})
}

func TestTeeReader(t *testing.T) {
	testContent := "test tee reader content"
	reader := strings.NewReader(testContent)
	var buf bytes.Buffer

	teeReader := TeeReader(reader, &buf)

	// Read from the tee reader
	data, err := io.ReadAll(teeReader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Verify that the data was read correctly
	if string(data) != testContent {
		t.Errorf("ReadAll() = %v, want %v", string(data), testContent)
	}

	// Verify that the data was also written to the buffer
	if buf.String() != testContent {
		t.Errorf("Buffer = %v, want %v", buf.String(), testContent)
	}
}
