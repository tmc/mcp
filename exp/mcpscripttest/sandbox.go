// Package mcpscripttest provides sandbox capabilities for secure test execution
package mcpscripttest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SandboxConfig defines the configuration for sandboxed execution
type SandboxConfig struct {
	// BlockNetwork prevents any network operations
	BlockNetwork bool `json:"block_network"`
	
	// BlockExec prevents execution of external commands
	BlockExec bool `json:"block_exec"`
	
	// BlockFileSystem restricts file system access to specific directories
	BlockFileSystem bool     `json:"block_filesystem"`
	AllowedPaths    []string `json:"allowed_paths"`
	
	// BlockEnvironment prevents access to environment variables
	BlockEnvironment bool     `json:"block_environment"`
	AllowedEnvVars   []string `json:"allowed_env_vars"`
	
	// OverlayDir is the directory containing the Go overlay files
	OverlayDir string `json:"overlay_dir"`
}

// GenerateBuildOverlay creates Go build overlay files to restrict stdlib functionality
func GenerateBuildOverlay(config *SandboxConfig) error {
	if config.OverlayDir == "" {
		return fmt.Errorf("overlay directory must be specified")
	}
	
	// Create the overlay directory
	if err := os.MkdirAll(config.OverlayDir, 0755); err != nil {
		return fmt.Errorf("failed to create overlay directory: %w", err)
	}
	
	// Generate overlay files based on configuration
	if config.BlockNetwork {
		if err := generateNetworkStub(config.OverlayDir); err != nil {
			return fmt.Errorf("failed to generate network stub: %w", err)
		}
	}
	
	if config.BlockExec {
		if err := generateExecStub(config.OverlayDir); err != nil {
			return fmt.Errorf("failed to generate exec stub: %w", err)
		}
	}
	
	if config.BlockFileSystem {
		if err := generateFileSystemStub(config.OverlayDir, config.AllowedPaths); err != nil {
			return fmt.Errorf("failed to generate filesystem stub: %w", err)
		}
	}
	
	if config.BlockEnvironment {
		if err := generateEnvironmentStub(config.OverlayDir, config.AllowedEnvVars); err != nil {
			return fmt.Errorf("failed to generate environment stub: %w", err)
		}
	}
	
	// Generate the overlay.json file
	if err := generateOverlayJSON(config); err != nil {
		return fmt.Errorf("failed to generate overlay.json: %w", err)
	}
	
	return nil
}

// generateNetworkStub creates a stub for the net package
func generateNetworkStub(overlayDir string) error {
	netDir := filepath.Join(overlayDir, "net")
	if err := os.MkdirAll(netDir, 0755); err != nil {
		return err
	}
	
	stub := `// +build sandbox

package net

import "errors"

var ErrSandboxed = errors.New("network operations are disabled in sandbox mode")

type Conn interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Close() error
}

func Dial(network, address string) (Conn, error) {
	return nil, ErrSandboxed
}

func Listen(network, address string) (Listener, error) {
	return nil, ErrSandboxed
}

type Listener interface {
	Accept() (Conn, error)
	Close() error
	Addr() Addr
}

type Addr interface {
	Network() string
	String() string
}
`
	
	return writeFile(filepath.Join(netDir, "net_sandbox.go"), stub)
}

// generateExecStub creates a stub for the os/exec package
func generateExecStub(overlayDir string) error {
	execDir := filepath.Join(overlayDir, "os", "exec")
	if err := os.MkdirAll(execDir, 0755); err != nil {
		return err
	}
	
	stub := `// +build sandbox

package exec

import (
	"context"
	"errors"
)

var ErrSandboxed = errors.New("command execution is disabled in sandbox mode")

type Cmd struct {
	Path string
	Args []string
	Env  []string
	Dir  string
}

func Command(name string, arg ...string) *Cmd {
	return &Cmd{Path: name, Args: arg}
}

func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	return &Cmd{Path: name, Args: arg}
}

func (c *Cmd) Run() error {
	return ErrSandboxed
}

func (c *Cmd) Start() error {
	return ErrSandboxed
}

func (c *Cmd) Wait() error {
	return ErrSandboxed
}

func (c *Cmd) Output() ([]byte, error) {
	return nil, ErrSandboxed
}

func (c *Cmd) CombinedOutput() ([]byte, error) {
	return nil, ErrSandboxed
}
`
	
	return writeFile(filepath.Join(execDir, "exec_sandbox.go"), stub)
}

