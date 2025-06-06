package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define command-line flags
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	format := flag.String("format", "text", "Output format (text, json, xml)")
	count := flag.Int("count", 1, "Number of times to repeat")
	timeout := flag.Float64("timeout", 30.0, "Operation timeout in seconds")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <message>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Check for message argument
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: message required")
		flag.Usage()
		os.Exit(1)
	}

	message := flag.Arg(0)

	// Process based on flags
	for i := 0; i < *count; i++ {
		if *verbose {
			fmt.Printf("[%d] ", i+1)
		}

		switch *format {
		case "json":
			fmt.Printf(`{"message": "%s", "index": %d}`, message, i+1)
		case "xml":
			fmt.Printf(`<message index="%d">%s</message>`, i+1, message)
		default:
			fmt.Print(message)
		}

		fmt.Println()
	}

	if *verbose {
		fmt.Printf("Completed in %.2f seconds\n", *timeout)
	}
}
