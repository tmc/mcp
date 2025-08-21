// Command mcp-shadow forwards MCP traffic to a primary server while shadowing to a secondary server
package main

import (
	"bufio"
	"context"
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
	timeout      = flag.Duration("timeout", 30*time.Second, "timeout for operations")
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
	shutdownOnce sync.Once
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc

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

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	server := &shadowServer{
		messages:      make(chan message, 100),
		shutdown:      make(chan struct{}),
		requestSpans:  make(map[string]string),
		responseSpans: make(map[string]string),
		traceBaggage:  *baggage,
		ctx:           ctx,
		cancel:        cancel,
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

	// Start input forwarder as goroutine
	server.wg.Add(1)
	go server.forwardInput()

	// Wait for shutdown or context cancellation
	select {
	case <-server.ctx.Done():
		if !*quiet {
			log.Println("Timeout reached, shutting down...")
		}
	case <-server.shutdown:
		// Normal shutdown
	}

	server.wg.Wait()
}

func (s *shadowServer) startServers() error {
	// Start primary server
	s.primary = exec.CommandContext(s.ctx, "sh", "-c", *primary)
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
	s.shadow = exec.CommandContext(s.ctx, "sh", "-c", *shadow)

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
	// Cancel context to signal all goroutines to stop
	s.cancel()

	// Close stdin pipes to signal servers to exit gracefully
	if s.primaryStdin != nil {
		s.primaryStdin.Close()
	}
	if s.shadowStdin != nil {
		s.shadowStdin.Close()
	}

	// Wait a short time for graceful shutdown
	time.Sleep(100 * time.Millisecond)

	// Force kill if still running
	if s.primary != nil && s.primary.Process != nil {
		s.primary.Process.Kill()
		s.primary.Wait() // Wait for process to exit
	}
	if s.shadow != nil && s.shadow.Process != nil {
		s.shadow.Process.Kill()
		s.shadow.Wait() // Wait for process to exit
	}

	// Close remaining pipes
	if s.primaryStdout != nil {
		s.primaryStdout.Close()
	}
	if s.primaryStderr != nil {
		s.primaryStderr.Close()
	}
	if s.shadowStdout != nil {
		s.shadowStdout.Close()
	}
	if s.shadowStderr != nil {
		s.shadowStderr.Close()
	}
}

func (s *shadowServer) forwardInput() {
	defer s.wg.Done()
	defer func() {
		// Signal shutdown when input is done
		s.shutdownOnce.Do(func() {
			close(s.shutdown)
		})
	}()

	scanner := bufio.NewScanner(os.Stdin)

	// Use a channel to read lines asynchronously
	lineCh := make(chan []byte)
	errCh := make(chan error, 1)

	go func() {
		defer close(lineCh)
		for scanner.Scan() {
			// Copy the bytes since scanner reuses the buffer
			line := make([]byte, len(scanner.Bytes()))
			copy(line, scanner.Bytes())

			select {
			case lineCh <- line:
			case <-s.ctx.Done():
				return
			}
		}
		if err := scanner.Err(); err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			if !*quiet {
				log.Println("Input forwarder cancelled")
			}
			return
		case err := <-errCh:
			if !*quiet {
				log.Printf("Scanner error: %v", err)
			}
			return
		case line, ok := <-lineCh:
			if !ok {
				// EOF reached - stdin closed
				if !*quiet {
					log.Println("Stdin closed, shutting down")
				}
				return
			}

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
					select {
					case s.messages <- msg:
					case <-s.ctx.Done():
						return
					}
				}

				// Forward to both servers
				if _, err := s.primaryStdin.Write(append(line, '\n')); err != nil {
					if !*quiet {
						log.Printf("Error writing to primary: %v", err)
					}
					return
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
	}
}

func (s *shadowServer) monitorOutput(reader io.Reader, source string, isStderr bool) {
	defer s.wg.Done()

	scanner := bufio.NewScanner(reader)
	
	// Use a channel-based approach for better cancellation handling
	lineCh := make(chan []byte)
	errCh := make(chan error, 1)
	
	go func() {
		defer close(lineCh)
		for scanner.Scan() {
			// Copy the bytes since scanner reuses the buffer
			line := make([]byte, len(scanner.Bytes()))
			copy(line, scanner.Bytes())
			
			select {
			case lineCh <- line:
			case <-s.ctx.Done():
				return
			}
		}
		if err := scanner.Err(); err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case err := <-errCh:
			if err != nil && !*quiet {
				log.Printf("Scanner error for %s: %v", source, err)
			}
			return
		case line, ok := <-lineCh:
			if !ok {
				// Scanner finished (EOF or error)
				return
			}
			s.processOutputLine(line, source, isStderr)
		}
	}
}

func (s *shadowServer) processOutputLine(line []byte, source string, isStderr bool) {
	// If stderr, just pass through
	if isStderr {
		if source == "primary" {
			fmt.Fprintln(os.Stderr, string(line))
		}
		return
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
			select {
			case s.messages <- msg:
			case <-s.ctx.Done():
				return
			}
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
		case <-s.ctx.Done():
			// Drain remaining messages before returning
			for {
				select {
				case msg := <-s.messages:
					line := formatMCPLine(msg)
					if *compareMode {
						fmt.Fprintln(file, line)
					} else {
						if !msg.isPrimary && msg.direction == "send" {
							line = "# " + line
						}
						fmt.Fprintln(file, line)
					}
				default:
					return
				}
			}
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
			// Drain remaining messages before returning
			for {
				select {
				case msg := <-s.messages:
					line := formatMCPLine(msg)
					if *compareMode {
						fmt.Fprintln(file, line)
					} else {
						if !msg.isPrimary && msg.direction == "send" {
							line = "# " + line
						}
						fmt.Fprintln(file, line)
					}
				default:
					return
				}
			}
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
