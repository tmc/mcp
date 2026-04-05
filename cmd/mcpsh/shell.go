package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

const completionAnnotation = "mcpsh_completion"

type interactiveShell struct {
	newRoot func() *cobra.Command
	prompt  string
	writer  io.Writer
	app     *app
}

type shellParseResult struct {
	args          []string
	words         []string
	current       string
	replaceStart  int
	replaceEnd    int
	trailingSpace bool
}

func newShellCommand(opts bootstrapOptions, app *app) *cobra.Command {
	return &cobra.Command{
		Use:     "shell",
		Short:   "Start an interactive shell",
		Long:    "Start an interactive shell with line editing and tab completion for discovered tools and flags.",
		GroupID: groupMeta,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveShell(cmd, opts, app)
		},
	}
}

func runInteractiveShell(cmd *cobra.Command, opts bootstrapOptions, app *app) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return errors.New("interactive shell requires a terminal")
	}

	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("enter raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)

	t := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{Reader: os.Stdin, Writer: os.Stdout}, shellPrompt(app))
	sh := &interactiveShell{
		newRoot: func() *cobra.Command {
			root := newRootCommand(opts, app)
			hideInteractiveFlags(root)
			root.SetContext(cmd.Context())
			return root
		},
		prompt: shellPrompt(app),
		writer: t,
		app:    app,
	}
	t.AutoCompleteCallback = sh.autoComplete
	printShellBanner(t, app)

	for {
		line, err := t.ReadLine()
		if errors.Is(err, io.EOF) {
			_, _ = io.WriteString(t, "\n")
			return nil
		}
		if err != nil {
			return err
		}
		exit, err := sh.runLine(cmd.Context(), line)
		if err != nil {
			_, _ = fmt.Fprintf(t, "error: %v\n", err)
			continue
		}
		if exit {
			return nil
		}
	}
}

func shellPrompt(app *app) string {
	if app != nil && len(app.servers) == 1 {
		srv := app.servers[0]
		if srv.info != nil && srv.info.ServerInfo.Name != "" {
			return srv.info.ServerInfo.Name + "> "
		}
	}
	return toolName + "> "
}

func printShellBanner(w io.Writer, app *app) {
	name := toolName
	if app != nil && len(app.servers) == 1 {
		srv := app.servers[0]
		if srv.info != nil && srv.info.ServerInfo.Name != "" {
			name = srv.info.ServerInfo.Name
		}
	}
	toolCount := 0
	if app != nil {
		toolCount = len(app.allTools())
	}
	_, _ = fmt.Fprintf(w, "%s interactive shell\n", name)
	if app != nil && len(app.servers) > 1 {
		_, _ = fmt.Fprintf(w, "%d servers, %d tools loaded. Type servers, tools, help, reload, clear, or quit.\n", len(app.servers), toolCount)
	} else {
		_, _ = fmt.Fprintf(w, "%d tools loaded. Tab completes commands, flags, and enum values. Type tools, help, reload, clear, or quit.\n", toolCount)
	}
}

func hideInteractiveFlags(root *cobra.Command) {
	if root == nil {
		return
	}
	root.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		_ = root.PersistentFlags().MarkHidden(flag.Name)
	})
}

func (s *interactiveShell) runLine(ctx context.Context, line string) (bool, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return false, nil
	}
	args, err := splitShellLine(line)
	if err != nil {
		return false, err
	}
	args = normalizeShellArgs(args, commandNames(s.newRoot()))
	if len(args) == 0 {
		return false, nil
	}
	switch args[0] {
	case "exit", "quit":
		return true, nil
	case "?":
		return false, s.printHelp(nil)
	case "help":
		return false, s.printHelp(args[1:])
	case "tools":
		return false, s.printTools()
	case "servers":
		return false, s.printServers()
	case "reload":
		return false, s.reloadTools(ctx)
	case "clear":
		return false, s.clearScreen()
	}

	root := s.newRoot()
	root.SetOut(s.writer)
	root.SetErr(s.writer)
	root.SetContext(ctx)
	if err := executeShellArgs(root, args); err != nil {
		return false, decorateShellError(err, args[0], root)
	}
	return false, nil
}

