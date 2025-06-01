package main

import (
	"encoding/json"
	"fmt"
	"reflect"

	"golang.org/x/exp/jsonrpc2"
)

func main() {
	// Create different ID types
	ids := []jsonrpc2.ID{
		{}, // empty
		jsonrpc2.Int64ID(1),
		jsonrpc2.StringID("test"),
	}

	for i, id := range ids {
		fmt.Printf("\nID %d:\n", i)
		fmt.Printf("  IsValid: %v\n", id.IsValid())
		fmt.Printf("  Raw: %v\n", id.Raw())
		fmt.Printf("  Type: %v\n", reflect.TypeOf(id.Raw()))

		// Check JSON marshaling
		data, err := json.Marshal(id)
		fmt.Printf("  JSON: %s (err: %v)\n", string(data), err)

		// Check the internal structure
		v := reflect.ValueOf(id)
		t := v.Type()
		fmt.Printf("  Struct fields:\n")
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)
			fmt.Printf("    %s: %v (exported: %v)\n", field.Name, value, field.IsExported())
		}
	}

	// Check if ID implements json.Marshaler
	id := jsonrpc2.ID{}
	if _, ok := interface{}(id).(json.Marshaler); ok {
		fmt.Println("\nID implements json.Marshaler")
	} else {
		fmt.Println("\nID does NOT implement json.Marshaler")
	}
}
