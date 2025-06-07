package mcp

import (
	"bufio"
	"bytes"
	"context"

	// "encoding/json" // Not directly needed here if sseRWCAdapter deals with bytes for JSON-RPC layer
	"fmt"
	"io"
	"log/slog"
	"net" // For net.Error in readLoop
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	// "golang.org/x/exp/jsonrpc2" // Only for Message type if sseRWCAdapter needs to parse/encode
)

// SSEClientTransport connects to an MCP server over Server-Sent Events.
type SSEClientTransport struct {
	sseURL *url.URL
	client *http.Client
	logger *slog.Logger
}

// NewSSEClientTransport creates a transport for an SSE client.
func NewSSEClientTransport(rawURL string, logger *slog.Logger) (*SSEClientTransport, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid SSE URL: %w", err)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &SSEClientTransport{
		sseURL: u,
		client: &http.Client{Timeout: 30 * time.Second /* Configurable */},
		logger: logger,
	}, nil
}

// Dial implements the mcp.Transport interface for SSE client.
// It establishes the SSE connection and returns an io.ReadWriteCloser adapter.
func (t *SSEClientTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	t.logger.DebugContext(ctx, "SSEClientTransport: Dialing", "url", t.sseURL.String())

	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, t.sseURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("sse client: create GET request: %w", err)
	}
	getReq.Header.Set("Accept", "text/event-stream")
	getReq.Header.Set("Cache-Control", "no-cache")

	sseResp, err := t.client.Do(getReq)
	if err != nil {
		return nil, fmt.Errorf("sse client: GET request failed: %w", err)
	}
	if sseResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(sseResp.Body)
		sseResp.Body.Close()
		return nil, fmt.Errorf("sse client: GET request status %d: %s", sseResp.StatusCode, string(bodyBytes))
	}

	sseReader := bufio.NewReader(sseResp.Body)
	var postURL *url.URL
	scanner := bufio.NewScanner(sseReader)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: endpoint") {
			if !scanner.Scan() {
				sseResp.Body.Close()
				return nil, fmt.Errorf("sse: unexpected EOF after endpoint event name")
			}
			dataLine := scanner.Text()
			if !strings.HasPrefix(dataLine, "data: ") {
				sseResp.Body.Close()
				return nil, fmt.Errorf("sse: expected data for endpoint event, got: %s", dataLine)
			}
			endpointPath := strings.TrimSpace(strings.TrimPrefix(dataLine, "data: "))
			postURL, err = t.sseURL.Parse(endpointPath)
			if err != nil {
				sseResp.Body.Close()
				return nil, fmt.Errorf("sse: invalid post endpoint URL '%s': %w", endpointPath, err)
			}
			t.logger.DebugContext(ctx, "SSEClient: Received POST endpoint", "url", postURL.String())
			break
		}
	}
	if errS := scanner.Err(); errS != nil {
		sseResp.Body.Close()
		return nil, fmt.Errorf("sse: scan endpoint event: %w", errS)
	}
	if postURL == nil {
		sseResp.Body.Close()
		return nil, fmt.Errorf("sse: no endpoint event received")
	}

	adapter := &sseRWCAdapter{
		ctx:         ctx,
		postURL:     postURL,
		sseReader:   sseReader,
		sseBody:     sseResp.Body,
		httpClient:  t.client,
		logger:      t.logger,
		readBuf:     new(bytes.Buffer),
		readChan:    make(chan []byte, 10),
		readErrChan: make(chan error, 1),
		closed:      make(chan struct{}),
	}

	go adapter.readLoop()

	return adapter, nil
}

// sseRWCAdapter adapts SSE client communication to an io.ReadWriteCloser.
type sseRWCAdapter struct {
	ctx        context.Context
	postURL    *url.URL
	sseReader  *bufio.Reader
	sseBody    io.Closer
	httpClient *http.Client
	logger     *slog.Logger

	readMu  sync.Mutex
	readBuf *bytes.Buffer

	readChan    chan []byte
	readErrChan chan error

	closeOnce sync.Once
	closed    chan struct{}
}

