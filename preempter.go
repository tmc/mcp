package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"golang.org/x/exp/jsonrpc2"
)

// CancellablePreempter handles `notifications/cancelled` MCP messages
// to cancel in-flight requests.
type CancellablePreempter struct {
	Conn   *jsonrpc2.Connection
	Logger *slog.Logger
}

// CancelledNotificationParams matches the MCP spec for `notifications/cancelled`.
type CancelledNotificationParams struct {
	RequestID json.RawMessage `json:"requestId"`
	Reason    string          `json:"reason,omitempty"`
}

// Preempt implements the jsonrpc2.Preempter interface.
func (p *CancellablePreempter) Preempt(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	logger := p.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if req.Method == string(MethodNotificationCancelled) {
		if p.Conn == nil {
			logger.ErrorContext(ctx, "CancellablePreempter: Connection is nil, cannot process cancellation", "method", req.Method)
			return nil, jsonrpc2.ErrNotHandled
		}

		var params CancelledNotificationParams
		if errUnmarshal := json.Unmarshal(req.Params, &params); errUnmarshal != nil {
			logger.ErrorContext(ctx, "Failed to unmarshal cancellation params", "error", errUnmarshal, "params", string(req.Params))
			return nil, fmt.Errorf("%w: invalid cancellation params: %s", jsonrpc2.ErrInvalidRequest, errUnmarshal.Error())
		}

		var rpcID jsonrpc2.ID
		if errUnmarshal := json.Unmarshal(params.RequestID, &rpcID); errUnmarshal != nil {
			logger.ErrorContext(ctx, "Failed to unmarshal cancellation requestId", "error", errUnmarshal, "requestId", string(params.RequestID))
			return nil, fmt.Errorf("%w: invalid requestId in cancellation: %s", jsonrpc2.ErrInvalidRequest, errUnmarshal.Error())
		}

		logger.InfoContext(ctx, "Received cancellation notification, attempting to cancel in-flight request",
			"request_id_to_cancel", rpcID, "reason", params.Reason)

		p.Conn.Cancel(rpcID)
		return nil, jsonrpc2.ErrNotHandled
	}
	return nil, jsonrpc2.ErrNotHandled
}
