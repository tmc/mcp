package mcp

import (
	"encoding/json"
	"fmt"
	"sync"
)

// NotifyHandler handles an MCP notification
type NotifyHandler func(method string, params json.RawMessage) error

// Dispatcher manages MCP notification routing
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string][]NotifyHandler
}

// List change notification methods
const (
	MethodRootsListChanged   = "notifications/roots/list_changed"
	MethodProgressChanged    = "notifications/progress_changed"
	MethodLoggingMessageSent = "notifications/logging_message_sent"
)

// NewDispatcher creates a new notification dispatcher
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string][]NotifyHandler),
	}
}

// Handle registers a handler for a notification method
func (d *Dispatcher) Handle(method string, h NotifyHandler) {
	d.mu.Lock()
	d.handlers[method] = append(d.handlers[method], h)
	d.mu.Unlock()
}

// Dispatch sends a notification to all registered handlers
func (d *Dispatcher) Dispatch(method string, params json.RawMessage) error {
	d.mu.RLock()
	handlers := d.handlers[method]
	d.mu.RUnlock()

	var lastErr error
	for _, h := range handlers {
		if err := h(method, params); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// NotifyListChanged sends a list changed notification
func (d *Dispatcher) NotifyListChanged(method string) error {
	return d.Dispatch(method, nil)
}

// NotifyProgress sends a progress notification
func (d *Dispatcher) NotifyProgress(token interface{}, progress float64, total *float64) error {
	params := struct {
		Token    interface{} `json:"token"`
		Progress float64     `json:"progress"`
		Total    *float64    `json:"total,omitempty"`
	}{
		Token:    token,
		Progress: progress,
		Total:    total,
	}
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("error marshaling progress params: %w", err)
	}
	return d.Dispatch(MethodProgressChanged, data)
}

// NotifyLoggingMessage sends a logging message notification
func (d *Dispatcher) NotifyLoggingMessage(level LoggingLevel, logger string, data interface{}) error {
	params := struct {
		Level  LoggingLevel `json:"level"`
		Logger string       `json:"logger"`
		Data   interface{}  `json:"data"`
	}{
		Level:  level,
		Logger: logger,
		Data:   data,
	}
	msgData, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("error marshaling logging message params: %w", err)
	}
	return d.Dispatch(MethodLoggingMessageSent, msgData)
}
