package mcpcorpus

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/tmc/mcp/mcptrace"
)

// Seed contains one tool's raw JSON object arguments. Name is the lowercase
// SHA-256 digest of Arguments, suitable as a deterministic filename.
type Seed struct {
	Name      string
	Arguments []byte
}

// Extract reads trace records and returns unique seeds for tool, sorted by Name.
// It rejects malformed selected requests, missing or null arguments, and invalid
// trace records. On any error it returns no seeds.
func Extract(r io.Reader, tool string) ([]Seed, error) {
	if r == nil || tool == "" {
		return nil, fmt.Errorf("extract: reader and tool name are required")
	}
	trace := mcptrace.NewReader(r)
	unique := map[string]Seed{}
	for record := 1; ; record++ {
		rec, err := trace.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("record %d: %w", record, err)
		}
		args, selected, err := arguments(rec.Message, tool)
		if err != nil {
			return nil, fmt.Errorf("record %d: %w", record, err)
		}
		if !selected {
			continue
		}
		name := fmt.Sprintf("%x", sha256.Sum256(args))
		if prior, ok := unique[name]; ok && !bytes.Equal(prior.Arguments, args) {
			return nil, fmt.Errorf("record %d: seed digest collision", record)
		}
		unique[name] = Seed{Name: name, Arguments: append([]byte(nil), args...)}
	}
	seeds := make([]Seed, 0, len(unique))
	for _, seed := range unique {
		seeds = append(seeds, seed)
	}
	sort.Slice(seeds, func(i, j int) bool { return seeds[i].Name < seeds[j].Name })
	return seeds, nil
}

func arguments(message []byte, tool string) (json.RawMessage, bool, error) {
	envelope, err := object(message)
	if err != nil {
		return nil, false, fmt.Errorf("message: %w", err)
	}
	method := envelope["method"]
	if method == nil {
		return nil, false, nil
	}
	var name string
	if json.Unmarshal(method, &name) != nil || bytes.Equal(method, []byte("null")) {
		return nil, false, fmt.Errorf("invalid method")
	}
	if name != "tools/call" {
		return nil, false, nil
	}
	params, err := object(envelope["params"])
	if err != nil {
		return nil, false, fmt.Errorf("tools/call params: %w", err)
	}
	var requested string
	if json.Unmarshal(params["name"], &requested) != nil || requested == "" {
		return nil, false, fmt.Errorf("tools/call requires a tool name")
	}
	if requested != tool {
		return nil, false, nil
	}
	var version string
	if json.Unmarshal(envelope["jsonrpc"], &version) != nil || version != "2.0" {
		return nil, false, fmt.Errorf("selected call requires jsonrpc 2.0")
	}
	id := bytes.TrimSpace(envelope["id"])
	if len(id) == 0 || bytes.Equal(id, []byte("null")) {
		return nil, false, fmt.Errorf("selected call requires a request id")
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(id))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return nil, false, fmt.Errorf("invalid request id")
	}
	switch value.(type) {
	case string, json.Number:
	default:
		return nil, false, fmt.Errorf("invalid request id")
	}
	if envelope["result"] != nil || envelope["error"] != nil {
		return nil, false, fmt.Errorf("selected call is also a response")
	}
	args := params["arguments"]
	if _, err := object(args); err != nil {
		return nil, false, fmt.Errorf("selected call arguments must be a JSON object: %w", err)
	}
	return args, true, nil
}

// object rejects duplicate keys instead of choosing an arbitrary interpretation.
func object(data []byte) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil {
		return nil, err
	}
	if token != json.Delim('{') {
		return nil, fmt.Errorf("expected object")
	}
	result := map[string]json.RawMessage{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("invalid object key")
		}
		if _, ok := result[key]; ok {
			return nil, fmt.Errorf("duplicate key %q", key)
		}
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return nil, err
		}
		result[key] = raw
	}
	if _, err := d.Token(); err != nil {
		return nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing object data")
	}
	return result, nil
}

// WriteFuzz writes a Go fuzz corpus file for a target with one []byte argument.
// Place it under testdata/fuzz/FuzzName/ to use it with go test.
func WriteFuzz(w io.Writer, seed Seed) error {
	if w == nil {
		return fmt.Errorf("write fuzz seed: nil writer")
	}
	if _, err := object(seed.Arguments); err != nil {
		return fmt.Errorf("write fuzz seed: %w", err)
	}
	data := []byte(fmt.Sprintf("go test fuzz v1\n[]byte(%q)\n", seed.Arguments))
	if err := write(w, data); err != nil {
		return fmt.Errorf("write fuzz seed: %w", err)
	}
	return nil
}

// WriteJSON writes the exact argument bytes followed by a newline.
func WriteJSON(w io.Writer, seed Seed) error {
	if w == nil {
		return fmt.Errorf("write JSON seed: nil writer")
	}
	if _, err := object(seed.Arguments); err != nil {
		return fmt.Errorf("write JSON seed: %w", err)
	}
	if err := write(w, append(append([]byte(nil), seed.Arguments...), '\n')); err != nil {
		return fmt.Errorf("write JSON seed: %w", err)
	}
	return nil
}

func write(w io.Writer, data []byte) error {
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}
