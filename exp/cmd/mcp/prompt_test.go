package main

import "testing"

func TestParsePromptArgs(t *testing.T) {
	args, err := parsePromptArgs([]string{"name=mesh", "enabled=true", "count=3"}, `{"mode":"fast"}`, false)
	if err != nil {
		t.Fatal(err)
	}
	if args["name"] != "mesh" {
		t.Fatalf("name=%v", args["name"])
	}
	if args["enabled"] != true {
		t.Fatalf("enabled=%v", args["enabled"])
	}
	if args["count"] != int64(3) {
		t.Fatalf("count=%T %v", args["count"], args["count"])
	}
	if args["mode"] != "fast" {
		t.Fatalf("mode=%v", args["mode"])
	}
}

func TestParsePromptArgsRejectsBadPair(t *testing.T) {
	if _, err := parsePromptArgs([]string{"bad"}, "", false); err == nil {
		t.Fatal("expected error")
	}
}
