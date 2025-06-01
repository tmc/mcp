// Package io provides utilities for I/O operations in MCP tools.
package io

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileReader abstracts reading content from a file or stdin.
type FileReader struct {
	Path   string
	reader io.Reader
}

// NewFileReader creates a new FileReader that reads from the specified path,
// or stdin if the path is empty or "-".
func NewFileReader(path string) (*FileReader, error) {
	if path == "" || path == "-" {
		return &FileReader{
			Path:   "stdin",
			reader: os.Stdin,
		}, nil
	}

	// Check if the path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	return &FileReader{
		Path:   path,
		reader: file,
	}, nil
}

// Read reads all content from the file or stdin.
func (r *FileReader) Read() ([]byte, error) {
	if closer, ok := r.reader.(io.Closer); ok && r.Path != "stdin" {
		defer closer.Close()
	}

	return io.ReadAll(r.reader)
}

// ReadLine reads a single line from the file or stdin.
func (r *FileReader) ReadLine() (string, error) {
	scanner := bufio.NewScanner(r.reader)
	if scanner.Scan() {
		return scanner.Text(), nil
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading line: %w", err)
	}

	return "", io.EOF
}

// Close closes the underlying file if applicable.
func (r *FileReader) Close() error {
	if closer, ok := r.reader.(io.Closer); ok && r.Path != "stdin" {
		return closer.Close()
	}
	return nil
}

// FileWriter abstracts writing content to a file or stdout.
type FileWriter struct {
	Path   string
	writer io.Writer
}

// NewFileWriter creates a new FileWriter that writes to the specified path,
// or stdout if the path is empty or "-".
func NewFileWriter(path string) (*FileWriter, error) {
	if path == "" || path == "-" {
		return &FileWriter{
			Path:   "stdout",
			writer: os.Stdout,
		}, nil
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("error creating directories: %w", err)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}

	return &FileWriter{
		Path:   path,
		writer: file,
	}, nil
}

// Write writes data to the file or stdout.
func (w *FileWriter) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

// WriteLine writes a line to the file or stdout.
func (w *FileWriter) WriteLine(line string) (int, error) {
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	return w.writer.Write([]byte(line))
}

// WriteJSON writes JSON data to the file or stdout with optional indentation.
func (w *FileWriter) WriteJSON(v interface{}, indent bool) error {
	var bytes []byte
	var err error

	if indent {
		bytes, err = json.MarshalIndent(v, "", "  ")
	} else {
		bytes, err = json.Marshal(v)
	}

	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	_, err = w.writer.Write(bytes)
	if err != nil {
		return fmt.Errorf("error writing JSON: %w", err)
	}

	// Add a newline for better formatting
	_, err = w.writer.Write([]byte("\n"))
	return err
}

// Close closes the underlying file if applicable.
func (w *FileWriter) Close() error {
	if closer, ok := w.writer.(io.Closer); ok && w.Path != "stdout" {
		return closer.Close()
	}
	return nil
}

// TeeReader creates a reader that will read from the provided reader and also write to the writer.
// This is useful for logging or spying on traffic.
func TeeReader(r io.Reader, w io.Writer) io.Reader {
	return io.TeeReader(r, w)
}
