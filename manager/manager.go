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

const hotKey = 0x01 // Ctrl-A

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

	sm.createServicePane()
	sm.runWindowSizeWatcher()
	sm.runInputReader()

	return sm
}

func (sm *sessionManager) createServicePane() {
	p := &StatusSession{}
	sm.sessions = append(sm.sessions, p)
	sm.selectSession(p)
}

func (sm *sessionManager) Create(argv []string) {
	var err error

	// 1. Open PTY master and slave
	master, slave, err := pty.Open()
	if err != nil {
		log.Printf("failed to start session: %v", err)
		return
	}
	defer slave.Close()

	// 2. Fork
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave

	if err = cmd.Start(); err != nil {
		log.Printf("failed to start session: %v", err)
		master.Close()
		return
	}

	s := &PtySession{Cmd: cmd, Master: master, ID: sm.nextSessionID}
	sm.nextSessionID++
	sm.sessions = append(sm.sessions, s)
	sm.selectSession(s)
	sm.runSessionOutputReader(s)

	ready := make(chan any, 1)
	sm.sessionWg.Go(func() {
		ready <- struct{}{}
		err = s.Cmd.Wait()
		if err != nil {
			log.Printf("session %d ended with error: %v", s.ID, err)
		}
		defer func() {
			master.Close()
			sm.close(s)
		}()
	})

	// Wait goroutine to start
	<-ready
}

func (sm *sessionManager) selectSession(s Session) {
	sm.mu.Lock()
	sm.activeSession = s
	sm.mu.Unlock()

	if ss, ok := s.(*StatusSession); ok {
		ss.Refresh(len(sm.sessions) - 1) // exclude status session
	}

	sm.clearPtyScreen()
}

func (sm *sessionManager) runSessionOutputReader(s Session) {
	go func() {
		tmp := make([]byte, 4096)

		for {
			n, err := s.Read(tmp)
			if err != nil {
				return
			}

			data := bytes.Clone(tmp[:n])
			s.WriteToBuffer(data)

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

func (sm *sessionManager) runInputReader() {
	go func() {
		buf := make([]byte, 1024)

		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}

			sm.parseInput(buf[:n])
		}
	}()
}

func (sm *sessionManager) parseInput(data []byte) {
	out := data[:0]

	for _, b := range data {
		if sm.isHotkey(rune(b)) {
			// If Ctrl-A is detected, switch to command mode
			sm.handleHotkey(rune(b))
		} else {
			// else send to active session
			out = append(out, b)
		}
	}

	sm.mu.Lock()
	activeSession := sm.activeSession
	sm.mu.Unlock()

	if len(out) > 0 {
		if s, ok := activeSession.(*PtySession); ok {
			s.Write(out)
		}
	}
}

func (sm *sessionManager) isHotkey(char rune) bool {
	return char == hotKey
}

func (sm *sessionManager) handleHotkey(hotKey rune) {
	if hotKey == 0x01 {
		sm.next()
	}
}

func (sm *sessionManager) close(s Session) {
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
