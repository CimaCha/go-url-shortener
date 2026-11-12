package main

import system "os"

func main() {
	system.Exit(1) // want "direct call to os.Exit from main"
	fake := struct{ Exit func(int) }{Exit: func(int) {}}
	fake.Exit(2)
}
