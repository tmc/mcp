package main

import (
	"fmt"
	"reflect"

	"github.com/tmc/mcp/exp/sourcereflect"
)

type User struct {
	ID        int                   `json:"id"`
	Username  string                `json:"username"`
	Email     string                `json:"email"`
	Age       int                   `json:"age,omitempty"`
	IsActive  bool                  `json:"is_active"`
	Tags      []string              `json:"tags"`
	Settings  map[string]string     `json:"settings"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func main() {
	// Generate schema from type
	schema, err := sourcereflect.FromType(reflect.TypeOf(User{}))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Convert to pretty JSON
	jsonStr, err := schema.ToPrettyJSON()
	if err != nil {
		fmt.Printf("Error converting to JSON: %v\n", err)
		return
	}

	fmt.Println("Generated JSON Schema:")
	fmt.Println(jsonStr)

	// Generate schema with caller information
	schemaWithCaller, err := sourcereflect.SchemaFromCaller(User{})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	jsonWithCaller, err := schemaWithCaller.ToPrettyJSON()
	if err != nil {
		fmt.Printf("Error converting to JSON: %v\n", err)
		return
	}

	fmt.Println("\nSchema with caller information:")
	fmt.Println(jsonWithCaller)
}