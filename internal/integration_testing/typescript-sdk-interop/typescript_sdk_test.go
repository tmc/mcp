package typescriptinterop

import (
	"bytes"
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

	serverPath := buildTestServer(t, moduleDir)
	clientDir, env := prepareTSClient(t, moduleDir, node, npm, "smoke.mjs")

	out := run(t, clientDir, env, node, "smoke.mjs", serverPath)
	t.Log(strings.TrimSpace(string(out)))
}

func TestTypeScriptSDKStreamableHTTPSmoke(t *testing.T) {
	moduleDir := currentDir(t)
	node := findNode(t)
	npm := findNPM(t, node)

	serverPath := buildTestServer(t, moduleDir)
	endpoint := startStreamableHTTPServer(t, serverPath)
	clientDir, env := prepareTSClient(t, moduleDir, node, npm, "streamable_http_smoke.mjs")

	out := run(t, clientDir, env, node, "streamable_http_smoke.mjs", endpoint)
	t.Log(strings.TrimSpace(string(out)))
}

func buildTestServer(t *testing.T, moduleDir string) string {
	t.Helper()
	serverPath := filepath.Join(t.TempDir(), "b8-smoke-server")
	if runtime.GOOS == "windows" {
		serverPath += ".exe"
	}
	run(t, moduleDir, nil, "go", "build", "-o", serverPath, "./testserver")
	return serverPath
}

func prepareTSClient(t *testing.T, moduleDir, node, npm, fixture string) (string, []string) {
	t.Helper()
	clientDir := filepath.Join(t.TempDir(), "ts-client")
	if err := os.Mkdir(clientDir, 0o755); err != nil {
		t.Fatalf("mkdir ts client dir: %v", err)
	}
	copyFile(t, filepath.Join(moduleDir, "testdata", "ts-client", "package.json"), filepath.Join(clientDir, "package.json"))
	copyFile(t, filepath.Join(moduleDir, "testdata", "ts-client", "package-lock.json"), filepath.Join(clientDir, "package-lock.json"))
	copyFile(t, filepath.Join(moduleDir, "testdata", "ts-client", fixture), filepath.Join(clientDir, fixture))

	env := pathEnvFor(node)
	run(t, clientDir, env, npm, "ci", "--ignore-scripts", "--no-audit", "--fund=false")
	return clientDir, env
}

func startStreamableHTTPServer(t *testing.T, serverPath string) string {
	t.Helper()
	urlFile := filepath.Join(t.TempDir(), "endpoint")
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, serverPath, "-http", "127.0.0.1:0", "-url-file", urlFile)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start streamable HTTP server: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			cancel()
			t.Fatalf("streamable HTTP server exited early: %v\n%s", err, out.String())
		default:
		}
		data, err := os.ReadFile(urlFile)
		if err == nil {
			endpoint := strings.TrimSpace(string(data))
			if endpoint != "" {
				t.Cleanup(func() {
					select {
					case err := <-done:
						if err != nil {
							t.Errorf("streamable HTTP server exited: %v\n%s", err, out.String())
						}
					default:
						cancel()
						<-done
					}
				})
				return endpoint
			}
		} else if !os.IsNotExist(err) {
			cancel()
			<-done
			t.Fatalf("read streamable HTTP endpoint: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	<-done
	t.Fatalf("streamable HTTP server did not write endpoint\n%s", out.String())
	return ""
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
