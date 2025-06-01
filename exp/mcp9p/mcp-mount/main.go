// mcp-mount - Plan9-style mounting for MCP namespaces
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// MountClient handles namespace mounting operations
type MountClient struct {
	nsServer string
	client   *http.Client
}

func NewMountClient(nsServer string) *MountClient {
	return &MountClient{
		nsServer: strings.TrimRight(nsServer, "/"),
		client:   &http.Client{},
	}
}

// Mount creates a mount point in the namespace
func (mc *MountClient) Mount(source, target string, options map[string]string) error {
	// First, check if source exists
	sourceInfo, err := mc.lookup(source)
	if err != nil {
		return fmt.Errorf("source not found: %w", err)
	}

	// Create mount entry
	entry := map[string]interface{}{
		"type": "mount",
		"metadata": map[string]string{
			"source":     source,
			"mounted_at": time.Now().Format(time.RFC3339),
		},
	}

	// Add options to metadata
	for k, v := range options {
		entry["metadata"].(map[string]string)[k] = v
	}

	// If source is a local service, we might need to start it
	if sourceInfo["type"] == "local" {
		if cmd, ok := sourceInfo["command"].(string); ok {
			// Start the local service if not already running
			entry["metadata"].(map[string]string)["command"] = cmd
			if args, ok := sourceInfo["args"].([]interface{}); ok {
				argsStr := make([]string, len(args))
				for i, arg := range args {
					argsStr[i] = arg.(string)
				}
				entry["metadata"].(map[string]string)["args"] = strings.Join(argsStr, ",")
			}
		}
	}

	// Register the mount
	return mc.register(target, entry)
}

// Bind creates a namespace binding (alias)
func (mc *MountClient) Bind(source, target string) error {
	entry := map[string]interface{}{
		"type": "bind",
		"metadata": map[string]string{
			"source": source,
		},
	}
	return mc.register(target, entry)
}

// Union creates a union mount (multiple sources)
func (mc *MountClient) Union(sources []string, target string) error {
	entry := map[string]interface{}{
		"type": "union",
		"metadata": map[string]string{
			"sources": strings.Join(sources, ","),
		},
	}
	return mc.register(target, entry)
}

// lookup performs a namespace lookup
func (mc *MountClient) lookup(path string) (map[string]interface{}, error) {
	resp, err := mc.client.Get(mc.nsServer + "/lookup?path=" + path)
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

// register creates an entry in the namespace
func (mc *MountClient) register(path string, entry map[string]interface{}) error {
	data, err := json.Marshal(map[string]interface{}{
		"path":  path,
		"entry": entry,
	})
	if err != nil {
		return err
	}

	resp, err := mc.client.Post(mc.nsServer+"/register", "application/json", strings.NewReader(string(data)))
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

// AutoMount automatically mounts local services and creates tunnels
func (mc *MountClient) AutoMount(path string, command string, args []string, transport string) error {
	// Register the local service
	entry := map[string]interface{}{
		"type":      "local",
		"transport": transport,
		"command":   command,
		"args":      args,
	}

	if err := mc.register(path, entry); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// If tunnel is requested, create it
	if os.Getenv("MCP_AUTO_TUNNEL") == "1" {
		tunnelCmd := exec.Command("mcp-tunnel",
			"-namespace", "ns://"+mc.nsServer+path+"-tunnel",
			"--", "ns://"+mc.nsServer+path)
		
		if err := tunnelCmd.Start(); err != nil {
			return fmt.Errorf("failed to start tunnel: %w", err)
		}

		// Give tunnel time to establish
		time.Sleep(2 * time.Second)
	}

	return nil
}

func main() {
	var (
		nsServer  = flag.String("ns", "localhost:9000", "Namespace server")
		mountType = flag.String("type", "mount", "Mount type: mount, bind, union, auto")
		transport = flag.String("transport", "stdio", "Transport for auto mounts")
		tunnel    = flag.Bool("tunnel", false, "Auto-create tunnel for local services")
	)
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "Usage: mcp-mount [options] <source> <target>\n")
		fmt.Fprintf(os.Stderr, "       mcp-mount -type auto <target> -- <command> [args...]\n")
		fmt.Fprintf(os.Stderr, "       mcp-mount -type union <target> <source1> <source2> ...\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	client := NewMountClient("http://" + *nsServer)

	switch *mountType {
	case "mount":
		source := flag.Arg(0)
		target := flag.Arg(1)
		
		options := make(map[string]string)
		if *tunnel {
			options["auto_tunnel"] = "true"
		}
		
		if err := client.Mount(source, target, options); err != nil {
			fmt.Fprintf(os.Stderr, "Mount failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Mounted %s at %s\n", source, target)

	case "bind":
		source := flag.Arg(0)
		target := flag.Arg(1)
		
		if err := client.Bind(source, target); err != nil {
			fmt.Fprintf(os.Stderr, "Bind failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Bound %s to %s\n", source, target)

	case "union":
		if flag.NArg() < 3 {
			fmt.Fprintf(os.Stderr, "Union mount requires at least 2 sources\n")
			os.Exit(1)
		}
		
		target := flag.Arg(0)
		sources := flag.Args()[1:]
		
		if err := client.Union(sources, target); err != nil {
			fmt.Fprintf(os.Stderr, "Union mount failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created union mount at %s from %v\n", target, sources)

	case "auto":
		target := flag.Arg(0)
		
		// Find the "--" separator
		var command string
		var args []string
		
		for i, arg := range os.Args {
			if arg == "--" && i+1 < len(os.Args) {
				command = os.Args[i+1]
				if i+2 < len(os.Args) {
					args = os.Args[i+2:]
				}
				break
			}
		}
		
		if command == "" {
			fmt.Fprintf(os.Stderr, "Auto mount requires command after --\n")
			os.Exit(1)
		}
		
		if *tunnel {
			os.Setenv("MCP_AUTO_TUNNEL", "1")
		}
		
		if err := client.AutoMount(target, command, args, *transport); err != nil {
			fmt.Fprintf(os.Stderr, "Auto mount failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Auto-mounted %s with command: %s %v\n", target, command, args)

	default:
		fmt.Fprintf(os.Stderr, "Unknown mount type: %s\n", *mountType)
		os.Exit(1)
	}
}