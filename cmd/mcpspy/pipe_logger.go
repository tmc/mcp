package main

import (
	"io"
	"os"
	"sync"
)

// handlePipeMode sets up mcpspy in pipe logger mode
func handlePipeMode(writer io.Writer, verbose, veryVerbose, prettyJSON bool, indent int, indentChar string, passThrough bool, forceUnbuffered bool) {
	// Create a WaitGroup to ensure all goroutines finish
	var wg sync.WaitGroup
	wg.Add(1)

	// Handle stdin → stdout (client input)
	go func() {
		defer wg.Done()

		// Standard directions for a pipe
		inDir := "recv"  // stdin → mcpspy is received (recv)
		outDir := "send" // mcpspy → stdout is sent (send)

		r := &teeReader{
			r:           os.Stdin,
			log:         writer,
			dir:         inDir,
			verbose:     verbose,
			veryVerbose: veryVerbose,
			prettyJSON:  prettyJSON,
			indentLevel: indent,
			indentChar:  indentChar,
			passThrough: passThrough,
		}
		w := &teeWriter{
			w:           os.Stdout,
			log:         nil,
			dir:         outDir,
			verbose:     false,
			veryVerbose: false,
			prettyJSON:  prettyJSON,
			indentLevel: indent,
			indentChar:  indentChar,
			passThrough: passThrough,
		}
		io.Copy(w, r)
	}()

	// Wait for all goroutines to finish
	wg.Wait()
}
