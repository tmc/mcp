package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
)

// Progress tracking and cancellation support for MCP operations

// ProgressToken represents a progress token that can be a string or number
type ProgressToken interface{}

// Progress represents a progress tracking object
type Progress struct {
	token   ProgressToken
	total   *float64
	value   float64
	message string
	logger  *slog.Logger
	mu      sync.RWMutex
}

// ProgressNotification represents a progress notification message
type ProgressNotification struct {
	ProgressToken ProgressToken `json:"progressToken"`
	Progress      float64       `json:"progress"`
	Total         *float64      `json:"total,omitempty"`
	Message       string        `json:"message,omitempty"`
}

// CancelledNotification represents a cancellation notification
type CancelledNotification struct {
	RequestID string `json:"requestId"`
	Reason    string `json:"reason,omitempty"`
}

// NewProgress creates a new progress tracker
func NewProgress(token ProgressToken, total *float64, logger *slog.Logger) *Progress {
	if logger == nil {
		logger = slog.Default()
	}

	return &Progress{
		token:  token,
		total:  total,
		value:  0,
		logger: logger,
	}
}

// Update updates the progress value and message
func (p *Progress) Update(value float64, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.value = value
	p.message = message

	p.logger.Debug("Progress updated",
		"token", p.token,
		"value", value,
		"total", p.total,
		"message", message,
	)
}

// Value returns the current progress value
func (p *Progress) Value() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.value
}

// Message returns the current progress message
func (p *Progress) Message() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.message
}

// Total returns the total progress value
func (p *Progress) Total() *float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.total
}

// IsComplete returns true if progress is complete (value >= total)
func (p *Progress) IsComplete() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.total == nil {
		return false // Cannot determine completion without total
	}

	return p.value >= *p.total
}

// ToNotification converts the progress to a notification message
func (p *Progress) ToNotification() *JSONRPCNotification {
	p.mu.RLock()
	defer p.mu.RUnlock()

	params := ProgressNotification{
		ProgressToken: p.token,
		Progress:      p.value,
		Total:         p.total,
		Message:       p.message,
	}

	paramsData, _ := json.Marshal(params)

	return &JSONRPCNotification{
		Method: string(MethodProgress),
		Params: json.RawMessage(paramsData),
	}
}

// ProgressManager manages multiple progress trackers
type ProgressManager struct {
	trackers map[ProgressToken]*Progress
	mu       sync.RWMutex
	logger   *slog.Logger
}

// NewProgressManager creates a new progress manager
func NewProgressManager(logger *slog.Logger) *ProgressManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &ProgressManager{
		trackers: make(map[ProgressToken]*Progress),
		logger:   logger,
	}
}

// CreateProgress creates and registers a new progress tracker
func (pm *ProgressManager) CreateProgress(token ProgressToken, total *float64) *Progress {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	progress := NewProgress(token, total, pm.logger)
	pm.trackers[token] = progress

	pm.logger.Debug("Progress tracker created", "token", token, "total", total)
	return progress
}

// GetProgress retrieves a progress tracker by token
func (pm *ProgressManager) GetProgress(token ProgressToken) (*Progress, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	progress, exists := pm.trackers[token]
	return progress, exists
}

// UpdateProgress updates a progress tracker
func (pm *ProgressManager) UpdateProgress(token ProgressToken, value float64, message string) bool {
	pm.mu.RLock()
	progress, exists := pm.trackers[token]
	pm.mu.RUnlock()

	if !exists {
		return false
	}

	progress.Update(value, message)
	return true
}

// RemoveProgress removes a progress tracker
func (pm *ProgressManager) RemoveProgress(token ProgressToken) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.trackers, token)
	pm.logger.Debug("Progress tracker removed", "token", token)
}

// ListProgress returns all active progress trackers
func (pm *ProgressManager) ListProgress() []*Progress {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var trackers []*Progress
	for _, progress := range pm.trackers {
		trackers = append(trackers, progress)
	}

	return trackers
}

// CancelManager manages operation cancellation
type CancelManager struct {
	cancelled map[string]context.CancelFunc
	reasons   map[string]string
	mu        sync.RWMutex
	logger    *slog.Logger
}

// NewCancelManager creates a new cancellation manager
func NewCancelManager(logger *slog.Logger) *CancelManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &CancelManager{
		cancelled: make(map[string]context.CancelFunc),
		reasons:   make(map[string]string),
		logger:    logger,
	}
}

// RegisterCancellation registers a cancellation function for a request
func (cm *CancelManager) RegisterCancellation(requestID string, cancelFunc context.CancelFunc) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cancelled[requestID] = cancelFunc
	cm.logger.Debug("Cancellation registered", "requestId", requestID)
}

// CancelRequest cancels a request by ID
func (cm *CancelManager) CancelRequest(requestID string, reason string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cancelFunc, exists := cm.cancelled[requestID]
	if !exists {
		return false
	}

	cm.reasons[requestID] = reason
	cancelFunc()

	cm.logger.Info("Request cancelled", "requestId", requestID, "reason", reason)
	return true
}

