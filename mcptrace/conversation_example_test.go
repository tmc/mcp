package mcptrace_test

import (
	"fmt"
	"strings"

	"github.com/tmc/mcp/mcptrace"
)

func ExampleConversation() {
	conversation := mcptrace.NewConversation(strings.NewReader(`mcp-send {"jsonrpc":"2.0","method":"notifications/initialized"}`), "one")
	event, _ := conversation.Next()
	fmt.Println(event.Kind)
	// Output: notification
}

func ExampleConversation_Next() {
	conversation := mcptrace.NewConversation(strings.NewReader(`mcp-send {"jsonrpc":"2.0","id":"a","method":"ping"}`), "one")
	_, _ = conversation.Next()
	event, _ := conversation.Next()
	fmt.Println(event.Kind, event.Diagnostic)
	// Output: outstanding no response observed before end of input
}

func ExampleEvent() {
	event := mcptrace.Event{Kind: "notification", Direction: "recv", Record: 1, Method: "notifications/initialized"}
	fmt.Println(event.Method)
	// Output: notifications/initialized
}

func ExampleEvent_String() {
	event := mcptrace.Event{Kind: "notification", Direction: "recv", Record: 1, Method: "notifications/initialized"}
	fmt.Println(event.String())
	// Output: 1 recv notification notifications/initialized
}