func (s *interactiveShell) printServers() error {
	if s.app == nil || len(s.app.servers) == 0 {
		_, _ = io.WriteString(s.writer, "no servers connected\n")
		return nil
	}
	for _, srv := range s.app.servers {
		name := srv.name
		info := ""
		if srv.info != nil && srv.info.ServerInfo.Name != "" {
			info = " (" + srv.info.ServerInfo.Name + ")"
		}
		_, _ = fmt.Fprintf(s.writer, "  %s%s  %d tools\n", name, info, len(srv.tools))
	}
	return nil
}

func (s *interactiveShell) printHelp(args []string) error {
	if len(args) == 0 {
		_, _ = fmt.Fprintf(s.writer, "Builtins: help [command], tools, servers, reload, clear, quit\n")
		_, _ = io.WriteString(s.writer, "Commands:\n")
		return s.printTools()
	}
	name := args[0]
	if text, ok := builtinHelp(name); ok {
		_, _ = io.WriteString(s.writer, text+"\n")
		return nil
	}
	root := s.newRoot()
	root.SetOut(s.writer)
	root.SetErr(s.writer)
	root.SetArgs([]string{name, "--help"})
	return root.Execute()
}

func (s *interactiveShell) printTools() error {
	root := s.newRoot()
	var names []string
	width := 0
	for _, name := range commandNames(root) {
		if isShellBuiltin(name) {
			continue
		}
		cmd := findSubcommand(root, name)
		if cmd == nil {
			continue
		}
		if len(name) > width {
			width = len(name)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		_, _ = io.WriteString(s.writer, "no commands loaded\n")
		return nil
	}
	sort.Strings(names)
	for _, name := range names {
		cmd := findSubcommand(root, name)
		if cmd == nil {
			continue
		}
		_, _ = fmt.Fprintf(s.writer, "  %-*s  %s\n", width, name, cmd.Short)
	}
	return nil
}

func (s *interactiveShell) reloadTools(ctx context.Context) error {
	if s.app == nil {
		_, _ = io.WriteString(s.writer, "no server configured\n")
		return nil
	}
	if err := s.app.reloadTools(ctx); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(s.writer, "reloaded %d tools\n", len(s.app.allTools()))
	return nil
}

func (s *interactiveShell) clearScreen() error {
	_, _ = io.WriteString(s.writer, "\x1b[2J\x1b[H")
	printShellBanner(s.writer, s.app)
	return nil
}

func builtinHelp(name string) (string, bool) {
	if name == "?" {
		name = "help"
	}
	switch name {
	case "help":
		return "help [command]  show shell help or command-specific help", true
	case "tools":
		return "tools           list discovered commands in compact form", true
	case "servers":
		return "servers         list connected MCP servers", true
	case "reload":
		return "reload          refresh discovered tools from the MCP server(s)", true
	case "clear":
		return "clear           clear the screen and reprint the shell banner", true
	case "quit", "exit":
		return "quit            leave the interactive shell", true
	default:
		return "", false
	}
}

func isShellBuiltin(name string) bool {
	switch name {
	case "help", "tools", "servers", "reload", "clear", "quit", "exit", "?":
		return true
	default:
		return false
	}
}

func normalizeShellArgs(args []string, candidates []string) []string {
	if len(args) != 1 {
		return args
	}
	token := args[0]
	for _, candidate := range candidates {
		if candidate == "" || len(token) != len(candidate)*2 {
			continue
		}
		if token == candidate+candidate {
			return []string{candidate}
		}
	}
	return args
}

func decorateShellError(err error, token string, root *cobra.Command) error {
	if err == nil {
		return nil
	}
	if !strings.Contains(err.Error(), "unknown command") {
		return err
	}
	suggestions := suggestCommands(token, commandNames(root))
	if len(suggestions) == 0 {
		return err
	}
	return fmt.Errorf("%w\ntry: %s", err, strings.Join(suggestions, ", "))
}

func executeShellLine(root *cobra.Command, line string) error {
	args, err := splitShellLine(line)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return nil
	}
	return executeShellArgs(root, args)
}

func executeShellArgs(root *cobra.Command, args []string) error {
	if args[0] == "shell" {
		return errors.New("already in interactive shell")
	}
	root.SetArgs(args)
	return root.Execute()
}

