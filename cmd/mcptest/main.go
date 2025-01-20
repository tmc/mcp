package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	update = flag.Bool("update", false, "update golden files")
	debug  = flag.Bool("debug", false, "enable debug logging")
)

// Entry represents a recorded MCP message
type Entry struct {
	Dir  string          `json:"dir"`            // "in" or "out"
	Data json.RawMessage `json:"data"`           // The raw message data
	Time time.Time       `json:"time,omitempty"` // When the message was recorded
}

func (e *Entry) WriteTo(w io.Writer) error {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

// LoadRecording loads a recording from a reader
func LoadRecording(r io.Reader) ([]Entry, error) {
	var entries []Entry
	decoder := json.NewDecoder(r)
	for {
		var entry Entry
		err := decoder.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] [files...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nIf no files are specified, all .txt files in testdata are processed.\n")
	}
	flag.Parse()

	// Default to all .txt files in testdata
	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"testdata/*.txt"}
	}

	var failed bool
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			log.Fatal(err)
		}
		for _, path := range matches {
			if err := runTest(path); err != nil {
				log.Printf("%s: %v", path, err)
				failed = true
			} else if *debug {
				log.Printf("%s: ok", path)
			}
		}
	}
	if failed {
		os.Exit(1)
	}
}

func runTest(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	entries, err := LoadRecording(f)
	if err != nil {
		return fmt.Errorf("loading recording: %v", err)
	}

	goldenPath := strings.TrimSuffix(path, ".txt") + ".golden"
	if *update {
		if *debug {
			log.Printf("Updating golden file: %s", goldenPath)
		}
		f, err := os.Create(goldenPath)
		if err != nil {
			return fmt.Errorf("creating golden file: %v", err)
		}
		defer f.Close()
		for _, e := range entries {
			if err := e.WriteTo(f); err != nil {
				return fmt.Errorf("writing golden file: %v", err)
			}
		}
		return nil
	}

	golden, err := os.Open(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no golden file %s (run with -update to create)", goldenPath)
		}
		return fmt.Errorf("opening golden file: %v", err)
	}
	defer golden.Close()

	goldenEntries, err := LoadRecording(golden)
	if err != nil {
		return fmt.Errorf("loading golden file: %v", err)
	}

	if len(entries) != len(goldenEntries) {
		return fmt.Errorf("got %d entries, want %d", len(entries), len(goldenEntries))
	}

	for i, got := range entries {
		want := goldenEntries[i]
		if got.Dir != want.Dir {
			return fmt.Errorf("entry %d: got dir %q, want %q", i, got.Dir, want.Dir)
		}
		// Compare JSON data ignoring whitespace
		var gotData, wantData interface{}
		if err := json.Unmarshal(got.Data, &gotData); err != nil {
			return fmt.Errorf("entry %d: invalid JSON in recording: %v", i, err)
		}
		if err := json.Unmarshal(want.Data, &wantData); err != nil {
			return fmt.Errorf("entry %d: invalid JSON in golden file: %v", i, err)
		}
		gotJSON, _ := json.Marshal(gotData)
		wantJSON, _ := json.Marshal(wantData)
		if string(gotJSON) != string(wantJSON) {
			return fmt.Errorf("entry %d: got data %s, want %s", i, gotJSON, wantJSON)
		}
	}
	return nil
}

func indent(s string) string {
	return "\t" + strings.ReplaceAll(s, "\n", "\n\t")
}
