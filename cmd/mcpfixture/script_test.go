package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestGeneratedScript(t *testing.T) {
	dir := t.TempDir()
	for name, path := range map[string]string{"mcpfixture": ".", "server": "./testdata/server"} {
		command := exec.Command("go", "build", "-o", filepath.Join(dir, name), path)
		command.Env = append(os.Environ(), "GOWORK=off")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", name, err, output)
		}
	}
	const trace = `mcp-send {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"demo","version":"1"}}}
mcp-recv {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-11-25","capabilities":{"tools":{}},"serverInfo":{"name":"demo","version":"1"}}}
mcp-send {"jsonrpc":"2.0","method":"notifications/initialized"}
mcp-send {"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{}}}
mcp-recv {"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"hello"}]}}`
	var generated, stderr bytes.Buffer
	if err := run([]string{"-record", "4"}, strings.NewReader(trace), &generated, &stderr); err != nil {
		t.Fatal(err)
	}
	archive := txtar.Parse(generated.Bytes())
	archive.Comment = append(append([]byte(nil), archive.Comment...), []byte("env MCP_RESPONSE=changed\n! exec mcpfixture check fixture.json -- $MCP_SERVER\nstderr 'observed result changed'\n")...)
	env := append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "MCP_SERVER="+filepath.Join(dir, "server"), "MCP_RESPONSE=")
	state, err := script.NewState(context.Background(), t.TempDir(), env)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.ExtractFiles(archive); err != nil {
		t.Fatal(err)
	}
	scripttest.Run(t, &script.Engine{Cmds: scripttest.DefaultCmds(), Conds: scripttest.DefaultConds()}, state, "generated.txt", bytes.NewReader(archive.Comment))
}

func TestRunErrors(t *testing.T) {
	for _, args := range [][]string{{}, {"-record", "0"}, {"check"}, {"check", "missing", "--", "server"}} {
		var out, stderr bytes.Buffer
		if err := run(args, strings.NewReader(""), &out, &stderr); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	var out, stderr bytes.Buffer
	if err := run([]string{"-h"}, strings.NewReader(""), &out, &stderr); err != nil {
		t.Fatal(err)
	}
}
