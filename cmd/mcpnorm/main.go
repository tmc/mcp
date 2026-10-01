// Command mcpnorm rebases and optionally sorts MCP trace records.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/tmc/mcp/mcptrace"
)

func main() {
	start := flag.Duration("start", 0, "timestamp to assign to the first timed record")
	absolute := flag.String("absolute", "", "Unix timestamp to assign to the first timed record")
	sortByTime := flag.Bool("sort", false, "sort records by timestamp")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] [trace]\n\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Reads an MCP trace from a file or stdin and writes normalized records to stdout.")
		flag.PrintDefaults()
	}
	flag.Parse()

	var input io.Reader = os.Stdin
	var file *os.File
	if flag.NArg() > 1 {
		log.Fatal("at most one input file is allowed")
	}
	if flag.NArg() == 1 {
		var err error
		file, err = os.Open(flag.Arg(0))
		if err != nil {
			log.Fatalf("open input: %v", err)
		}
		defer file.Close()
		input = file
	}

	base := *start
	if *absolute != "" {
		seconds, err := strconv.ParseFloat(*absolute, 64)
		if err != nil {
			log.Fatalf("invalid absolute timestamp: %v", err)
		}
		base = time.Duration(seconds * float64(time.Second))
	}
	if err := normalize(input, os.Stdout, base, *sortByTime); err != nil {
		log.Fatalf("normalize trace: %v", err)
	}
}

func normalize(input io.Reader, output io.Writer, base time.Duration, sortByTime bool) error {
	reader := mcptrace.NewReader(input)
	var records []mcptrace.Record
	var first time.Time
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if !record.Time.IsZero() {
			if first.IsZero() {
				first = record.Time
			}
			record.Time = baseToTime(base).Add(record.Time.Sub(first))
		}
		records = append(records, record)
	}
	if sortByTime {
		sort.SliceStable(records, func(i, j int) bool {
			a, b := records[i].Time, records[j].Time
			if a.IsZero() {
				return !b.IsZero()
			}
			if b.IsZero() {
				return false
			}
			return a.Before(b)
		})
	}
	writer := mcptrace.NewWriter(output)
	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}

func baseToTime(base time.Duration) time.Time {
	return time.Unix(0, int64(base))
}
