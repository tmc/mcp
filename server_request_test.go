// Copyright 2025 The MCP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.

package mcp

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/exp/jsonrpc2"
)

// TestServerRequestGuardsWithoutConnection verifies that server-initiated
// requests fail cleanly when no client connection is established.
func TestServerRequestGuardsWithoutConnection(t *testing.T) {
	server := NewServer("test-server", "1.0.0")
	ctx := context.Background()

	if _, err := server.CreateMessage(ctx, CreateMessageRequest{}); err == nil {
		t.Error("CreateMessage succeeded without connection")
	}
	if _, err := server.Elicit(ctx, ElicitRequest{Message: "x"}); err == nil {
		t.Error("Elicit succeeded without connection")
	}
	if _, err := server.ListRoots(ctx); err == nil {
		t.Error("ListRoots succeeded without connection")
	}
}

// TestServerRequestGuardsUnsupportedCapability verifies that server-initiated
// requests return ErrUnsupported when the client did not advertise the matching
// capability. A non-nil sentinel connection lets the call reach the capability
// gate, which returns before the connection is ever used.
func TestServerRequestGuardsUnsupportedCapability(t *testing.T) {
	server := NewServer("test-server", "1.0.0")
	server.mu.Lock()
	server.conn = &jsonrpc2.Connection{}
	server.clientCaps = ClientCapabilities{}
	server.mu.Unlock()

	ctx := context.Background()
	if _, err := server.CreateMessage(ctx, CreateMessageRequest{}); !errors.Is(err, ErrUnsupported) {
		t.Errorf("CreateMessage error = %v, want ErrUnsupported", err)
	}
	if _, err := server.Elicit(ctx, ElicitRequest{Message: "x"}); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Elicit error = %v, want ErrUnsupported", err)
	}
	if _, err := server.ListRoots(ctx); !errors.Is(err, ErrUnsupported) {
		t.Errorf("ListRoots error = %v, want ErrUnsupported", err)
	}
}

// TestElicitModeGatingUnsupported verifies that elicitation mode inference and
// capability gating reject mode/capability mismatches before any request is
// sent. Only mismatch cases are exercised here so the sentinel connection is
// never dereferenced; the accepted paths are covered by the streamable e2e
// tests.
func TestElicitModeGatingUnsupported(t *testing.T) {
	formOnly := ClientCapabilities{Elicitation: &ElicitationCapabilities{Form: &struct{}{}}}
	urlOnly := ClientCapabilities{Elicitation: &ElicitationCapabilities{URL: &struct{}{}}}

	tests := []struct {
		name    string
		caps    ClientCapabilities
		request ElicitRequest
	}{
		{name: "url inferred but unsupported", caps: formOnly, request: ElicitRequest{URL: "https://x"}},
		{name: "form requested but only url", caps: urlOnly, request: ElicitRequest{Message: "x", Mode: "form"}},
		{name: "unsupported mode", caps: formOnly, request: ElicitRequest{Message: "x", Mode: "bogus"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := NewServer("test-server", "1.0.0")
			server.mu.Lock()
			server.conn = &jsonrpc2.Connection{}
			server.clientCaps = tc.caps
			server.mu.Unlock()

			if _, err := server.Elicit(context.Background(), tc.request); err == nil {
				t.Fatalf("Elicit accepted %+v, want error", tc.request)
			}
		})
	}
}
