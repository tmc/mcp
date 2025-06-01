// Command mcp-shadow forwards MCP traffic to a primary server while shadowing to a secondary server
package main

import (
	"bufio"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	primary      = flag.String("primary", "", "primary server command to execute")
	shadow       = flag.String("shadow", "", "shadow server command to execute")
	outFile      = flag.String("o", "", "output recording file")
	verbose      = flag.Bool("v", false, "verbose mode")
	quiet        = flag.Bool("q", false, "quiet mode")
	genTrace     = flag.Bool("trace", false, "generate OpenTelemetry trace context")
	baggage      = flag.String("baggage", "", "trace-level baggage (key=value,key=value)")
	timeout      = flag.Duration("timeout", 5*time.Second, "timeout for shadow server responses")
	splitMode    = flag.String("split-mode", "shadow", "split mode: shadow, random, round-robin")
	splitPercent = flag.Float64("split-percent", 100.0, "percentage of traffic to shadow (0-100)")
	compareMode  = flag.Bool("compare", false, "output both primary and shadow responses in enhanced mcptrace format")
)

type message struct {
	raw          []byte
	timestamp    time.Time
	direction    string // "recv", "send", "recv-shadow", or "send-shadow"
	spanID       string
	linksTo      string
	baggage      string
	isPrimary    bool
	isShadow     bool
	originalSpan string // for linking shadow responses to original requests
}

type shadowServer struct {
	primary  *exec.Cmd
	shadow   *exec.Cmd
	stdin    io.Writer
	stdout   io.Reader
	stderr   io.Reader
	messages chan message
	shutdown chan struct{}
	wg       sync.WaitGroup

	// For primary server
	primaryStdin  io.WriteCloser
	primaryStdout io.ReadCloser
	primaryStderr io.ReadCloser

	// For shadow server
	shadowStdin  io.WriteCloser
	shadowStdout io.ReadCloser
	shadowStderr io.ReadCloser

	// Trace context
	traceID      string
	traceParent  string
	traceBaggage string

	// Span tracking for comparison
	requestSpans  map[string]string // maps request IDs to span IDs
	responseSpans map[string]string // maps response IDs to span IDs for comparison
	spanLock      sync.RWMutex
}

func main() {
	flag.Parse()

	if *primary == "" || *shadow == "" {
		log.Fatal("Both -primary and -shadow flags are required")
	}

	server := &shadowServer{
		messages:      make(chan message, 100),
		shutdown:      make(chan struct{}),
		requestSpans:  make(map[string]string),
		responseSpans: make(map[string]string),
		traceBaggage:  *baggage,
	}

	// Generate trace context if requested
	if *genTrace {
		server.traceID = generateTraceID()
		server.traceParent = fmt.Sprintf("00-%s-%s-01", server.traceID, generateSpanID())
		if !*quiet {
			log.Printf("Generated trace context: %s", server.traceParent)
		}
	}

	// Start servers
	if err := server.startServers(); err != nil {
		log.Fatal(err)
	}
	defer server.stop()

	// Start message recorder
	if *outFile != "" {
		server.wg.Add(1)
		go server.recordMessages()
	}

	// Start server output monitors
	server.wg.Add(4) // 2 for each server (stdout/stderr)
	go server.monitorOutput(server.primaryStdout, "primary", false)
	go server.monitorOutput(server.primaryStderr, "primary", true)
	go server.monitorOutput(server.shadowStdout, "shadow", false)
	go server.monitorOutput(server.shadowStderr, "shadow", true)

	// Start input forwarder
	server.forwardInput()

	// Wait for shutdown
	close(server.shutdown)
	server.wg.Wait()
}

func (s *shadowServer) startServers() error {
	// Start primary server
	s.primary = exec.Command("sh", "-c", *primary)
	var err error

	s.primaryStdin, err = s.primary.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating primary stdin pipe: %w", err)
	}

	s.primaryStdout, err = s.primary.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating primary stdout pipe: %w", err)
	}

	s.primaryStderr, err = s.primary.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating primary stderr pipe: %w", err)
	}

	if err := s.primary.Start(); err != nil {
		return fmt.Errorf("starting primary server: %w", err)
	}

	// Start shadow server
	s.shadow = exec.Command("sh", "-c", *shadow)

	s.shadowStdin, err = s.shadow.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating shadow stdin pipe: %w", err)
	}

	s.shadowStdout, err = s.shadow.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating shadow stdout pipe: %w", err)
	}

	s.shadowStderr, err = s.shadow.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating shadow stderr pipe: %w", err)
	}

	if err := s.shadow.Start(); err != nil {
		return fmt.Errorf("starting shadow server: %w", err)
	}

	if !*quiet {
		log.Println("Started primary and shadow servers")
	}

	return nil
}

func (s *shadowServer) stop() {
	if s.primary != nil && s.primary.Process != nil {
		s.primary.Process.Kill()
	}
	if s.shadow != nil && s.shadow.Process != nil {
		s.shadow.Process.Kill()
	}
}

