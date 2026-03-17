package mcpspy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const specDocumentVersion = "0.1.0"

// SpecOptions configures a SpecTracker.
type SpecOptions struct {
	Path string
	Name string
}

// SpecDocument is the live .mcpspec document built from observed traffic.
type SpecDocument struct {
	SpecVersion string         `json:"specVersion"`
	Server      SpecServer     `json:"server"`
	Tools       []SpecTool     `json:"tools,omitempty"`
	Resources   []SpecResource `json:"resources,omitempty"`
	Prompts     []SpecPrompt   `json:"prompts,omitempty"`
}

// SpecServer describes the observed server.
type SpecServer struct {
	Name        string `json:"name,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}

// SpecTool describes an observed tool.
type SpecTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
	ReturnType  json.RawMessage `json:"returnType,omitempty"`
}

// SpecResource describes an observed resource.
type SpecResource struct {
	URI         string         `json:"uri"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	MimeType    string         `json:"mimeType,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// SpecPrompt describes an observed prompt.
type SpecPrompt struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Arguments   []SpecPromptArgument `json:"arguments,omitempty"`
}

// SpecPromptArgument describes an observed prompt argument.
type SpecPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// SpecSnapshot is the current state exposed to the UI.
type SpecSnapshot struct {
	Path    string       `json:"path,omitempty"`
	Size    int64        `json:"size,omitempty"`
	ModTime time.Time    `json:"mod_time,omitempty"`
	Spec    SpecDocument `json:"spec"`
	Text    string       `json:"text,omitempty"`
}

// SpecTracker builds a live spec from recorder events.
type SpecTracker struct {
	path string

	mu        sync.Mutex
	server    SpecServer
	tools     map[string]*toolState
	resources map[string]*resourceState
	prompts   map[string]*promptState
	pending   map[string]pendingRequest
	lastJSON  []byte
	lastMod   time.Time
	subs      map[int]chan SpecSnapshot
	nextSub   int
	cancel    func()
	done      chan struct{}
}

type pendingRequest struct {
	Method      string
	ToolName    string
	ResourceURI string
	PromptName  string
}

type toolState struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	InferredIn  *schemaNode
	ReturnType  *schemaNode
}

type resourceState struct {
	URI         string
	Name        string
	Description string
	MimeType    string
	Annotations map[string]any
}

type promptState struct {
	Name        string
	Description string
	Arguments   map[string]*promptArgumentState
}

type promptArgumentState struct {
	Name        string
	Description string
	Required    bool
}

type schemaNode struct {
	observations int
	types        map[string]bool
	properties   map[string]*schemaNode
	propSeen     map[string]int
	items        *schemaNode
}

type rpcMessage struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type initializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
	Instructions string `json:"instructions"`
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type toolsListResult struct {
	Tools []struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		InputSchema json.RawMessage `json:"inputSchema"`
	} `json:"tools"`
}

type resourcesListResult struct {
	Resources []struct {
		URI         string         `json:"uri"`
		Name        string         `json:"name"`
		Description string         `json:"description"`
		MimeType    string         `json:"mimeType"`
		Annotations map[string]any `json:"annotations"`
	} `json:"resources"`
}

type resourcesReadParams struct {
	URI string `json:"uri"`
}

type resourcesReadResult struct {
	Contents []struct {
		URI      string `json:"uri"`
		MimeType string `json:"mimeType"`
	} `json:"contents"`
}

type promptsListResult struct {
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

type promptsGetParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// NewSpecTracker starts a live spec tracker fed by recorder events.
func NewSpecTracker(recorder *Recorder, opts SpecOptions) *SpecTracker {
	s := &SpecTracker{
		path:      opts.Path,
		server:    SpecServer{Name: opts.Name},
		tools:     make(map[string]*toolState),
		resources: make(map[string]*resourceState),
		prompts:   make(map[string]*promptState),
		pending:   make(map[string]pendingRequest),
		subs:      make(map[int]chan SpecSnapshot),
		done:      make(chan struct{}),
	}
	ch, cancel := recorder.Subscribe()
	s.cancel = cancel
	go s.run(ch)
	return s
}

// Close stops the tracker.
func (s *SpecTracker) Close() {
	if s == nil {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	<-s.done
}

// Snapshot returns the current spec snapshot.
func (s *SpecTracker) Snapshot() SpecSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

// Subscribe returns a live stream of spec snapshots.
func (s *SpecTracker) Subscribe() (<-chan SpecSnapshot, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextSub
	s.nextSub++
	ch := make(chan SpecSnapshot, 16)
	s.subs[id] = ch
	cancel := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if ch, ok := s.subs[id]; ok {
			delete(s.subs, id)
			close(ch)
		}
	}
	return ch, cancel
}

func (s *SpecTracker) run(events <-chan Event) {
	defer close(s.done)
	for ev := range events {
		s.observe(ev)
	}
}

func (s *SpecTracker) observe(ev Event) {
	if len(ev.Parsed) == 0 {
		return
	}

	var msg rpcMessage
	if err := json.Unmarshal(ev.Parsed, &msg); err != nil {
		return
	}

	s.mu.Lock()
	changed := false
	if msg.Method != "" {
		changed = s.observeRequestLocked(msg) || changed
	} else if len(msg.ID) > 0 {
		changed = s.observeResponseLocked(msg) || changed
	}
	if !changed {
		s.mu.Unlock()
		return
	}

	if err := s.persistLocked(); err != nil {
		s.mu.Unlock()
		return
	}
	snapshot := s.snapshotLocked()
	subs := make([]chan SpecSnapshot, 0, len(s.subs))
	for _, ch := range s.subs {
		subs = append(subs, ch)
	}
	s.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- snapshot:
		default:
		}
	}
}

func (s *SpecTracker) observeRequestLocked(msg rpcMessage) bool {
	changed := false
	id := string(bytes.TrimSpace(msg.ID))
	if id != "" {
		s.pending[id] = pendingRequest{Method: msg.Method}
	}

	switch msg.Method {
	case "initialize":
		var params initializeParams
		if err := json.Unmarshal(msg.Params, &params); err == nil && params.ProtocolVersion != "" {
			// Keep the document version stable; protocol version is only observed for context today.
		}
	case "tools/call":
		var params toolsCallParams
		if err := json.Unmarshal(msg.Params, &params); err != nil || params.Name == "" {
			break
		}
		if id != "" {
			req := s.pending[id]
			req.ToolName = params.Name
			s.pending[id] = req
		}
		tool := s.ensureToolLocked(params.Name)
		var args any
		if decodeJSON(msg.Params, &args) {
			if object, ok := args.(map[string]any); ok {
				if value, ok := object["arguments"]; ok {
					if tool.InferredIn == nil {
						tool.InferredIn = newSchemaNode()
					}
					tool.InferredIn.observe(value)
					changed = true
				}
			}
		}
	case "resources/read":
		var params resourcesReadParams
		if err := json.Unmarshal(msg.Params, &params); err != nil || params.URI == "" {
			break
		}
		if id != "" {
			req := s.pending[id]
			req.ResourceURI = params.URI
			s.pending[id] = req
		}
		resource := s.ensureResourceLocked(params.URI)
		if resource.Name == "" {
			resource.Name = params.URI
			changed = true
		}
	case "prompts/get":
		var params promptsGetParams
		if err := json.Unmarshal(msg.Params, &params); err != nil || params.Name == "" {
			break
		}
		if id != "" {
			req := s.pending[id]
			req.PromptName = params.Name
			s.pending[id] = req
		}
		prompt := s.ensurePromptLocked(params.Name)
		changed = s.observePromptArgumentsLocked(prompt, params.Arguments) || changed
	}
	return changed
}

func (s *SpecTracker) observeResponseLocked(msg rpcMessage) bool {
	id := string(bytes.TrimSpace(msg.ID))
	req, ok := s.pending[id]
	if !ok {
		return false
	}
	delete(s.pending, id)

	changed := false
	switch req.Method {
	case "initialize":
		var result initializeResult
		if err := json.Unmarshal(msg.Result, &result); err != nil {
			break
		}
		if result.ServerInfo.Name != "" && s.server.Name != result.ServerInfo.Name {
			s.server.Name = result.ServerInfo.Name
			changed = true
		}
		if result.ServerInfo.Version != "" && s.server.Version != result.ServerInfo.Version {
			s.server.Version = result.ServerInfo.Version
			changed = true
		}
		if result.Instructions != "" && s.server.Description == "" {
			s.server.Description = result.Instructions
			changed = true
		}
	case "tools/list":
		var result toolsListResult
		if err := json.Unmarshal(msg.Result, &result); err != nil {
			break
		}
		for _, entry := range result.Tools {
			if entry.Name == "" {
				continue
			}
			tool := s.ensureToolLocked(entry.Name)
			if entry.Description != "" && tool.Description == "" {
				tool.Description = entry.Description
				changed = true
			}
			if len(entry.InputSchema) > 0 && !bytes.Equal(bytes.TrimSpace(tool.InputSchema), bytes.TrimSpace(entry.InputSchema)) {
				tool.InputSchema = append(tool.InputSchema[:0], entry.InputSchema...)
				changed = true
			}
		}
	case "tools/call":
		if req.ToolName == "" || len(msg.Result) == 0 {
			break
		}
		tool := s.ensureToolLocked(req.ToolName)
		if tool.ReturnType == nil {
			tool.ReturnType = newSchemaNode()
		}
		var result any
		if decodeJSON(msg.Result, &result) {
			tool.ReturnType.observe(result)
			changed = true
		}
	case "resources/list":
		var result resourcesListResult
		if err := json.Unmarshal(msg.Result, &result); err != nil {
			break
		}
		for _, entry := range result.Resources {
			if entry.URI == "" {
				continue
			}
			resource := s.ensureResourceLocked(entry.URI)
			if entry.Name != "" && resource.Name == "" {
				resource.Name = entry.Name
				changed = true
			}
			if entry.Description != "" && resource.Description == "" {
				resource.Description = entry.Description
				changed = true
			}
			if entry.MimeType != "" && resource.MimeType == "" {
				resource.MimeType = entry.MimeType
				changed = true
			}
			if len(entry.Annotations) > 0 && len(resource.Annotations) == 0 {
				resource.Annotations = cloneMap(entry.Annotations)
				changed = true
			}
		}
	case "resources/read":
		if req.ResourceURI == "" {
			break
		}
		var result resourcesReadResult
		if err := json.Unmarshal(msg.Result, &result); err != nil {
			break
		}
		resource := s.ensureResourceLocked(req.ResourceURI)
		for _, content := range result.Contents {
			if content.URI != "" && content.URI != req.ResourceURI {
				continue
			}
			if content.MimeType != "" && resource.MimeType == "" {
				resource.MimeType = content.MimeType
				changed = true
			}
		}
	case "prompts/list":
		var result promptsListResult
		if err := json.Unmarshal(msg.Result, &result); err != nil {
			break
		}
		for _, entry := range result.Prompts {
			if entry.Name == "" {
				continue
			}
			prompt := s.ensurePromptLocked(entry.Name)
			if entry.Description != "" && prompt.Description == "" {
				prompt.Description = entry.Description
				changed = true
			}
			for _, arg := range entry.Arguments {
				state := prompt.ensureArgument(arg.Name)
				if arg.Description != "" && state.Description == "" {
					state.Description = arg.Description
					changed = true
				}
				if arg.Required && !state.Required {
					state.Required = true
					changed = true
				}
			}
		}
	case "prompts/get":
		if req.PromptName == "" {
			break
		}
		// The prompt arguments were already captured from the request.
		_ = req.PromptName
	}
	return changed
}

func (s *SpecTracker) persistLocked() error {
	document := s.documentLocked()
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal spec: %w", err)
	}
	if bytes.Equal(data, s.lastJSON) {
		return nil
	}
	s.lastJSON = append(s.lastJSON[:0], data...)
	s.lastMod = time.Now()

	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("create spec dir: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(append([]byte(nil), data...), '\n'), 0644); err != nil {
		return fmt.Errorf("write spec: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename spec: %w", err)
	}
	info, err := os.Stat(s.path)
	if err == nil {
		s.lastMod = info.ModTime()
	}
	return nil
}

func (s *SpecTracker) snapshotLocked() SpecSnapshot {
	document := s.documentLocked()
	snapshot := SpecSnapshot{
		Path: s.path,
		Spec: document,
	}
	if len(s.lastJSON) > 0 {
		snapshot.Text = string(s.lastJSON)
	}
	if s.path == "" {
		return snapshot
	}
	info, err := os.Stat(s.path)
	if err == nil {
		snapshot.Size = info.Size()
		snapshot.ModTime = info.ModTime()
		return snapshot
	}
	if len(s.lastJSON) > 0 {
		snapshot.Size = int64(len(s.lastJSON) + 1)
		snapshot.ModTime = s.lastMod
	}
	return snapshot
}

func (s *SpecTracker) documentLocked() SpecDocument {
	document := SpecDocument{
		SpecVersion: specDocumentVersion,
		Server:      s.server,
	}

	toolNames := make([]string, 0, len(s.tools))
	for name := range s.tools {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)
	for _, name := range toolNames {
		state := s.tools[name]
		tool := SpecTool{
			Name:        state.Name,
			Description: state.Description,
		}
		if len(state.InputSchema) > 0 {
			tool.InputSchema = append(json.RawMessage(nil), state.InputSchema...)
		} else if state.InferredIn != nil {
			tool.InputSchema = state.InferredIn.marshal()
		}
		if state.ReturnType != nil {
			tool.ReturnType = state.ReturnType.marshal()
		}
		document.Tools = append(document.Tools, tool)
	}

	resourceURIs := make([]string, 0, len(s.resources))
	for uri := range s.resources {
		resourceURIs = append(resourceURIs, uri)
	}
	sort.Strings(resourceURIs)
	for _, uri := range resourceURIs {
		state := s.resources[uri]
		resource := SpecResource{
			URI:         state.URI,
			Name:        state.Name,
			Description: state.Description,
			MimeType:    state.MimeType,
		}
		if len(state.Annotations) > 0 {
			resource.Annotations = cloneMap(state.Annotations)
		}
		document.Resources = append(document.Resources, resource)
	}

	promptNames := make([]string, 0, len(s.prompts))
	for name := range s.prompts {
		promptNames = append(promptNames, name)
	}
	sort.Strings(promptNames)
	for _, name := range promptNames {
		state := s.prompts[name]
		prompt := SpecPrompt{
			Name:        state.Name,
			Description: state.Description,
		}
		argNames := make([]string, 0, len(state.Arguments))
		for arg := range state.Arguments {
			argNames = append(argNames, arg)
		}
		sort.Strings(argNames)
		for _, argName := range argNames {
			arg := state.Arguments[argName]
			prompt.Arguments = append(prompt.Arguments, SpecPromptArgument{
				Name:        arg.Name,
				Description: arg.Description,
				Required:    arg.Required,
			})
		}
		document.Prompts = append(document.Prompts, prompt)
	}

	return document
}

func (s *SpecTracker) ensureToolLocked(name string) *toolState {
	tool := s.tools[name]
	if tool == nil {
		tool = &toolState{Name: name}
		s.tools[name] = tool
	}
	return tool
}

func (s *SpecTracker) ensureResourceLocked(uri string) *resourceState {
	resource := s.resources[uri]
	if resource == nil {
		resource = &resourceState{URI: uri}
		s.resources[uri] = resource
	}
	return resource
}

func (s *SpecTracker) ensurePromptLocked(name string) *promptState {
	prompt := s.prompts[name]
	if prompt == nil {
		prompt = &promptState{Name: name, Arguments: make(map[string]*promptArgumentState)}
		s.prompts[name] = prompt
	}
	return prompt
}

func (s *SpecTracker) observePromptArgumentsLocked(prompt *promptState, raw json.RawMessage) bool {
	var data any
	if !decodeJSON(raw, &data) {
		return false
	}
	object, ok := data.(map[string]any)
	if !ok {
		return false
	}
	changed := false
	for name := range object {
		arg := prompt.ensureArgument(name)
		if !arg.Required {
			arg.Required = true
			changed = true
		}
	}
	return changed
}

func (p *promptState) ensureArgument(name string) *promptArgumentState {
	arg := p.Arguments[name]
	if arg == nil {
		arg = &promptArgumentState{Name: name}
		p.Arguments[name] = arg
	}
	return arg
}

func newSchemaNode() *schemaNode {
	return &schemaNode{
		types:      make(map[string]bool),
		properties: make(map[string]*schemaNode),
		propSeen:   make(map[string]int),
	}
}

func (n *schemaNode) observe(value any) {
	n.observations++
	switch v := value.(type) {
	case nil:
		n.types["null"] = true
	case bool:
		n.types["boolean"] = true
	case string:
		n.types["string"] = true
	case float64:
		if math.Trunc(v) == v {
			n.types["integer"] = true
		} else {
			n.types["number"] = true
		}
	case []any:
		n.types["array"] = true
		if len(v) == 0 {
			return
		}
		if n.items == nil {
			n.items = newSchemaNode()
		}
		for _, item := range v {
			n.items.observe(item)
		}
	case map[string]any:
		n.types["object"] = true
		for key, item := range v {
			n.propSeen[key]++
			child := n.properties[key]
			if child == nil {
				child = newSchemaNode()
				n.properties[key] = child
			}
			child.observe(item)
		}
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return
		}
		var decoded any
		if decodeJSON(raw, &decoded) {
			n.observe(decoded)
		}
	}
}

func (n *schemaNode) merge(other *schemaNode) {
	if other == nil {
		return
	}
	n.observations += other.observations
	for typ := range other.types {
		n.types[typ] = true
	}
	for key, count := range other.propSeen {
		n.propSeen[key] += count
	}
	for key, child := range other.properties {
		dst := n.properties[key]
		if dst == nil {
			dst = newSchemaNode()
			n.properties[key] = dst
		}
		dst.merge(child)
	}
	if other.items != nil {
		if n.items == nil {
			n.items = newSchemaNode()
		}
		n.items.merge(other.items)
	}
}

func (n *schemaNode) marshal() json.RawMessage {
	value := n.value()
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return data
}

func (n *schemaNode) value() any {
	out := make(map[string]any)
	if types := schemaTypes(n.types); len(types) > 0 {
		if len(types) == 1 {
			out["type"] = types[0]
		} else {
			out["type"] = types
		}
	}
	if n.types["object"] {
		keys := make([]string, 0, len(n.properties))
		for key := range n.properties {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) > 0 {
			props := make(map[string]any, len(keys))
			required := make([]string, 0, len(keys))
			for _, key := range keys {
				props[key] = n.properties[key].value()
				if n.propSeen[key] == n.observations {
					required = append(required, key)
				}
			}
			out["properties"] = props
			if len(required) > 0 {
				out["required"] = required
			}
		}
	}
	if n.types["array"] && n.items != nil {
		out["items"] = n.items.value()
	}
	return out
}

func schemaTypes(types map[string]bool) []string {
	if len(types) == 0 {
		return nil
	}
	order := []string{"array", "boolean", "integer", "null", "number", "object", "string"}
	out := make([]string, 0, len(types))
	for _, typ := range order {
		if typ == "integer" && types["number"] {
			continue
		}
		if types[typ] {
			out = append(out, typ)
		}
	}
	return out
}

func decodeJSON(raw []byte, dst any) bool {
	if len(bytes.TrimSpace(raw)) == 0 {
		return false
	}
	return json.Unmarshal(raw, dst) == nil
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	keys := make([]string, 0, len(src))
	for key := range src {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		dst[key] = src[key]
	}
	return dst
}

// SpecFilenameFor returns the default .mcpspec path for a wrapped tool.
func SpecFilenameFor(tool string) string {
	name := sanitizeSpecName(tool)
	dir, err := os.UserHomeDir()
	if err == nil && dir != "" {
		return filepath.Join(dir, ".mcpspy", "specs", name+".mcpspec")
	}
	return filepath.Join(".mcpspy", "specs", name+".mcpspec")
}

func sanitizeSpecName(tool string) string {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return "stdin"
	}
	tool = filepath.Base(tool)
	if ext := filepath.Ext(tool); ext != "" {
		tool = strings.TrimSuffix(tool, ext)
	}
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		" ", "-",
	)
	tool = replacer.Replace(tool)
	tool = strings.Trim(tool, ".-")
	if tool == "" {
		return "stdin"
	}
	return tool
}
