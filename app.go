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
	pane, err := sm.Create(manager.PanePositionEnum.FullScreen, []string{"bash", "-c", "while true; do date; sleep 1 ;done"})
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	sm.Select(pane)

	pane, err = sm.Create(manager.PanePositionEnum.FullScreen, []string{"/bin/zsh"})
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	sm.Select(pane)

	pane, err = sm.Create(manager.PanePositionEnum.Left, []string{"/bin/zsh"})
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	sm.Select(pane)

	pane, err = sm.Create(manager.PanePositionEnum.Right, []string{"/bin/zsh"})
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	sm.Select(pane)

	// Put terminal into raw mode
	terminalPtr := int(os.Stdin.Fd())
	if term.IsTerminal(terminalPtr) {
		oldState, _ := term.MakeRaw(terminalPtr)
		defer restoreTerminal(terminalPtr, oldState)
		runForceExitHandler(terminalPtr, oldState)
	}

	sm.Wait()
	fmt.Println("All sessions finished. Exiting.")
}

func runForceExitHandler(terminalPtr int, oldState *term.State) {
	killSignal := make(chan os.Signal, 1)
	signal.Notify(killSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-killSignal
		fmt.Println("Killed. Exiting.")
		restoreTerminal(terminalPtr, oldState)
		os.Exit(1)
	}()
}

func restoreTerminal(terminalPtr int, oldState *term.State) {
	term.Restore(terminalPtr, oldState)
}
