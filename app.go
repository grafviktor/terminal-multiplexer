package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"pty-wrapper/manager"
	"syscall"

	"golang.org/x/term"
)

func main() {
	sm := manager.New()
	session, err := sm.Create([]string{"bash", "-c", "while true; do date; sleep 1 ;done"})
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	sm.Select(session)

	session, err = sm.Create([]string{"/bin/zsh"})
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	sm.Select(session)

	// Put terminal into raw mode
	if term.IsTerminal(int(os.Stdin.Fd())) {
		old, _ := term.MakeRaw(int(os.Stdin.Fd()))
		dyingSignal := make(chan os.Signal, 1)
		signal.Notify(dyingSignal, os.Interrupt, syscall.SIGTERM)

		// Restore terminal on interrupt or terminate
		go func() {
			<-dyingSignal
			term.Restore(int(os.Stdin.Fd()), old)
			os.Exit(0)
		}()
	}

	// Wait for all sessions to finish
	sm.Wait()
	fmt.Println("All sessions finished. Exiting.")
}