// IsCancelled checks if a request has been cancelled
func (cm *CancelManager) IsCancelled(requestID string) (bool, string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	reason, cancelled := cm.reasons[requestID]
	return cancelled, reason
}

// UnregisterCancellation removes a cancellation registration
func (cm *CancelManager) UnregisterCancellation(requestID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.cancelled, requestID)
	delete(cm.reasons, requestID)
	cm.logger.Debug("Cancellation unregistered", "requestId", requestID)
}

// CreateCancelledNotification creates a cancellation notification
func (cm *CancelManager) CreateCancelledNotification(requestID string) *JSONRPCNotification {
	cm.mu.RLock()
	reason := cm.reasons[requestID]
	cm.mu.RUnlock()

	params := CancelledNotification{
		RequestID: requestID,
		Reason:    reason,
	}

	paramsData, _ := json.Marshal(params)

	return &JSONRPCNotification{
		Method: string(MethodNotificationCancelled),
		Params: json.RawMessage(paramsData),
	}
}

// ContextWithProgress adds progress tracking to a context
func ContextWithProgress(ctx context.Context, progress *Progress) context.Context {
	return context.WithValue(ctx, progressKey, progress)
}

// ProgressFromContext retrieves progress tracker from context
func ProgressFromContext(ctx context.Context) (*Progress, bool) {
	progress, ok := ctx.Value(progressKey).(*Progress)
	return progress, ok
}

// ContextWithCancelManager adds cancel manager to a context
func ContextWithCancelManager(ctx context.Context, cm *CancelManager) context.Context {
	return context.WithValue(ctx, cancelManagerKey, cm)
}

// CancelManagerFromContext retrieves cancel manager from context
func CancelManagerFromContext(ctx context.Context) (*CancelManager, bool) {
	cm, ok := ctx.Value(cancelManagerKey).(*CancelManager)
	return cm, ok
}

// WithProgress is a helper to create a context with progress tracking
func WithProgress(ctx context.Context, token ProgressToken, total *float64) (context.Context, *Progress) {
	progress := NewProgress(token, total, slog.Default())
	return ContextWithProgress(ctx, progress), progress
}

// WithCancellation is a helper to create a cancellable context
func WithCancellation(ctx context.Context, requestID string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	// Add to cancel manager if available
	if cm, ok := CancelManagerFromContext(ctx); ok {
		cm.RegisterCancellation(requestID, cancel)
	}

	return ctx, cancel
}

// UpdateProgressInContext updates progress if available in context
func UpdateProgressInContext(ctx context.Context, value float64, message string) bool {
	if progress, ok := ProgressFromContext(ctx); ok {
		progress.Update(value, message)
		return true
	}
	return false
}

// CheckCancellation checks if the context has been cancelled and returns appropriate error
func CheckCancellation(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// ProgressCallback is a function type for progress callbacks
type ProgressCallback func(token ProgressToken, value float64, total *float64, message string)

// CancellationCallback is a function type for cancellation callbacks
type CancellationCallback func(requestID string, reason string)

// ProgressHandler handles progress notifications
type ProgressHandler struct {
	callbacks []ProgressCallback
	mu        sync.RWMutex
}

// NewProgressHandler creates a new progress handler
func NewProgressHandler() *ProgressHandler {
	return &ProgressHandler{}
}

// AddCallback adds a progress callback
func (ph *ProgressHandler) AddCallback(callback ProgressCallback) {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.callbacks = append(ph.callbacks, callback)
}

// NotifyProgress notifies all callbacks of progress update
func (ph *ProgressHandler) NotifyProgress(token ProgressToken, value float64, total *float64, message string) {
	ph.mu.RLock()
	callbacks := make([]ProgressCallback, len(ph.callbacks))
	copy(callbacks, ph.callbacks)
	ph.mu.RUnlock()

	for _, callback := range callbacks {
		cb := callback
		safeGo(nil, "progress callback", func() {
			cb(token, value, total, message)
		})
	}
}

// CancellationHandler handles cancellation notifications
type CancellationHandler struct {
	callbacks []CancellationCallback
	mu        sync.RWMutex
}

// NewCancellationHandler creates a new cancellation handler
func NewCancellationHandler() *CancellationHandler {
	return &CancellationHandler{}
}

// AddCallback adds a cancellation callback
func (ch *CancellationHandler) AddCallback(callback CancellationCallback) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.callbacks = append(ch.callbacks, callback)
}

// NotifyCancellation notifies all callbacks of cancellation
func (ch *CancellationHandler) NotifyCancellation(requestID string, reason string) {
	ch.mu.RLock()
	callbacks := make([]CancellationCallback, len(ch.callbacks))
	copy(callbacks, ch.callbacks)
	ch.mu.RUnlock()

	for _, callback := range callbacks {
		cb := callback
		safeGo(nil, "cancellation callback", func() {
			cb(requestID, reason)
		})
	}
}
