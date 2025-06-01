// Command mcptrace-to-otel converts MCPTrace files to OpenTelemetry format
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	inputFile   = flag.String("f", "", "input MCPTrace file")
	outputType  = flag.String("type", "stdout", "output type: stdout, otlp-grpc, otlp-http, jaeger, zipkin")
	endpoint    = flag.String("endpoint", "", "endpoint for exporter (required for otlp-*, jaeger, zipkin)")
	serviceName = flag.String("service", "mcp-trace", "service name for traces")
	verbose     = flag.Bool("v", false, "verbose output")
	batchSize   = flag.Int("batch", 100, "batch size for exports")
	timeout     = flag.Duration("timeout", 10*time.Second, "export timeout")
	insecure    = flag.Bool("insecure", false, "use insecure connection for OTLP")
)

// MCPMessage represents a parsed MCP trace line
type MCPMessage struct {
	Direction string                 `json:"direction"` // "recv" or "send"
	JSON      map[string]interface{} `json:"json"`
	Timestamp time.Time              `json:"timestamp"`
	SpanID    string                 `json:"span_id,omitempty"`
	LinksTo   string                 `json:"links_to,omitempty"`
	Baggage   map[string]string      `json:"baggage,omitempty"`
}

// TraceHeader represents the MCPTrace header
type TraceHeader struct {
	Version     string
	TraceParent string
	TraceState  string
	Baggage     map[string]string
}

// Regular expressions for parsing
var (
	headerRegex = regexp.MustCompile(`^# mcptrace:v(\d+)(?:\s+traceparent=([^\s]+))?(?:\s+tracestate=([^\s]+))?(?:\s+baggage=([^\s]+))?`)
	lineRegex   = regexp.MustCompile(`^mcp-(recv|send)\s+(.+?)\s+#\s+([\d.]+)(.*)$`)
	// Also support commented shadow lines
	shadowLineRegex = regexp.MustCompile(`^#\s+mcp-(recv|send)\s+(.+?)\s+#\s+([\d.]+)(.*)$`)
)

func main() {
	flag.Parse()

	if *inputFile == "" {
		log.Fatal("Input file required (-f)")
	}

	ctx := context.Background()

	// Initialize the trace provider
	tp, err := initTraceProvider(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize trace provider: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down trace provider: %v", err)
		}
	}()

	// Process the trace file
	if err := processTraceFile(ctx, *inputFile); err != nil {
		log.Fatalf("Failed to process trace file: %v", err)
	}
}

func initTraceProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	var err error

	switch *outputType {
	case "stdout":
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "otlp-grpc":
		if *endpoint == "" {
			return nil, fmt.Errorf("endpoint required for OTLP gRPC exporter")
		}
		var opts []otlptracegrpc.Option
		opts = append(opts, otlptracegrpc.WithEndpoint(*endpoint))
		if *insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		client := otlptracegrpc.NewClient(opts...)
		exporter, err = otlptrace.New(ctx, client)
	case "otlp-http":
		if *endpoint == "" {
			return nil, fmt.Errorf("endpoint required for OTLP HTTP exporter")
		}
		var opts []otlptracehttp.Option
		opts = append(opts, otlptracehttp.WithEndpoint(*endpoint))
		if *insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		client := otlptracehttp.NewClient(opts...)
		exporter, err = otlptrace.New(ctx, client)
	case "jaeger":
		if *endpoint == "" {
			return nil, fmt.Errorf("endpoint required for Jaeger exporter")
		}
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(*endpoint)))
	case "zipkin":
		if *endpoint == "" {
			return nil, fmt.Errorf("endpoint required for Zipkin exporter")
		}
		exporter, err = zipkin.New(*endpoint)
	default:
		return nil, fmt.Errorf("unknown output type: %s", *outputType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create resource
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(*serviceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(*timeout),
			sdktrace.WithMaxExportBatchSize(*batchSize),
		),
		sdktrace.WithResource(r),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func processTraceFile(ctx context.Context, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	tracer := otel.Tracer("mcptrace-converter")

	var header *TraceHeader
	messages := make(map[string]*MCPMessage) // spanID -> message
	links := make(map[string][]string)       // spanID -> linked spanIDs
	
	// First pass: parse all messages and build link graph
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Parse header
		if strings.HasPrefix(line, "# mcptrace:") {
			header = parseHeader(line)
			continue
		}

		// Parse regular or shadow lines
		msg := parseLine(line)
		if msg != nil {
			messages[msg.SpanID] = msg
			if msg.LinksTo != "" {
				links[msg.LinksTo] = append(links[msg.LinksTo], msg.SpanID)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Second pass: create spans
	rootCtx := ctx
	var rootSpan trace.Span

	// If we have trace context from header, use it
	if header != nil && header.TraceParent != "" {
		rootCtx, rootSpan = createRootSpan(ctx, tracer, header)
		defer rootSpan.End()
	}

	// Group messages by request ID for creating spans
	requestSpans := make(map[string]trace.Span)
	
	for spanID, msg := range messages {
		// Skip if this is a linked span (shadow response)
		if msg.LinksTo != "" {
			continue
		}

		// Create or get span for this request
		span := createSpanForMessage(rootCtx, tracer, msg, header)
		
		// Add events for linked messages (e.g., shadow responses)
		if linkedSpans, ok := links[spanID]; ok {
			if *verbose {
				log.Printf("Found %d linked spans for %s", len(linkedSpans), spanID)
			}
			for _, linkedID := range linkedSpans {
				if linkedMsg, exists := messages[linkedID]; exists {
					if *verbose {
						log.Printf("Adding linked event from %s to span %s", linkedID, spanID)
					}
					addLinkedEvent(span, linkedMsg)
				}
			}
		} else if *verbose {
			log.Printf("No linked spans found for %s", spanID)
		}

		// Store span for potential future use
		if id, ok := msg.JSON["id"]; ok {
			requestSpans[fmt.Sprint(id)] = span
		}

		// End span after processing
		span.End()
	}

	return nil
}

func parseHeader(line string) *TraceHeader {
	matches := headerRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil
	}

	header := &TraceHeader{
		Version: matches[1],
		Baggage: make(map[string]string),
	}

	if len(matches) > 2 && matches[2] != "" {
		header.TraceParent = matches[2]
	}
	if len(matches) > 3 && matches[3] != "" {
		header.TraceState = matches[3]
	}
	if len(matches) > 4 && matches[4] != "" {
		parseBaggage(matches[4], header.Baggage)
	}

	return header
}

func parseLine(line string) *MCPMessage {
	// Try regular line first
	matches := lineRegex.FindStringSubmatch(line)
	if len(matches) == 0 {
		// Try shadow line
		matches = shadowLineRegex.FindStringSubmatch(line)
		if len(matches) == 0 {
			return nil
		}
	}

	msg := &MCPMessage{
		Direction: matches[1],
		Baggage:   make(map[string]string),
	}

	// Parse JSON
	if err := json.Unmarshal([]byte(matches[2]), &msg.JSON); err != nil {
		if *verbose {
			log.Printf("Failed to parse JSON: %v", err)
		}
		return nil
	}

	// Parse timestamp
	timestampFloat, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		if *verbose {
			log.Printf("Failed to parse timestamp: %v", err)
		}
		return nil
	}
	msg.Timestamp = time.Unix(int64(timestampFloat), int64((timestampFloat-float64(int64(timestampFloat)))*1e9))

	// Parse optional fields from the extra part
	if len(matches) > 4 && matches[4] != "" {
		parseExtraFields(matches[4], msg)
	}

	return msg
}

// parseExtraFields parses the optional fields from trace lines
func parseExtraFields(extra string, msg *MCPMessage) {
	// Split extra fields by space and parse each one
	fields := strings.Fields(extra)
	for _, field := range fields {
		if strings.HasPrefix(field, "spanid=") {
			msg.SpanID = strings.TrimPrefix(field, "spanid=")
		} else if strings.HasPrefix(field, "linksto=") {
			msg.LinksTo = strings.TrimPrefix(field, "linksto=")
		} else if strings.HasPrefix(field, "baggage=") {
			baggageStr := strings.TrimPrefix(field, "baggage=")
			parseBaggage(baggageStr, msg.Baggage)
		}
	}
}

func parseBaggage(baggageStr string, baggage map[string]string) {
	pairs := strings.Split(baggageStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			baggage[kv[0]] = kv[1]
		}
	}
}

