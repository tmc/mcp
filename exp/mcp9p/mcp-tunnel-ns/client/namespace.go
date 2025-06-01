package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// NamespaceClient handles namespace lookups
type NamespaceClient struct {
	serverURL string
	client    *http.Client
}

func NewNamespaceClient(serverURL string) *NamespaceClient {
	return &NamespaceClient{
		serverURL: strings.TrimRight(serverURL, "/"),
		client:    &http.Client{},
	}
}

// ResolveService looks up a service in the namespace
func (nc *NamespaceClient) ResolveService(path string) (*ServiceInfo, error) {
	// Try to lookup the path
	resp, err := nc.client.Get(nc.serverURL + "/lookup?path=" + path)
	if err != nil {
		return nil, fmt.Errorf("namespace lookup failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service not found in namespace: %s", path)
	}

	var entry map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to ServiceInfo
	info := &ServiceInfo{
		Path: path,
		Type: entry["type"].(string),
	}

	if transport, ok := entry["transport"].(string); ok {
		info.Transport = transport
	}
	if address, ok := entry["address"].(string); ok {
		info.Address = address
	}
	if command, ok := entry["command"].(string); ok {
		info.Command = command
	}
	if args, ok := entry["args"].([]interface{}); ok {
		info.Args = make([]string, len(args))
		for i, arg := range args {
			info.Args[i] = arg.(string)
		}
	}

	return info, nil
}

// RegisterService registers a service in the namespace
func (nc *NamespaceClient) RegisterService(path string, info *ServiceInfo) error {
	entry := map[string]interface{}{
		"type":      info.Type,
		"transport": info.Transport,
	}

	if info.Address != "" {
		entry["address"] = info.Address
	}
	if info.Command != "" {
		entry["command"] = info.Command
	}
	if len(info.Args) > 0 {
		entry["args"] = info.Args
	}
	if info.TunnelURL != "" {
		if entry["metadata"] == nil {
			entry["metadata"] = make(map[string]string)
		}
		entry["metadata"].(map[string]string)["tunnel_url"] = info.TunnelURL
	}

	data, err := json.Marshal(map[string]interface{}{
		"path":  path,
		"entry": entry,
	})
	if err != nil {
		return err
	}

	resp, err := nc.client.Post(nc.serverURL+"/register", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed")
	}

	return nil
}

// ServiceInfo contains resolved service information
type ServiceInfo struct {
	Path      string
	Type      string   // "local", "remote", "tunneled"
	Transport string   // "stdio", "http", "sse"
	Address   string   // For remote services
	Command   string   // For local services
	Args      []string // For local services
	TunnelURL string   // For tunneled services
}

// ParseNamespaceURI parses URIs like "ns://namespace.local/services/echo"
func ParseNamespaceURI(uri string) (server, path string, err error) {
	if !strings.HasPrefix(uri, "ns://") {
		return "", "", fmt.Errorf("invalid namespace URI: %s", uri)
	}

	uri = strings.TrimPrefix(uri, "ns://")
	parts := strings.SplitN(uri, "/", 2)
	
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid namespace URI: missing path")
	}

	return "http://" + parts[0], "/" + parts[1], nil
}