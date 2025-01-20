package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/google/go-cmp/cmp"
)

var (
	serverCmd = flag.String("s", "", "server command to run")
	inFile    = flag.String("f", "", "recording file to verify against")
	debug     = flag.Bool("debug", false, "enable debug logging")
)

// Entry represents a recorded MCP message
type Entry struct {
	Dir  string          `json:"dir"`            // "in" or "out"
	Data json.RawMessage `json:"data"`           // The raw message data
	Time time.Time       `json:"time,omitempty"` // When the message was recorded
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
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nVerifies that a server responds correctly to a recorded MCP session.\n")
	}
	flag.Parse()

	if *serverCmd == "" || *inFile == "" {
		flag.Usage()
		log.Fatal("must specify both -s and -f")
	}

	f, err := os.Open(*inFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	entries, err := LoadRecording(f)
	if err != nil {
		log.Fatal(err)
	}

	if *debug {
		log.Printf("Running server command: %s", *serverCmd)
		log.Printf("Verifying against recording: %s", *inFile)
		log.Printf("Found %d entries in recording", len(entries))
	}

	cmd := exec.Command("sh", "-c", *serverCmd)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer cmd.Process.Kill()

	for i, e := range entries {
		if *debug {
			log.Printf("Processing entry %d: %s", i, e.Dir)
		}

		switch e.Dir {
		case "in":
			if *debug {
				log.Printf("Sending: %s", e.Data)
			}
			if _, err := fmt.Fprintf(stdin, "%s\n", e.Data); err != nil {
				log.Fatalf("Error sending input: %v", err)
			}

		case "out":
			if *debug {
				log.Printf("Expecting: %s", e.Data)
			}
			var buf bytes.Buffer
			if _, err := io.CopyN(&buf, stdout, int64(len(e.Data)+1)); err != nil {
				log.Fatalf("Error reading response: %v", err)
			}
			got := bytes.TrimSpace(buf.Bytes())

			if *debug {
				log.Printf("Received: %s", got)
			}

			// Compare as JSON to ignore formatting differences
			var gotJSON, wantJSON interface{}
			if err := json.Unmarshal(got, &gotJSON); err != nil {
				log.Fatalf("Invalid JSON response: %v\nResponse: %s", err, got)
			}
			if err := json.Unmarshal(e.Data, &wantJSON); err != nil {
				log.Fatalf("Invalid JSON in recording: %v\nRecording: %s", err, e.Data)
			}
			if diff := cmp.Diff(wantJSON, gotJSON); diff != "" {
				log.Fatalf("Response mismatch (-want +got):\n%s", diff)
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		if *debug {
			log.Printf("Server exited with error: %v", err)
		}
		log.Fatal("Server exited with error")
	}

	fmt.Println("Verification successful!")
}
