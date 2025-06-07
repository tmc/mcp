package modelcontextprotocol

// --- ClientCapabilities Options ---
type ClientCapabilityOption func(*ClientCapabilities)
type RootsClientCapabilityOption func(*RootsClientCapability)

func NewClientCapabilities(opts ...ClientCapabilityOption) *ClientCapabilities {
	cc := &ClientCapabilities{} // Experimental map is nil by default, initialized on first WithClientExperimental
	for _, opt := range opts {
		opt(cc)
	}
	return cc
}
func WithClientExperimental(key string, value any) ClientCapabilityOption {
	return func(cc *ClientCapabilities) {
		if cc.Experimental == nil {
			cc.Experimental = make(map[string]any)
		}
		cc.Experimental[key] = value
	}
}
func WithClientSampling() ClientCapabilityOption {
	return func(cc *ClientCapabilities) { cc.Sampling = &struct{}{} }
}
func WithClientRoots(rootsOpts ...RootsClientCapabilityOption) ClientCapabilityOption {
	return func(cc *ClientCapabilities) {
		if cc.Roots == nil {
			cc.Roots = &RootsClientCapability{}
		}
		for _, opt := range rootsOpts {
			opt(cc.Roots)
		}
	}
}
func WithRootsListChanged(enabled bool) RootsClientCapabilityOption {
	return func(rc *RootsClientCapability) {
		if enabled {
			val := true
			rc.ListChanged = &val
		} else {
			rc.ListChanged = nil
		}
	}
}

// --- ServerCapabilities Options ---
type ServerCapabilityOption func(*ServerCapabilities)
type PromptsServerCapabilityOption func(*PromptsServerCapability)
type ResourcesServerCapabilityOption func(*ResourcesServerCapability)
type ToolsServerCapabilityOption func(*ToolsServerCapability)

func NewServerCapabilities(opts ...ServerCapabilityOption) *ServerCapabilities {
	sc := &ServerCapabilities{} // Experimental map is nil by default
	for _, opt := range opts {
		opt(sc)
	}
	return sc
}
func WithServerLogging() ServerCapabilityOption {
	return func(sc *ServerCapabilities) { sc.Logging = &struct{}{} }
}
func WithServerCompletions() ServerCapabilityOption {
	return func(sc *ServerCapabilities) { sc.Completions = &struct{}{} }
}
func WithServerPrompts(promptOpts ...PromptsServerCapabilityOption) ServerCapabilityOption {
	return func(sc *ServerCapabilities) {
		if sc.Prompts == nil {
			sc.Prompts = &PromptsServerCapability{}
		}
		for _, opt := range promptOpts {
			opt(sc.Prompts)
		}
	}
}
func WithPromptsListChanged(enabled bool) PromptsServerCapabilityOption {
	return func(pc *PromptsServerCapability) {
		if enabled {
			val := true
			pc.ListChanged = &val
		} else {
			pc.ListChanged = nil
		}
	}
}
func WithServerResources(resourceOpts ...ResourcesServerCapabilityOption) ServerCapabilityOption {
	return func(sc *ServerCapabilities) {
		if sc.Resources == nil {
			sc.Resources = &ResourcesServerCapability{}
		}
		for _, opt := range resourceOpts {
			opt(sc.Resources)
		}
	}
}
func WithResourcesSubscription(enabled bool) ResourcesServerCapabilityOption {
	return func(rsc *ResourcesServerCapability) {
		if enabled {
			val := true
			rsc.Subscribe = &val
		} else {
			rsc.Subscribe = nil
		}
	}
}
func WithResourcesListChanged(enabled bool) ResourcesServerCapabilityOption {
	return func(rsc *ResourcesServerCapability) {
		if enabled {
			val := true
			rsc.ListChanged = &val
		} else {
			rsc.ListChanged = nil
		}
	}
}
func WithServerTools(toolOpts ...ToolsServerCapabilityOption) ServerCapabilityOption {
	return func(sc *ServerCapabilities) {
		if sc.Tools == nil {
			sc.Tools = &ToolsServerCapability{}
		}
		for _, opt := range toolOpts {
			opt(sc.Tools)
		}
	}
}
func WithToolsListChanged(enabled bool) ToolsServerCapabilityOption {
	return func(tc *ToolsServerCapability) {
		if enabled {
			val := true
			tc.ListChanged = &val
		} else {
			tc.ListChanged = nil
		}
	}
}
func WithServerExperimental(key string, value any) ServerCapabilityOption {
	return func(sc *ServerCapabilities) {
		if sc.Experimental == nil {
			sc.Experimental = make(map[string]any)
		}
		sc.Experimental[key] = value
	}
}

