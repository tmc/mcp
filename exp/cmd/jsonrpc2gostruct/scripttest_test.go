package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestScripts(t *testing.T) {
	// First build the tool
	cmd := exec.Command("go", "build")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build tool: %v", err)
	}

	dir := t.TempDir()
	testdata := filepath.Join(dir, "testdata")
	if err := os.MkdirAll(testdata, 0755); err != nil {
		t.Fatal(err)
	}
	e := script.NewEngine()
	e.Cmds["jsonrpc2gostruct"] = script.Command(script.CmdUsage{
		Summary: "jsonrpc2gostruct",
		//Usage:   "jsonrpc2gostruct [flags] <jsonrpc2 file>",
		Args: []string{"jsonrpc2 file"},
		//RegexpArgs : []string{"jsonrpc2 file"},
	}, func(args []string) error {
		if len(args) != 1 {
			return script.NewError("jsonrpc2gostruct: missing jsonrpc2 file")
		}
		jsonrpc2File := args[0]
		outputFile := e.FlagValue("output").String()
		pkgName := e.FlagValue("package").String()
		jsonrpc2FileFlag := e.FlagValue("jsonrpc2").String()

		cmd := exec.Command("jsonrpc2gostruct", "-o", outputFile, "-p", pkgName, jsonrpc2FileFlag)
		cmd.Dir = testdata
		if err := cmd.Run(); err != nil {
			return err
		}
		return nil
	})
	scripttest.Test(t, e, nil, "testdata/*.txt")
}
