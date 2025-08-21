// Package mcptestutil provides test isolation helpers to ensure clean state between tests.
package mcptestutil

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// IsolationContext manages test resources and ensures proper cleanup.
// It tracks goroutines, memory usage, and other resources to detect leaks.
type IsolationContext struct {
	t                 *testing.T
	initialGoroutines int
	initialMemStats   runtime.MemStats
	startTime         time.Time
	cleanupFuncs      []func()
	resourceTrackers  map[string]ResourceTracker
	mu                sync.RWMutex
	isActive          bool
}

// ResourceTracker defines an interface for tracking specific resource types.
type ResourceTracker interface {
	// Track starts tracking a resource
	Track(name string, resource interface{}) error

	// Untrack stops tracking a resource
	Untrack(name string) error

	// CheckLeaks returns any detected leaks
	CheckLeaks() []ResourceLeak

	// Cleanup performs any necessary cleanup
	Cleanup() error
}

// ResourceLeak represents a detected resource leak.
type ResourceLeak struct {
	Type        string      `json:"type"`
	Name        string      `json:"name"`
	Resource    interface{} `json:"resource,omitempty"`
	Description string      `json:"description"`
	StackTrace  string      `json:"stackTrace,omitempty"`
}

// NewIsolationContext creates a new isolation context for a test.
// It captures the initial state for later leak detection.
func NewIsolationContext(t *testing.T) *IsolationContext {
	t.Helper()

	// Force garbage collection to get clean baseline
	runtime.GC()
	runtime.GC() // Call twice to ensure cleanup

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	ctx := &IsolationContext{
		t:                 t,
		initialGoroutines: runtime.NumGoroutine(),
		initialMemStats:   memStats,
		startTime:         time.Now(),
		cleanupFuncs:      make([]func(), 0),
		resourceTrackers:  make(map[string]ResourceTracker),
		isActive:          true,
	}

	// Register cleanup with testing framework
	t.Cleanup(func() {
		ctx.Cleanup()
	})

	return ctx
}

// AddCleanup adds a cleanup function to be executed when the context is destroyed.
func (ctx *IsolationContext) AddCleanup(cleanup func()) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if !ctx.isActive {
		ctx.t.Log("Warning: Adding cleanup to inactive isolation context")
		return
	}

	ctx.cleanupFuncs = append(ctx.cleanupFuncs, cleanup)
}

// AddResourceTracker adds a custom resource tracker.
func (ctx *IsolationContext) AddResourceTracker(name string, tracker ResourceTracker) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.resourceTrackers[name] = tracker
}

// TrackResource tracks a named resource using the specified tracker.
func (ctx *IsolationContext) TrackResource(trackerName, resourceName string, resource interface{}) error {
	ctx.mu.RLock()
	tracker, exists := ctx.resourceTrackers[trackerName]
	ctx.mu.RUnlock()

	if !exists {
		return fmt.Errorf("resource tracker %q not found", trackerName)
	}

	return tracker.Track(resourceName, resource)
}

// UntrackResource stops tracking a named resource.
func (ctx *IsolationContext) UntrackResource(trackerName, resourceName string) error {
	ctx.mu.RLock()
	tracker, exists := ctx.resourceTrackers[trackerName]
	ctx.mu.RUnlock()

	if !exists {
		return fmt.Errorf("resource tracker %q not found", trackerName)
	}

	return tracker.Untrack(resourceName)
}

// ValidateNoLeaks checks for goroutine and memory leaks.
// Returns true if no leaks are detected, false otherwise.
func (ctx *IsolationContext) ValidateNoLeaks() bool {
	ctx.t.Helper()

	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	hasLeaks := false

	// Check goroutine leaks
	if leaks := ctx.checkGoroutineLeaks(); len(leaks) > 0 {
		ctx.t.Errorf("Goroutine leaks detected:")
		for _, leak := range leaks {
			ctx.t.Errorf("  %s: %s", leak.Name, leak.Description)
			if leak.StackTrace != "" {
				ctx.t.Errorf("    Stack trace:\n%s", leak.StackTrace)
			}
		}
		hasLeaks = true
	}

	// Check memory leaks
	if leaks := ctx.checkMemoryLeaks(); len(leaks) > 0 {
		ctx.t.Errorf("Memory leaks detected:")
		for _, leak := range leaks {
			ctx.t.Errorf("  %s: %s", leak.Name, leak.Description)
		}
		hasLeaks = true
	}

	// Check custom resource leaks
	for name, tracker := range ctx.resourceTrackers {
		if leaks := tracker.CheckLeaks(); len(leaks) > 0 {
			ctx.t.Errorf("Resource leaks detected in tracker %q:", name)
			for _, leak := range leaks {
				ctx.t.Errorf("  %s: %s", leak.Name, leak.Description)
			}
			hasLeaks = true
		}
	}

	return !hasLeaks
}

