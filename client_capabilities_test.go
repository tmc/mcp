// Copyright 2025 The MCP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.

package mcp

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/exp/jsonrpc2"
)

// captureServer records the params of each request it receives.
type captureServer struct {
	mu     sync.Mutex
	params map[string]json.RawMessage
}

func (s *captureServer) handle(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	s.mu.Lock()
	if s.params == nil {
		s.params = make(map[string]json.RawMessage)
	}
	s.params[req.Method] = req.Params
	s.mu.Unlock()
	switch req.Method {
	case "initialize":
		return InitializeResult{ProtocolVersion: LATEST_PROTOCOL_VERSION}, nil
	default:
		return struct{}{}, nil
	}
}

func newCaptureClient(t *testing.T) (*Client, *captureServer) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	srv := &captureServer{}
	ready := make(chan struct{})
	go func() {
		ready <- struct{}{}
		conn, err := jsonrpc2.Dial(context.Background(), &ReadWriteCloserTransport{serverConn}, jsonrpc2.ConnectionOptions{
			Framer:  jsonrpc2.RawFramer(),
			Handler: jsonrpc2.HandlerFunc(srv.handle),
		})
		if err != nil {
			return
		}
		conn.Wait()
	}()
	<-ready
	time.Sleep(10 * time.Millisecond)
	client, err := NewClient(&ReadWriteCloserTransport{clientConn})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client, srv
}

// TestClientAdvertisesCapabilitiesFromHandlers verifies that registering typed
// server-request handlers causes the matching capability to be advertised at
// initialize time.
func TestClientAdvertisesCapabilitiesFromHandlers(t *testing.T) {
	client, srv := newCaptureClient(t)

	client.OnSampling(func(context.Context, CreateMessageRequest) (*CreateMessageResult, error) {
		return &CreateMessageResult{}, nil
	})
	client.OnElicit(func(context.Context, ElicitRequest) (*ElicitResult, error) {
		return &ElicitResult{Action: "accept"}, nil
	}, ElicitModeForm, ElicitModeURL)
	client.OnListRoots(func(context.Context) (*ListRootsResult, error) {
		return &ListRootsResult{}, nil
	})

	if _, err := client.Initialize(context.Background(), InitializeRequest{}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	srv.mu.Lock()
	raw := srv.params["initialize"]
	srv.mu.Unlock()

	var req InitializeRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("decode initialize params: %v", err)
	}
	if req.Capabilities.Sampling == nil {
		t.Error("sampling capability not advertised after OnSampling")
	}
	if req.Capabilities.Roots == nil {
		t.Error("roots capability not advertised after OnListRoots")
	}
	if req.Capabilities.Elicitation == nil {
		t.Fatal("elicitation capability not advertised after OnElicit")
	}
	if req.Capabilities.Elicitation.Form == nil {
		t.Error("elicitation form mode not advertised")
	}
	if req.Capabilities.Elicitation.URL == nil {
		t.Error("elicitation url mode not advertised")
	}
}

// TestClientCapabilitiesNotClobbered verifies that caller-supplied capabilities
// take precedence over handler-derived advertisement.
func TestClientCapabilitiesNotClobbered(t *testing.T) {
	client, srv := newCaptureClient(t)

	// Register a sampling handler but explicitly suppress the capability by
	// supplying capabilities that omit sampling.
	client.OnSampling(func(context.Context, CreateMessageRequest) (*CreateMessageResult, error) {
		return &CreateMessageResult{}, nil
	})

	caller := InitializeRequest{
		Capabilities: ClientCapabilities{
			Experimental: map[string]any{"x": true},
		},
	}
	if _, err := client.Initialize(context.Background(), caller); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	srv.mu.Lock()
	raw := srv.params["initialize"]
	srv.mu.Unlock()

	var req InitializeRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("decode initialize params: %v", err)
	}
	// Sampling was nil in the caller-supplied caps, so the handler-derived
	// advertisement still fills it (per-field merge), which is the documented
	// behavior. Experimental must be preserved.
	if req.Capabilities.Experimental["x"] != true {
		t.Error("caller-supplied experimental capability was lost")
	}
	if req.Capabilities.Sampling == nil {
		t.Error("sampling capability should be merged when caller left it nil")
	}
}

// TestClientNoHandlersNoCapabilities verifies that a client with no typed
// handlers advertises no server-request capabilities.
func TestClientNoHandlersNoCapabilities(t *testing.T) {
	client, srv := newCaptureClient(t)
	if _, err := client.Initialize(context.Background(), InitializeRequest{}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	srv.mu.Lock()
	raw := srv.params["initialize"]
	srv.mu.Unlock()
	var req InitializeRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("decode initialize params: %v", err)
	}
	if req.Capabilities.Sampling != nil || req.Capabilities.Elicitation != nil || req.Capabilities.Roots != nil {
		t.Errorf("unexpected capabilities advertised without handlers: %+v", req.Capabilities)
	}
}
