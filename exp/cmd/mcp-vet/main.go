// mcp-vet probes an MCP server and reports common issues and anti-patterns.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	timeout    = flag.Duration("timeout", 15*time.Second, "Timeout for server operations")
	verbose    = flag.Bool("v", false, "Verbose output")
	jsonOutput = flag.Bool("json", false, "Output results as JSON")
	exitCode   = flag.Bool("exit-code", true, "Exit with non-zero status if issues found")
	checkLevel = flag.String("level", "warn", "Minimum level to report: info, warn, error")
	noColor    = flag.Bool("no-color", false, "Disable color output")
)

// Severity levels for findings.
type Severity int

const (
	Info  Severity = iota
	Warn
	Error
)

func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warn:
		return "warn"
	case Error:
		return "error"
	}
	return "unknown"
}

// Finding represents a single diagnostic finding.
type Finding struct {
	Severity Severity `json:"severity"`
	Check    string   `json:"check"`
	Target   string   `json:"target,omitempty"` // tool/resource/prompt name
	Message  string   `json:"message"`
	Hint     string   `json:"hint,omitempty"`
}

// Results collects all findings from a vet run.
type Results struct {
	ServerName    string    `json:"server_name,omitempty"`
	ServerVersion string    `json:"server_version,omitempty"`
	Protocol      string    `json:"protocol,omitempty"`
	Findings      []Finding `json:"findings"`
	Errors        int       `json:"errors"`
	Warnings      int       `json:"warnings"`
	Infos         int       `json:"infos"`
}

func (r *Results) add(f Finding) {
	r.Findings = append(r.Findings, f)
	switch f.Severity {
	case Error:
		r.Errors++
	case Warn:
		r.Warnings++
	case Info:
		r.Infos++
	}
}

// jsonrpcMsg is the base structure for all JSON-RPC 2.0 messages.
type jsonrpcMsg struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// probe holds the state for a single server probe session.
type probe struct {
	stdin  io.WriteCloser
	stdout *bufio.Reader
	nextID int
	ctx    context.Context
	v      bool
}

func newProbe(ctx context.Context, args []string, verbose bool) (*probe, func(), error) {
	if len(args) == 0 {
		return nil, nil, fmt.Errorf("no server command provided")
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr // let server stderr through

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start server: %w", err)
	}

	cleanup := func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}

	return &probe{
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		ctx:    ctx,
		v:      verbose,
	}, cleanup, nil
}

func (p *probe) send(method string, params any) (int, error) {
	p.nextID++
	id := p.nextID

	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return 0, err
		}
		rawParams = b
	}

	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if rawParams != nil {
		msg["params"] = rawParams
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}
	if p.v {
		fmt.Fprintf(os.Stderr, "→ %s\n", data)
	}
	_, err = fmt.Fprintf(p.stdin, "%s\n", data)
	return id, err
}

func (p *probe) notify(method string, params any) error {
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}
		msg["params"] = b
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if p.v {
		fmt.Fprintf(os.Stderr, "→ (notify) %s\n", data)
	}
	_, err = fmt.Fprintf(p.stdin, "%s\n", data)
	return err
}

func (p *probe) receive() (*jsonrpcMsg, error) {
	for {
		line, err := p.stdout.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if p.v {
			fmt.Fprintf(os.Stderr, "← %s\n", line)
		}
		var msg jsonrpcMsg
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}
		// Skip server-initiated notifications
		if msg.Method != "" && msg.ID == nil {
			continue
		}
		return &msg, nil
	}
}

func (p *probe) call(method string, params any) (*jsonrpcMsg, error) {
	if _, err := p.send(method, params); err != nil {
		return nil, err
	}
	return p.receive()
}

// --- vet checks ---

// vetServer runs all checks and returns Results.
func vetServer(ctx context.Context, args []string) *Results {
	results := &Results{}

	p, cleanup, err := newProbe(ctx, args, *verbose)
	if err != nil {
		results.add(Finding{
			Severity: Error,
			Check:    "startup",
			Message:  fmt.Sprintf("failed to start server: %v", err),
		})
		return results
	}
	defer cleanup()

	// 1. Initialize handshake
	initResult := checkInitialize(p, results)
	if initResult == nil {
		return results
	}

	results.ServerName = initResult.serverName
	results.ServerVersion = initResult.serverVersion
	results.Protocol = initResult.protocolVersion

	// Send initialized notification
	_ = p.notify("notifications/initialized", nil)

	// 2. Check tools
	tools := checkTools(p, results)

	// 3. Check prompts
	checkPrompts(p, results)

	// 4. Check resources
	checkResources(p, results)

	// 5. Cross-cutting checks on tool/prompt/resource names
	checkNamingConventions(tools, results)

	// 6. Ping check
	checkPing(p, results)

	// 7. Unknown method handling
	checkUnknownMethod(p, results)

	return results
}

