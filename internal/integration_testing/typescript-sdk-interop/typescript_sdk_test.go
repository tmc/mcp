package typescriptinterop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestTypeScriptSDKStdioSmoke(t *testing.T) {
	moduleDir := currentDir(t)
	node := findNode(t)
	npm := findNPM(t, node)

	tmp := t.TempDir()
	serverPath := filepath.Join(tmp, "b8-stdio-smoke-server")
	if runtime.GOOS == "windows" {
		serverPath += ".exe"
	}

	run(t, moduleDir, nil, "go", "build", "-o", serverPath, "./testserver")

	clientDir := filepath.Join(tmp, "ts-client")
	if err := os.Mkdir(clientDir, 0o755); err != nil {
		t.Fatalf("mkdir ts client dir: %v", err)
	}
	copyFile(t, filepath.Join(moduleDir, "testdata", "ts-client", "package.json"), filepath.Join(clientDir, "package.json"))
	copyFile(t, filepath.Join(moduleDir, "testdata", "ts-client", "package-lock.json"), filepath.Join(clientDir, "package-lock.json"))
	copyFile(t, filepath.Join(moduleDir, "testdata", "ts-client", "smoke.mjs"), filepath.Join(clientDir, "smoke.mjs"))

	env := pathEnvFor(node)
	run(t, clientDir, env, npm, "ci", "--ignore-scripts", "--no-audit", "--fund=false")

	out := run(t, clientDir, env, node, "smoke.mjs", serverPath)
	t.Log(strings.TrimSpace(string(out)))
}

func currentDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	return filepath.Dir(file)
}

func findNode(t *testing.T) string {
	t.Helper()
	return findExecutable(t, "node", []string{
		os.Getenv("MCP_NODE"),
		filepath.Join(os.Getenv("HOME"), ".nvm", "versions", "node", "v24.14.1", "bin", "node"),
	})
}

func findNPM(t *testing.T, node string) string {
	t.Helper()
	return findExecutable(t, "npm", []string{
		os.Getenv("MCP_NPM"),
		filepath.Join(filepath.Dir(node), "npm"),
	})
}

func findExecutable(t *testing.T, name string, candidates []string) string {
	t.Helper()
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if executableWorks(candidate, "--version") {
			return candidate
		}
	}
	path, err := exec.LookPath(name)
	if err == nil && executableWorks(path, "--version") {
		return path
	}
	t.Skipf("%s not available; set MCP_%s to run TypeScript SDK interop smoke", name, strings.ToUpper(name))
	return ""
}

func executableWorks(name string, args ...string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Run() == nil
}

func pathEnvFor(node string) []string {
	nodeDir := filepath.Dir(node)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "PATH=") {
			return []string{"PATH=" + nodeDir + string(os.PathListSeparator) + strings.TrimPrefix(env, "PATH=")}
		}
	}
	return []string{"PATH=" + nodeDir}
}

func run(t *testing.T, dir string, extraEnv []string, name string, args ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s timed out: %v\n%s", commandLine(name, args), ctx.Err(), out)
	}
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", commandLine(name, args), err, out)
	}
	return out
}

func commandLine(name string, args []string) string {
	return strings.Join(append([]string{name}, args...), " ")
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}