// --- Prompt Options ---
type PromptOption func(*Prompt)
type PromptArgumentOption func(*PromptArgument)

func NewPrompt(name string, opts ...PromptOption) Prompt {
	p := Prompt{Name: name} // Arguments slice will be nil initially
	for _, opt := range opts {
		opt(&p)
	}
	return p
}
func WithPromptDescription(desc string) PromptOption {
	return func(p *Prompt) { p.Description = &desc }
}
func WithPromptArgument(name string, argOpts ...PromptArgumentOption) PromptOption {
	return func(p *Prompt) {
		arg := PromptArgument{Name: name}
		for _, opt := range argOpts {
			opt(&arg)
		}
		if p.Arguments == nil {
			p.Arguments = make([]*PromptArgument, 0)
		} // Initialize if nil
		p.Arguments = append(p.Arguments, &arg)
	}
}
func WithPromptArgumentDescription(desc string) PromptArgumentOption {
	return func(pa *PromptArgument) { pa.Description = &desc }
}
func WithPromptArgumentRequired(req bool) PromptArgumentOption {
	return func(pa *PromptArgument) {
		if req {
			val := true
			pa.Required = &val
		} else {
			pa.Required = nil
		}
	}
}

// --- Tool Options (base version) ---
type ToolOption func(*Tool)

func NewTool(name string, inputSchema ToolSchema, opts ...ToolOption) Tool {
	t := Tool{Name: name, InputSchema: inputSchema}
	for _, opt := range opts {
		opt(&t)
	}
	return t
}
func WithToolDescription(desc string) ToolOption { return func(t *Tool) { t.Description = &desc } }
func WithToolAnnotations(annotations ToolAnnotations) ToolOption {
	return func(t *Tool) { t.Annotations = &annotations }
}

// --- Resource Options ---
type ResourceOption func(*Resource)

func NewResource(uri, name string, opts ...ResourceOption) Resource {
	r := Resource{URI: uri, Name: name}
	for _, opt := range opts {
		opt(&r)
	}
	return r
}
func WithResourceDescription(desc string) ResourceOption {
	return func(r *Resource) { r.Description = &desc }
}
func WithResourceMimeType(mime string) ResourceOption {
	return func(r *Resource) { r.MimeType = &mime }
}
func WithResourceSize(size int64) ResourceOption { return func(r *Resource) { r.Size = &size } }
func WithResourceAnnotations(annotations Annotations) ResourceOption {
	return func(r *Resource) { r.Annotations = &annotations }
}

// --- ResourceTemplate Options ---
type ResourceTemplateOption func(*ResourceTemplate)

func NewResourceTemplate(uriTemplate, name string, opts ...ResourceTemplateOption) ResourceTemplate {
	rt := ResourceTemplate{URITemplate: uriTemplate, Name: name}
	for _, opt := range opts {
		opt(&rt)
	}
	return rt
}
func WithResourceTemplateDescription(desc string) ResourceTemplateOption {
	return func(rt *ResourceTemplate) { rt.Description = &desc }
}
func WithResourceTemplateMimeType(mime string) ResourceTemplateOption {
	return func(rt *ResourceTemplate) { rt.MimeType = &mime }
}
func WithResourceTemplateAnnotations(annotations Annotations) ResourceTemplateOption {
	return func(rt *ResourceTemplate) { rt.Annotations = &annotations }
}
