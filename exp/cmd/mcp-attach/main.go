// Command mcp-attach "attaches" a shell session to a specific mcpd instance.
//
// It does this by starting a new shell with the MCP_SOCKET_PATH environment
// variable set to the target mcpd's socket, allowing MCP tools to communicate
// with that specific mcpd instance.
//
// Usage:
//
//	mcp-attach [flags] <service-name|pid|socket-path>
//
// For example:
//
//	mcp-attach myweather      # Looks for service named "myweather" via ~/.srv/mcp/myweather
//	mcp-attach 12345          # Attaches to mcpd with PID 12345 via ~/.mcpd/sock.12345
//	mcp-attach /path/to/sock  # Uses the specified socket path directly
//
// Flags:
//
//	-shell string     Shell to execute (default: $SHELL or /bin/sh)
//	-socket-env string Environment variable to set (default: MCP_SOCKET_PATH)
//	-root-dir string   Root directory for service/socket discovery (default: ~/.mcpd or $XDG_RUNTIME_DIR/mcpd)
//	-v                 Verbose mode
//
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

var (
	shellFlag     = flag.String("shell", "", "Shell to execute (default: $SHELL or /bin/sh)")
	socketEnvFlag = flag.String("socket-env", "MCP_SOCKET_PATH", "Environment variable to set")
	rootDirFlag   = flag.String("root-dir", "", "Root directory for service/socket discovery")
	srvDirFlag    = flag.String("srv-dir", "", "Root directory for service discovery")
	verboseFlag   = flag.Bool("v", false, "Verbose mode")
)

// SocketInfo stores the discovered socket path and metadata
type SocketInfo struct {
	Path        string // Full path to socket
	ServiceName string // Name of the service (if available)
	PID         int    // PID of the mcpd process (if available)
}

func main() {
	log.SetFlags(0)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <service-name|pid|socket-path>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s myweather      # Looks for service named \"myweather\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s 12345          # Attaches to mcpd with PID 12345\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s /path/to/sock  # Uses the specified socket path directly\n", os.Args[0])
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	target := flag.Arg(0)

	socketInfo, err := resolveSocket(target)
	if err != nil {
		log.Fatalf("Error resolving socket: %v", err)
	}

	if *verboseFlag {
		log.Printf("Using socket: %s", socketInfo.Path)
		if socketInfo.ServiceName != "" {
			log.Printf("Service: %s", socketInfo.ServiceName)
		}
		if socketInfo.PID > 0 {
			log.Printf("mcpd PID: %d", socketInfo.PID)
		}
	}

	// Get shell to execute
	shell := *shellFlag
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
	}

	// Copy current environment and add MCP_SOCKET_PATH
	env := os.Environ()
	env = append(env, fmt.Sprintf("%s=%s", *socketEnvFlag, socketInfo.Path))

	// Create shell command
	cmd := exec.Command(shell)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if *verboseFlag {
		log.Printf("Starting shell: %s", shell)
		log.Printf("Setting %s=%s", *socketEnvFlag, socketInfo.Path)
	}

	// Execute the shell
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Error starting shell: %v", err)
	}

	// Wait for shell to exit
	err = cmd.Wait()

	// Pass on the exit status
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			os.Exit(status.ExitStatus())
		}
	} else if err != nil {
		log.Fatalf("Error waiting for shell: %v", err)
	}
}

// resolveSocket determines the socket path from the target
func resolveSocket(target string) (*SocketInfo, error) {
	// Case 1: Direct socket path
	if strings.HasPrefix(target, "/") || strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../") {
		// Check if file exists and is a socket
		fi, err := os.Stat(target)
		if err != nil {
			return nil, fmt.Errorf("socket path error: %w", err)
		}
		mode := fi.Mode()
		if mode&os.ModeSocket == 0 {
			return nil, fmt.Errorf("not a socket: %s", target)
		}
		return &SocketInfo{Path: target}, nil
	}

	// Case 2: PID number
	if isPID(target) {
		pid, _ := strconv.Atoi(target)
		sockPath, err := findSocketByPID(pid)
		if err != nil {
			return nil, err
		}
		return &SocketInfo{
			Path: sockPath,
			PID:  pid,
		}, nil
	}

	// Case 3: Service name
	sockPath, err := findSocketByServiceName(target)
	if err != nil {
		return nil, err
	}
	return &SocketInfo{
		Path:        sockPath,
		ServiceName: target,
	}, nil
}

