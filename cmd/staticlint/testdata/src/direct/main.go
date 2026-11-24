package main

import "os"

func main() {
	os.Exit(1) // want "direct call to os.Exit from main"
}

func other() {
	os.Exit(2)
}
