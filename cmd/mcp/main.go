package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/tmc/mcp/tools/internal/mcp"
)

var (
	outFile = flag.String("f", "", "output recording file")
)

func main() {
	flag.Parse()
	if *outFile == "" {
		log.Fatal("must specify -f")
	}
	if flag.NArg() < 1 {
		log.Fatal("must specify command to run")
	}

	f, err := os.Create(*outFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

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
		r:   os.Stdin,
		w:   stdin,
		log: f,
		dir: "in",
	}
	teeOut := &teeRecorder{
		r:   stdout,
		w:   os.Stdout,
		log: f,
		dir: "out",
	}

	// Copy stdin to command's stdin and stdout to our stdout
	go io.Copy(teeIn, teeIn)
	io.Copy(teeOut, teeOut)

	if err := cmd.Wait(); err != nil {
		log.Printf("Command exited with error: %v", err)
	}
}

type teeRecorder struct {
	r   io.Reader
	w   io.Writer
	log io.Writer
	dir string
}

func (t *teeRecorder) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		entry := mcp.Entry{Dir: t.dir, Data: p[:n]}
		entry.WriteTo(t.log)
	}
	return
}

func (t *teeRecorder) Write(p []byte) (n int, err error) {
	n, err = t.w.Write(p)
	if n > 0 {
		entry := mcp.Entry{Dir: t.dir, Data: p[:n]}
		entry.WriteTo(t.log)
	}
	return
}