// isPID checks if the string is a valid PID
func isPID(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// findSocketByPID looks for a socket associated with the given PID
func findSocketByPID(pid int) (string, error) {
	// Determine socket root directory
	rootDir := getMCPDRootDir()
	
	// Try different possible socket patterns
	patterns := []string{
		filepath.Join(rootDir, fmt.Sprintf("sock.%d", pid)),
		filepath.Join(rootDir, fmt.Sprintf("%d.sock", pid)),
		filepath.Join(rootDir, fmt.Sprintf("mcpd.%d.sock", pid)),
	}

	for _, pattern := range patterns {
		if *verboseFlag {
			log.Printf("Checking for socket at: %s", pattern)
		}
		if fileExists(pattern) && isSocket(pattern) {
			return pattern, nil
		}
	}

	return "", fmt.Errorf("no socket found for PID %d in %s", pid, rootDir)
}

// findSocketByServiceName looks for a socket associated with the given service name
func findSocketByServiceName(name string) (string, error) {
	// Check in ~/.srv/mcp directory first
	srvDir := getSrvRootDir()
	srvPath := filepath.Join(srvDir, name)
	
	if *verboseFlag {
		log.Printf("Checking for service at: %s", srvPath)
	}
	
	if fileExists(srvPath) {
		// Read the socket path from the service file
		content, err := os.ReadFile(srvPath)
		if err != nil {
			return "", fmt.Errorf("error reading service file %s: %w", srvPath, err)
		}
		sockPath := strings.TrimSpace(string(content))
		if isSocket(sockPath) {
			return sockPath, nil
		}
		return "", fmt.Errorf("service file %s contains invalid socket path: %s", srvPath, sockPath)
	}

	// If not in service dir, check if service file in ~/.mcpd
	mcpdRoot := getMCPDRootDir()
	sockPath := filepath.Join(mcpdRoot, fmt.Sprintf("svc.%s.sock", name))
	
	if *verboseFlag {
		log.Printf("Checking for service socket at: %s", sockPath)
	}
	
	if fileExists(sockPath) && isSocket(sockPath) {
		return sockPath, nil
	}

	// Check for 'current' symlink
	if name == "current" {
		currentPath := filepath.Join(mcpdRoot, "current")
		if *verboseFlag {
			log.Printf("Checking for 'current' symlink at: %s", currentPath)
		}
		if isSymlink(currentPath) {
			// Read the symlink
			target, err := os.Readlink(currentPath)
			if err != nil {
				return "", fmt.Errorf("error reading 'current' symlink: %w", err)
			}
			
			// If target is relative, make it absolute
			if !filepath.IsAbs(target) {
				target = filepath.Join(mcpdRoot, target)
			}
			
			if isSocket(target) {
				return target, nil
			}
			return "", fmt.Errorf("'current' symlink points to non-socket: %s", target)
		}
	}

	return "", fmt.Errorf("no socket found for service '%s'", name)
}

// getMCPDRootDir returns the root directory for mcpd sockets
func getMCPDRootDir() string {
	if *rootDirFlag != "" {
		return *rootDirFlag
	}
	
	// Use XDG_RUNTIME_DIR if available
	if xdgDir := os.Getenv("XDG_RUNTIME_DIR"); xdgDir != "" {
		return filepath.Join(xdgDir, "mcpd")
	}
	
	// Fall back to ~/.mcpd
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: couldn't determine home directory: %v", err)
		return "/tmp/mcpd" // Last resort
	}
	return filepath.Join(home, ".mcpd")
}

// getSrvRootDir returns the root directory for service files
func getSrvRootDir() string {
	if *srvDirFlag != "" {
		return *srvDirFlag
	}
	
	// Fall back to ~/.srv/mcp
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: couldn't determine home directory: %v", err)
		return "/tmp/srv/mcp" // Last resort
	}
	return filepath.Join(home, ".srv", "mcp")
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isSocket checks if a path is a Unix domain socket
func isSocket(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSocket != 0
}

// isSymlink checks if a path is a symlink
func isSymlink(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0
}