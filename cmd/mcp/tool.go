package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tmc/mcp"
	"github.com/tmc/mcp/internal/mcpcli"
	"golang.org/x/term"
)

type jsonSchema struct {
	Type        any                    `json:"type,omitempty"`
	Description string                 `json:"description,omitempty"`
	Properties  map[string]*jsonSchema `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Enum        []any                  `json:"enum,omitempty"`
	Items       *jsonSchema            `json:"items,omitempty"`
	Default     any                    `json:"default,omitempty"`
}

type toolCommandBuilder struct {
	app          *app
	tool         mcp.Tool
	schema       *jsonSchema
	properties   []*propertyBinding
	required     map[string]bool
	jsonFlagName string
}

type propertyBinding struct {
	key         string
	flagName    string
	usage       string
	kind        string
	enumValues  []string
	stringValue *string
	boolValue   *bool
	intValue    *int64
	floatValue  *float64
	jsonValue   *string
}

func newToolCommand(ctx context.Context, a *app, opts bootstrapOptions) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "tool",
		Short: "Discover and call MCP tools",
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tools",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := cmdContext(cmd, a.cfg.Timeout)
			defer cancel()
			sess, err := a.session(ctx)
			if err != nil {
				return err
			}
			tools, err := sess.ListToolsAll(ctx)
			if err != nil {
				return err
			}
			if a.output == mcpcli.OutputJSON || a.output == mcpcli.OutputNDJSON {
				data, err := json.MarshalIndent(tools, "", "  ")
				if err != nil {
					return err
				}
				return mcpcli.WriteOutput("", data)
			}
			for _, tool := range tools {
				if tool.Description != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", cobraName(tool.Name), tool.Description)
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), cobraName(tool.Name))
				}
			}
			return nil
		},
	}
	callCmd := &cobra.Command{
		Use:   "call",
		Short: "Call a discovered tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(listCmd, callCmd)

	if opts.Config.Cmd == "" && opts.Config.HTTPURL == "" && opts.Config.SSEURL == "" {
		return cmd, nil
	}
	sess, err := a.session(ctx)
	if err != nil {
		return nil, err
	}
	tools, err := sess.ListToolsAll(ctx)
	if err != nil {
		return nil, err
	}
	used := make(map[string]int)
	for _, tool := range tools {
		builder := newToolCommandBuilder(a, tool)
		callCmd.AddCommand(builder.command(used))
	}
	return cmd, nil
}

func newToolCommandBuilder(a *app, tool mcp.Tool) *toolCommandBuilder {
	builder := &toolCommandBuilder{
		app:          a,
		tool:         tool,
		required:     make(map[string]bool),
		jsonFlagName: "json",
	}
	if len(tool.InputSchema) > 0 {
		var schema jsonSchema
		if err := json.Unmarshal(tool.InputSchema, &schema); err == nil {
			builder.schema = &schema
			for _, name := range schema.Required {
				builder.required[name] = true
			}
		}
	}
	return builder
}

func (b *toolCommandBuilder) command(used map[string]int) *cobra.Command {
	name := uniqueName(cobraName(b.tool.Name), used)
	cmd := &cobra.Command{
		Use:           name,
		Short:         shortToolHelp(b.tool),
		Long:          longToolHelp(b.tool),
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.run(cmd)
		},
	}
	jsonInput := ""
	cmd.Flags().StringVar(&jsonInput, b.jsonFlagName, "", "raw JSON object for tool arguments")
	if b.schema != nil && b.schema.isObject() {
		b.addSchemaFlags(cmd)
	}
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if jsonInput == "" {
			return nil
		}
		if _, err := decodeJSONObject(jsonInput); err != nil {
			return fmt.Errorf("parse --json: %w", err)
		}
		return nil
	}
	return cmd
}

func (b *toolCommandBuilder) run(cmd *cobra.Command) error {
	jsonInput, err := cmd.Flags().GetString(b.jsonFlagName)
	if err != nil {
		return err
	}
	if jsonInput == "" && !term.IsTerminal(int(os.Stdin.Fd())) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		jsonInput = strings.TrimSpace(string(data))
	}
	args, err := decodeJSONObject(jsonInput)
	if err != nil {
		return fmt.Errorf("parse --json: %w", err)
	}
	if args == nil {
		args = make(map[string]any)
	}
	for _, binding := range b.properties {
		if !cmd.Flags().Changed(binding.flagName) {
			continue
		}
		value, err := binding.value()
		if err != nil {
			return fmt.Errorf("flag --%s: %w", binding.flagName, err)
		}
		args[binding.key] = value
	}
	for name := range b.required {
		if _, ok := args[name]; !ok {
			return fmt.Errorf("missing required argument %q", name)
		}
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return err
	}
	ctx, cancel := cmdContext(cmd, b.app.cfg.Timeout)
	defer cancel()
	sess, err := b.app.session(ctx)
	if err != nil {
		return err
	}
	result, err := sess.Client().CallTool(ctx, mcp.CallToolRequest{
		Name:      b.tool.Name,
		Arguments: raw,
	})
	if err != nil {
		return err
	}
	data, err := mcpcli.RenderToolResult(result, b.app.output)
	if err != nil {
		return err
	}
	return mcpcli.WriteOutput("", data)
}

func (b *toolCommandBuilder) addSchemaFlags(cmd *cobra.Command) {
	keys := make([]string, 0, len(b.schema.Properties))
	for key := range b.schema.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	usedFlags := map[string]int{b.jsonFlagName: 1}
	for _, key := range keys {
		prop := b.schema.Properties[key]
		binding := bindProperty(key, prop, b.required[key], usedFlags)
		if binding == nil {
			continue
		}
		b.properties = append(b.properties, binding)
		addFlag(cmd.Flags(), binding)
		if len(binding.enumValues) > 0 {
			values := append([]string(nil), binding.enumValues...)
			cmd.RegisterFlagCompletionFunc(binding.flagName, func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
				return values, cobra.ShellCompDirectiveNoFileComp
			})
		}
	}
}

func bindProperty(key string, schema *jsonSchema, required bool, used map[string]int) *propertyBinding {
	binding := &propertyBinding{
		key:      key,
		flagName: uniqueName(flagName(key), used),
		usage:    flagUsage(schema, required),
	}
	switch schema.kind() {
	case "string":
		value := ""
		if s, ok := schema.Default.(string); ok {
			value = s
		}
		binding.kind = "string"
		binding.stringValue = &value
		binding.enumValues = enumStrings(schema.Enum)
	case "boolean":
		value := false
		if v, ok := schema.Default.(bool); ok {
			value = v
		}
		binding.kind = "bool"
		binding.boolValue = &value
	case "integer":
		value := int64(0)
		if v, ok := schema.Default.(float64); ok {
			value = int64(v)
		}
		binding.kind = "int"
		binding.intValue = &value
	case "number":
		value := 0.0
		if v, ok := schema.Default.(float64); ok {
			value = v
		}
		binding.kind = "float"
		binding.floatValue = &value
	default:
		value := ""
		binding.kind = "json"
		binding.jsonValue = &value
	}
	return binding
}

func addFlag(flags *pflag.FlagSet, binding *propertyBinding) {
	switch binding.kind {
	case "string":
		flags.StringVar(binding.stringValue, binding.flagName, *binding.stringValue, binding.usage)
	case "bool":
		flags.BoolVar(binding.boolValue, binding.flagName, *binding.boolValue, binding.usage)
	case "int":
		flags.Int64Var(binding.intValue, binding.flagName, *binding.intValue, binding.usage)
	case "float":
		flags.Float64Var(binding.floatValue, binding.flagName, *binding.floatValue, binding.usage)
	default:
		flags.StringVar(binding.jsonValue, binding.flagName, "", binding.usage)
	}
}

func (b *propertyBinding) value() (any, error) {
	switch b.kind {
	case "string":
		return *b.stringValue, nil
	case "bool":
		return *b.boolValue, nil
	case "int":
		return *b.intValue, nil
	case "float":
		return *b.floatValue, nil
	default:
		var out any
		if err := json.Unmarshal([]byte(*b.jsonValue), &out); err != nil {
			return nil, err
		}
		return out, nil
	}
}

func shortToolHelp(tool mcp.Tool) string {
	if tool.Description != "" {
		return tool.Description
	}
	return fmt.Sprintf("Call MCP tool %q", tool.Name)
}

func longToolHelp(tool mcp.Tool) string {
	if len(tool.InputSchema) == 0 {
		return shortToolHelp(tool)
	}
	return shortToolHelp(tool) + "\n\nInput schema:\n" + string(tool.InputSchema)
}

func flagUsage(schema *jsonSchema, required bool) string {
	if schema == nil {
		if required {
			return "required"
		}
		return ""
	}
	var parts []string
	if schema.Description != "" {
		parts = append(parts, schema.Description)
	}
	if kind := schema.kind(); kind != "" {
		if kind == "json" {
			parts = append(parts, "JSON value")
		} else {
			parts = append(parts, kind)
		}
	}
	if len(schema.Enum) > 0 {
		parts = append(parts, "choices: "+strings.Join(enumStrings(schema.Enum), ", "))
	}
	if required {
		parts = append(parts, "required")
	}
	return strings.Join(parts, "; ")
}

func enumStrings(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case string:
			out = append(out, v)
		case float64:
			out = append(out, strconv.FormatFloat(v, 'f', -1, 64))
		case bool:
			out = append(out, strconv.FormatBool(v))
		}
	}
	sort.Strings(out)
	return out
}

func (s *jsonSchema) kind() string {
	if s == nil {
		return ""
	}
	switch v := s.Type.(type) {
	case string:
		return v
	case []any:
		for _, item := range v {
			if name, ok := item.(string); ok && name != "null" {
				return name
			}
		}
	}
	if len(s.Properties) > 0 {
		return "object"
	}
	if s.Items != nil {
		return "array"
	}
	return "json"
}

func (s *jsonSchema) isObject() bool { return s != nil && s.kind() == "object" }

var nonName = regexp.MustCompile(`[^a-zA-Z0-9-]+`)

func flagName(key string) string {
	key = splitName(key)
	key = nonName.ReplaceAllString(key, "-")
	key = strings.Trim(key, "-")
	if key == "" {
		return "arg"
	}
	return key
}

func cobraName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = splitName(strings.ReplaceAll(name, "/", "-"))
	name = nonName.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return "tool"
	}
	return name
}

func splitName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	var prev rune
	var havePrev bool
	for i, r := range s {
		switch {
		case r == '_' || r == '-' || r == '/' || unicode.IsSpace(r):
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
			havePrev = false
			prev = 0
			continue
		case i > 0 && havePrev && unicode.IsUpper(r):
			nextLower := false
			if j := i + len(string(r)); j < len(s) {
				next, _ := utf8.DecodeRuneInString(s[j:])
				nextLower = unicode.IsLower(next)
			}
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				if b.Len() > 0 && !lastDash {
					b.WriteByte('-')
					lastDash = true
				}
			}
		}
		b.WriteRune(unicode.ToLower(r))
		lastDash = false
		prev = r
		havePrev = true
	}
	return b.String()
}

func uniqueName(name string, used map[string]int) string {
	if used[name] == 0 {
		used[name] = 1
		return name
	}
	used[name]++
	return fmt.Sprintf("%s-%d", name, used[name])
}

func decodeJSONObject(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = make(map[string]any)
	}
	return out, nil
}