func (s *shadowServer) forwardInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Check if it's a JSON message
		if strings.HasPrefix(strings.TrimSpace(string(line)), "{") {
			// Record incoming message
			spanID := generateSpanID()
			s.recordRequestSpan(line, spanID)

			msg := message{
				raw:       line,
				timestamp: time.Now(),
				direction: "recv",
				spanID:    spanID,
				baggage:   s.traceBaggage,
			}

			if *outFile != "" {
				s.messages <- msg
			}

			// Removed redundant shadow recv message - input is identical for both servers

			// Forward to both servers
			if _, err := s.primaryStdin.Write(append(line, '\n')); err != nil {
				if !*quiet {
					log.Printf("Error writing to primary: %v", err)
				}
			}

			// Decide whether to shadow based on split mode
			if shouldShadow() || *compareMode {
				// In compare mode, always send to shadow server
				if _, err := s.shadowStdin.Write(append(line, '\n')); err != nil {
					if !*quiet {
						log.Printf("Error writing to shadow: %v", err)
					}
				}
			}
		}
	}

	// Close stdin pipes
	s.primaryStdin.Close()
	s.shadowStdin.Close()
}

func (s *shadowServer) monitorOutput(reader io.Reader, source string, isStderr bool) {
	defer s.wg.Done()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()

		// If stderr, just pass through
		if isStderr {
			if source == "primary" {
				fmt.Fprintln(os.Stderr, string(line))
			}
			continue
		}

		// Process stdout
		if strings.HasPrefix(strings.TrimSpace(string(line)), "{") {
			// This is a JSON response
			isPrimary := source == "primary"
			isShadow := source == "shadow"
			spanID := generateSpanID()
			linksTo := s.findRequestSpan(line)

			direction := "send"
			if isShadow && *compareMode {
				direction = "send-shadow"
			}

			msg := message{
				raw:       line,
				timestamp: time.Now(),
				direction: direction,
				spanID:    spanID,
				linksTo:   linksTo,
				baggage:   s.traceBaggage,
				isPrimary: isPrimary,
				isShadow:  isShadow,
			}

			// In compare mode, track response spans for correlation
			if *compareMode {
				var msgData map[string]interface{}
				if err := json.Unmarshal(line, &msgData); err == nil {
					if id, ok := msgData["id"]; ok {
						s.spanLock.Lock()
						if isShadow {
							msg.originalSpan = s.responseSpans[fmt.Sprint(id)]
							if msg.originalSpan != "" {
								msg.linksTo = msg.originalSpan
							}
						} else if isPrimary {
							s.responseSpans[fmt.Sprint(id)] = spanID
						}
						s.spanLock.Unlock()
					}
				}
			}

			if isShadow {
				if *compareMode {
					// In compare mode, add additional metadata
					msg.baggage = fmt.Sprintf("%s,shadow=true,compare=true", s.traceBaggage)
				} else {
					msg.baggage = fmt.Sprintf("%s,shadow=true", s.traceBaggage)
				}
			}

			if *outFile != "" {
				s.messages <- msg
			}

			// Forward primary output to stdout
			if isPrimary {
				fmt.Println(string(line))
			}
		} else {
			// Non-JSON output - only forward primary
			if source == "primary" {
				fmt.Println(string(line))
			}
		}
	}
}

func (s *shadowServer) recordMessages() {
	defer s.wg.Done()

	file, err := os.Create(*outFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	// Write header
	header := "# mcptrace:v1"
	if s.traceParent != "" {
		header += fmt.Sprintf(" traceparent=%s", s.traceParent)
	}
	if s.traceBaggage != "" {
		header += fmt.Sprintf(" baggage=%s", s.traceBaggage)
	}
	if *compareMode {
		// Add compare mode indicator to the header
		header += " compare=true"
	}
	fmt.Fprintln(file, header)

	for {
		select {
		case msg := <-s.messages:
			line := formatMCPLine(msg)
			if *compareMode {
				// In compare mode, don't comment out shadow responses
				// The comparison tools will handle them
				fmt.Fprintln(file, line)
			} else {
				// Legacy behavior - comment out shadow responses
				if !msg.isPrimary && msg.direction == "send" {
					line = "# " + line
				}
				fmt.Fprintln(file, line)
			}
			file.Sync()

		case <-s.shutdown:
			return
		}
	}
}

func formatMCPLine(msg message) string {
	line := fmt.Sprintf("mcp-%s %s # %.3f",
		msg.direction,
		string(msg.raw),
		float64(msg.timestamp.UnixNano())/1e9)

	if msg.spanID != "" {
		line += fmt.Sprintf(" spanid=%s", msg.spanID)
	}

	if msg.linksTo != "" {
		line += fmt.Sprintf(" linksto=%s", msg.linksTo)
	}

	if msg.baggage != "" {
		line += fmt.Sprintf(" baggage=%s", msg.baggage)
	}

	return line
}

func (s *shadowServer) recordRequestSpan(data []byte, spanID string) {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	if id, ok := msg["id"]; ok {
		s.spanLock.Lock()
		s.requestSpans[fmt.Sprint(id)] = spanID
		s.spanLock.Unlock()
	}
}

func (s *shadowServer) findRequestSpan(data []byte) string {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return ""
	}

	if id, ok := msg["id"]; ok {
		s.spanLock.RLock()
		span := s.requestSpans[fmt.Sprint(id)]
		s.spanLock.RUnlock()
		return span
	}

	return ""
}

func shouldShadow() bool {
	if *splitMode == "shadow" {
		// Always shadow
		return true
	}

	if *splitMode == "random" {
		// Random sampling based on percentage
		return (rand.Float64() * 100.0) < *splitPercent
	}

	// Other modes not implemented yet
	return true
}

func generateTraceID() string {
	bytes := make([]byte, 16)
	cryptorand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateSpanID() string {
	bytes := make([]byte, 8)
	cryptorand.Read(bytes)
	return hex.EncodeToString(bytes)
}
