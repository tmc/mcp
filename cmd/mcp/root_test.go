package main

import "testing"

func TestRootURI(t *testing.T) {
	got := rootURI(".")
	if got == "." {
		t.Fatal("expected absolute file URI")
	}
	if got[:7] != "file://" {
		t.Fatalf("uri=%q", got)
	}
}

func TestRootURILeavesExplicitURI(t *testing.T) {
	if got := rootURI("file:///tmp/demo"); got != "file:///tmp/demo" {
		t.Fatalf("uri=%q", got)
	}
}
