package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/tmc/mcp/internal/mcptracediff"
)

func diffCalls(left, right string, out, stderr io.Writer, opts mcptracediff.Options, jsonOutput bool) int {
	a, err := os.Open(left)
	if err != nil {
		fmt.Fprintln(stderr, "mcpdiff:", err)
		return 2
	}
	defer a.Close()
	b, err := os.Open(right)
	if err != nil {
		fmt.Fprintln(stderr, "mcpdiff:", err)
		return 2
	}
	defer b.Close()
	differences, err := mcptracediff.Compare(a, b, opts)
	if err != nil {
		fmt.Fprintln(stderr, "mcpdiff:", err)
		return 2
	}
	if jsonOutput {
		if err := json.NewEncoder(out).Encode(differences); err != nil {
			fmt.Fprintln(stderr, "mcpdiff:", err)
			return 2
		}
	} else {
		for _, d := range differences {
			if _, err := fmt.Fprintf(out, "%s %s session=%q %s left=%d right=%d: %s\n", d.Direction, d.Method, d.Session, d.Kind, d.LeftRecord, d.RightRecord, d.Detail); err != nil {
				fmt.Fprintln(stderr, "mcpdiff:", err)
				return 2
			}
			if d.Kind == "result" {
				if _, err := fmt.Fprintf(out, "- %s\n+ %s\n", d.LeftResponse, d.RightResponse); err != nil {
					fmt.Fprintln(stderr, "mcpdiff:", err)
					return 2
				}
			}
		}
	}
	if len(differences) > 0 {
		return 1
	}
	return 0
}
