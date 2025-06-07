package main

import (
	"fmt"
	"os/exec"
)

func main() {
	// This simulates what a scripttest does - executing another program
	cmd := exec.Command("./simple_program", "arg1", "arg2")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Output: %s\n", output)
}

func runTest() {
	// Another function to show internal call graph
	fmt.Println("Running test...")
}
