package mcptrace

import (
	"io"
	"strings"
	"testing"
)

const benchmarkMessage = `mcp-send {"jsonrpc":"2.0","id":1,"method":"ping"}`

func BenchmarkReader(b *testing.B) {
	input := benchmarkMessage + "\n"
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader(strings.NewReader(input))
		if _, err := r.Read(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriter(b *testing.B) {
	writer := NewWriter(io.Discard)
	record := Record{Direction: "send", Message: []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)}
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkMessage) + 1))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := writer.Write(record); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRecordOverhead(b *testing.B) {
	input := benchmarkMessage + "\n"
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := ReadCloser(io.NopCloser(strings.NewReader(input)), NewWriter(io.Discard), "recv")
		if _, err := io.Copy(io.Discard, r); err != nil {
			b.Fatal(err)
		}
		if err := r.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
