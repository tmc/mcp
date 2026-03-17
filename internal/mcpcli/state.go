package mcpcli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/tmc/mcp"
)

type stateFile struct {
	Roots []mcp.Root `json:"roots"`
}

// StateStore persists local CLI state such as the configured root set.
type StateStore struct {
	path string
	mu   sync.Mutex
}

// OpenStateStore opens or creates the CLI state store rooted at dir.
func OpenStateStore(dir string) (*StateStore, error) {
	if dir == "" {
		return nil, errors.New("missing state directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &StateStore{path: filepath.Join(dir, "state.json")}, nil
}

// List returns the persisted roots.
func (s *StateStore) List() ([]mcp.Root, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.read()
	if err != nil {
		return nil, err
	}
	roots := append([]mcp.Root(nil), state.Roots...)
	sort.Slice(roots, func(i, j int) bool { return roots[i].URI < roots[j].URI })
	return roots, nil
}

// AddRoot persists root if it does not already exist.
func (s *StateStore) AddRoot(root mcp.Root) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.read()
	if err != nil {
		return err
	}
	root.URI = strings.TrimSpace(root.URI)
	if root.URI == "" {
		return errors.New("root URI is required")
	}
	for _, existing := range state.Roots {
		if existing.URI == root.URI {
			return nil
		}
	}
	state.Roots = append(state.Roots, root)
	return s.write(state)
}

// RemoveRoot removes rootURI if present.
func (s *StateStore) RemoveRoot(rootURI string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.read()
	if err != nil {
		return err
	}
	filtered := state.Roots[:0]
	for _, root := range state.Roots {
		if root.URI != rootURI {
			filtered = append(filtered, root)
		}
	}
	state.Roots = filtered
	return s.write(state)
}

func (s *StateStore) read() (*stateFile, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return &stateFile{}, nil
	}
	if err != nil {
		return nil, err
	}
	var state stateFile
	if len(data) == 0 {
		return &state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *StateStore) write(state *stateFile) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
