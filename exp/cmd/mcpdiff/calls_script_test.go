package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestCallScripts(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "mcpdiff")
	command := exec.Command("go", "build", "-o", binary, ".")
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	scripttest.Test(t, context.Background(), &script.Engine{Cmds: scripttest.DefaultCmds(), Conds: scripttest.DefaultConds()}, append(os.Environ(), "MCPDIFF="+binary), "testdata/calls/*.txt")
}