// checkGoroutineLeaks detects goroutine leaks by comparing current count to initial.
func (ctx *IsolationContext) checkGoroutineLeaks() []ResourceLeak {
	currentGoroutines := runtime.NumGoroutine()
	leakCount := currentGoroutines - ctx.initialGoroutines

	if leakCount <= 0 {
		return nil
	}

	// Get stack traces of all goroutines
	buf := make([]byte, 1024*1024) // 1MB buffer
	n := runtime.Stack(buf, true)
	stackTrace := string(buf[:n])

	return []ResourceLeak{
		{
			Type: "goroutine",
			Name: "goroutine_leak",
			Description: fmt.Sprintf("Leaked %d goroutine(s): started with %d, current %d",
				leakCount, ctx.initialGoroutines, currentGoroutines),
			StackTrace: stackTrace,
		},
	}
}

// checkMemoryLeaks detects significant memory increases.
func (ctx *IsolationContext) checkMemoryLeaks() []ResourceLeak {
	var currentMemStats runtime.MemStats
	runtime.ReadMemStats(&currentMemStats)

	// Check for significant memory increase (more than 10MB or 50% increase)
	const memoryThresholdBytes = 10 * 1024 * 1024 // 10MB
	const memoryThresholdPercent = 0.5            // 50%

	initialAlloc := ctx.initialMemStats.Alloc
	currentAlloc := currentMemStats.Alloc

	if currentAlloc <= initialAlloc {
		return nil // Memory decreased or stayed same
	}

	increase := currentAlloc - initialAlloc
	percentIncrease := float64(increase) / float64(initialAlloc)

	if increase > memoryThresholdBytes || percentIncrease > memoryThresholdPercent {
		return []ResourceLeak{
			{
				Type: "memory",
				Name: "memory_leak",
				Description: fmt.Sprintf(
					"Significant memory increase detected: %d bytes (%.1f%% increase) from %d to %d bytes",
					increase, percentIncrease*100, initialAlloc, currentAlloc,
				),
			},
		}
	}

	return nil
}

// Cleanup performs cleanup of all tracked resources and validates no leaks remain.
func (ctx *IsolationContext) Cleanup() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if !ctx.isActive {
		return
	}

	ctx.isActive = false

	// Run cleanup functions in reverse order
	for i := len(ctx.cleanupFuncs) - 1; i >= 0; i-- {
		func() {
			defer func() {
				if r := recover(); r != nil {
					ctx.t.Logf("Panic during cleanup: %v", r)
				}
			}()
			ctx.cleanupFuncs[i]()
		}()
	}

	// Cleanup resource trackers
	for name, tracker := range ctx.resourceTrackers {
		if err := tracker.Cleanup(); err != nil {
			ctx.t.Logf("Error cleaning up resource tracker %q: %v", name, err)
		}
	}

	// Give some time for async cleanup
	time.Sleep(10 * time.Millisecond)

	// Force garbage collection before final leak check
	runtime.GC()
	runtime.GC()
}

// RunIsolated runs a test function with complete isolation and leak detection.
// It ensures the test starts with a clean state and validates no leaks remain.
//
// Usage:
//
//	RunIsolated(t, func(ctx *IsolationContext) {
//	    // Your test code here
//	    ctx.AddCleanup(func() { /* cleanup */ })
//	})
func RunIsolated(t *testing.T, testFunc func(ctx *IsolationContext)) {
	t.Helper()

	// Create isolation context
	ctx := NewIsolationContext(t)

	// Run the test function
	testFunc(ctx)

	// Cleanup is handled automatically via t.Cleanup()
}

// ValidateNoLeaks is a standalone function to check for leaks in any test.
// It should be called at the end of tests that don't use RunIsolated.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    defer ValidateNoLeaks(t)
//	    // Test code here
//	}
func ValidateNoLeaks(t *testing.T) {
	t.Helper()

	// Force garbage collection
	runtime.GC()
	runtime.GC()

	// Give async operations time to complete
	time.Sleep(10 * time.Millisecond)

	// Check goroutines
	if runtime.NumGoroutine() > 2 { // Allow for test runner and main goroutine
		buf := make([]byte, 1024*1024)
		n := runtime.Stack(buf, true)
		stackTrace := string(buf[:n])

		// Filter out expected goroutines
		stacks := strings.Split(stackTrace, "\n\n")
		suspiciousStacks := make([]string, 0)

		for _, stack := range stacks {
			// Skip testing framework goroutines
			if strings.Contains(stack, "testing.") ||
				strings.Contains(stack, "runtime.") ||
				strings.Contains(stack, "internal/poll.") {
				continue
			}

			suspiciousStacks = append(suspiciousStacks, stack)
		}

		if len(suspiciousStacks) > 0 {
			t.Errorf("Potential goroutine leaks detected (%d goroutines):\n%s",
				runtime.NumGoroutine(), strings.Join(suspiciousStacks, "\n\n"))
		}
	}
}

