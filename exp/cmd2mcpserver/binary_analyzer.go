package cmd2mcpserver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BinaryAnalyzer analyzes Go binaries to extract information
type BinaryAnalyzer struct {
	binaryPath string
	ModuleInfo *ModuleInfo
}

// NewBinaryAnalyzer creates a new binary analyzer
func NewBinaryAnalyzer(binaryPath string) *BinaryAnalyzer {
	return &BinaryAnalyzer{
		binaryPath: binaryPath,
	}
}

// GetModuleInfo extracts module information from a Go binary
func (ba *BinaryAnalyzer) GetModuleInfo() (*ModuleInfo, error) {
	cmd := exec.Command("go", "version", "-v", "-m", ba.binaryPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get module info: %w", err)
	}

	info := &ModuleInfo{}
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Parse Go version
		if strings.HasPrefix(line, ba.binaryPath+":") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				info.GoVersion = parts[1]
			}
		}
		
		// Parse module path
		if strings.HasPrefix(line, "path\t") {
			info.ModulePath = strings.TrimPrefix(line, "path\t")
		}
		
		// Parse module version
		if strings.HasPrefix(line, "mod\t") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				info.ModuleName = parts[1]
				if len(parts) >= 3 {
					info.ModuleVersion = parts[2]
				}
			}
		}
		
		// Parse build settings
		if strings.HasPrefix(line, "build\t") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimPrefix(parts[0], "build\t")
				value := parts[1]
				
				switch key {
				case "GOOS":
					info.GOOS = value
				case "GOARCH":
					info.GOARCH = value
				}
			}
		}
	}
	
	return info, nil
}

// GetPackageDoc retrieves package documentation using go doc
func (ba *BinaryAnalyzer) GetPackageDoc(importPath string) (string, error) {
	cmd := exec.Command("go", "doc", "-all", importPath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get package doc: %w", err)
	}
	
	return string(output), nil
}

// GetSourcePath attempts to find the source path for a binary's module
func (ba *BinaryAnalyzer) GetSourcePath() (string, error) {
	info, err := ba.GetModuleInfo()
	if err != nil {
		return "", err
	}

	if info.ModulePath == "" {
		return "", fmt.Errorf("no module path found")
	}

	// Try multiple strategies to find source

	// 1. Try to find source in current workspace
	cmd := exec.Command("go", "list", "-f", "{{.Dir}}", info.ModulePath)
	output, err := cmd.Output()
	if err == nil {
		sourceDir := strings.TrimSpace(string(output))
		if sourceDir != "" {
			return sourceDir, nil
		}
	}

	// 2. Try to find in module cache using module@version
	if info.ModuleName != "" && info.ModuleVersion != "" {
		moduleSpec := fmt.Sprintf("%s@%s", info.ModuleName, info.ModuleVersion)
		cmd = exec.Command("go", "list", "-m", "-f", "{{.Dir}}", moduleSpec)
		output, err = cmd.Output()
		if err == nil {
			cacheDir := strings.TrimSpace(string(output))
			if cacheDir != "" {
				// The cache dir will be something like:
				// /Users/username/go/pkg/mod/github.com/user/module@v1.2.3
				// We need to find the actual module path within it
				return ba.findModulePathInCache(cacheDir, info.ModulePath)
			}
		}
	}

	// 3. Try to find in GOMODCACHE by constructing the path
	modCacheDir := ba.getModCacheDir()
	if modCacheDir != "" && info.ModuleName != "" && info.ModuleVersion != "" {
		// Construct the expected cache path
		cachePath := filepath.Join(modCacheDir, info.ModuleName+"@"+info.ModuleVersion)
		if stat, err := os.Stat(cachePath); err == nil && stat.IsDir() {
			return ba.findModulePathInCache(cachePath, info.ModulePath)
		}
	}

	return "", fmt.Errorf("could not find source directory for module")
}

