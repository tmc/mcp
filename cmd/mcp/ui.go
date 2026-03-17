package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
	"github.com/tmc/mcp/internal/mcpcli"
)

type uiSnapshot struct {
	roots     []string
	tools     []string
	resources []string
	prompts   []string
	tasks     []string
	err       error
}

type uiSnapshotMsg uiSnapshot
type uiEventMsg string
type uiTickMsg time.Time

type uiModel struct {
	app         *app
	session     *mcpcli.Session
	events      <-chan mcpcli.Event
	unsubscribe func()
	focus       int
	width       int
	height      int
	roots       []string
	tools       []string
	resources   []string
	prompts     []string
	tasks       []string
	logs        []string
	err         error
}

func newUICommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:     "ui",
		Aliases: []string{"top"},
		Short:   "Open an interactive MCP dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := a.session(context.Background())
			if err != nil {
				return err
			}
			events, unsubscribe := sess.Subscribe(64)
			defer unsubscribe()
			model := uiModel{
				app:         a,
				session:     sess,
				events:      events,
				unsubscribe: unsubscribe,
			}
			_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
			return err
		},
	}
}

func (m uiModel) Init() tea.Cmd {
	return tea.Batch(m.refreshCmd(), m.waitEventCmd(), tickCmd())
}

func (m uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.focus = (m.focus + 1) % 6
		case "shift+tab", "left", "h":
			m.focus = (m.focus + 5) % 6
		case "r":
			return m, m.refreshCmd()
		}
	case uiSnapshotMsg:
		m.roots = msg.roots
		m.tools = msg.tools
		m.resources = msg.resources
		m.prompts = msg.prompts
		m.tasks = msg.tasks
		m.err = msg.err
	case uiEventMsg:
		if msg != "" {
			m.logs = append([]string{string(msg)}, m.logs...)
			if len(m.logs) > 20 {
				m.logs = m.logs[:20]
			}
		}
		return m, m.waitEventCmd()
	case uiTickMsg:
		return m, tea.Batch(m.refreshCmd(), tickCmd())
	}
	return m, nil
}

func (m uiModel) View() string {
	if m.width == 0 {
		return "loading..."
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("mcp ui")
	help := lipgloss.NewStyle().Faint(true).Render("tab: cycle  r: refresh  q: quit")
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", help)
	if m.err != nil {
		header += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.err.Error())
	}

	leftWidth := max(30, m.width/3)
	rightWidth := m.width - leftWidth - 3
	sectionHeight := max(6, (m.height-4)/3)

	left := lipgloss.JoinVertical(lipgloss.Left,
		m.section("Roots", m.roots, 0, leftWidth, sectionHeight),
		m.section("Tasks", m.tasks, 1, leftWidth, sectionHeight),
		m.section("Logs", m.logs, 2, leftWidth, sectionHeight),
	)
	right := lipgloss.JoinVertical(lipgloss.Left,
		m.section("Tools", m.tools, 3, rightWidth, sectionHeight),
		m.section("Resources", m.resources, 4, rightWidth, sectionHeight),
		m.section("Prompts", m.prompts, 5, rightWidth, sectionHeight),
	)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, "   ", right)
	return header + "\n\n" + body
}

func (m uiModel) section(title string, lines []string, focus, width, height int) string {
	border := lipgloss.NormalBorder()
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(border).
		Padding(0, 1)
	if m.focus == focus {
		style = style.BorderForeground(lipgloss.Color("12"))
	} else {
		style = style.BorderForeground(lipgloss.Color("8"))
	}
	if len(lines) == 0 {
		lines = []string{"(empty)"}
	}
	if len(lines) > height-2 {
		lines = lines[:height-2]
	}
	return style.Render(title + "\n" + strings.Join(lines, "\n"))
}

func (m uiModel) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.cfg.Timeout)
		defer cancel()
		snapshot := uiSnapshot{
			roots:     mustRoots(m.app),
			tools:     mustTools(ctx, m.session),
			resources: mustResources(ctx, m.session),
			prompts:   mustPrompts(ctx, m.session),
			tasks:     mustTasks(ctx, m.session),
		}
		return uiSnapshotMsg(snapshot)
	}
}

func (m uiModel) waitEventCmd() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events
		if !ok {
			return uiEventMsg("")
		}
		if len(event.Params) == 0 {
			return uiEventMsg(event.Method)
		}
		return uiEventMsg(fmt.Sprintf("%s %s", event.Method, string(event.Params)))
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return uiTickMsg(t) })
}

func mustRoots(a *app) []string {
	store, err := a.stateStore()
	if err != nil {
		return nil
	}
	roots, err := store.List()
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		label := root.URI
		if root.Name != "" {
			label = root.Name + " -> " + root.URI
		}
		out = append(out, label)
	}
	return out
}

func mustTools(ctx context.Context, sess *mcpcli.Session) []string {
	tools, err := sess.ListToolsAll(ctx)
	if err != nil {
		return []string{err.Error()}
	}
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, cobraName(tool.Name))
	}
	return out
}

func mustResources(ctx context.Context, sess *mcpcli.Session) []string {
	resources, err := sess.ListResourcesAll(ctx)
	if err != nil {
		return []string{err.Error()}
	}
	out := make([]string, 0, len(resources))
	for _, resource := range resources {
		out = append(out, resource.URI)
	}
	return out
}

func mustPrompts(ctx context.Context, sess *mcpcli.Session) []string {
	prompts, err := sess.ListPromptsAll(ctx)
	if err != nil {
		return []string{err.Error()}
	}
	out := make([]string, 0, len(prompts))
	for _, prompt := range prompts {
		out = append(out, prompt.Name)
	}
	return out
}

func mustTasks(ctx context.Context, sess *mcpcli.Session) []string {
	if !sess.Supports("tasks") {
		return []string{"(unsupported)"}
	}
	var tasks mcp.ListTasksResult
	if err := sess.CallRaw(ctx, string(mcp.MethodTasksList), mcp.ListTasksRequest{}, &tasks); err != nil {
		return []string{err.Error()}
	}
	out := make([]string, 0, len(tasks.Tasks))
	for _, task := range tasks.Tasks {
		out = append(out, fmt.Sprintf("%s %s", task.TaskID, task.Status))
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func _json(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}