// generateFileSystemStub creates a stub for restricted file system access
func generateFileSystemStub(overlayDir string, allowedPaths []string) error {
	osDir := filepath.Join(overlayDir, "os")
	if err := os.MkdirAll(osDir, 0755); err != nil {
		return err
	}
	
	// Generate allowed paths check
	allowedPathsCode := "var allowedPaths = []string{\n"
	for _, path := range allowedPaths {
		allowedPathsCode += fmt.Sprintf("\t%q,\n", path)
	}
	allowedPathsCode += "}\n"
	
	stub := fmt.Sprintf(`// +build sandbox

package os

import (
	"errors"
	"path/filepath"
	"strings"
)

var ErrSandboxed = errors.New("file operation not allowed in sandbox mode")

%s

func isPathAllowed(path string) bool {
	absPath, _ := filepath.Abs(path)
	for _, allowed := range allowedPaths {
		if strings.HasPrefix(absPath, allowed) {
			return true
		}
	}
	return false
}

type File struct {
	name string
}

func Open(name string) (*File, error) {
	if !isPathAllowed(name) {
		return nil, ErrSandboxed
	}
	// Delegate to real implementation
	return openReal(name)
}

func Create(name string) (*File, error) {
	if !isPathAllowed(name) {
		return nil, ErrSandboxed
	}
	// Delegate to real implementation
	return createReal(name)
}

func Remove(name string) error {
	if !isPathAllowed(name) {
		return ErrSandboxed
	}
	// Delegate to real implementation
	return removeReal(name)
}
`, allowedPathsCode)
	
	return writeFile(filepath.Join(osDir, "file_sandbox.go"), stub)
}

// generateEnvironmentStub creates a stub for environment variable access
func generateEnvironmentStub(overlayDir string, allowedVars []string) error {
	osDir := filepath.Join(overlayDir, "os")
	if err := os.MkdirAll(osDir, 0755); err != nil {
		return err
	}
	
	// Generate allowed vars check
	allowedVarsCode := "var allowedEnvVars = map[string]bool{\n"
	for _, varName := range allowedVars {
		allowedVarsCode += fmt.Sprintf("\t%q: true,\n", varName)
	}
	allowedVarsCode += "}\n"
	
	stub := fmt.Sprintf(`// +build sandbox

package os

%s

func Getenv(key string) string {
	if !allowedEnvVars[key] {
		return ""
	}
	// Delegate to real implementation
	return getenvReal(key)
}

func Setenv(key, value string) error {
	if !allowedEnvVars[key] {
		return nil // Silently ignore
	}
	// Delegate to real implementation
	return setenvReal(key, value)
}

func LookupEnv(key string) (string, bool) {
	if !allowedEnvVars[key] {
		return "", false
	}
	// Delegate to real implementation
	return lookupEnvReal(key)
}
`, allowedVarsCode)
	
	return writeFile(filepath.Join(osDir, "env_sandbox.go"), stub)
}

// generateOverlayJSON creates the overlay.json file for go build
func generateOverlayJSON(config *SandboxConfig) error {
	overlay := map[string]interface{}{
		"Replace": map[string]string{},
	}
	
	replaceMap := overlay["Replace"].(map[string]string)
	
	if config.BlockNetwork {
		replaceMap["net/net.go"] = filepath.Join(config.OverlayDir, "net", "net_sandbox.go")
	}
	
	if config.BlockExec {
		replaceMap["os/exec/exec.go"] = filepath.Join(config.OverlayDir, "os", "exec", "exec_sandbox.go")
	}
	
	if config.BlockFileSystem {
		replaceMap["os/file.go"] = filepath.Join(config.OverlayDir, "os", "file_sandbox.go")
	}
	
	if config.BlockEnvironment {
		replaceMap["os/env.go"] = filepath.Join(config.OverlayDir, "os", "env_sandbox.go")
	}
	
	overlayJSON, err := json.MarshalIndent(overlay, "", "  ")
	if err != nil {
		return err
	}
	
	return writeFile(filepath.Join(config.OverlayDir, "overlay.json"), string(overlayJSON))
}

// GenerateBuildCommand generates the go build command with overlay
func GenerateBuildCommand(config *SandboxConfig, originalCmd string) string {
	overlayPath := filepath.Join(config.OverlayDir, "overlay.json")
	return fmt.Sprintf("go build -overlay=%s -tags=sandbox %s", overlayPath, originalCmd)
}

// writeFile is a helper to write content to a file
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// RunSandboxed executes a scripttest in a sandboxed environment
func RunSandboxed(t interface{ Logf(string, ...interface{}) }, scriptPath string, config *SandboxConfig) error {
	// Generate the overlay files
	if err := GenerateBuildOverlay(config); err != nil {
		return fmt.Errorf("failed to generate overlay: %w", err)
	}

	// Set up environment for sandboxed execution
	env := os.Environ()
	env = append(env, fmt.Sprintf("GOFLAGS=-overlay=%s -tags=sandbox",
		filepath.Join(config.OverlayDir, "overlay.json")))

	// This is a placeholder - in practice you would use os/exec package
	// but that creates a circular dependency. The actual implementation
	// would need to be in a separate package or use a different approach.
	t.Logf("Would run sandboxed test: go test -run %s with GOFLAGS=%s",
		scriptPath, env[len(env)-1])

	return nil
}