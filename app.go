package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"pty-wrapper/manager"
	"syscall"
	"time"

	"golang.org/x/term"

	"net/http"
	_ "net/http/pprof"
)

func main() {
	startPprof()
	startGoroutineLeak()

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
	dyingSignal := make(chan os.Signal, 1)
	signal.Notify(dyingSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-dyingSignal
		fmt.Println("Killed. Exiting.")
		restoreTerminal(terminalPtr, oldState)
	}()
}

func restoreTerminal(terminalPtr int, oldState *term.State) {
	term.Restore(terminalPtr, oldState)
}

func startPprof() {
	// go install github.com/goccy/go-graphviz/cmd/dot@latest
	// go tool pprof http://localhost:6060/debug/pprof/goroutineleak
	// open your browser and navigate to http://localhost:6060/debug/pprof/ to see available profiles
	// then run the command below to check a specific profile, for example goroutineleak
	// go tool pprof -http=:8080 http://localhost:6060/debug/pprof/goroutineleak
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}

func startGoroutineLeak() {
	// Short burst; repeat a few times so GC has something to find.
	for range 100 {
		ch := make(chan struct{}) // local, will go out of scope
		go func() {
			// Block forever trying to send; no receiver exists and 'c' will become unreachable.
			ch <- struct{}{}
		}()
	}
	// Give them a moment to start and block.
	time.Sleep(200 * time.Millisecond)
}
