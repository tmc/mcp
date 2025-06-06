package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"time"
)

func main() {
	var (
		input       = flag.String("input", "-", "Input file (- for stdin)")
		output      = flag.String("output", "-", "Output file (- for stdout)")
		format      = flag.String("format", "rfc3339", "Output timestamp format")
		base        = flag.String("base", "", "Base time for relative timestamps (RFC3339)")
		relative    = flag.Bool("relative", false, "Convert to relative timestamps")
		pattern     = flag.String("pattern", "", "Custom timestamp pattern (regex)")
		parseLayout = flag.String("parse", "", "Custom parse layout (Go time format)")
		verbose     = flag.Bool("verbose", false, "Verbose output")
		dryRun      = flag.Bool("dry-run", false, "Show what would be changed without modifying")
	)
	flag.Parse()

	// Setup normalizer
	normalizer := &TimestampNormalizer{
		OutputFormat: *format,
		Relative:     *relative,
		ParseLayout:  *parseLayout,
		Verbose:      *verbose,
		DryRun:       *dryRun,
	}

	// Parse base time if provided
	if *base != "" {
		baseTime, err := time.Parse(time.RFC3339, *base)
		if err != nil {
			log.Fatalf("Invalid base time: %v", err)
		}
		normalizer.BaseTime = baseTime
	} else if *relative {
		normalizer.BaseTime = time.Now()
	}

	// Setup custom pattern if provided
	if *pattern != "" {
		re, err := regexp.Compile(*pattern)
		if err != nil {
			log.Fatalf("Invalid timestamp pattern: %v", err)
		}
		normalizer.CustomPattern = re
	}

	// Setup input
	var reader io.Reader
	if *input == "-" {
		reader = os.Stdin
	} else {
		file, err := os.Open(*input)
		if err != nil {
			log.Fatalf("Failed to open input file: %v", err)
		}
		defer file.Close()
		reader = file
	}

	// Setup output
	var writer io.Writer
	if *output == "-" {
		writer = os.Stdout
	} else {
		file, err := os.Create(*output)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer file.Close()
		writer = file
	}

	// Process input
	stats, err := normalizer.Process(reader, writer)
	if err != nil {
		log.Fatalf("Processing failed: %v", err)
	}

	// Print statistics if verbose
	if *verbose {
		fmt.Fprintf(os.Stderr, "\nProcessed %d lines, normalized %d timestamps\n",
			stats.LinesProcessed, stats.TimestampsNormalized)
		if stats.ParseErrors > 0 {
			fmt.Fprintf(os.Stderr, "Parse errors: %d\n", stats.ParseErrors)
		}
	}
}

// TimestampNormalizer normalizes timestamps in text
type TimestampNormalizer struct {
	OutputFormat  string
	BaseTime      time.Time
	Relative      bool
	CustomPattern *regexp.Regexp
	ParseLayout   string
	Verbose       bool
	DryRun        bool

	patterns  []*timestampPattern
	firstTime *time.Time
}

// Statistics tracks processing statistics
type Statistics struct {
	LinesProcessed       int
	TimestampsNormalized int
	ParseErrors          int
}

type timestampPattern struct {
	regex  *regexp.Regexp
	layout string
	name   string
}

// Process reads input and writes normalized output
func (n *TimestampNormalizer) Process(reader io.Reader, writer io.Writer) (*Statistics, error) {
	n.setupPatterns()

	stats := &Statistics{}
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		stats.LinesProcessed++
		line := scanner.Text()

		normalized, changed := n.normalizeLine(line, stats)

		if n.DryRun && changed {
			fmt.Fprintf(os.Stderr, "Would change:\n  %s\n  %s\n", line, normalized)
		} else {
			fmt.Fprintln(writer, normalized)
		}
	}

	return stats, scanner.Err()
}

func (n *TimestampNormalizer) setupPatterns() {
	n.patterns = []*timestampPattern{}

	// Add custom pattern if provided
	if n.CustomPattern != nil && n.ParseLayout != "" {
		n.patterns = append(n.patterns, &timestampPattern{
			regex:  n.CustomPattern,
			layout: n.ParseLayout,
			name:   "custom",
		})
	}

	// Add common timestamp patterns
	n.patterns = append(n.patterns, []*timestampPattern{
		{
			regex:  regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})`),
			layout: time.RFC3339Nano,
			name:   "RFC3339",
		},
		{
			regex:  regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`),
			layout: "2006-01-02 15:04:05",
			name:   "SQL datetime",
		},
		{
			regex:  regexp.MustCompile(`\d{2}/\w{3}/\d{4}:\d{2}:\d{2}:\d{2} [+-]\d{4}`),
			layout: "02/Jan/2006:15:04:05 -0700",
			name:   "Common Log Format",
		},
		{
			regex:  regexp.MustCompile(`\w{3} \d{2} \d{2}:\d{2}:\d{2}`),
			layout: "Jan 02 15:04:05",
			name:   "Syslog",
		},
		{
			regex:  regexp.MustCompile(`\d{10}(?:\.\d+)?`),
			layout: "unix",
			name:   "Unix timestamp",
		},
		{
			regex:  regexp.MustCompile(`\d{13}`),
			layout: "unix_ms",
			name:   "Unix milliseconds",
		},
	}...)
}

