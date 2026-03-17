package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestOpenRequiresListen(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-open")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected command to fail")
	}
	if !strings.Contains(string(out), "-open requires -l") {
		t.Fatalf("output=%q", string(out))
	}
}

func TestListenKeepsRecording(t *testing.T) {
	tmp := t.TempDir()
	recording := filepath.Join(tmp, "out.mcp")
	specPath := filepath.Join(tmp, "out.mcpspec")
	cmd := exec.Command("go", "run", ".", "-l", "-http", "127.0.0.1:0", "-f", recording, "--", "cat")
	cmd.Dir = "."
	cmd.Stdin = strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"echo\",\"arguments\":{\"message\":\"ping\"}}}\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run mcpspy: %v\nstderr:\n%s", err, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "\"tools/call\"") {
		t.Fatalf("stdout=%q", got)
	}
	data, err := os.ReadFile(recording)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "mcp-recv") || !strings.Contains(text, "mcp-send") {
		t.Fatalf("recording=%q", text)
	}
	specData, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(specData), "\"echo\"") {
		t.Fatalf("spec=%q", string(specData))
	}
	if !strings.Contains(stderr.String(), "http://127.0.0.1:") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestSignalStartsUI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signals differ on windows")
	}
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "mcpspy")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = "."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build mcpspy: %v\n%s", err, out)
	}

	recording := filepath.Join(tmp, "out.mcp")
	cmd := exec.Command(bin, "-f", recording, "--", "cat")
	cmd.Dir = "."
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)
	if err := cmd.Process.Signal(syscall.SIGUSR1); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if _, err := io.WriteString(stdin, "{\"method\":\"ping\"}\n"); err != nil {
		t.Fatalf("write stdin: %v\nstderr:\n%s", err, stderr.String())
	}
	_ = stdin.Close()

	if err := cmd.Wait(); err != nil {
		t.Fatalf("wait mcpspy: %v\nstderr:\n%s", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "http://127.0.0.1:") {
		t.Fatalf("stderr=%q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "{\"method\":\"ping\"}") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}
