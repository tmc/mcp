package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestScripts(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "mcpinspect")
	command := exec.Command("go", "build", "-o", binary, ".")
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build mcpinspect: %v\n%s", err, output)
	}
	engine := &script.Engine{Cmds: scripttest.DefaultCmds(), Conds: scripttest.DefaultConds()}
	engine.Cmds["inspectstdin"] = script.Command(script.CmdUsage{Summary: "inspect a trace on stdin", Args: "file"}, func(s *script.State, args ...string) (script.WaitFunc, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected one input file")
		}
		data, err := os.ReadFile(s.Path(args[0]))
		if err != nil {
			return nil, err
		}
		return func(*script.State) (string, string, error) {
			cmd := exec.CommandContext(s.Context(), binary)
			cmd.Stdin = bytes.NewReader(data)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			return stdout.String(), stderr.String(), err
		}, nil
	})
	scripttest.Test(t, context.Background(), engine, append(os.Environ(), "MCPINSPECT="+binary), "testdata/*.txt")
}