func (s *interactiveShell) autoComplete(line string, pos int, key rune) (string, int, bool) {
	if key != '\t' {
		return "", 0, false
	}
	if pos < 0 || pos > len(line) {
		pos = len(line)
	}
	parsed, err := parseShellLine(line[:pos])
	if err != nil {
		return "", 0, false
	}

	root := s.newRoot()
	suggestions := completionCandidates(root, parsed)
	if len(suggestions) == 0 {
		return "", 0, false
	}
	sort.Strings(suggestions)
	prefix := longestCommonPrefix(suggestions)
	if prefix == "" {
		prefix = parsed.current
	}
	if len(suggestions) == 1 {
		prefix = suggestions[0]
		if !strings.HasSuffix(prefix, " ") {
			prefix += " "
		}
	}
	if prefix == parsed.current {
		s.printSuggestions(suggestions)
		return line, pos, true
	}

	newLine := line[:parsed.replaceStart] + prefix + line[pos:]
	return newLine, parsed.replaceStart + len(prefix), true
}

func (s *interactiveShell) printSuggestions(suggestions []string) {
	if len(suggestions) == 0 || s.writer == nil {
		return
	}
	_, _ = io.WriteString(s.writer, "\n")
	_, _ = io.WriteString(s.writer, formatSuggestions(suggestions))
	_, _ = io.WriteString(s.writer, "\n")
}

func completionCandidates(root *cobra.Command, parsed shellParseResult) []string {
	args := append([]string(nil), parsed.args...)
	if len(args) == 0 {
		return matchingStrings(commandNames(root), parsed.current)
	}
	if len(args) == 1 && !parsed.trailingSpace && !strings.HasPrefix(parsed.current, "-") && findSubcommand(root, args[0]) == nil {
		return matchingStrings(commandNames(root), parsed.current)
	}

	cmd := findSubcommand(root, args[0])
	if cmd == nil {
		return matchingStrings(commandNames(root), parsed.current)
	}
	if values := flagValueCandidates(cmd, args, parsed.current); len(values) > 0 {
		return values
	}
	if parsed.current == "" || strings.HasPrefix(parsed.current, "-") {
		return matchingStrings(flagNames(root, cmd), parsed.current)
	}
	return nil
}

func commandNames(root *cobra.Command) []string {
	seen := make(map[string]bool)
	var names []string
	add := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	for _, cmd := range root.Commands() {
		if cmd.Hidden {
			continue
		}
		name := cmd.Name()
		if name == "shell" {
			continue
		}
		add(name)
	}
	for _, name := range []string{"help", "tools", "servers", "reload", "clear", "quit", "exit", "?"} {
		add(name)
	}
	sort.Strings(names)
	return names
}

func findSubcommand(root *cobra.Command, name string) *cobra.Command {
	for _, cmd := range root.Commands() {
		if cmd.Hidden {
			continue
		}
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func flagValueCandidates(cmd *cobra.Command, args []string, current string) []string {
	if len(args) == 0 {
		return nil
	}
	last := args[len(args)-1]
	if strings.HasPrefix(last, "-") {
		flag := lookupFlag(cmd, last)
		if flag == nil || !flagRequiresValue(flag) {
			return nil
		}
		return matchingStrings(flagCompletionValues(flag), current)
	}
	if len(args) < 2 {
		return nil
	}
	prev := args[len(args)-2]
	if !strings.HasPrefix(prev, "-") {
		return nil
	}
	flag := lookupFlag(cmd, prev)
	if flag == nil || !flagRequiresValue(flag) {
		return nil
	}
	return matchingStrings(flagCompletionValues(flag), current)
}

func lookupFlag(cmd *cobra.Command, token string) *pflag.Flag {
	name := strings.TrimLeft(token, "-")
	if flag := cmd.Flags().Lookup(name); flag != nil {
		return flag
	}
	return cmd.InheritedFlags().Lookup(name)
}

func flagRequiresValue(flag *pflag.Flag) bool {
	if flag == nil {
		return false
	}
	return flag.Value.Type() != "bool"
}

func flagCompletionValues(flag *pflag.Flag) []string {
	if flag == nil || flag.Annotations == nil {
		return nil
	}
	return flag.Annotations[completionAnnotation]
}

func flagNames(root, cmd *cobra.Command) []string {
	seen := make(map[string]bool)
	var names []string
	add := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(flag *pflag.Flag) {
			if flag.Hidden {
				return
			}
			name := "--" + flag.Name
			if seen[name] {
				return
			}
			seen[name] = true
			names = append(names, name)
		})
	}
	add(root.PersistentFlags())
	add(cmd.Flags())
	sort.Strings(names)
	return names
}

