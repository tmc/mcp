// Example of using the MCP namespace system programmatically
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// NamespaceClient provides methods to interact with the namespace server
type NamespaceClient struct {
	serverURL string
	client    *http.Client
}

// Entry represents a namespace entry
type Entry struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Transport string            `json:"transport,omitempty"`
	Address   string            `json:"address,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

func NewNamespaceClient(serverURL string) *NamespaceClient {
	return &NamespaceClient{
		serverURL: serverURL,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (nc *NamespaceClient) Register(path string, entry *Entry) error {
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
		return fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	return nil
}

func (nc *NamespaceClient) Lookup(path string) (*Entry, error) {
	resp, err := nc.client.Get(nc.serverURL + "/lookup?path=" + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lookup failed with status %d", resp.StatusCode)
	}

	var entry Entry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

func (nc *NamespaceClient) List(path string) ([]*Entry, error) {
	resp, err := nc.client.Get(nc.serverURL + "/list?path=" + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list failed with status %d", resp.StatusCode)
	}

	var entries []*Entry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func main() {
	// Create namespace client
	client := NewNamespaceClient("http://localhost:9000")

	// Register a local service
	echoService := &Entry{
		Type:      "local",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"@modelcontextprotocol/server-echo", "stdio"},
		Metadata: map[string]string{
			"description": "Echo service for testing",
			"version":     "1.0.0",
		},
	}

	if err := client.Register("/services/echo", echoService); err != nil {
		log.Printf("Failed to register echo service: %v", err)
	} else {
		log.Println("Registered echo service")
	}

	// Register a remote service
	apiService := &Entry{
		Type:      "remote",
		Transport: "http",
		Address:   "https://api.example.com/mcp",
		Metadata: map[string]string{
			"auth":     "required",
			"protocol": "mcp-1.0",
		},
	}

	if err := client.Register("/remote/api", apiService); err != nil {
		log.Printf("Failed to register API service: %v", err)
	} else {
		log.Println("Registered API service")
	}

	// Create a namespace
	aiNamespace := &Entry{
		Type: "namespace",
		Metadata: map[string]string{
			"description": "AI and ML services",
		},
	}

	if err := client.Register("/services/ai", aiNamespace); err != nil {
		log.Printf("Failed to create AI namespace: %v", err)
	} else {
		log.Println("Created AI namespace")
	}

	// Register a service in the namespace
	gptService := &Entry{
		Type:      "remote",
		Transport: "http",
		Address:   "https://gpt.api/mcp",
		Metadata: map[string]string{
			"model":      "gpt-4",
			"rate_limit": "100/min",
		},
	}

	if err := client.Register("/services/ai/gpt", gptService); err != nil {
		log.Printf("Failed to register GPT service: %v", err)
	} else {
		log.Println("Registered GPT service")
	}

	// List all services
	fmt.Println("\nListing all services:")
	entries, err := client.List("/services")
	if err != nil {
		log.Printf("Failed to list services: %v", err)
	} else {
		for _, entry := range entries {
			fmt.Printf("- %s (%s)\n", entry.Name, entry.Type)
		}
	}

	// Lookup a specific service
	fmt.Println("\nLooking up echo service:")
	entry, err := client.Lookup("/services/echo")
	if err != nil {
		log.Printf("Failed to lookup echo service: %v", err)
	} else {
		fmt.Printf("Found: %s\n", entry.Name)
		fmt.Printf("Type: %s\n", entry.Type)
		fmt.Printf("Transport: %s\n", entry.Transport)
		fmt.Printf("Command: %s %v\n", entry.Command, entry.Args)
		fmt.Printf("Metadata: %v\n", entry.Metadata)
	}

	// Demonstrate namespace URI parsing
	fmt.Println("\nParsing namespace URI:")
	uri := "ns://localhost:9000/services/echo"
	if server, path, err := parseNamespaceURI(uri); err == nil {
		fmt.Printf("URI: %s\n", uri)
		fmt.Printf("Server: %s\n", server)
		fmt.Printf("Path: %s\n", path)
	}
}

// parseNamespaceURI parses a namespace URI
func parseNamespaceURI(uri string) (server, path string, err error) {
	if len(uri) < 5 || uri[:5] != "ns://" {
		return "", "", fmt.Errorf("invalid namespace URI")
	}

	uri = uri[5:]
	for i, ch := range uri {
		if ch == '/' {
			return "http://" + uri[:i], uri[i:], nil
		}
	}

	return "", "", fmt.Errorf("invalid namespace URI: no path")
}
