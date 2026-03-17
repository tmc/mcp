package mcpspy

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuntimeRegistrationAndPeerOpen(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmp)

	rec := New(io.Discard, Options{})
	rt, err := NewRuntime(rec, RuntimeOptions{
		SessionID: "session-a",
		Command:   []string{"mcpspy", "--", "cat"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()

	var opened int
	if err := rt.Start(func() (string, error) {
		opened++
		return "http://127.0.0.1:1234", rt.UpdateUIURL("http://127.0.0.1:1234")
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(tmp, "mcpspy", rt.Status().ID+".json")); err != nil {
		t.Fatal(err)
	}

	url, err := OpenPeerUI(rt.Status())
	if err != nil {
		t.Fatal(err)
	}
	if url != "http://127.0.0.1:1234" {
		t.Fatalf("url=%q", url)
	}
	if opened != 1 {
		t.Fatalf("opened=%d, want 1", opened)
	}
}

func TestPruneStale(t *testing.T) {
	tmp := t.TempDir()
	info := InstanceInfo{
		ID:        "old",
		PID:       999999,
		Heartbeat: time.Now().Add(-staleAfter - time.Second),
	}
	data, err := jsonMarshal(info)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(tmp, "old.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := pruneStale(tmp); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stale registry still exists: %v", err)
	}
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