type initInfo struct {
	serverName      string
	serverVersion   string
	protocolVersion string
	capabilities    map[string]json.RawMessage
}

// checkInitialize verifies the server responds correctly to initialize.
func checkInitialize(p *probe, results *Results) *initInfo {
	resp, err := p.call("initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"clientInfo": map[string]any{
			"name":    "mcp-vet",
			"version": "0.1.0",
		},
		"capabilities": map[string]any{},
	})
	if err != nil {
		results.add(Finding{
			Severity: Error,
			Check:    "initialize",
			Message:  fmt.Sprintf("initialize failed: %v", err),
		})
		return nil
	}

	if resp.Error != nil {
		results.add(Finding{
			Severity: Error,
			Check:    "initialize",
			Message:  fmt.Sprintf("server returned error: code=%d %s", resp.Error.Code, resp.Error.Message),
		})
		return nil
	}

	var init struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
		Capabilities map[string]json.RawMessage `json:"capabilities"`
		Instructions string                      `json:"instructions"`
	}
	if err := json.Unmarshal(resp.Result, &init); err != nil {
		results.add(Finding{
			Severity: Error,
			Check:    "initialize",
			Message:  fmt.Sprintf("failed to parse initialize result: %v", err),
		})
		return nil
	}

	// Check protocol version
	known := map[string]bool{
		"2025-03-26": true,
		"2024-11-05": true,
	}
	if !known[init.ProtocolVersion] {
		results.add(Finding{
			Severity: Warn,
			Check:    "initialize/protocolVersion",
			Message:  fmt.Sprintf("unknown protocol version %q", init.ProtocolVersion),
			Hint:     "expected 2025-03-26 or 2024-11-05",
		})
	}

	// Check serverInfo
	if init.ServerInfo.Name == "" {
		results.add(Finding{
			Severity: Warn,
			Check:    "initialize/serverInfo",
			Message:  "serverInfo.name is empty",
			Hint:     "provide a descriptive server name",
		})
	}
	if init.ServerInfo.Version == "" {
		results.add(Finding{
			Severity: Info,
			Check:    "initialize/serverInfo",
			Message:  "serverInfo.version is empty",
		})
	}

	return &initInfo{
		serverName:      init.ServerInfo.Name,
		serverVersion:   init.ServerInfo.Version,
		protocolVersion: init.ProtocolVersion,
		capabilities:    init.Capabilities,
	}
}

// toolInfo holds parsed tool metadata.
type toolInfo struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

