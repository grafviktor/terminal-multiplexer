package main

import (
	"os"
	"pty-wrapper/manager"

	"golang.org/x/term"
)

func main() {
	sm := manager.New()
	sm.Create([]string{"/bin/bash"})
	sm.Create([]string{"/bin/zsh"})

	// Put terminal into raw mode
	if term.IsTerminal(int(os.Stdin.Fd())) {
		old, _ := term.MakeRaw(int(os.Stdin.Fd()))
		defer term.Restore(int(os.Stdin.Fd()), old)
	}

	// Wait for all sessions to finish
	sm.Wait()
}
