package mcptrace_test

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/tmc/mcp/mcptrace"
)

func ExampleNewConversation() {
	c := mcptrace.NewConversation(strings.NewReader("mcp-send {\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\"}\nmcp-recv {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}\n"), "demo")
	for {
		e, err := c.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Println(e.String())
	}
	// Output:
	// 1 send request session=demo id=1 ping
	// 2 recv response session=demo id=1 ping request=1
}

func TestConversation(t *testing.T) {
	tests := []struct {
		name, input string
		kinds       []string
		matches     []int
		diagnostics []bool
	}{
		{"escaped ID", `mcp-send {"jsonrpc":"2.0","id":"a","method":"ping"}
mcp-recv {"jsonrpc":"2.0","id":"\u0061","result":{}}`, []string{"request", "response"}, []int{0, 1}, nil},
		{"conflicting scope", `mcp-send {"jsonrpc":"2.0","id":1,"method":"ping"} # 1 session=a session=b`, []string{"diagnostic"}, []int{0}, []bool{true}},
		{"invalid params", `mcp-send {"jsonrpc":"2.0","id":1,"method":"ping","params":null}`, []string{"diagnostic"}, []int{0}, []bool{true}},
		{"invalid cancellation", `mcp-send {"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":null}}`, []string{"diagnostic"}, []int{0}, []bool{true}},

		{"opposite initiators", `mcp-send {"jsonrpc":"2.0","id":1,"method":"a"}
mcp-recv {"jsonrpc":"2.0","id":1,"method":"b"}
mcp-recv {"jsonrpc":"2.0","id":1,"result":{}}
mcp-send {"jsonrpc":"2.0","id":1,"result":{}}`, []string{"request", "request", "response", "response"}, []int{0, 0, 1, 2}, nil},
		{"ID types", `mcp-send {"jsonrpc":"2.0","id":1,"method":"a"}
mcp-recv {"jsonrpc":"2.0","id":"1","result":{}}`, []string{"request", "response", "outstanding"}, []int{0, 0, 0}, []bool{false, true, true}},
		{"sessions", `mcp-send {"jsonrpc":"2.0","id":1,"method":"a"} # 1 session=a
mcp-send {"jsonrpc":"2.0","id":1,"method":"b"} # 1 session=b
mcp-recv {"jsonrpc":"2.0","id":1,"result":{}} # 2 session=b`, []string{"request", "request", "response", "outstanding"}, []int{0, 0, 2, 0}, nil},
		{"duplicate", `mcp-send {"jsonrpc":"2.0","id":1,"method":"a"}
mcp-send {"jsonrpc":"2.0","id":1,"method":"b"}
mcp-recv {"jsonrpc":"2.0","id":1,"result":{}}`, []string{"request", "request", "response", "outstanding"}, []int{0, 0, 0, 0}, []bool{false, true, true, true}},
		{"cancellation stays pending", `mcp-send {"jsonrpc":"2.0","id":1,"method":"a"}
mcp-send {"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":1}}
mcp-recv {"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":"cancelled"}}`, []string{"request", "cancellation", "response"}, []int{0, 1, 1}, nil},
		{"invalid shapes", `mcp-send {"jsonrpc":"2.0","id":1,"method":"a","result":{}}
mcp-recv {"jsonrpc":"2.0","id":1,"error":null}
mcp-recv [{"jsonrpc":"2.0","id":1,"result":{}}]`, []string{"diagnostic", "diagnostic", "diagnostic"}, []int{0, 0, 0}, []bool{true, true, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mcptrace.NewConversation(strings.NewReader(tt.input), "")
			for i, kind := range tt.kinds {
				e, err := c.Next()
				if err != nil {
					t.Fatal(err)
				}
				if e.Kind != kind || e.RequestRecord != tt.matches[i] {
					t.Fatalf("event %d: %#v", i, e)
				}
				if tt.diagnostics != nil && (e.Diagnostic != "") != tt.diagnostics[i] {
					t.Fatalf("diagnostic %d: %#v", i, e)
				}
			}
			if _, err := c.Next(); err != io.EOF {
				t.Fatalf("end: %v", err)
			}
		})
	}
}

func TestConversationTimeAndTruncation(t *testing.T) {
	for _, timed := range []bool{false, true} {
		input := "mcp-send {\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\"}"
		if timed {
			input += " # 1"
		}
		input += "\nmcp-recv {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}"
		if timed {
			input += " # 2"
		}
		c := mcptrace.NewConversation(strings.NewReader(input), "")
		_, _ = c.Next()
		e, err := c.Next()
		if err != nil || (e.Duration != nil) != timed {
			t.Fatalf("duration: %#v %v", e, err)
		}
	}
	c := mcptrace.NewConversation(strings.NewReader("mcp-send {\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\"}\nmcp-recv {"), "")
	_, _ = c.Next()
	if _, err := c.Next(); err == nil || err == io.EOF {
		t.Fatalf("truncation error: %v", err)
	}
}