func (n *TimestampNormalizer) normalizeLine(line string, stats *Statistics) (string, bool) {
	changed := false
	result := line

	for _, pattern := range n.patterns {
		matches := pattern.regex.FindAllStringSubmatchIndex(result, -1)

		// Process matches in reverse order to maintain positions
		for i := len(matches) - 1; i >= 0; i-- {
			match := matches[i]
			if len(match) < 2 {
				continue
			}

			start, end := match[0], match[1]
			timestampStr := result[start:end]

			// Parse timestamp
			parsedTime, err := n.parseTimestamp(timestampStr, pattern.layout)
			if err != nil {
				if n.Verbose {
					fmt.Fprintf(os.Stderr, "Failed to parse %s as %s: %v\n",
						timestampStr, pattern.name, err)
				}
				stats.ParseErrors++
				continue
			}

			// Store first timestamp for relative calculations
			if n.firstTime == nil {
				n.firstTime = &parsedTime
			}

			// Format timestamp
			formatted := n.formatTimestamp(parsedTime)

			// Replace in line
			result = result[:start] + formatted + result[end:]
			changed = true
			stats.TimestampsNormalized++
		}
	}

	return result, changed
}

func (n *TimestampNormalizer) parseTimestamp(str string, layout string) (time.Time, error) {
	switch layout {
	case "unix":
		// Parse as Unix timestamp (seconds)
		var seconds int64
		if _, err := fmt.Sscanf(str, "%d", &seconds); err != nil {
			return time.Time{}, err
		}
		return time.Unix(seconds, 0), nil

	case "unix_ms":
		// Parse as Unix timestamp (milliseconds)
		var millis int64
		if _, err := fmt.Sscanf(str, "%d", &millis); err != nil {
			return time.Time{}, err
		}
		return time.Unix(millis/1000, (millis%1000)*1000000), nil

	default:
		// Parse with Go time layout
		return time.Parse(layout, str)
	}
}

func (n *TimestampNormalizer) formatTimestamp(t time.Time) string {
	if n.Relative {
		var duration time.Duration
		if n.firstTime != nil {
			duration = t.Sub(*n.firstTime)
		} else {
			duration = t.Sub(n.BaseTime)
		}

		// Format duration nicely
		return formatDuration(duration)
	}

	// Format according to output format
	switch n.OutputFormat {
	case "rfc3339":
		return t.Format(time.RFC3339)
	case "rfc3339nano":
		return t.Format(time.RFC3339Nano)
	case "unix":
		return fmt.Sprintf("%d", t.Unix())
	case "unix_ms":
		return fmt.Sprintf("%d", t.UnixNano()/1000000)
	case "sql":
		return t.Format("2006-01-02 15:04:05")
	default:
		// Try as custom format
		return t.Format(n.OutputFormat)
	}
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	// Format as +/- HH:MM:SS.mmm
	sign := ""
	if d < 0 {
		sign = "-"
		d = -d
	} else {
		sign = "+"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	millis := int(d.Milliseconds()) % 1000

	if hours > 0 {
		return fmt.Sprintf("%s%02d:%02d:%02d.%03d", sign, hours, minutes, seconds, millis)
	}
	if minutes > 0 {
		return fmt.Sprintf("%s%02d:%02d.%03d", sign, minutes, seconds, millis)
	}
	return fmt.Sprintf("%s%d.%03ds", sign, seconds, millis)
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `tsnorm - Normalize timestamps in text

Usage:
  tsnorm [options] < input.log > output.log

Examples:
  tsnorm < app.log                           # Normalize to RFC3339
  tsnorm -relative < trace.log               # Convert to relative times
  tsnorm -format unix < events.log           # Convert to Unix timestamps
  tsnorm -pattern '\d{8}T\d{6}' -parse '20060102T150405'  # Custom format

Output Formats:
  rfc3339      - RFC3339 format (default)
  rfc3339nano  - RFC3339 with nanoseconds
  unix         - Unix timestamp (seconds)
  unix_ms      - Unix timestamp (milliseconds)
  sql          - SQL datetime format
  <custom>     - Custom Go time format

Options:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
The tool automatically detects common timestamp formats:
  - RFC3339 (2006-01-02T15:04:05Z)
  - SQL datetime (2006-01-02 15:04:05)
  - Common Log Format (02/Jan/2006:15:04:05 -0700)
  - Syslog (Jan 02 15:04:05)
  - Unix timestamps (seconds and milliseconds)

For custom formats, use -pattern with a regex and -parse with a Go time layout.
`)
	}
}
