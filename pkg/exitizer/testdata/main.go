package main

import "os"

// A
type A struct{}

// Exit
func (a A) Exit() {}

// Exit
func Exit() {}

func main() {
	os.Exit(1) // want "os.Exit call"
	Exit()
	a := A{}
	a.Exit()
}
