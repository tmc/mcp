package mcpcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Context identifies the visibility under which a catalog was captured.
// Scope and Principal are labels chosen by the caller, never credentials.
// Client describes relevant client capabilities or configuration.
type Context struct {
	Scope     string `json:"scope"`
	Principal string `json:"principal"`
	Client    string `json:"client"`
}

// Snapshot is a versioned tool catalog. Format must be 1.
// Initialize contains negotiated protocol and server capabilities.
type Snapshot struct {
	Format     int                   `json:"format"`
	Context    Context               `json:"context"`
	Initialize *mcp.InitializeResult `json:"initialize"`
	Tools      []*mcp.Tool           `json:"tools"`
}

// Capture lists all tools using the SDK's pagination support. It does not call
// tools. The caller must supply accurate visibility labels and keep the server's
// catalog stable during capture; the protocol does not provide atomic snapshots.
func Capture(ctx context.Context, session *mcp.ClientSession, visibility Context) (*Snapshot, error) {
	if session == nil || session.InitializeResult() == nil {
		return nil, fmt.Errorf("capture: uninitialized session")
	}
	s := &Snapshot{Format: 1, Context: visibility, Initialize: session.InitializeResult(), Tools: []*mcp.Tool{}}
	if s.Initialize.Capabilities != nil && s.Initialize.Capabilities.Tools != nil {
		for tool, err := range session.Tools(ctx, nil) {
			if err != nil {
				return nil, fmt.Errorf("list tools: %w", err)
			}
			s.Tools = append(s.Tools, tool)
		}
	}
	// Detach SDK-owned objects and normalize the ordering.
	data, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("capture: %w", err)
	}
	return readBytes(data)
}

func readBytes(data []byte) (*Snapshot, error) {
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	if err := normalize(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Read reads one JSON snapshot and rejects trailing data or invalid catalogs.
func Read(r io.Reader) (*Snapshot, error) {
	var s Snapshot
	d := json.NewDecoder(r)
	if err := d.Decode(&s); err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("read catalog: trailing data")
	}
	if err := normalize(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Write writes a deterministically ordered JSON snapshot without modifying s.
func Write(w io.Writer, s *Snapshot) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("write catalog: %w", err)
	}
	copy, err := readBytes(data)
	if err != nil {
		return err
	}
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	if err := e.Encode(copy); err != nil {
		return fmt.Errorf("write catalog: %w", err)
	}
	return nil
}
func normalize(s *Snapshot) error {
	if s.Format != 1 || s.Initialize == nil || s.Initialize.ProtocolVersion == "" || s.Initialize.ServerInfo == nil {
		return fmt.Errorf("invalid catalog header")
	}
	seen := map[string]bool{}
	for _, t := range s.Tools {
		if t == nil || t.Name == "" || seen[t.Name] {
			return fmt.Errorf("invalid or duplicate tool name")
		}
		seen[t.Name] = true
	}
	sort.Slice(s.Tools, func(i, j int) bool { return s.Tools[i].Name < s.Tools[j].Name })
	return nil
}
