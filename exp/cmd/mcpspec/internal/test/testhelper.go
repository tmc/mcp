package test

import (
	"os"
	"testing"
)

// ReadFile is a helper function to read a file in the test environment.
func ReadFile(t *testing.T, path string) ([]byte, error) {
	return os.ReadFile(path)
}