func matchingStrings(candidates []string, prefix string) []string {
	if prefix == "" {
		return append([]string(nil), candidates...)
	}
	var matches []string
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, prefix) {
			matches = append(matches, candidate)
		}
	}
	return matches
}

func suggestCommands(token string, candidates []string) []string {
	if token == "" {
		return nil
	}
	type suggestion struct {
		name  string
		score int
	}
	var suggestions []suggestion
	for _, candidate := range candidates {
		score, ok := commandScore(token, candidate)
		if !ok {
			continue
		}
		suggestions = append(suggestions, suggestion{name: candidate, score: score})
	}
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].score == suggestions[j].score {
			return suggestions[i].name < suggestions[j].name
		}
		return suggestions[i].score < suggestions[j].score
	})
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}
	out := make([]string, 0, len(suggestions))
	for _, suggestion := range suggestions {
		out = append(out, suggestion.name)
	}
	return out
}

func commandScore(token, candidate string) (int, bool) {
	switch {
	case token == candidate:
		return 0, true
	case strings.HasPrefix(candidate, token):
		return 1 + len(candidate) - len(token), true
	case strings.HasPrefix(token, candidate):
		return 2 + len(token) - len(candidate), true
	case strings.Contains(candidate, token):
		return 4 + len(candidate) - len(token), true
	default:
		d := editDistance(token, candidate)
		if d > 3 {
			return 0, false
		}
		return 8 + d, true
	}
}

func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return len(b)
	}
	if b == "" {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr := make([]int, len(b)+1)
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min3(
				prev[j]+1,
				curr[j-1]+1,
				prev[j-1]+cost,
			)
		}
		prev = curr
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a > b {
		a = b
	}
	if a > c {
		a = c
	}
	return a
}

func longestCommonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	prefix := values[0]
	for _, value := range values[1:] {
		for !strings.HasPrefix(value, prefix) {
			if prefix == "" {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func formatSuggestions(values []string) string {
	if len(values) == 0 {
		return ""
	}
	var b strings.Builder
	for i, value := range values {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(value)
	}
	return b.String()
}

func splitShellLine(line string) ([]string, error) {
	parsed, err := parseShellLine(line)
	if err != nil {
		return nil, err
	}
	return parsed.words, nil
}

func parseShellLine(line string) (shellParseResult, error) {
	var (
		args      []string
		buf       strings.Builder
		start     = -1
		quote     rune
		escape    bool
		inToken   bool
		lastSpace = true
	)
	for i, r := range line {
		switch {
		case escape:
			if !inToken {
				start = i
				inToken = true
			}
			buf.WriteRune(r)
			escape = false
			lastSpace = false
		case quote != 0:
			switch r {
			case '\\':
				escape = true
			case quote:
				quote = 0
			default:
				buf.WriteRune(r)
			}
			lastSpace = false
		case r == '\\':
			if !inToken {
				start = i
				inToken = true
			}
			escape = true
			lastSpace = false
		case r == '\'' || r == '"':
			if !inToken {
				start = i
				inToken = true
			}
			quote = r
			lastSpace = false
		case unicode.IsSpace(r):
			if inToken {
				args = append(args, buf.String())
				buf.Reset()
				inToken = false
				start = -1
			}
			lastSpace = true
		default:
			if !inToken {
				start = i
				inToken = true
			}
			buf.WriteRune(r)
			lastSpace = false
		}
	}
	if escape {
		buf.WriteRune('\\')
	}
	if quote != 0 {
		return shellParseResult{}, errors.New("unterminated quote")
	}

	result := shellParseResult{trailingSpace: lastSpace}
	if inToken {
		args = append(args, buf.String())
		result.words = append(result.words, args...)
		result.current = buf.String()
		result.replaceStart = start
		result.replaceEnd = len(line)
		result.args = append(result.args, args[:len(args)-1]...)
		return result, nil
	}
	result.args = args
	result.words = append(result.words, args...)
	result.replaceStart = len(line)
	result.replaceEnd = len(line)
	return result, nil
}
