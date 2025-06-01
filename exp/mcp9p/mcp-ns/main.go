// mcp-ns - Command-line client for MCP namespace operations
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Client for namespace operations
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

// Register adds an entry to the namespace
func (c *NamespaceClient) Register(path string, entry interface{}) error {
	data, err := json.Marshal(map[string]interface{}{
		"path":  path,
		"entry": entry,
	})
	if err != nil {
		return err
	}

	resp, err := c.client.Post(c.serverURL+"/register", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed: %s", string(body))
	}

	return nil
}

// Lookup finds an entry by path
func (c *NamespaceClient) Lookup(path string) (map[string]interface{}, error) {
	resp, err := c.client.Get(c.serverURL + "/lookup?path=" + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("lookup failed: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// List returns entries in a namespace
func (c *NamespaceClient) List(path string) ([]map[string]interface{}, error) {
	resp, err := c.client.Get(c.serverURL + "/list?path=" + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list failed: %s", string(body))
	}

	var result []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func main() {
	var (
		server = flag.String("s", "http://localhost:9000", "Namespace server URL")
		cmd    = flag.String("c", "", "Command: register, lookup, list, bind")
	)
	flag.Parse()

	if *cmd == "" {
		fmt.Fprintf(os.Stderr, "Usage: mcp-ns -c <command> [options] <path> [args]\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  register <path> -type <type> -transport <transport> -address <address>\n")
		fmt.Fprintf(os.Stderr, "  register <path> -type local -command <cmd> -args <args...>\n")
		fmt.Fprintf(os.Stderr, "  lookup <path>\n")
		fmt.Fprintf(os.Stderr, "  list <path>\n")
		fmt.Fprintf(os.Stderr, "  bind <source> <target>\n")
		os.Exit(1)
	}

	client := NewNamespaceClient(*server)

	switch *cmd {
	case "register":
		if flag.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "Error: path required\n")
			os.Exit(1)
		}
		
		path := flag.Arg(0)
		
		// Parse additional flags for registration
		regCmd := flag.NewFlagSet("register", flag.ExitOnError)
		entryType := regCmd.String("type", "", "Entry type (local, remote, namespace)")
		transport := regCmd.String("transport", "", "Transport type (stdio, http, sse)")
		address := regCmd.String("address", "", "Service address")
		command := regCmd.String("command", "", "Command for local services")
		args := regCmd.String("args", "", "Arguments for command (comma-separated)")
		metadata := regCmd.String("metadata", "", "Metadata (key=value,key=value)")
		
		regCmd.Parse(flag.Args()[1:])
		
		entry := map[string]interface{}{
			"type": *entryType,
		}
		
		if *transport != "" {
			entry["transport"] = *transport
		}
		if *address != "" {
			entry["address"] = *address
		}
		if *command != "" {
			entry["command"] = *command
		}
		if *args != "" {
			entry["args"] = strings.Split(*args, ",")
		}
		if *metadata != "" {
			meta := make(map[string]string)
			for _, pair := range strings.Split(*metadata, ",") {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) == 2 {
					meta[parts[0]] = parts[1]
				}
			}
			entry["metadata"] = meta
		}
		
		if err := client.Register(path, entry); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Println("Registered successfully")
		
	case "lookup":
		if flag.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "Error: path required\n")
			os.Exit(1)
		}
		
		path := flag.Arg(0)
		entry, err := client.Lookup(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		
		data, _ := json.MarshalIndent(entry, "", "  ")
		fmt.Println(string(data))
		
	case "list":
		path := "/"
		if flag.NArg() > 0 {
			path = flag.Arg(0)
		}
		
		entries, err := client.List(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		
		for _, entry := range entries {
			name := entry["name"].(string)
			entryType := entry["type"].(string)
			
			if entryType == "namespace" {
				fmt.Printf("%s/\n", name)
			} else {
				fmt.Printf("%s (%s)\n", name, entryType)
			}
		}
		
	case "bind":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "Error: source and target paths required\n")
			os.Exit(1)
		}
		
		source := flag.Arg(0)
		target := flag.Arg(1)
		
		// This would call a mount endpoint on the server
		fmt.Printf("Binding %s to %s (not implemented yet)\n", source, target)
		
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *cmd)
		os.Exit(1)
	}
}