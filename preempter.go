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
	RequestID interface{} `json:"requestId"`
	Reason    string      `json:"reason,omitempty"`
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

		logger.InfoContext(ctx, "Debug: cancellation params received", "params", string(req.Params), "requestId_raw", params.RequestID)

		// Convert the requestId to jsonrpc2.ID
		var rpcID jsonrpc2.ID
		switch v := params.RequestID.(type) {
		case float64:
			// JSON numbers unmarshal as float64
			rpcID = jsonrpc2.Int64ID(int64(v))
		case string:
			rpcID = jsonrpc2.StringID(v)
		case nil:
			logger.ErrorContext(ctx, "RequestID is nil in cancellation")
			return nil, fmt.Errorf("%w: nil requestId in cancellation", jsonrpc2.ErrInvalidRequest)
		default:
			logger.ErrorContext(ctx, "Unexpected requestId type", "type", fmt.Sprintf("%T", v), "value", v)
			return nil, fmt.Errorf("%w: invalid requestId type %T", jsonrpc2.ErrInvalidRequest, v)
		}

		logger.InfoContext(ctx, "Received cancellation notification, attempting to cancel in-flight request",
			"request_id_to_cancel", rpcID, "reason", params.Reason)

		p.Conn.Cancel(rpcID)
		return nil, jsonrpc2.ErrNotHandled
	}
	return nil, jsonrpc2.ErrNotHandled
}