// checkTools inspects tools/list and individual tool schemas.
func checkTools(p *probe, results *Results) []toolInfo {
	resp, err := p.call("tools/list", map[string]any{})
	if err != nil {
		results.add(Finding{
			Severity: Warn,
			Check:    "tools/list",
			Message:  fmt.Sprintf("tools/list failed: %v", err),
		})
		return nil
	}
	if resp.Error != nil {
		// Not all servers have tools - method not found is acceptable
		if resp.Error.Code == -32601 {
			results.add(Finding{
				Severity: Info,
				Check:    "tools/list",
				Message:  "server does not support tools",
			})
		} else {
			results.add(Finding{
				Severity: Warn,
				Check:    "tools/list",
				Message:  fmt.Sprintf("tools/list error: code=%d %s", resp.Error.Code, resp.Error.Message),
			})
		}
		return nil
	}

	var result struct {
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"inputSchema"`
		} `json:"tools"`
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		results.add(Finding{
			Severity: Error,
			Check:    "tools/list",
			Message:  fmt.Sprintf("failed to parse tools/list result: %v", err),
		})
		return nil
	}

	var tools []toolInfo
	for _, t := range result.Tools {
		tools = append(tools, toolInfo{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	vetTools(tools, results)
	return tools
}

// vetTools checks individual tool definitions.
func vetTools(tools []toolInfo, results *Results) {
	seen := map[string]bool{}
	for _, t := range tools {
		target := "tool:" + t.Name

		// Duplicate names
		if seen[t.Name] {
			results.add(Finding{
				Severity: Error,
				Check:    "tools/duplicate",
				Target:   target,
				Message:  fmt.Sprintf("duplicate tool name %q", t.Name),
			})
		}
		seen[t.Name] = true

		// Empty name
		if t.Name == "" {
			results.add(Finding{
				Severity: Error,
				Check:    "tools/name",
				Target:   target,
				Message:  "tool has empty name",
			})
			continue
		}

		// Description
		if t.Description == "" {
			results.add(Finding{
				Severity: Warn,
				Check:    "tools/description",
				Target:   target,
				Message:  fmt.Sprintf("tool %q has no description", t.Name),
				Hint:     "descriptions help LLMs understand when to use a tool",
			})
		} else {
			vetDescription("tool", t.Name, t.Description, results)
		}

		// InputSchema
		if len(t.InputSchema) == 0 || string(t.InputSchema) == "null" {
			results.add(Finding{
				Severity: Warn,
				Check:    "tools/inputSchema",
				Target:   target,
				Message:  fmt.Sprintf("tool %q has no inputSchema", t.Name),
				Hint:     `provide {"type":"object","properties":{},"required":[]} at minimum`,
			})
		} else {
			vetInputSchema(t.Name, t.InputSchema, results)
		}
	}
}

// vetInputSchema checks a tool's JSON Schema for common issues.
func vetInputSchema(toolName string, schema json.RawMessage, results *Results) {
	target := "tool:" + toolName

	var s map[string]json.RawMessage
	if err := json.Unmarshal(schema, &s); err != nil {
		results.add(Finding{
			Severity: Error,
			Check:    "tools/inputSchema",
			Target:   target,
			Message:  fmt.Sprintf("tool %q has invalid inputSchema JSON: %v", toolName, err),
		})
		return
	}

	// Must have "type"
	typeVal, hasType := s["type"]
	if !hasType {
		results.add(Finding{
			Severity: Warn,
			Check:    "tools/inputSchema/type",
			Target:   target,
			Message:  fmt.Sprintf("tool %q inputSchema missing top-level 'type' field", toolName),
			Hint:     `add "type": "object"`,
		})
	} else {
		var typStr string
		_ = json.Unmarshal(typeVal, &typStr)
		if typStr != "object" {
			results.add(Finding{
				Severity: Warn,
				Check:    "tools/inputSchema/type",
				Target:   target,
				Message:  fmt.Sprintf("tool %q inputSchema top-level type is %q, expected 'object'", toolName, typStr),
			})
		}
	}

	// Check properties descriptions
	if propsRaw, ok := s["properties"]; ok {
		var props map[string]json.RawMessage
		if err := json.Unmarshal(propsRaw, &props); err == nil {
			for propName, propSchema := range props {
				var prop map[string]json.RawMessage
				if err := json.Unmarshal(propSchema, &prop); err != nil {
					continue
				}
				if _, hasDesc := prop["description"]; !hasDesc {
					results.add(Finding{
						Severity: Info,
						Check:    "tools/inputSchema/property",
						Target:   target,
						Message:  fmt.Sprintf("tool %q parameter %q has no description", toolName, propName),
						Hint:     "parameter descriptions improve LLM argument generation",
					})
				}
			}
		}
	}
}

// checkPrompts inspects prompts/list results.
func checkPrompts(p *probe, results *Results) {
	resp, err := p.call("prompts/list", map[string]any{})
	if err != nil {
		return // not a hard failure
	}
	if resp.Error != nil {
		if resp.Error.Code != -32601 {
			results.add(Finding{
				Severity: Info,
				Check:    "prompts/list",
				Message:  fmt.Sprintf("prompts/list error: %s", resp.Error.Message),
			})
		}
		return
	}

	var result struct {
		Prompts []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Arguments   []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Required    bool   `json:"required"`
			} `json:"arguments"`
		} `json:"prompts"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return
	}

	seen := map[string]bool{}
	for _, pr := range result.Prompts {
		target := "prompt:" + pr.Name

		if seen[pr.Name] {
			results.add(Finding{
				Severity: Error,
				Check:    "prompts/duplicate",
				Target:   target,
				Message:  fmt.Sprintf("duplicate prompt name %q", pr.Name),
			})
		}
		seen[pr.Name] = true

		if pr.Description == "" {
			results.add(Finding{
				Severity: Warn,
				Check:    "prompts/description",
				Target:   target,
				Message:  fmt.Sprintf("prompt %q has no description", pr.Name),
			})
		} else {
			vetDescription("prompt", pr.Name, pr.Description, results)
		}

		for _, arg := range pr.Arguments {
			if arg.Description == "" {
				results.add(Finding{
					Severity: Info,
					Check:    "prompts/argument",
					Target:   target,
					Message:  fmt.Sprintf("prompt %q argument %q has no description", pr.Name, arg.Name),
				})
			}
		}
	}
}

