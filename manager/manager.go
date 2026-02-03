package manager

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

type sessionManager struct {
	nextSessionID int
	sessionWg     sync.WaitGroup

	mu            sync.Mutex
	activeSession Session
	sessions      []Session
}

func New() *sessionManager {
	sm := &sessionManager{
		sessions:  []Session{},
		sessionWg: sync.WaitGroup{},
	}

	sm.runWindowSizeWatcher()
	sm.runUserStdInReader()

	return sm
}

func (sm *sessionManager) Create(argv []string) {
	ready := make(chan any, 1)
	// Do i need a separate thread here?
	sm.sessionWg.Go(func() {
		var err error

		// 1. Open PTY master and slave
		master, slave, err := pty.Open()
		if err != nil {
			log.Printf("failed to start session: %v", err)
			return
		}
		defer master.Close()

		// 2. Fork
		cmd := exec.Command(argv[0], argv[1:]...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
		cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave

		if err := cmd.Start(); err != nil {
			_ = master.Close()
			_ = slave.Close()
			log.Printf("failed to start session: %v", err)
			return
		}
		_ = slave.Close()

		s := &PtySession{Cmd: cmd, Master: master, ID: sm.nextSessionID}
		sm.nextSessionID++
		sm.sessions = append(sm.sessions, s)
		sm.selectSession(s)
		sm.runSessionStdOutReader(s)
		ready <- struct{}{} // notify session is ready

		err = s.Cmd.Wait()
		if err != nil {
			log.Printf("session %d ended with error: %v", s.ID, err)
		}

		sm.cleanup(s)
	})

	<-ready
}

func (sm *sessionManager) selectSession(s Session) {
	sm.mu.Lock()
	sm.activeSession = s
	sm.mu.Unlock()
	sm.clearPtyScreen()
}

func (sm *sessionManager) runSessionStdOutReader(s Session) {
	go func() {
		tmp := make([]byte, 4096)

		for {
			n, err := s.Read(tmp)
			if err != nil {
				return
			}

			data := bytes.Clone(tmp[:n])
			s.Write(data)

			sm.mu.Lock()
			if sm.activeSession == s {
				_, _ = os.Stdout.Write(data)
			}
			sm.mu.Unlock()
		}
	}()
}

func (sm *sessionManager) next() {
	if len(sm.sessions) < 1 {
		return
	}

	firstSessionID := 0
	lastSessionID := len(sm.sessions) - 1
	currentSessionID := 0
	for i := range sm.sessions {
		if sm.sessions[i] == sm.activeSession {
			currentSessionID = i + 1

			if currentSessionID > lastSessionID {
				currentSessionID = firstSessionID
			}

			break
		}
	}

	sm.selectSession(sm.sessions[currentSessionID])
}

func (sm *sessionManager) runUserStdInReader() {
	go func() {
		buf := make([]byte, 1024)

		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}

			// If Ctrl-A is detected, switch to command mode
			sm.parseInput(buf[:n])
			// else send to active session
		}
	}()
}

func (sm *sessionManager) parseInput(data []byte) {
	out := data[:0]

	for _, b := range data {
		if b == 0x01 {
			sm.next()
		} else {
			out = append(out, b)
		}
	}

	sm.mu.Lock()
	activeSession := sm.activeSession
	sm.mu.Unlock()

	if len(out) > 0 && activeSession != nil {
		_, _ = activeSession.(*PtySession).Master.Write(out)
	}
}

/* utils */

func (sm *sessionManager) cleanup(s Session) {
	sm.mu.Lock()
	for i, sess := range sm.sessions {
		if sess == s {
			sm.sessions = append(sm.sessions[:i], sm.sessions[i+1:]...)
			break
		}
	}

	if len(sm.sessions) > 0 {
		sm.activeSession = sm.sessions[0]
		sm.clearPtyScreen()
	}
	sm.mu.Unlock()
}

func (sm *sessionManager) Wait() {
	sm.sessionWg.Wait()
}

func (sm *sessionManager) clearPtyScreen() {
	if ps, ok := sm.activeSession.(*PtySession); ok {
		ps.ClearAndRedraw()
	}
}

func (sm *sessionManager) runWindowSizeWatcher() {
	// Propagate resizes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if ps, ok := sm.activeSession.(*PtySession); ok {
				ps.ClearAndRedraw()
			}
		}
	}()
}
