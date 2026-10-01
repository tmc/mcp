package mcptest_test

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcptest"
)

func Example() {
	server := mcp.NewServer(&mcp.Implementation{Name: "example", Version: "1"}, nil)
	client := mcp.NewClient(&mcp.Implementation{Name: "example", Version: "1"}, nil)
	fixture, err := mcptest.New(context.Background(), server, client)
	if err != nil {
		panic(err)
	}
	defer fixture.Close(context.Background())
	tools, err := fixture.Client.ListTools(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(tools.Tools))
	// Output: 0
}

func ExampleNew() {
	server, client := peers()
	fixture, err := mcptest.New(context.Background(), server, client)
	if err != nil {
		panic(err)
	}
	defer fixture.Close(context.Background())
	fmt.Println(fixture.Client != nil)
	// Output: true
}

func ExampleFixture_Close() {
	server, client := peers()
	fixture, err := mcptest.New(context.Background(), server, client)
	if err != nil {
		panic(err)
	}
	fmt.Println(fixture.Close(context.Background()))
	// Output: <nil>
}

// exampleTest stands in for testing.T in this runnable example.
type exampleTest struct{ cleanup []func() }

func (*exampleTest) Helper()                           {}
func (t *exampleTest) Cleanup(f func())                { t.cleanup = append(t.cleanup, f) }
func (*exampleTest) Fatalf(format string, args ...any) { panic(fmt.Sprintf(format, args...)) }
func (*exampleTest) Errorf(format string, args ...any) { panic(fmt.Sprintf(format, args...)) }

func ExampleConnect() {
	t := new(exampleTest) // In a Test function, use its *testing.T directly.
	defer func() {
		for i := len(t.cleanup) - 1; i >= 0; i-- {
			t.cleanup[i]()
		}
	}()
	server, client := peers()
	fixture := mcptest.Connect(t, server, client)
	tools, err := fixture.Client.ListTools(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(tools.Tools))
	// Output: 0
}
