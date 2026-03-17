package mcpcli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tmc/mcp"
)

// OutputMode controls CLI rendering format.
type OutputMode string

const (
	OutputText   OutputMode = "text"
	OutputJSON   OutputMode = "json"
	OutputNDJSON OutputMode = "ndjson"
)

// RenderToolResult returns either plain text or structured JSON for a tool result.
func RenderToolResult(result *mcp.CallToolResult, mode OutputMode) ([]byte, error) {
	if mode == OutputJSON || mode == OutputNDJSON {
		return json.MarshalIndent(result, "", "  ")
	}
	text := textOnlyResult(result)
	if text != "" {
		return []byte(text), nil
	}
	return json.MarshalIndent(result, "", "  ")
}

// RenderPromptResult returns a terminal-friendly prompt transcript or JSON.
func RenderPromptResult(result *mcp.GetPromptResult, mode OutputMode) ([]byte, error) {
	if mode == OutputJSON || mode == OutputNDJSON {
		return json.MarshalIndent(result, "", "  ")
	}
	var b strings.Builder
	for i, msg := range result.Messages {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "[%s]\n", msg.Role)
		for _, item := range msg.Content {
			switch v := item.(type) {
			case map[string]any:
				if v["type"] == "text" {
					if text, ok := v["text"].(string); ok {
						b.WriteString(text)
						continue
					}
				}
				raw, _ := json.MarshalIndent(v, "", "  ")
				b.Write(raw)
			default:
				raw, _ := json.MarshalIndent(v, "", "  ")
				b.Write(raw)
			}
		}
	}
	return []byte(b.String()), nil
}

// RenderResourceResult renders a resource read result.
func RenderResourceResult(result *mcp.ReadResourceResult, mode OutputMode) ([]byte, error) {
	if mode == OutputJSON || mode == OutputNDJSON {
		return json.MarshalIndent(result, "", "  ")
	}
	var b bytes.Buffer
	for i, content := range result.Contents {
		if i > 0 {
			b.WriteString("\n")
		}
		switch v := content.(type) {
		case mcp.TextResourceContents:
			b.WriteString(v.Text)
		case mcp.BlobResourceContents:
			decoded, err := base64.StdEncoding.DecodeString(v.Blob)
			if err != nil {
				return nil, err
			}
			b.Write(decoded)
		default:
			raw, _ := json.MarshalIndent(v, "", "  ")
			b.Write(raw)
		}
	}
	return b.Bytes(), nil
}

func textOnlyResult(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	lines := make([]string, 0, len(result.Content))
	for _, item := range result.Content {
		m, ok := item.(map[string]any)
		if !ok {
			return ""
		}
		if kind, _ := m["type"].(string); kind != "text" {
			return ""
		}
		text, ok := m["text"].(string)
		if !ok {
			return ""
		}
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n")
}

// WriteOutput writes rendered output with a trailing newline for text/json modes.
func WriteOutput(path string, data []byte) error {
	if path == "" {
		if len(data) == 0 {
			return nil
		}
		if bytes.HasSuffix(data, []byte("\n")) {
			_, err := os.Stdout.Write(data)
			return err
		}
		_, err := fmt.Fprintln(os.Stdout, string(data))
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ParseOutputMode validates output mode.
func ParseOutputMode(s string) (OutputMode, error) {
	switch OutputMode(s) {
	case "", OutputText:
		return OutputText, nil
	case OutputJSON:
		return OutputJSON, nil
	case OutputNDJSON:
		return OutputNDJSON, nil
	default:
		return "", errors.New("invalid output mode")
	}
}
