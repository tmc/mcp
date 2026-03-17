package mcpcli

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/tmc/mcp"
)

// CommandTransport starts command and uses its stdin/stdout as an MCP transport.
func CommandTransport(command string, stderr io.Writer) mcp.Transport {
	return mcp.TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
		return startCommand(ctx, command, stderr)
	})
}

type commandConn struct {
	cmd    *os.Process
	wait   func() error
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (c *commandConn) Read(p []byte) (int, error) {
	return c.stdout.Read(p)
}

func (c *commandConn) Write(p []byte) (int, error) {
	return c.stdin.Write(p)
}

func (c *commandConn) Close() error {
	_ = c.stdin.Close()
	_ = c.stdout.Close()
	if c.cmd != nil {
		_ = c.cmd.Kill()
	}
	if c.wait != nil {
		_ = c.wait()
	}
	return nil
}

func startCommand(ctx context.Context, command string, stderr io.Writer) (io.ReadWriteCloser, error) {
	if strings.TrimSpace(command) == "" {
		return nil, errors.New("empty command")
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-lc", command)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &commandConn{
		cmd:    cmd.Process,
		wait:   cmd.Wait,
		stdin:  stdin,
		stdout: stdout,
	}, nil
}
