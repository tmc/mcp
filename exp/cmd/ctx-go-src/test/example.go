package test

import "fmt"

// Example function with a txtar-like comment
func Example() {
	// This looks like -- a header --
	fmt.Println("Hello")

	/* Even this:
	-- multiline.go --
	should be escaped
	*/
}

// -- this_should_be_escaped.go --
func ShouldEscape() {
	fmt.Println("-- another file --")
}
