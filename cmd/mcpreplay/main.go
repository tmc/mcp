package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

var (
	dryRun = flag.Bool("n", false, "print messages instead of sending them")
	delay  = flag.Duration("d", 0, "delay between messages")
	debug  = flag.Bool("debug", false, "enable debug logging")
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
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] recording\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nReplays recorded MCP messages to stdout.\n")
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	if *debug {
		log.Printf("Reading recording from: %s", flag.Arg(0))
		if *dryRun {
			log.Printf("Dry run mode enabled")
		}
		if *delay > 0 {
			log.Printf("Using delay: %v", *delay)
		}
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	entries, err := LoadRecording(f)
	if err != nil {
		log.Fatal(err)
	}

	if *debug {
		log.Printf("Found %d entries in recording", len(entries))
	}

	for i, e := range entries {
		if e.Dir != "in" {
			if *debug {
				log.Printf("Skipping entry %d: %s", i, e.Dir)
			}
			continue
		}

		if *debug {
			log.Printf("Processing entry %d: %s", i, e.Dir)
		}

		if *dryRun {
			fmt.Printf("Would send: %s\n", e.Data)
			continue
		}

		// Pretty print the JSON if in debug mode
		if *debug {
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, e.Data, "", "  "); err == nil {
				log.Printf("Sending:\n%s", prettyJSON.String())
			} else {
				log.Printf("Sending: %s", e.Data)
			}
		}

		if _, err := fmt.Printf("%s\n", e.Data); err != nil {
			log.Fatalf("Error sending message: %v", err)
		}

		if *delay > 0 {
			time.Sleep(*delay)
		}
	}

	if *debug {
		log.Printf("Replay complete")
	}
}
