package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
)

func main() {
	// Create a simple buffer to capture what would be sent
	var buf bytes.Buffer

	// Create a test message
	msg := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test","version":"1.0.0"},"capabilities":{}}}`

	// Simulate what RawFramer should do
	buf.Write([]byte(msg))

	// Check if it adds a newline
	fmt.Printf("Buffer contents: %q\n", buf.String())
	fmt.Printf("Buffer length: %d\n", buf.Len())

	// Check last character
	if buf.Len() > 0 {
		lastChar := buf.Bytes()[buf.Len()-1]
		fmt.Printf("Last character: %q (0x%x)\n", lastChar, lastChar)
	}

	// Write to stdout to see what gets sent
	log.Println("Writing to stdout...")
	os.Stdout.Write(buf.Bytes())
	fmt.Println() // Add explicit newline
}