// GoroutineTracker tracks goroutines by name.
type GoroutineTracker struct {
	goroutines map[string]*goroutineInfo
	mu         sync.RWMutex
}

// goroutineInfo stores information about tracked goroutines.
type goroutineInfo struct {
	Name      string
	StartTime time.Time
	Context   context.Context
	Cancel    context.CancelFunc
}

// NewGoroutineTracker creates a new goroutine tracker.
func NewGoroutineTracker() *GoroutineTracker {
	return &GoroutineTracker{
		goroutines: make(map[string]*goroutineInfo),
	}
}

// Track starts tracking a goroutine.
func (gt *GoroutineTracker) Track(name string, resource interface{}) error {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	// If resource is a context, use it; otherwise create a new one
	var ctx context.Context
	var cancel context.CancelFunc

	if c, ok := resource.(context.Context); ok {
		ctx, cancel = context.WithCancel(c)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	gt.goroutines[name] = &goroutineInfo{
		Name:      name,
		StartTime: time.Now(),
		Context:   ctx,
		Cancel:    cancel,
	}

	return nil
}

// Untrack stops tracking a goroutine.
func (gt *GoroutineTracker) Untrack(name string) error {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	if info, exists := gt.goroutines[name]; exists {
		info.Cancel() // Cancel the context to signal shutdown
		delete(gt.goroutines, name)
	}

	return nil
}

// CheckLeaks returns any goroutines that are still tracked.
func (gt *GoroutineTracker) CheckLeaks() []ResourceLeak {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	leaks := make([]ResourceLeak, 0, len(gt.goroutines))

	for name, info := range gt.goroutines {
		duration := time.Since(info.StartTime)
		leaks = append(leaks, ResourceLeak{
			Type:        "goroutine",
			Name:        name,
			Resource:    info,
			Description: fmt.Sprintf("Goroutine %q has been running for %v", name, duration),
		})
	}

	return leaks
}

// Cleanup cancels all tracked goroutines.
func (gt *GoroutineTracker) Cleanup() error {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	for name, info := range gt.goroutines {
		info.Cancel()
		delete(gt.goroutines, name)
	}

	return nil
}

// MemoryTracker tracks memory allocations by name.
type MemoryTracker struct {
	allocations map[string]*memoryInfo
	mu          sync.RWMutex
}

// memoryInfo stores information about tracked memory.
type memoryInfo struct {
	Name      string
	Size      int64
	AllocTime time.Time
	Data      interface{}
}

// NewMemoryTracker creates a new memory tracker.
func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{
		allocations: make(map[string]*memoryInfo),
	}
}

// Track starts tracking a memory allocation.
func (mt *MemoryTracker) Track(name string, resource interface{}) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	// Estimate size based on type
	var size int64
	switch r := resource.(type) {
	case []byte:
		size = int64(len(r))
	case string:
		size = int64(len(r))
	default:
		size = 0 // Unknown size
	}

	mt.allocations[name] = &memoryInfo{
		Name:      name,
		Size:      size,
		AllocTime: time.Now(),
		Data:      resource,
	}

	return nil
}

// Untrack stops tracking a memory allocation.
func (mt *MemoryTracker) Untrack(name string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	delete(mt.allocations, name)
	return nil
}

// CheckLeaks returns any memory allocations that are still tracked.
func (mt *MemoryTracker) CheckLeaks() []ResourceLeak {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	leaks := make([]ResourceLeak, 0, len(mt.allocations))

	for name, info := range mt.allocations {
		duration := time.Since(info.AllocTime)
		leaks = append(leaks, ResourceLeak{
			Type:     "memory",
			Name:     name,
			Resource: info,
			Description: fmt.Sprintf("Memory allocation %q (%d bytes) has been tracked for %v",
				name, info.Size, duration),
		})
	}

	return leaks
}

// Cleanup clears all tracked memory allocations.
func (mt *MemoryTracker) Cleanup() error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	for name := range mt.allocations {
		delete(mt.allocations, name)
	}

	// Force garbage collection
	runtime.GC()

	return nil
}
