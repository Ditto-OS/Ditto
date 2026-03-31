package main

import "fmt"

func main() {
	// Basic output
	fmt.Println("Hello from Go!")
	fmt.Println("Running on Ditto embedded interpreter")

	// Variables
	name := "Ditto"
	version := "0.1.0"
	fmt.Println("Project:", name, "v"+version)

	// Arithmetic
	a := 10
	b := 20
	fmt.Println("Sum:", a+b)

	// Array
	colors := []string{"red", "green", "blue"}
	fmt.Println("Colors:", len(colors), "items")

	// Loop
	for i := 0; i < 3; i++ {
		fmt.Println("Counting:", i)
	}

	// If statement
	x := 10
	if x > 5 {
		fmt.Println("x is greater than 5")
	}
}
