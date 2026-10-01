package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestScript(t *testing.T) {
	paths, err := filepath.Glob("testdata/*.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			archive := txtar.Parse(data)
			state, err := script.NewState(context.Background(), t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := state.ExtractFiles(archive); err != nil {
				t.Fatal(err)
			}
			engine := script.NewEngine()
			engine.Cmds["mcpcorpus"] = script.Command(script.CmdUsage{Summary: "run mcpcorpus", Args: "exit-code stdin-file args..."}, func(s *script.State, args ...string) (script.WaitFunc, error) {
				if len(args) < 2 {
					return nil, fmt.Errorf("require expected exit and stdin file")
				}
				want, err := strconv.Atoi(args[0])
				if err != nil {
					return nil, err
				}
				input, err := os.ReadFile(s.Path(args[1]))
				if err != nil {
					return nil, err
				}
				command := append([]string(nil), args[2:]...)
				for i, arg := range command {
					if i > 0 && command[i-1] == "-out" {
						command[i] = s.Path(arg)
					} else if strings.HasSuffix(arg, ".mcp") {
						command[i] = s.Path(arg)
					}
				}
				return func(*script.State) (string, string, error) {
					var out, errout bytes.Buffer
					got := run(s.Context(), command, bytes.NewReader(input), &out, &errout)
					var err error
					if got != want {
						err = fmt.Errorf("exit %d, want %d", got, want)
					}
					return out.String(), errout.String(), err
				}, nil
			})
			scripttest.Run(t, engine, state, path, strings.NewReader(string(archive.Comment)))
		})
	}
}