// getModCacheDir returns the Go module cache directory
func (ba *BinaryAnalyzer) getModCacheDir() string {
	// Try GOMODCACHE env var first
	if modCache := os.Getenv("GOMODCACHE"); modCache != "" {
		return modCache
	}

	// Get from go env
	cmd := exec.Command("go", "env", "GOMODCACHE")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	// Fallback to default
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		if home := os.Getenv("HOME"); home != "" {
			gopath = filepath.Join(home, "go")
		}
	}

	if gopath != "" {
		return filepath.Join(gopath, "pkg", "mod")
	}

	return ""
}

// findModulePathInCache finds the actual module path within a cache directory
func (ba *BinaryAnalyzer) findModulePathInCache(cacheDir, modulePath string) (string, error) {
	// The module path might be nested within the cache directory
	// For example: github.com/user/project/cmd/tool
	// In cache: /path/to/cache/github.com/user/project@v1.0.0/cmd/tool

	// First try the cache directory itself
	if _, err := os.Stat(filepath.Join(cacheDir, "go.mod")); err == nil {
		return cacheDir, nil
	}

	// Try to find the specific subpath within the module
	info, err := ba.GetModuleInfo()
	if err != nil {
		return cacheDir, nil
	}

	if strings.HasPrefix(modulePath, info.ModuleName) {
		subPath := strings.TrimPrefix(modulePath, info.ModuleName)
		subPath = strings.TrimPrefix(subPath, "/")

		if subPath != "" {
			fullPath := filepath.Join(cacheDir, subPath)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath, nil
			}
		}
	}

	return cacheDir, nil
}

// ExtractBinaryInfo combines all analysis methods to get comprehensive binary info
func (ba *BinaryAnalyzer) ExtractBinaryInfo() (*BinaryInfo, error) {
	moduleInfo, err := ba.GetModuleInfo()
	if err != nil {
		return nil, err
	}
	
	binaryInfo := &BinaryInfo{
		Path:       ba.binaryPath,
		Name:       filepath.Base(ba.binaryPath),
		ModuleInfo: moduleInfo,
	}
	
	// Try to get source path
	if sourcePath, err := ba.GetSourcePath(); err == nil {
		binaryInfo.SourcePath = sourcePath
	}
	
	// Try to get package documentation
	if moduleInfo.ModulePath != "" {
		if doc, err := ba.GetPackageDoc(moduleInfo.ModulePath); err == nil {
			binaryInfo.PackageDoc = doc
			// Extract description from first line of doc
			if lines := strings.Split(doc, "\n"); len(lines) > 0 {
				binaryInfo.Description = strings.TrimSpace(lines[0])
			}
		}
	}
	
	return binaryInfo, nil
}

// ModuleInfo represents Go module information
type ModuleInfo struct {
	ModulePath    string
	ModuleName    string
	ModuleVersion string
	GoVersion     string
	GOOS          string
	GOARCH        string
}

// BinaryInfo represents comprehensive binary information
type BinaryInfo struct {
	Path        string
	Name        string
	Description string
	SourcePath  string
	PackageDoc  string
	ModuleInfo  *ModuleInfo
}

// ExtractFlagsFromDoc attempts to extract flag definitions from go doc output
func ExtractFlagsFromDoc(doc string) []FlagDef {
	var flags []FlagDef
	
	// Look for flag definitions in documentation
	// This is a simple heuristic - could be improved
	lines := strings.Split(doc, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Look for patterns like: -name string
		if strings.HasPrefix(line, "-") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				flagName := strings.TrimPrefix(parts[0], "-")
				flagType := parts[1]
				
				// Convert type to our format
				var ourType string
				switch flagType {
				case "string":
					ourType = "string"
				case "int", "int64":
					ourType = "integer"
				case "bool":
					ourType = "boolean"
				case "float64":
					ourType = "number"
				default:
					ourType = "string"
				}
				
				flag := FlagDef{
					Name: flagName,
					Type: ourType,
				}
				
				// Try to extract description
				if len(parts) > 2 {
					flag.Description = strings.Join(parts[2:], " ")
				}
				
				flags = append(flags, flag)
			}
		}
	}
	
	return flags
}