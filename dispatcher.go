// Copyright 2025 The MCP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tmc/mcp/modelcontextprotocol"
)

// Dispatcher manages MCP notification routing.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string][]any
}

// NewDispatcher creates a new notification dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{handlers: make(map[string][]any)}
}

// Handle registers a handler for a notification method.
func (d *Dispatcher) Handle(method string, h NotificationHandlerFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[method] = append(d.handlers[method], h)
}

// Dispatch sends a notification to all registered handlers.
func (d *Dispatcher) Dispatch(ctx context.Context, method string, params json.RawMessage) error {
	d.mu.RLock()
	handlers, ok := d.handlers[method]
	d.mu.RUnlock()
	if !ok {
		return nil
	}

	var allErrors []error
	for _, h := range handlers {
		handler, ok := h.(NotificationHandlerFunc)
		if !ok {
			continue
		}
		if err := handler(ctx, method, params); err != nil {
			allErrors = append(allErrors, fmt.Errorf("handler error for %s: %w", method, err))
		}
	}
	if len(allErrors) > 0 {
		return fmt.Errorf("multiple handler errors: %v", allErrors)
	}
	return nil
}

// NotifyListChanged sends a list changed notification.
func (d *Dispatcher) NotifyListChanged(ctx context.Context, method Method) error {
	data, _ := json.Marshal(struct{}{})
	return d.Dispatch(ctx, string(method), data)
}

// NotifyProgress sends a progress notification.
func (d *Dispatcher) NotifyProgress(ctx context.Context, token any, progress float64, total *float64) error {
	params := modelcontextprotocol.ProgressNotificationParams{
		ProgressToken: token,
		Progress:      progress,
		Total:         total,
	}
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return d.Dispatch(ctx, string(MethodProgress), data)
}

// NotifyLoggingMessage sends a logging message notification.
func (d *Dispatcher) NotifyLoggingMessage(ctx context.Context, level LoggingLevel, logger string, data any) error {
	// Marshal the data properly
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	params := modelcontextprotocol.LoggingMessageNotificationParams{
		Level:  modelcontextprotocol.LoggingLevel(level),
		Logger: &logger,
		Data:   dataJSON,
	}
	paramData, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return d.Dispatch(ctx, string(MethodLogging), paramData)
}

// NotifyElicitationComplete sends an out-of-band elicitation completion notification.
func (d *Dispatcher) NotifyElicitationComplete(ctx context.Context, elicitationID string) error {
	if elicitationID == "" {
		return NewParameterError(string(MethodElicitationComplete), "elicitationId", "missing required elicitation id", nil)
	}
	params := modelcontextprotocol.ElicitationCompleteNotificationParams{
		ElicitationID: elicitationID,
	}
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return d.Dispatch(ctx, string(MethodElicitationComplete), data)
}
