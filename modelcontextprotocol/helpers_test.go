package modelcontextprotocol

import (
	"testing"
)

func TestNewHelpers(t *testing.T) {
	t.Run("NewTextContent", func(t *testing.T) {
		tc := NewTextContent("test text")
		if tc.Type != ContentTypeText {
			t.Errorf("Expected type %s, got %s", ContentTypeText, tc.Type)
		}
		if tc.Text != "test text" {
			t.Errorf("Expected text 'test text', got %s", tc.Text)
		}
	})

	t.Run("NewImageContent", func(t *testing.T) {
		ic := NewImageContent("base64data", "image/png")
		if ic.Type != ContentTypeImage {
			t.Errorf("Expected type %s, got %s", ContentTypeImage, ic.Type)
		}
		if ic.Data != "base64data" {
			t.Errorf("Expected data 'base64data', got %s", ic.Data)
		}
		if ic.MimeType != "image/png" {
			t.Errorf("Expected mimeType 'image/png', got %s", ic.MimeType)
		}
	})

	t.Run("NewAudioContent", func(t *testing.T) {
		ac := NewAudioContent("base64audio", "audio/mp3")
		if ac.Type != ContentTypeAudio {
			t.Errorf("Expected type %s, got %s", ContentTypeAudio, ac.Type)
		}
		if ac.Data != "base64audio" {
			t.Errorf("Expected data 'base64audio', got %s", ac.Data)
		}
		if ac.MimeType != "audio/mp3" {
			t.Errorf("Expected mimeType 'audio/mp3', got %s", ac.MimeType)
		}
	})

	t.Run("NewEmbeddedResource", func(t *testing.T) {
		rc := NewTextResourceContents("/path/to/file", "file content")
		er := NewEmbeddedResource(rc)
		if er.Type != ContentTypeResource {
			t.Errorf("Expected type %s, got %s", ContentTypeResource, er.Type)
		}
		if er.Resource == nil {
			t.Error("Expected resource to be set")
		}
	})

	t.Run("NewPromptReference", func(t *testing.T) {
		pr := NewPromptReference("my-prompt")
		if pr.Type != ReferenceTypePrompt {
			t.Errorf("Expected type %s, got %s", ReferenceTypePrompt, pr.Type)
		}
		if pr.Name != "my-prompt" {
			t.Errorf("Expected name 'my-prompt', got %s", pr.Name)
		}
	})

	t.Run("NewResourceReference", func(t *testing.T) {
		rr := NewResourceReference("/path/to/resource")
		if rr.Type != ReferenceTypeResource {
			t.Errorf("Expected type %s, got %s", ReferenceTypeResource, rr.Type)
		}
		if rr.URI != "/path/to/resource" {
			t.Errorf("Expected URI '/path/to/resource', got %s", rr.URI)
		}
	})

	t.Run("NewTextResourceContents", func(t *testing.T) {
		trc := NewTextResourceContents("/file.txt", "content", "text/plain")
		if trc.URI != "/file.txt" {
			t.Errorf("Expected URI '/file.txt', got %s", trc.URI)
		}
		if trc.Text != "content" {
			t.Errorf("Expected text 'content', got %s", trc.Text)
		}
		if trc.MimeType == nil || *trc.MimeType != "text/plain" {
			t.Error("Expected mimeType 'text/plain'")
		}
	})

	t.Run("NewBlobResourceContents", func(t *testing.T) {
		brc := NewBlobResourceContents("/file.bin", "base64data", "application/octet-stream")
		if brc.URI != "/file.bin" {
			t.Errorf("Expected URI '/file.bin', got %s", brc.URI)
		}
		if brc.Blob != "base64data" {
			t.Errorf("Expected blob 'base64data', got %s", brc.Blob)
		}
		if brc.MimeType == nil || *brc.MimeType != "application/octet-stream" {
			t.Error("Expected mimeType 'application/octet-stream'")
		}
	})
}

func TestCapabilitiesHelpers(t *testing.T) {
	t.Run("ClientCapabilities nil checks", func(t *testing.T) {
		// Test completely nil capabilities
		var cc *ClientCapabilities
		if cc.SupportsSampling() {
			t.Error("nil capabilities should not support sampling")
		}
		if cc.SupportsRootListChanged() {
			t.Error("nil capabilities should not support root list changed")
		}
		// Test nil experimental getter
		val, ok := cc.GetClientExperimental("test")
		if ok || val != nil {
			t.Error("nil capabilities should return false and nil for experimental")
		}
	})

	t.Run("ClientCapabilities nil experimental map", func(t *testing.T) {
		cc := &ClientCapabilities{}
		val, ok := cc.GetClientExperimental("test")
		if ok || val != nil {
			t.Error("nil experimental map should return false and nil")
		}
	})

	t.Run("ServerCapabilities nil checks", func(t *testing.T) {
		// Test completely nil capabilities
		var sc *ServerCapabilities
		if sc.SupportsLogging() {
			t.Error("nil capabilities should not support logging")
		}
		if sc.SupportsCompletions() {
			t.Error("nil capabilities should not support completions")
		}
		if sc.SupportsPromptListChanged() {
			t.Error("nil capabilities should not support prompt list changed")
		}
		if sc.SupportsResourceSubscription() {
			t.Error("nil capabilities should not support resource subscription")
		}
		if sc.SupportsResourceListChanged() {
			t.Error("nil capabilities should not support resource list changed")
		}
		if sc.SupportsToolListChanged() {
			t.Error("nil capabilities should not support tool list changed")
		}
		// Test nil experimental getter
		val, ok := sc.GetServerExperimental("test")
		if ok || val != nil {
			t.Error("nil capabilities should return false and nil for experimental")
		}
	})

	t.Run("ServerCapabilities nil experimental map", func(t *testing.T) {
		sc := &ServerCapabilities{}
		val, ok := sc.GetServerExperimental("test")
		if ok || val != nil {
			t.Error("nil experimental map should return false and nil")
		}
	})
}

func TestContentHelpersWithAnnotations(t *testing.T) {
	priority := 1.0
	annotations := Annotations{
		Audience: []Role{RoleUser, RoleAssistant},
		Priority: &priority,
	}

	t.Run("NewImageContent with annotations", func(t *testing.T) {
		ic := NewImageContent("data", "image/png", annotations)
		if ic.Annotations == nil {
			t.Error("Expected annotations to be set")
		} else if ic.Annotations.Priority == nil || *ic.Annotations.Priority != priority {
			t.Error("Expected priority to be set correctly")
		}
	})

	t.Run("NewAudioContent with annotations", func(t *testing.T) {
		ac := NewAudioContent("data", "audio/mp3", annotations)
		if ac.Annotations == nil {
			t.Error("Expected annotations to be set")
		} else if ac.Annotations.Priority == nil || *ac.Annotations.Priority != priority {
			t.Error("Expected priority to be set correctly")
		}
	})

	t.Run("NewEmbeddedResource with annotations", func(t *testing.T) {
		rc := NewTextResourceContents("/file", "content")
		er := NewEmbeddedResource(rc, annotations)
		if er.Annotations == nil {
			t.Error("Expected annotations to be set")
		} else if er.Annotations.Priority == nil || *er.Annotations.Priority != priority {
			t.Error("Expected priority to be set correctly")
		}
	})
}
