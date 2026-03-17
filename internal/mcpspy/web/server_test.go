package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tmc/mcp/internal/mcpspy"
)

func TestServerEndpoints(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	rec := mcpspy.New(io.Discard, mcpspy.Options{})
	spec := mcpspy.NewSpecTracker(rec, mcpspy.SpecOptions{})
	defer spec.Close()
	rt, err := mcpspy.NewRuntime(rec, mcpspy.RuntimeOptions{
		SessionID: "session-web",
		Command:   []string{"mcpspy"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	srv := New(rec, spec, rt, Options{Addr: "127.0.0.1:0"})
	if err := rt.Start(srv.Start); err != nil {
		t.Fatal(err)
	}
	url, err := srv.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	resp, err := http.Get(url + "/api/info")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var info map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info["self"] == nil {
		t.Fatal("missing self info")
	}

	resp, err = http.Get(url + "/api/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	if _, err := rec.Writer("recv", io.Discard).Write([]byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"echo\",\"arguments\":{\"message\":\"hi\"}}}\n")); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, err = http.Get(url + "/api/spec")
		if err != nil {
			t.Fatal(err)
		}
		var snapshot struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
			resp.Body.Close()
			t.Fatal(err)
		}
		resp.Body.Close()
		if strings.Contains(snapshot.Text, "\"echo\"") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("spec snapshot=%q", snapshot.Text)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