func (s *sseRWCAdapter) readLoop() {
	defer close(s.readChan)
	defer s.logger.DebugContext(s.ctx, "sseRWCAdapter: readLoop finished")

	scanner := bufio.NewScanner(s.sseReader)
	var currentEventDataLines []string

	for {
		select {
		case <-s.closed:
			s.readErrChan <- io.ErrClosedPipe
			return
		case <-s.ctx.Done():
			s.readErrChan <- s.ctx.Err()
			return
		default:
		}

		if conn, ok := s.sseBody.(interface{ SetReadDeadline(time.Time) error }); ok {
			conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		}

		if !scanner.Scan() {
			err := scanner.Err()
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				s.logger.DebugContext(s.ctx, "sseRWCAdapter: readLoop scan timeout")
				continue
			}
			if err == nil {
				err = io.EOF
			}
			s.logger.ErrorContext(s.ctx, "sseRWCAdapter: readLoop scanner error", "error", err)
			s.readErrChan <- err
			return
		}

		line := scanner.Text()
		s.logger.DebugContext(s.ctx, "sseRWCAdapter: readLoop got line", "line", line)

		if line == "" {
			if len(currentEventDataLines) > 0 {
				fullMessage := strings.Join(currentEventDataLines, "")
				currentEventDataLines = nil

				select {
				case s.readChan <- []byte(fullMessage + "\n"):
					s.logger.DebugContext(s.ctx, "sseRWCAdapter: readLoop sent message to readChan", "data_len", len(fullMessage))
				case <-s.closed:
					s.readErrChan <- io.ErrClosedPipe
					return
				case <-s.ctx.Done():
					s.readErrChan <- s.ctx.Err()
					return
				}
			}
		} else if strings.HasPrefix(line, "data:") {
			currentEventDataLines = append(currentEventDataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		} else if strings.HasPrefix(line, "event:") {
			if len(currentEventDataLines) > 0 {
				s.logger.WarnContext(s.ctx, "sseRWCAdapter: data received before non-message event, sending previous data", "event", line)
				fullMessage := strings.Join(currentEventDataLines, "")
				currentEventDataLines = nil
				select {
				case s.readChan <- []byte(fullMessage + "\n"):
				case <-s.closed:
					s.readErrChan <- io.ErrClosedPipe
					return
				case <-s.ctx.Done():
					s.readErrChan <- s.ctx.Err()
					return
				default:
					s.logger.WarnContext(s.ctx, "sseRWCAdapter: readChan full, dropping message before non-message event")
				}
			}
			s.logger.DebugContext(s.ctx, "sseRWCAdapter: readLoop received SSE event type", "event_line", line)
		}
	}
}

func (s *sseRWCAdapter) Read(p []byte) (n int, err error) {
	s.readMu.Lock()
	defer s.readMu.Unlock()

	if s.readBuf.Len() > 0 {
		n, err = s.readBuf.Read(p)
		s.logger.DebugContext(s.ctx, "sseRWCAdapter: Read from buffer", "bytes_read", n, "error", err)
		if err == io.EOF && s.readBuf.Len() > 0 {
			return n, nil
		}
		return n, err
	}

	select {
	case <-s.closed:
		return 0, io.ErrClosedPipe
	case <-s.ctx.Done():
		return 0, s.ctx.Err()
	case data, ok := <-s.readChan:
		if !ok {
			select {
			case err = <-s.readErrChan:
				if err == nil {
					err = io.EOF
				}
			default:
				err = io.EOF
			}
			s.logger.DebugContext(s.ctx, "sseRWCAdapter: Read returning", "error", err, "readChan_closed", true)
			return 0, err
		}
		s.logger.DebugContext(s.ctx, "sseRWCAdapter: Read got data from readChan", "data_len", len(data))

		s.readBuf.Reset() // Ensure buffer is clean before writing new data
		s.readBuf.Write(data)
		n, err = s.readBuf.Read(p)
		if err == io.EOF && s.readBuf.Len() > 0 { // If buffer was emptied but still data left
			return n, nil
		}
		return n, err
	}
}

func (s *sseRWCAdapter) Write(p []byte) (n int, err error) {
	select {
	case <-s.closed:
		return 0, io.ErrClosedPipe
	case <-s.ctx.Done():
		return 0, s.ctx.Err()
	default:
	}

	dataToSend := bytes.TrimSuffix(p, []byte{'\n'})
	s.logger.DebugContext(s.ctx, "sseRWCAdapter: Writing (POSTing) data", "url", s.postURL.String(), "data_len_original", len(p), "data_len_tosend", len(dataToSend))

	postReq, err := http.NewRequestWithContext(s.ctx, http.MethodPost, s.postURL.String(), bytes.NewReader(dataToSend))
	if err != nil {
		s.logger.ErrorContext(s.ctx, "sseRWCAdapter: Failed to create POST request", "error", err)
		return 0, fmt.Errorf("sse client: create POST: %w", err)
	}
	postReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(postReq)
	if err != nil {
		s.logger.ErrorContext(s.ctx, "sseRWCAdapter: POST request failed", "error", err)
		return 0, fmt.Errorf("sse client: POST failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		s.logger.ErrorContext(s.ctx, "sseRWCAdapter: POST request returned error status", "status_code", resp.StatusCode, "body", string(bodyBytes))
		return 0, fmt.Errorf("sse client: POST status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	s.logger.DebugContext(s.ctx, "sseRWCAdapter: Write (POST) successful", "bytes_posted", len(dataToSend), "reported_written", len(p))
	return len(p), nil
}

func (s *sseRWCAdapter) Close() error {
	s.logger.DebugContext(s.ctx, "sseRWCAdapter: Close called")
	s.closeOnce.Do(func() {
		close(s.closed)
		s.sseBody.Close()
		s.logger.InfoContext(s.ctx, "sseRWCAdapter: Closed")
	})
	return nil
}

type SSEServerTransport struct {
	rwc    io.ReadWriteCloser
	logger *slog.Logger
}

func NewSSEServerTransport(rwc io.ReadWriteCloser, logger *slog.Logger) *SSEServerTransport {
	if logger == nil {
		logger = slog.Default()
	}
	return &SSEServerTransport{rwc: rwc, logger: logger}
}

func (t *SSEServerTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	t.logger.InfoContext(ctx, "SSEServerTransport: Dial called")
	if t.rwc == nil {
		return nil, fmt.Errorf("SSEServerTransport: RWC not initialized by HTTP handler")
	}
	return t.rwc, nil
}
