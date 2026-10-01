package mcpcorpus_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/tmc/mcp/mcpcorpus"
)

func Example() {
	trace := `mcp-send {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"q":"a"}}}`
	seeds, err := mcpcorpus.Extract(strings.NewReader(trace), "search")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(len(seeds), string(seeds[0].Arguments))
	// Output: 1 {"q":"a"}
}
func ExampleExtract() {
	seeds, err := mcpcorpus.Extract(strings.NewReader(`mcp-send {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{}}}`), "search")
	fmt.Println(len(seeds), err)
	// Output: 1 <nil>
}
func ExampleWriteFuzz() {
	err := mcpcorpus.WriteFuzz(os.Stdout, mcpcorpus.Seed{Arguments: []byte(`{"q":"a"}`)})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// go test fuzz v1
	// []byte("{\"q\":\"a\"}")
}
func ExampleWriteJSON() {
	err := mcpcorpus.WriteJSON(os.Stdout, mcpcorpus.Seed{Arguments: []byte(`{ "q": "a" }`)})
	if err != nil {
		fmt.Println(err)
	}
	// Output: { "q": "a" }
}
func ExampleSeed() {
	seed := mcpcorpus.Seed{Arguments: []byte(`{}`)}
	fmt.Println(string(seed.Arguments))
	// Output: {}
}