// checkResources inspects resources/list results.
func checkResources(p *probe, results *Results) {
	resp, err := p.call("resources/list", map[string]any{})
	if err != nil {
		return
	}
	if resp.Error != nil {
		if resp.Error.Code != -32601 {
			results.add(Finding{
				Severity: Info,
				Check:    "resources/list",
				Message:  fmt.Sprintf("resources/list error: %s", resp.Error.Message),
			})
		}
		return
	}

	var result struct {
		Resources []struct {
			URI         string `json:"uri"`
			Name        string `json:"name"`
			Description string `json:"description"`
			MimeType    string `json:"mimeType"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return
	}

	seen := map[string]bool{}
	for _, r := range result.Resources {
		target := "resource:" + r.URI

		if r.URI == "" {
			results.add(Finding{
				Severity: Error,
				Check:    "resources/uri",
				Target:   target,
				Message:  "resource has empty URI",
			})
			continue
		}

		if seen[r.URI] {
			results.add(Finding{
				Severity: Error,
				Check:    "resources/duplicate",
				Target:   target,
				Message:  fmt.Sprintf("duplicate resource URI %q", r.URI),
			})
		}
		seen[r.URI] = true

		if r.Description == "" {
			results.add(Finding{
				Severity: Info,
				Check:    "resources/description",
				Target:   target,
				Message:  fmt.Sprintf("resource %q has no description", r.URI),
			})
		}

		if r.MimeType == "" {
			results.add(Finding{
				Severity: Info,
				Check:    "resources/mimeType",
				Target:   target,
				Message:  fmt.Sprintf("resource %q has no mimeType", r.URI),
				Hint:     "mimeType helps clients display resources correctly",
			})
		}
	}
}

var toolNameRE = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// checkNamingConventions checks tool names for common anti-patterns.
func checkNamingConventions(tools []toolInfo, results *Results) {
	for _, t := range tools {
		target := "tool:" + t.Name

		// Mixed case tool names
		if !toolNameRE.MatchString(t.Name) {
			results.add(Finding{
				Severity: Warn,
				Check:    "naming/tool",
				Target:   target,
				Message:  fmt.Sprintf("tool name %q does not match recommended pattern [a-z][a-z0-9_-]*", t.Name),
				Hint:     "lowercase names with underscores or hyphens are clearest for LLMs",
			})
		}

		// Very long names
		if len(t.Name) > 64 {
			results.add(Finding{
				Severity: Warn,
				Check:    "naming/toolLength",
				Target:   target,
				Message:  fmt.Sprintf("tool name %q is very long (%d chars)", t.Name, len(t.Name)),
				Hint:     "keep tool names under 64 characters",
			})
		}

		// Vague names
		vague := []string{"do", "run", "execute", "process", "handle", "action", "perform", "go", "call"}
		for _, v := range vague {
			if strings.EqualFold(t.Name, v) {
				results.add(Finding{
					Severity: Warn,
					Check:    "naming/toolVague",
					Target:   target,
					Message:  fmt.Sprintf("tool name %q is too generic", t.Name),
					Hint:     "use a specific verb that describes the action",
				})
				break
			}
		}
	}
}

// checkPing tests the ping method.
func checkPing(p *probe, results *Results) {
	resp, err := p.call("ping", nil)
	if err != nil {
		results.add(Finding{
			Severity: Info,
			Check:    "ping",
			Message:  fmt.Sprintf("ping failed: %v", err),
		})
		return
	}
	if resp.Error != nil && resp.Error.Code != -32601 {
		results.add(Finding{
			Severity: Info,
			Check:    "ping",
			Message:  fmt.Sprintf("ping returned error: %s", resp.Error.Message),
		})
	}
}

// checkUnknownMethod verifies the server returns a proper -32601 for unknown methods.
func checkUnknownMethod(p *probe, results *Results) {
	resp, err := p.call("mcp_vet/nonexistent_method_12345", nil)
	if err != nil {
		return // connection issue, already reported
	}
	if resp.Error == nil {
		results.add(Finding{
			Severity: Warn,
			Check:    "protocol/methodNotFound",
			Message:  "server returned success for an unknown method (should return -32601)",
			Hint:     "return JSON-RPC error -32601 for unknown methods",
		})
		return
	}
	if resp.Error.Code != -32601 {
		results.add(Finding{
			Severity: Info,
			Check:    "protocol/methodNotFound",
			Message:  fmt.Sprintf("server returned code %d for unknown method (expected -32601)", resp.Error.Code),
		})
	}
}

// vetDescription checks description text for common issues.
func vetDescription(kind, name, desc string, results *Results) {
	target := kind + ":" + name

	// Very short descriptions are often useless
	if utf8.RuneCountInString(desc) < 10 {
		results.add(Finding{
			Severity: Info,
			Check:    "description/length",
			Target:   target,
			Message:  fmt.Sprintf("%s %q description is very short (%d chars)", kind, name, len(desc)),
			Hint:     "more descriptive text helps LLMs use your tools correctly",
		})
	}

	// Starts with lowercase (minor style)
	r, _ := utf8.DecodeRuneInString(desc)
	if unicode.IsLower(r) {
		results.add(Finding{
			Severity: Info,
			Check:    "description/style",
			Target:   target,
			Message:  fmt.Sprintf("%s %q description starts with lowercase", kind, name),
		})
	}

	// Ends with period - common style inconsistency
	trimmed := strings.TrimSpace(desc)
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '.' {
		// Ending with period is fine; just a note if mixing styles would be flagged
		// Don't flag this - periods at end are perfectly acceptable.
	}

	// Detect placeholder text
	placeholders := []string{"todo", "fixme", "tbd", "placeholder", "description here"}
	lower := strings.ToLower(desc)
	for _, ph := range placeholders {
		if strings.Contains(lower, ph) {
			results.add(Finding{
				Severity: Warn,
				Check:    "description/placeholder",
				Target:   target,
				Message:  fmt.Sprintf("%s %q description contains placeholder text %q", kind, name, ph),
			})
			break
		}
	}
}

// --- output formatting ---

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorGray   = "\033[90m"
)

func color(s, c string) string {
	if *noColor {
		return s
	}
	return c + s + colorReset
}

func severityColor(s Severity) string {
	switch s {
	case Error:
		return colorRed
	case Warn:
		return colorYellow
	case Info:
		return colorCyan
	}
	return ""
}

func printResults(r *Results, minLevel Severity) {
	if *jsonOutput {
		filtered := &Results{
			ServerName:    r.ServerName,
			ServerVersion: r.ServerVersion,
			Protocol:      r.Protocol,
		}
		for _, f := range r.Findings {
			if f.Severity >= minLevel {
				filtered.add(f)
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(filtered)
		return
	}

	// Header
	serverLine := r.ServerName
	if r.ServerVersion != "" {
		serverLine += " " + r.ServerVersion
	}
	if serverLine != "" {
		fmt.Printf("%s %s (protocol %s)\n\n", color("Server:", colorBold), serverLine, r.Protocol)
	}

	// Group by severity
	var byLevel [3][]Finding
	for _, f := range r.Findings {
		if f.Severity >= minLevel {
			byLevel[f.Severity] = append(byLevel[f.Severity], f)
		}
	}

	// Sort each group by check name for stable output
	for i := range byLevel {
		sort.Slice(byLevel[i], func(a, b int) bool {
			if byLevel[i][a].Check != byLevel[i][b].Check {
				return byLevel[i][a].Check < byLevel[i][b].Check
			}
			return byLevel[i][a].Target < byLevel[i][b].Target
		})
	}

	total := 0
	for sev := Error; sev >= Info; sev-- {
		for _, f := range byLevel[sev] {
			label := color(strings.ToUpper(f.Severity.String()), severityColor(f.Severity))
			loc := f.Check
			if f.Target != "" {
				loc += " " + color(f.Target, colorGray)
			}
			fmt.Printf("%s  %s: %s\n", label, loc, f.Message)
			if f.Hint != "" && minLevel <= Info {
				fmt.Printf("       %s\n", color("hint: "+f.Hint, colorGray))
			}
			total++
		}
	}

	if total == 0 {
		fmt.Println(color("No issues found.", colorBold))
	} else {
		fmt.Printf("\n%d issue(s): %d error(s), %d warning(s), %d info\n",
			r.Errors+r.Warnings+r.Infos, r.Errors, r.Warnings, r.Infos)
	}
}

func parseSeverity(s string) Severity {
	switch strings.ToLower(s) {
	case "error":
		return Error
	case "warn", "warning":
		return Warn
	default:
		return Info
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mcp-vet [flags] -- <server-command> [args...]\n\n")
		fmt.Fprintf(os.Stderr, "mcp-vet probes an MCP server and reports common issues.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  mcp-vet -- go run ./my-server\n")
		fmt.Fprintf(os.Stderr, "  mcp-vet -level=error -- ./my-server --config prod.json\n")
		fmt.Fprintf(os.Stderr, "  mcp-vet -json -- npx @modelcontextprotocol/server-everything stdio\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	minLevel := parseSeverity(*checkLevel)

	results := vetServer(ctx, args)

	printResults(results, minLevel)

	if *exitCode && results.Errors > 0 {
		os.Exit(1)
	}
}
