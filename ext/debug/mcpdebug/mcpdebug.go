package mcpdebug

import (
	"context"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Instance holds all debug information associated with an MCP instance.
type Instance struct {
	StartTime     time.Time
	ServerAddress string

	serveMu              sync.Mutex
	debugAddress         string
	listenedDebugAddress string
	State                *State
}

// State holds debugging information related to MCP server/client state.
type State struct {
	mu      sync.Mutex
	servers []*Server
	clients []*Client
}

// Server represents a debug view of an MCP server instance.
type Server struct {
	ID           string
	Name         string
	Version      string
	StartTime    time.Time
	Capabilities ServerCapabilities
	Tools        []Tool
	Resources    []Resource
	Prompts      []Prompt
}

// Client represents a debug view of an MCP client instance.
type Client struct {
	ID           string
	StartTime    time.Time
	ServerInfo   Implementation
	Capabilities ServerCapabilities
	Initialized  bool
}

// Types mirrored from main package to avoid import cycles
type ServerCapabilities struct {
	Experimental map[string]any `json:"experimental,omitempty"`
	Tools        *struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"tools,omitempty"`
	Resources *struct {
		Subscribe   bool `json:"subscribe,omitempty"`
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"resources,omitempty"`
	Prompts *struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"prompts,omitempty"`
}

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"inputSchema,omitempty"`
}

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// NewInstance creates a new debug instance
func NewInstance() *Instance {
	return &Instance{
		StartTime: time.Now(),
		State:     &State{},
	}
}

// AddServer adds a server to the debug state.
func (s *State) AddServer(name, version string, capabilities ServerCapabilities, tools []Tool, resources []Resource, prompts []Prompt) {
	s.mu.Lock()
	defer s.mu.Unlock()

	debugServer := &Server{
		ID:           fmt.Sprintf("server-%d", len(s.servers)),
		Name:         name,
		Version:      version,
		StartTime:    time.Now(),
		Capabilities: capabilities,
		Tools:        tools,
		Resources:    resources,
		Prompts:      prompts,
	}

	s.servers = append(s.servers, debugServer)
}

// AddClient adds a client to the debug state.
func (s *State) AddClient(serverInfo Implementation, capabilities ServerCapabilities, initialized bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	debugClient := &Client{
		ID:           fmt.Sprintf("client-%d", len(s.clients)),
		StartTime:    time.Now(),
		ServerInfo:   serverInfo,
		Capabilities: capabilities,
		Initialized:  initialized,
	}

	s.clients = append(s.clients, debugClient)
}

// Servers returns the current list of debug servers.
func (s *State) Servers() []*Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	servers := make([]*Server, len(s.servers))
	copy(servers, s.servers)
	return servers
}

// Clients returns the current list of debug clients.
func (s *State) Clients() []*Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	clients := make([]*Client, len(s.clients))
	copy(clients, s.clients)
	return clients
}

// Serve starts and runs a debug server in the background on the given addr.
func (i *Instance) Serve(ctx context.Context, addr string) (string, error) {
	if addr == "" {
		return "", nil
	}
	i.serveMu.Lock()
	defer i.serveMu.Unlock()

	if i.listenedDebugAddress != "" {
		// Already serving. Return the bound address.
		return i.listenedDebugAddress, nil
	}

	i.debugAddress = addr
	listener, err := net.Listen("tcp", i.debugAddress)
	if err != nil {
		return "", err
	}
	i.listenedDebugAddress = listener.Addr().String()

	port := listener.Addr().(*net.TCPAddr).Port
	if strings.HasSuffix(i.debugAddress, ":0") {
		fmt.Printf("MCP debug server listening at http://localhost:%d\n", port)
	}

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", render(MainTmpl, func(*http.Request) any { return i }))
		mux.HandleFunc("/debug/", render(DebugPageTmpl, nil))
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		mux.HandleFunc("/servers/", render(ServersTmpl, i.getServers))
		mux.HandleFunc("/clients/", render(ClientsTmpl, i.getClients))
		mux.HandleFunc("/memory", render(MemoryTmpl, getMemory))

		// Internal debugging helpers.
		mux.HandleFunc("/gc", func(w http.ResponseWriter, r *http.Request) {
			runtime.GC()
			runtime.GC()
			runtime.GC()
			http.Redirect(w, r, "/memory", http.StatusTemporaryRedirect)
		})

		if err := http.Serve(listener, mux); err != nil {
			fmt.Printf("MCP debug server failed: %v\n", err)
			return
		}
	}()
	return i.listenedDebugAddress, nil
}

