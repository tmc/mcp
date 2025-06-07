package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Simple program running...")
	processArgs()
}

func processArgs() {
	fmt.Printf("Args: %v\n", os.Args[1:])
}

func helperFunc() {
	fmt.Println("Helper function")
}
