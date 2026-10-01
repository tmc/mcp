package main

import (
	"strings"
	"testing"
)

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
}
func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}
func assertNotEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got == want {
		t.Fatalf("got %v, want values to differ", got)
	}
}
func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("%q does not contain %q", got, want)
	}
}

type tempDir struct{ Path string }

func newTempDir(t *testing.T) *tempDir { t.Helper(); return &tempDir{Path: t.TempDir()} }
func (*tempDir) Cleanup()              {}