func (i *Instance) getServers(r *http.Request) any {
	return i.State.Servers()
}

func (i *Instance) getClients(r *http.Request) any {
	return i.State.Clients()
}

func getMemory(_ *http.Request) any {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

type dataFunc func(*http.Request) any

func render(tmpl *template.Template, fun dataFunc) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var data any
		if fun != nil {
			data = fun(r)
		}
		if err := tmpl.Execute(w, data); err != nil {
			fmt.Printf("Template execution error: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func commas(s string) string {
	for i := len(s); i > 3; {
		i -= 3
		s = s[:i] + "," + s[i:]
	}
	return s
}

func fuint64(v uint64) string {
	return commas(strconv.FormatUint(v, 10))
}

func fuint32(v uint32) string {
	return commas(strconv.FormatUint(uint64(v), 10))
}

var BaseTemplate = template.Must(template.New("").Parse(`
<html>
<head>
<title>{{template "title" .}}</title>
<style>
.profile-name{
	display:inline-block;
	width:6rem;
}
td.value {
	text-align: right;
}
body {
	font-family: sans-serif;
	font-size: 1rem;
	line-height: normal;
}
table {
	border-collapse: collapse;
}
th, td {
	border: 1px solid #ddd;
	padding: 8px;
}
th {
	background-color: #f2f2f2;
}
</style>
{{block "head" .}}{{end}}
</head>
<body>
<a href="/">Main</a>
<a href="/servers/">Servers</a>
<a href="/clients/">Clients</a>
<a href="/memory">Memory</a>
<a href="/debug/pprof">Profiling</a>
<hr>
<h1>{{template "title" .}}</h1>
{{block "body" .}}
Unknown page
{{end}}
</body>
</html>
`)).Funcs(template.FuncMap{
	"fuint64": fuint64,
	"fuint32": fuint32,
})

var MainTmpl = template.Must(template.Must(BaseTemplate.Clone()).Parse(`
{{define "title"}}MCP Debug Server{{end}}
{{define "body"}}
<h2>MCP Servers</h2>
<ul>{{range .State.Servers}}<li><a href="/servers/">{{.Name}} v{{.Version}}</a> ({{.ID}})</li>{{end}}</ul>
<h2>MCP Clients</h2>
<ul>{{range .State.Clients}}<li><a href="/clients/">{{.ID}}</a></li>{{end}}</ul>
<h2>System Information</h2>
<p>Started: {{.StartTime.Format "2006-01-02 15:04:05"}}</p>
{{if .ServerAddress}}<p>Server Address: {{.ServerAddress}}</p>{{end}}
{{end}}
`))

var DebugPageTmpl = template.Must(template.Must(BaseTemplate.Clone()).Parse(`
{{define "title"}}MCP Debug Pages{{end}}
{{define "body"}}
<a href="/debug/pprof">Profiling</a>
{{end}}
`))

var ServersTmpl = template.Must(template.Must(BaseTemplate.Clone()).Parse(`
{{define "title"}}MCP Servers{{end}}
{{define "body"}}
<table>
<tr><th>ID</th><th>Name</th><th>Version</th><th>Started</th><th>Tools</th><th>Resources</th><th>Prompts</th></tr>
{{range .}}
<tr>
<td>{{.ID}}</td>
<td>{{.Name}}</td>
<td>{{.Version}}</td>
<td>{{.StartTime.Format "15:04:05"}}</td>
<td>{{len .Tools}}</td>
<td>{{len .Resources}}</td>
<td>{{len .Prompts}}</td>
</tr>
{{end}}
</table>

{{range .}}
<h3>{{.Name}} ({{.ID}})</h3>
<h4>Capabilities</h4>
<ul>
{{if .Capabilities.Tools}}<li>Tools: {{if .Capabilities.Tools.ListChanged}}list_changed{{end}}</li>{{end}}
{{if .Capabilities.Resources}}<li>Resources: {{if .Capabilities.Resources.Subscribe}}subscribe{{end}} {{if .Capabilities.Resources.ListChanged}}list_changed{{end}}</li>{{end}}
{{if .Capabilities.Prompts}}<li>Prompts: {{if .Capabilities.Prompts.ListChanged}}list_changed{{end}}</li>{{end}}
</ul>

{{if .Tools}}
<h4>Tools</h4>
<ul>{{range .Tools}}<li><strong>{{.Name}}</strong>: {{.Description}}</li>{{end}}</ul>
{{end}}

{{if .Resources}}
<h4>Resources</h4>
<ul>{{range .Resources}}<li><strong>{{.Name}}</strong>: {{.Description}} ({{.URI}})</li>{{end}}</ul>
{{end}}

{{if .Prompts}}
<h4>Prompts</h4>
<ul>{{range .Prompts}}<li><strong>{{.Name}}</strong>: {{.Description}}</li>{{end}}</ul>
{{end}}
{{end}}
{{end}}
`))

var ClientsTmpl = template.Must(template.Must(BaseTemplate.Clone()).Parse(`
{{define "title"}}MCP Clients{{end}}
{{define "body"}}
<table>
<tr><th>ID</th><th>Started</th><th>Initialized</th><th>Server</th></tr>
{{range .}}
<tr>
<td>{{.ID}}</td>
<td>{{.StartTime.Format "15:04:05"}}</td>
<td>{{.Initialized}}</td>
<td>{{.ServerInfo.Name}} v{{.ServerInfo.Version}}</td>
</tr>
{{end}}
</table>

{{range .}}
<h3>{{.ID}}</h3>
<h4>Server Information</h4>
<p>Name: {{.ServerInfo.Name}}</p>
<p>Version: {{.ServerInfo.Version}}</p>
<p>Initialized: {{.Initialized}}</p>

<h4>Server Capabilities</h4>
<ul>
{{if .Capabilities.Tools}}<li>Tools: {{if .Capabilities.Tools.ListChanged}}list_changed{{end}}</li>{{end}}
{{if .Capabilities.Resources}}<li>Resources: {{if .Capabilities.Resources.Subscribe}}subscribe{{end}} {{if .Capabilities.Resources.ListChanged}}list_changed{{end}}</li>{{end}}
{{if .Capabilities.Prompts}}<li>Prompts: {{if .Capabilities.Prompts.ListChanged}}list_changed{{end}}</li>{{end}}
</ul>
{{end}}
{{end}}
`))

var MemoryTmpl = template.Must(template.Must(BaseTemplate.Clone()).Parse(`
{{define "title"}}MCP Memory Usage{{end}}
{{define "head"}}<meta http-equiv="refresh" content="5">{{end}}
{{define "body"}}
<form action="/gc"><input type="submit" value="Run garbage collector"/></form>
<h2>Stats</h2>
<table>
<tr><td class="label">Allocated bytes</td><td class="value">{{fuint64 .HeapAlloc}}</td></tr>
<tr><td class="label">Total allocated bytes</td><td class="value">{{fuint64 .TotalAlloc}}</td></tr>
<tr><td class="label">System bytes</td><td class="value">{{fuint64 .Sys}}</td></tr>
<tr><td class="label">Heap system bytes</td><td class="value">{{fuint64 .HeapSys}}</td></tr>
<tr><td class="label">Malloc calls</td><td class="value">{{fuint64 .Mallocs}}</td></tr>
<tr><td class="label">Frees</td><td class="value">{{fuint64 .Frees}}</td></tr>
<tr><td class="label">Idle heap bytes</td><td class="value">{{fuint64 .HeapIdle}}</td></tr>
<tr><td class="label">In use bytes</td><td class="value">{{fuint64 .HeapInuse}}</td></tr>
<tr><td class="label">Released to system bytes</td><td class="value">{{fuint64 .HeapReleased}}</td></tr>
<tr><td class="label">Heap object count</td><td class="value">{{fuint64 .HeapObjects}}</td></tr>
<tr><td class="label">Stack in use bytes</td><td class="value">{{fuint64 .StackInuse}}</td></tr>
<tr><td class="label">Stack from system bytes</td><td class="value">{{fuint64 .StackSys}}</td></tr>
<tr><td class="label">Bucket hash bytes</td><td class="value">{{fuint64 .BuckHashSys}}</td></tr>
<tr><td class="label">GC metadata bytes</td><td class="value">{{fuint64 .GCSys}}</td></tr>
<tr><td class="label">Off heap bytes</td><td class="value">{{fuint64 .OtherSys}}</td></tr>
</table>
<h2>By size</h2>
<table>
<tr><th>Size</th><th>Mallocs</th><th>Frees</th></tr>
{{range .BySize}}<tr><td class="value">{{fuint32 .Size}}</td><td class="value">{{fuint64 .Mallocs}}</td><td class="value">{{fuint64 .Frees}}</td></tr>{{end}}
</table>
{{end}}
`))
