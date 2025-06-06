package sourcereflect_test

import (
	"fmt"
	"reflect"

	sourcereflect "github.com/tmc/mcp/exp/sourcereflect"
)

func ExampleFromType() {
	type Person struct {
		Name  string `json:"name"`
		Age   int    `json:"age,omitempty"`
		Email string `json:"email"`
	}

	schema, err := sourcereflect.FromType(reflect.TypeOf(Person{}))
	if err != nil {
		panic(err)
	}

	json, _ := schema.ToPrettyJSON()
	fmt.Println(json)
}

func ExampleFromValue() {
	user := struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{
		ID:   1,
		Name: "Alice",
	}

	schema, err := sourcereflect.FromValue(user)
	if err != nil {
		panic(err)
	}

	fmt.Println(schema.Type)
	fmt.Println(len(schema.Properties))
	// Output:
	// object
	// 2
}

func ExampleSchemaFromCaller() {
	type Config struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	schema, err := sourcereflect.SchemaFromCaller(Config{})
	if err != nil {
		panic(err)
	}

	// Schema includes source location information
	if loc, ok := schema.Additional["$sourceLocation"]; ok {
		fmt.Println("Has source location:", loc != nil)
	}
	// Output:
	// Has source location: true
}

func ExampleSchemaBuilder() {
	schema := sourcereflect.NewSchemaBuilder().
		WithType("object").
		WithTitle("User").
		WithProperty("username", &sourcereflect.Schema{Type: "string"}).
		WithProperty("email", &sourcereflect.Schema{Type: "string", Format: "email"}).
		WithRequired("username", "email").
		Build()

	fmt.Println(schema.Type)
	fmt.Println(schema.Title)
	fmt.Println(len(schema.Required))
	// Output:
	// object
	// User
	// 2
}
