package mcp

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Handler handles an MCP notification
type Handler func(method string, params json.RawMessage) error

// Dispatcher manages MCP notification routing
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// List change notification methods
const (
	MethodRootsListChanged    = "notifications/roots/list_changed"
	MethodResourceListChanged = "notifications/resources/list_changed"
	MethodPromptListChanged   = "notifications/prompts/list_changed"
	MethodToolListChanged     = "notifications/tools/list_changed"
)

// Progress and logging notification methods
const (
	MethodProgress = "notifications/progress"
	MethodLogging  = "notifications/message"
)

// Other notification methods
const (
	MethodCancelled   = "notifications/cancelled"
	MethodInitialized = "notifications/initialized"
)

// NewDispatcher creates a new notification dispatcher
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string][]Handler),
	}
}

// Handle registers a handler for a notification method
func (d *Dispatcher) Handle(method string, h Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[method] = append(d.handlers[method], h)
}

// Dispatch sends a notification to all registered handlers
func (d *Dispatcher) Dispatch(method string, params json.RawMessage) error {
	d.mu.RLock()
	handlers := d.handlers[method]
	d.mu.RUnlock()

	for _, h := range handlers {
		if err := h(method, params); err != nil {
			return fmt.Errorf("handler error: %w", err)
		}
	}
	return nil
}

// List change notifications
func (d *Dispatcher) NotifyListChanged(method string) error {
	return d.Dispatch(method, nil)
}

// Progress notifications
func (d *Dispatcher) NotifyProgress(token interface{}, progress float64, total *float64) error {
	params := struct {
		Token    interface{} `json:"progressToken"`
		Progress float64     `json:"progress"`
		Total    *float64    `json:"total,omitempty"`
	}{
		Token:    token,
		Progress: progress,
		Total:    total,
	}
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return d.Dispatch(MethodProgress, data)
}

// Logging notifications
func (d *Dispatcher) NotifyLoggingMessage(level LoggingLevel, logger string, data interface{}) error {
	params := struct {
		Level  LoggingLevel `json:"level"`
		Logger string       `json:"logger,omitempty"`
		Data   interface{}  `json:"data"`
	}{
		Level:  level,
		Logger: logger,
		Data:   data,
	}
	msgData, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return d.Dispatch(MethodLogging, msgData)
}
