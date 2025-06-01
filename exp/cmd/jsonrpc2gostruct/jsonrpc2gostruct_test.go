// Package main implements tests for the jsonrpc2gostruct tool
package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"jsonrpc2gostruct": func() int {
			// This function behaves like main() for the command being tested
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = []string{"jsonrpc2gostruct"}
			os.Args = append(os.Args, os.Args[1:]...)
			main()
			return 0
		},
	}))
}

func TestJsonRpc2GoStruct(t *testing.T) {
	// Ensure we have the tool built
	cmd := exec.Command("go", "build")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build tool: %v\n%s", err, out)
	}

	// Run the tests
	testscript.Run(t, testscript.Params{
		Dir:           "testdata",
		UpdateScripts: false, // Set to true to update scripts
	})
}
