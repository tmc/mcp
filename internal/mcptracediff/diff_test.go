package mcptracediff

import (
	"fmt"
	"strings"
	"testing"
)

const baseline = `mcp-send {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"n":9007199254740993}}}
mcp-recv {"jsonrpc":"2.0","id":1,"result":{"content":[],"n":9007199254740993}}
`

func TestCompare(t *testing.T) {
	tests := []struct {
		name, left, right string
		want              int
		err               string
		timing            bool
	}{
		{"IDs ignored", baseline, strings.ReplaceAll(baseline, `"id":1`, `"id":"other"`), 0, "", false},
		{"large integer result", baseline, strings.Replace(baseline, `"content":[],"n":9007199254740993`, `"content":[],"n":9007199254740992`, 1), 1, "", false},
		{"large integer argument", baseline, strings.Replace(baseline, `"n":9007199254740993`, `"n":9007199254740992`, 1), 2, "", false},
		{"string number distinct", baseline, strings.Replace(baseline, `"n":9007199254740993`, `"n":"9007199254740993"`, 1), 2, "", false},
		{"error result", baseline, strings.Replace(baseline, `"result":{"content":[],"n":9007199254740993}`, `"error":{"code":-1,"message":"failed"}`, 1), 1, "", false},
		{"outstanding", baseline, strings.Split(baseline, "\n")[0], 0, "no response observed", false},
		{"truncated", baseline, baseline + "mcp-recv {", 0, "incomplete or invalid", false},
		{"no timing by default", baseline, baseline, 0, "", false},
		{"missing time requested", baseline, baseline, 0, "duration unavailable", true},
		{"repeated sequential", baseline + baseline, baseline + baseline, 0, "", false},
		{"unequal repeated count", baseline + baseline, baseline, 1, "", false},
		{"duplicate JSON", baseline, strings.Replace(baseline, `"method":"tools/call"`, `"method":"ping","method":"tools/call"`, 1), 0, "duplicate JSON", false},
		{"notifications ignored", baseline, baseline + `mcp-recv {"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`, 0, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			differences, err := Compare(strings.NewReader(tt.left), strings.NewReader(tt.right), Options{Timing: tt.timing})
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("error=%v want %q", err, tt.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(differences) != tt.want {
				t.Fatalf("differences=%#v", differences)
			}
		})
	}
}

func TestConcurrentAndOppositeDirections(t *testing.T) {
	trace := `mcp-send {"jsonrpc":"2.0","id":1,"method":"a","params":{"n":1}}
mcp-send {"jsonrpc":"2.0","id":2,"method":"a","params":{"n":2}}
mcp-recv {"jsonrpc":"2.0","id":1,"method":"a","params":{"n":1}}
mcp-recv {"jsonrpc":"2.0","id":2,"result":{"n":2}}
mcp-send {"jsonrpc":"2.0","id":1,"result":{"n":3}}
mcp-recv {"jsonrpc":"2.0","id":1,"result":{"n":1}}`
	if differences, err := Compare(strings.NewReader(trace), strings.NewReader(trace), Options{}); err != nil || len(differences) != 0 {
		t.Fatalf("out of order opposite calls: %#v %v", differences, err)
	}
	ambiguous := strings.Replace(trace, `"id":2,"method":"a","params":{"n":2}`, `"id":2,"method":"a","params":{"n":1}`, 1)
	if _, err := Compare(strings.NewReader(trace), strings.NewReader(ambiguous), Options{}); err == nil || !strings.Contains(err.Error(), "overlapping identical") {
		t.Fatalf("ambiguity=%v", err)
	}
}

func TestSessionsAndTiming(t *testing.T) {
	timed := strings.ReplaceAll(baseline, "}\n", "} # 1 session=a\n")
	changed := strings.Replace(timed, `"n":9007199254740993}} # 1 session=a`, `"n":9007199254740993}} # 2 session=a`, 1)
	if _, err := Compare(strings.NewReader(timed), strings.NewReader(changed), Options{Timing: true}); err != nil {
		t.Fatal(err)
	}
	other := strings.ReplaceAll(timed, "session=a", "session=b")
	if differences, err := Compare(strings.NewReader(timed), strings.NewReader(other), Options{}); err != nil || len(differences) != 2 {
		t.Fatalf("session identity: %#v %v", differences, err)
	}
	nanos := strings.Replace(timed, "# 1 session=a", "# 1.000000001 session=a", 1)
	if differences, err := Compare(strings.NewReader(timed), strings.NewReader(nanos), Options{}); err != nil || len(differences) != 0 {
		t.Fatalf("clock independent comparison: %#v %v", differences, err)
	}
	if differences, err := Compare(strings.NewReader(timed), strings.NewReader(nanos), Options{Timing: true}); err == nil || !strings.Contains(err.Error(), "precedes request") {
		t.Fatalf("nanosecond negative duration: %#v %v", differences, err)
	}
}

func TestCanonicalNumbers(t *testing.T) {
	for _, pair := range [][2]string{{"1", "1.0"}, {"1e1000000000", "10e999999999"}, {"-0", "0"}, {"100", "1e2"}, {"100", "1e+2"}, {"0.001", "1e-3"}} {
		a, err := canonical([]byte(pair[0]))
		if err != nil {
			t.Fatal(err)
		}
		b, err := canonical([]byte(pair[1]))
		if err != nil || a != b {
			t.Fatalf("%v: %s %s %v", pair, a, b, err)
		}
	}
}

func ExampleCompare() {
	differences, err := Compare(strings.NewReader(baseline), strings.NewReader(baseline), Options{})
	fmt.Println(len(differences), err)
	// Output: 0 <nil>
}

func ExampleOptions() {
	options := Options{Session: "demo", Timing: false}
	fmt.Println(options.Session, options.Timing)
	// Output: demo false
}

func ExampleDifference() {
	difference := Difference{Kind: "result", Method: "tools/call", LeftRecord: 4, RightRecord: 4}
	fmt.Println(difference.Kind, difference.Method)
	// Output: result tools/call
}
