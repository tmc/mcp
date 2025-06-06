// Package sourcereflect provides functionality to generate JSON schemas from Go types
// using reflection. It supports both compile-time type information and runtime caller
// context to enrich schemas with source location metadata.
//
// Basic usage:
//
//	type User struct {
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	    Age   int    `json:"age,omitempty"`
//	}
//
//	schema, err := sourcereflect.FromType(reflect.TypeOf(User{}))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	jsonStr, _ := schema.ToPrettyJSON()
//	fmt.Println(jsonStr)
//
// Using caller context:
//
//	schema, err := sourcereflect.SchemaFromCaller(User{})
//	// schema will include source location metadata
//
// The package supports:
// - Basic Go types (string, int, float, bool)
// - Structs with JSON tags
// - Slices and arrays
// - Maps with string keys
// - Nested types
// - Optional fields (using omitempty tag)
// - Source location tracking
package sourcereflect
