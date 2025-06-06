package sourcereflect_test

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

var updateFlag = flag.Bool("update", false, "update test expectations")

func TestScriptSimple(t *testing.T) {
	// Build the binary once
	tempDir := t.TempDir()
	sourcereflectPath := filepath.Join(tempDir, "sourcereflect")

	cmd := exec.Command("go", "build", "-o", sourcereflectPath, "./cmd/sourcereflect")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build sourcereflect: %v\n%s", err, output)
	}

	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			// Copy the binary to the test environment
			env.Setenv("PATH", tempDir+":"+env.Getenv("PATH"))
			return nil
		},
	})
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
