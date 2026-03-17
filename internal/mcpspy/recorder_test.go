package mcpspy

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestRecorderSplitJSON(t *testing.T) {
	var log bytes.Buffer
	rec := New(&log, Options{
		TimeNow: func() time.Time { return time.Unix(1, 200000000) },
	})
	w := rec.Writer("recv", io.Discard)
	if _, err := w.Write([]byte("{\"a\":1")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("}\n")); err != nil {
		t.Fatal(err)
	}
	snapshot := rec.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("snapshot len=%d, want 1", len(snapshot))
	}
	if got := string(snapshot[0].Raw); got != "{\"a\":1}" {
		t.Fatalf("raw=%q", got)
	}
	if !bytes.Contains(log.Bytes(), []byte("mcp-recv {\"a\":1} # 1.200")) {
		t.Fatalf("log=%q", log.String())
	}
}

func TestRecorderEvictsOldest(t *testing.T) {
	rec := New(io.Discard, Options{
		BufferSize: 2,
		TimeNow:    func() time.Time { return time.Unix(1, 0) },
	})
	w := rec.Writer("send", io.Discard)
	for _, line := range []string{"{\"id\":1}\n", "{\"id\":2}\n", "{\"id\":3}\n"} {
		if _, err := w.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := rec.Snapshot()
	if len(snapshot) != 2 {
		t.Fatalf("snapshot len=%d, want 2", len(snapshot))
	}
	if got := string(snapshot[0].Raw); got != "{\"id\":2}" {
		t.Fatalf("first raw=%q", got)
	}
	if got := string(snapshot[1].Raw); got != "{\"id\":3}" {
		t.Fatalf("second raw=%q", got)
	}
}

func TestSubscribeReceivesEvents(t *testing.T) {
	rec := New(io.Discard, Options{
		TimeNow: func() time.Time { return time.Unix(1, 0) },
	})
	ch, cancel := rec.Subscribe()
	defer cancel()
	if _, err := rec.Writer("send", io.Discard).Write([]byte("{\"ok\":true}\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-ch:
		if ev.Seq == 0 {
			t.Fatal("missing sequence number")
		}
		if ev.Direction != "send" {
			t.Fatalf("direction=%q", ev.Direction)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}
