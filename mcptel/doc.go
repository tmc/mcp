// Package mcptel is the telemetry package for the mcp project.
package mcptel

import "context"

// Tracer is an interface for creating spans.
type Tracer interface {
	Start(ctx context.Context, spanName string, opts ...SpanStartOption) (context.Context, Span)
}

// SpanStartOption is a function that modifies the span start options.
type SpanStartOption any

// Span is an interface for a span.
type Span interface {
	End(opts ...SpanEndOption)
	SetName(name string)
	AddEvent(name string, opts ...EventOption)
	SetAttributes(attrs ...KeyValue)
}

// EventOption is a function that modifies the event options.
type EventOption any

// SpanEndOption is a function that modifies the span end options.
type SpanEndOption any

// KeyValue is a key-value pair.
type KeyValue struct {
	Key   string
	Value any
}

type spanKey struct{}

// WithSpan adds a span to the context.
func WithSpan(ctx context.Context, span Span) context.Context {
	return context.WithValue(ctx, spanKey{}, span)
}

func SpanFromContext(ctx context.Context) Span {
	span, ok := ctx.Value(spanKey{}).(Span)
	if !ok {
		return nil
	}
	return span
}

// CurrentSpan returns the current span from the context.
func CurrentSpan(ctx context.Context) (Span, context.Context) {
	span := SpanFromContext(ctx)
	return span, ctx
}