func createRootSpan(ctx context.Context, tracer trace.Tracer, header *TraceHeader) (context.Context, trace.Span) {
	// Parse W3C trace context
	parts := strings.Split(header.TraceParent, "-")
	if len(parts) != 4 {
		return tracer.Start(ctx, "mcp-trace-root")
	}

	// Extract trace ID and span ID
	traceID, _ := trace.TraceIDFromHex(parts[1])
	spanID, _ := trace.SpanIDFromHex(parts[2])
	
	// Create span context
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	// Create context with remote span context
	ctx = trace.ContextWithRemoteSpanContext(ctx, spanCtx)
	
	// Start new span as child
	return tracer.Start(ctx, "mcp-trace",
		trace.WithAttributes(
			attribute.String("mcp.trace.version", header.Version),
		))
}

func createSpanForMessage(ctx context.Context, tracer trace.Tracer, msg *MCPMessage, header *TraceHeader) trace.Span {
	// Determine span name
	spanName := fmt.Sprintf("mcp.%s", msg.Direction)
	if method, ok := msg.JSON["method"].(string); ok {
		spanName = fmt.Sprintf("mcp.%s.%s", msg.Direction, method)
	}

	// Create span with timestamp
	opts := []trace.SpanStartOption{
		trace.WithTimestamp(msg.Timestamp),
		trace.WithSpanKind(getSpanKind(msg)),
	}

	// If we have a span ID from the trace, use it
	if msg.SpanID != "" {
		if _, err := trace.SpanIDFromHex(msg.SpanID); err == nil {
			// Note: OpenTelemetry doesn't allow setting custom span IDs directly
			// We'll add it as an attribute instead
			opts = append(opts, trace.WithAttributes(
				attribute.String("mcp.span_id", msg.SpanID),
			))
		}
	}

	_, span := tracer.Start(ctx, spanName, opts...)

	// Add attributes
	addAttributes(span, msg)

	return span
}

func getSpanKind(msg *MCPMessage) trace.SpanKind {
	if msg.Direction == "recv" {
		return trace.SpanKindServer
	}
	return trace.SpanKindClient
}

func addAttributes(span trace.Span, msg *MCPMessage) {
	// Basic attributes
	span.SetAttributes(
		attribute.String("mcp.direction", msg.Direction),
		attribute.String("mcp.timestamp", msg.Timestamp.Format(time.RFC3339Nano)),
	)

	// JSON-RPC attributes
	if jsonrpc, ok := msg.JSON["jsonrpc"].(string); ok {
		span.SetAttributes(attribute.String("rpc.jsonrpc.version", jsonrpc))
	}
	if method, ok := msg.JSON["method"].(string); ok {
		span.SetAttributes(
			attribute.String("rpc.method", method),
			attribute.String("mcp.method", method),
		)
	}
	if id, ok := msg.JSON["id"]; ok {
		span.SetAttributes(attribute.String("rpc.jsonrpc.request_id", fmt.Sprint(id)))
	}
	if err, ok := msg.JSON["error"]; ok {
		span.SetAttributes(attribute.String("rpc.jsonrpc.error_message", fmt.Sprint(err)))
		span.SetStatus(1, "JSON-RPC error") // 1 = Error status
	}

	// Baggage attributes
	for k, v := range msg.Baggage {
		span.SetAttributes(attribute.String(fmt.Sprintf("mcp.baggage.%s", k), v))
	}

	// Shadow/link attributes
	if msg.LinksTo != "" {
		span.SetAttributes(attribute.String("mcp.links_to", msg.LinksTo))
	}
}

func addLinkedEvent(span trace.Span, linkedMsg *MCPMessage) {
	eventName := fmt.Sprintf("shadow_%s", linkedMsg.Direction)
	
	attrs := []attribute.KeyValue{
		attribute.String("mcp.shadow.span_id", linkedMsg.SpanID),
		attribute.String("mcp.shadow.direction", linkedMsg.Direction),
	}

	// Add shadow response details
	if result, ok := linkedMsg.JSON["result"]; ok {
		resultJSON, _ := json.Marshal(result)
		attrs = append(attrs, attribute.String("mcp.shadow.result", string(resultJSON)))
	}
	if error, ok := linkedMsg.JSON["error"]; ok {
		errorJSON, _ := json.Marshal(error)
		attrs = append(attrs, attribute.String("mcp.shadow.error", string(errorJSON)))
	}

	// Add shadow baggage
	for k, v := range linkedMsg.Baggage {
		attrs = append(attrs, attribute.String(fmt.Sprintf("mcp.shadow.baggage.%s", k), v))
	}

	span.AddEvent(eventName, 
		trace.WithTimestamp(linkedMsg.Timestamp),
		trace.WithAttributes(attrs...),
	)
}