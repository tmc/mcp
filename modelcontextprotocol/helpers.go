package modelcontextprotocol

// --- Capability Checking Helpers ---
func (cc *ClientCapabilities) SupportsSampling() bool { return cc != nil && cc.Sampling != nil }
func (cc *ClientCapabilities) SupportsRootListChanged() bool {
	return cc != nil && cc.Roots != nil && cc.Roots.ListChanged != nil && *cc.Roots.ListChanged
}
func (cc *ClientCapabilities) GetClientExperimental(key string) (any, bool) {
	if cc == nil || cc.Experimental == nil {
		return nil, false
	}
	val, ok := cc.Experimental[key]
	return val, ok
}
func (sc *ServerCapabilities) SupportsLogging() bool     { return sc != nil && sc.Logging != nil }
func (sc *ServerCapabilities) SupportsCompletions() bool { return sc != nil && sc.Completions != nil }
func (sc *ServerCapabilities) SupportsPromptListChanged() bool {
	return sc != nil && sc.Prompts != nil && sc.Prompts.ListChanged != nil && *sc.Prompts.ListChanged
}
func (sc *ServerCapabilities) SupportsResourceSubscription() bool {
	return sc != nil && sc.Resources != nil && sc.Resources.Subscribe != nil && *sc.Resources.Subscribe
}
func (sc *ServerCapabilities) SupportsResourceListChanged() bool {
	return sc != nil && sc.Resources != nil && sc.Resources.ListChanged != nil && *sc.Resources.ListChanged
}
func (sc *ServerCapabilities) SupportsToolListChanged() bool {
	return sc != nil && sc.Tools != nil && sc.Tools.ListChanged != nil && *sc.Tools.ListChanged
}
func (sc *ServerCapabilities) GetServerExperimental(key string) (any, bool) {
	if sc == nil || sc.Experimental == nil {
		return nil, false
	}
	val, ok := sc.Experimental[key]
	return val, ok
}

// --- Content Creation Helpers ---
func NewTextContent(text string, annotations ...Annotations) TextContent {
	var ann *Annotations
	if len(annotations) > 0 {
		ann = &annotations[0]
	}
	return TextContent{Type: ContentTypeText, Text: text, Annotations: ann}
}
func NewImageContent(data string, mimeType string, annotations ...Annotations) ImageContent {
	var ann *Annotations
	if len(annotations) > 0 {
		ann = &annotations[0]
	}
	return ImageContent{Type: ContentTypeImage, Data: data, MimeType: mimeType, Annotations: ann}
}
func NewAudioContent(data string, mimeType string, annotations ...Annotations) AudioContent {
	var ann *Annotations
	if len(annotations) > 0 {
		ann = &annotations[0]
	}
	return AudioContent{Type: ContentTypeAudio, Data: data, MimeType: mimeType, Annotations: ann}
}
func NewEmbeddedResource(rc ResourceContents, annotations ...Annotations) EmbeddedResource {
	var ann *Annotations
	if len(annotations) > 0 {
		ann = &annotations[0]
	}
	return EmbeddedResource{Type: ContentTypeResource, Resource: rc, Annotations: ann}
}
func NewTextResourceContents(uri, text string, mimeType ...string) TextResourceContents {
	var mt *string
	if len(mimeType) > 0 {
		mtS := mimeType[0]
		mt = &mtS
	}
	return TextResourceContents{BaseResourceContents: BaseResourceContents{URI: uri, MimeType: mt}, Text: text}
}
func NewBlobResourceContents(uri, blobData string, mimeType ...string) BlobResourceContents {
	var mt *string
	if len(mimeType) > 0 {
		mtS := mimeType[0]
		mt = &mtS
	}
	return BlobResourceContents{BaseResourceContents: BaseResourceContents{URI: uri, MimeType: mt}, Blob: blobData}
}

// --- Reference Creation Helpers ---
func NewPromptReference(name string) PromptReference {
	return PromptReference{Type: ReferenceTypePrompt, Name: name}
}
func NewResourceReference(uri string) ResourceReference {
	return ResourceReference{Type: ReferenceTypeResource, URI: uri}
}
