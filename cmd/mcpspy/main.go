package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

var (
	outFile = flag.String("f", "", "output recording file")
	debug   = flag.Bool("debug", false, "enable debug logging")
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

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] command [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *outFile == "" {
		log.Fatal("must specify -f")
	}
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	f, err := os.Create(*outFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if *debug {
		log.Printf("Recording MCP traffic to %s", *outFile)
		log.Printf("Running command: %s %v", flag.Arg(0), flag.Args()[1:])
	}

	// Start the command
	cmd := exec.Command(flag.Arg(0), flag.Args()[1:]...)
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

	// Create tee recorders for stdin and stdout
	teeIn := &teeRecorder{
		r:     os.Stdin,
		w:     stdin,
		log:   f,
		dir:   "in",
		debug: *debug,
	}
	teeOut := &teeRecorder{
		r:     stdout,
		w:     os.Stdout,
		log:   f,
		dir:   "out",
		debug: *debug,
	}

	// Copy stdin to command's stdin and stdout to our stdout
	go io.Copy(teeIn, teeIn)
	io.Copy(teeOut, teeOut)

	if err := cmd.Wait(); err != nil {
		if *debug {
			log.Printf("Command exited with error: %v", err)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

type teeRecorder struct {
	r     io.Reader
	w     io.Writer
	log   io.Writer
	dir   string
	debug bool
}

func (t *teeRecorder) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		// Try to parse as JSON to ensure it's valid
		var raw json.RawMessage
		if err := json.Unmarshal(p[:n], &raw); err == nil {
			entry := Entry{Dir: t.dir, Data: raw}
			if err := entry.WriteTo(t.log); err != nil && t.debug {
				log.Printf("Error writing entry: %v", err)
			}
		} else if t.debug {
			log.Printf("Invalid JSON in %s: %v", t.dir, err)
		}
	}
	return
}

func (t *teeRecorder) Write(p []byte) (n int, err error) {
	n, err = t.w.Write(p)
	if n > 0 {
		// Try to parse as JSON to ensure it's valid
		var raw json.RawMessage
		if err := json.Unmarshal(p[:n], &raw); err == nil {
			entry := Entry{Dir: t.dir, Data: raw}
			if err := entry.WriteTo(t.log); err != nil && t.debug {
				log.Printf("Error writing entry: %v", err)
			}
		} else if t.debug {
			log.Printf("Invalid JSON in %s: %v", t.dir, err)
		}
	}
	return
}
