package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNormalize(t *testing.T) {
	input := "mcp-send {\"id\":2} # 200.500\nmcp-recv {\"id\":1} # 200.750\n"
	var got bytes.Buffer
	if err := normalize(strings.NewReader(input), &got, 2*time.Second, false); err != nil {
		t.Fatal(err)
	}
	want := "mcp-send {\"id\":2} # 2.000\nmcp-recv {\"id\":1} # 2.250\n"
	if got.String() != want {
		t.Fatalf("normalize() = %q, want %q", got.String(), want)
	}
}

func TestNormalizeSortsStably(t *testing.T) {
	input := "mcp-send {\"id\":2} # 3.000\nmcp-send {\"id\":1} # 1.000\n"
	var got bytes.Buffer
	if err := normalize(strings.NewReader(input), &got, 0, true); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.String(), "mcp-send {\"id\":1}") {
		t.Fatalf("normalize() did not sort records: %q", got.String())
	}
}
