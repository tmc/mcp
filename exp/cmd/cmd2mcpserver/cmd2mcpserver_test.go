package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"cmd2mcpserver": main,
	})
}

func TestScript(t *testing.T) {
	toolDir := t.TempDir()
	toolPath := filepath.Join(toolDir, "cmd2mcpserver")
	cmd := exec.Command("go", "build", "-o", toolPath, ".")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build cmd2mcpserver: %v\n%s", err, output)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	sourceDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			env.Setenv("GOCACHE", filepath.Join(cache, "go-build"))
			env.Setenv("PATH", toolDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			env.Setenv("CMD2MCPSERVER_SOURCE", sourceDir)
			return nil
		},
	})
}
