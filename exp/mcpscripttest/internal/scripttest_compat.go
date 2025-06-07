package internal

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/txtar"
	"rsc.io/script"
	"rsc.io/script/scripttest"
)

// synctestCompatibleTest runs scripttest but skips deadline logic under synctest
func synctestCompatibleTest(t *testing.T, ctx context.Context, engine *script.Engine, env []string, pattern string) {
	// Under synctest, skip the deadline logic in scripttest.Test that causes issues
	// The problem is t.Deadline() returns real time but time.Until() uses synthetic time
	if hasSynctest {
		testWithoutDeadline(t, ctx, engine, env, pattern)
	} else {
		scripttest.Test(t, ctx, engine, env, pattern)
	}
}

// testWithoutDeadline is a copy of scripttest.Test but without the deadline logic
func testWithoutDeadline(t *testing.T, ctx context.Context, engine *script.Engine, env []string, pattern string) {
	files, _ := filepath.Glob(pattern)
	if len(files) == 0 {
		t.Fatal("no testdata")
	}
	
	for _, file := range files {
		file := file
		name := strings.TrimSuffix(filepath.Base(file), ".txt")
		t.Run(name, func(t *testing.T) {
			workdir := t.TempDir()
			s, err := script.NewState(ctx, workdir, env)
			if err != nil {
				t.Fatal(err)
			}

			// Unpack archive
			a, err := txtar.ParseFile(file)
			if err != nil {
				t.Fatal(err)
			}
			initScriptDirs(t, s)
			if err := s.ExtractFiles(a); err != nil {
				t.Fatal(err)
			}

			t.Log(time.Now().UTC().Format(time.RFC3339))
			work, _ := s.LookupEnv("WORK")
			t.Logf("$WORK=%s", work)

			scripttest.Run(t, engine, s, file, bytes.NewReader(a.Comment))
		})
	}
}

// Copy of scripttest.initScriptDirs to avoid import issues
func initScriptDirs(t testing.TB, s *script.State) {
	must := func(err error) {
		if err != nil {
			t.Helper()
			t.Fatal(err)
		}
	}

	work := s.Getwd()
	must(s.Setenv("WORK", work))
	must(os.MkdirAll(filepath.Join(work, "tmp"), 0777))
	must(s.Setenv(tempEnvName(), filepath.Join(work, "tmp")))
}

func tempEnvName() string {
	switch runtime.GOOS {
	case "windows":
		return "TMP"
	case "plan9":
		return "TMPDIR"
	default:
		return "TMPDIR"
	}
}