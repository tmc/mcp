package mcp

import (
	"fmt"
	"log/slog"
	"runtime/debug"
)

// recoverPanic converts a recovered panic value into an error. It is used at
// the top of goroutines that run caller-supplied code (handlers, callbacks) so
// a single panic degrades one request instead of crashing the whole process: a
// recover only catches panics in its own goroutine, so each spawned goroutine
// needs its own.
func recoverPanic(r any) error {
	if r == nil {
		return nil
	}
	if err, ok := r.(error); ok {
		return fmt.Errorf("panic: %w", err)
	}
	return fmt.Errorf("panic: %v", r)
}

// safeGo runs fn in a new goroutine with panic recovery. A panic is logged via
// logger (or slog.Default if nil) together with the stack, and does not
// propagate. Use it for fire-and-forget callbacks where there is no channel to
// report the error on; when the caller waits on a result channel, recover
// inline instead so the panic can be surfaced as an error (see server.go and
// the timeout middleware).
func safeGo(logger *slog.Logger, what string, fn func()) {
	if logger == nil {
		logger = slog.Default()
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("recovered panic in "+what,
					"error", recoverPanic(r),
					"stack", string(debug.Stack()))
			}
		}()
		fn()
	}()
}
