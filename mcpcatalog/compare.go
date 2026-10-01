package mcpcatalog

import (
	"encoding/json"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Kind classifies a declared API difference.
type Kind string

const (
	// Incompatible means a supported schema or catalog change breaks the old API.
	Incompatible Kind = "incompatible"
	// Unknown means compatibility cannot be established by this checker.
	Unknown Kind = "unknown"
	// Changed means a difference without a detected schema incompatibility.
	Changed Kind = "changed"
)

// Change identifies a tool or catalog difference. Path is a schema field or
// catalog field; Message explains the classification.
type Change struct {
	Kind    Kind   `json:"kind"`
	Tool    string `json:"tool,omitempty"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Compare compares declared tool APIs. Input acceptance may widen; output
// possibilities may narrow. Unknown schemas and mismatched visibility contexts
// prevent a compatibility conclusion. An empty result only establishes the
// bounded checks described in the package documentation.
func Compare(old, new *Snapshot) []Change {
	var result []Change
	add := func(k Kind, t, p, m string) { result = append(result, Change{k, t, p, m}) }
	if old == nil || new == nil {
		add(Unknown, "", "catalog", "missing snapshot")
		return result
	}
	if normalizeCopy(old) != nil || normalizeCopy(new) != nil {
		add(Unknown, "", "catalog", "invalid snapshot")
		return result
	}
	if old.Context != new.Context || old.Context.Scope == "" || old.Context.Principal == "" || old.Context.Client == "" || old.Initialize.ProtocolVersion != new.Initialize.ProtocolVersion || old.Initialize.ServerInfo.Name != new.Initialize.ServerInfo.Name || !equalJSON(old.Initialize.Capabilities, new.Initialize.Capabilities) {
		add(Unknown, "", "context", "visibility, protocol, server identity, or capabilities differ or are unspecified")
		return result
	}
	before := map[string]*toolPair{}
	for _, t := range old.Tools {
		before[t.Name] = &toolPair{old: t}
	}
	for _, t := range new.Tools {
		if before[t.Name] == nil {
			before[t.Name] = &toolPair{}
		}
		before[t.Name].new = t
	}
	names := make([]string, 0, len(before))
	for n := range before {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		pair := before[name]
		if pair.old == nil {
			add(Changed, name, "tool", "tool added")
			continue
		}
		if pair.new == nil {
			add(Incompatible, name, "tool", "tool removed")
			continue
		}
		for _, field := range []string{"inputSchema", "outputSchema"} {
			a, b := pair.old.InputSchema, pair.new.InputSchema
			if field == "outputSchema" {
				a, b = pair.old.OutputSchema, pair.new.OutputSchema
			}
			if field == "outputSchema" && a == nil && b == nil {
				continue
			}
			sa, oka := parseSchema(a)
			sb, okb := parseSchema(b)
			if !oka || !okb {
				add(Unknown, name, field, "schema is absent or outside the supported subset")
				continue
			}
			source, target := sa, sb
			if field == "outputSchema" {
				source, target = sb, sa
			}
			if !subset(source, target) {
				add(Incompatible, name, field, "accepted inputs narrow or possible outputs widen")
			} else if !equalJSON(a, b) {
				add(Changed, name, field, "schema changed within the supported compatibility rules")
			}
		}
		a, b := *pair.old, *pair.new
		a.InputSchema = nil
		a.OutputSchema = nil
		b.InputSchema = nil
		b.OutputSchema = nil
		if !equalJSON(a, b) {
			add(Changed, name, "metadata", "tool metadata changed")
		}
	}
	return result
}

func equalJSON(a, b any) bool {
	x, e := json.Marshal(a)
	y, f := json.Marshal(b)
	return e == nil && f == nil && string(x) == string(y)
}
func normalizeCopy(s *Snapshot) error {
	_, err := json.Marshal(s)
	if err != nil {
		return err
	}
	copy := *s
	copy.Tools = append(copy.Tools[:0:0], s.Tools...)
	return normalize(&copy)
}

type toolPair struct{ old, new *mcp.Tool }
