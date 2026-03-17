package mcpcli

import (
	"path/filepath"
	"testing"

	"github.com/tmc/mcp"
)

func TestStateStoreRoots(t *testing.T) {
	store, err := OpenStateStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddRoot(mcp.Root{URI: "file:///tmp/project", Name: "project"}); err != nil {
		t.Fatal(err)
	}
	if err := store.AddRoot(mcp.Root{URI: "file:///tmp/project"}); err != nil {
		t.Fatal(err)
	}
	roots, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 {
		t.Fatalf("roots=%v", roots)
	}
	if roots[0].URI != "file:///tmp/project" {
		t.Fatalf("uri=%q", roots[0].URI)
	}
	if err := store.RemoveRoot("file:///tmp/project"); err != nil {
		t.Fatal(err)
	}
	roots, err = store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 0 {
		t.Fatalf("roots after remove=%v", roots)
	}
}

func TestOpenStateStoreCreatesFileArea(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "state")
	if _, err := OpenStateStore(dir); err != nil {
		t.Fatal(err)
	}
}
