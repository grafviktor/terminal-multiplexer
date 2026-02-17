package manager

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

const hotKey = 0x01 // Ctrl-A

type sessionManager struct {
	nextSessionID int
	sessionWg     sync.WaitGroup
	uiDirty       chan Session
	isModeAppCursor bool
	isModeAppKeypad bool

	mu            sync.Mutex
	activeSession Session
	sessions      []Session
}

func New() *sessionManager {
	sm := &sessionManager{
		sessions:  []Session{},
		sessionWg: sync.WaitGroup{},
		uiDirty:   make(chan Session, 1),
	}

	sm.createServicePane()
	sm.runWindowSizeWatcher()
	sm.runStdInReader()
	sm.runRenderer()

	return sm
}

func (sm *sessionManager) createServicePane() {
	rows, cols := sm.getSize()
	p := &StatusSession{cols: cols, rows: rows}
	sm.sessions = append(sm.sessions, p)
	sm.Select(p)
}

func (sm *sessionManager) Create(argv []string) (Session, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	session, err := NewPtySession(sm.nextSessionID, cmd)
	if err != nil {
		return nil, err
	}

	rows, cols := sm.getSize()
	session.SetSize(cols, rows)
	sm.nextSessionID++
	sm.sessions = append(sm.sessions, session)
	sm.runStdOutReader(session)

	ready := make(chan any, 1)
	sm.sessionWg.Go(func() {
		ready <- struct{}{}
		waitErr := cmd.Wait()
		if waitErr != nil {
			log.Printf("session %d ended with error: %v", session.ID, waitErr)
		}
		defer sm.close(session)
	})

	// Wait session goroutine to start
	<-ready
	return session, nil
}

func (sm *sessionManager) Select(s Session) {
	if ss, ok := s.(*StatusSession); ok {
		sessionInfo := SessionInfo{
			sessionCount: len(sm.sessions) - 1,
		}
		ss.Refresh(sessionInfo) // exclude status session
	}

	sm.mu.Lock()
	sm.activeSession = s
	sm.mu.Unlock()

	sm.render(true)
}

func (sm *sessionManager) runStdInReader() {
	go func() {
		buf := make([]byte, 1024)

		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}

			sm.parseStdIn(buf[:n])
			sm.uiDirty <- sm.activeSession
		}
	}()
}

func (sm *sessionManager) runStdOutReader(s Session) {
	go func() {
		tmp := make([]byte, 4096)

		for {
			n, err := s.Read(tmp)
			if err != nil {
				return
			}

			data := bytes.Clone(tmp[:n])
			s.WriteBackground(data)
			sm.uiDirty <- s
		}
	}()
}

func (sm *sessionManager) runWindowSizeWatcher() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			for _, s := range sm.sessions {
				rows, cols := sm.getSize()
				s.SetSize(cols, rows)
			}

			sm.activeSession.Render()
		}
	}()
}

func (sm *sessionManager) runRenderer() {
	go func() {
		for {
			// Wait for something to be written to the screen
			s := <-sm.uiDirty

			for {
				select {
				case s = <-sm.uiDirty:
				default:
					goto RENDER
				}
			}

		RENDER:
			if s == sm.activeSession {
				sm.render(false)
			}
		}
	}()
}

func (sm *sessionManager) render(shouldInvalidate bool) {
	sm.mu.Lock()
	s := sm.activeSession
	sm.mu.Unlock()
	// Clears the previous screen of the session.
	if shouldInvalidate {
		s.Invalidate()
	}

	sm.setTerminalMode(s)
	s.Render()
}

func (sm *sessionManager) setTerminalMode(s Session) {
	ptySession, ok := s.(*PtySession)
	if ok {
		mode := ptySession.Term.Mode()
		isModeAppCursor := mode & vt10x.ModeAppCursor != 0
		isModeAppKeypad := mode & vt10x.ModeAppKeypad != 0

		if sm.isModeAppCursor != isModeAppCursor {
			sm.isModeAppCursor = isModeAppCursor
			if isModeAppCursor {
				sm.enableAppCursorMode()
			} else {
				sm.disableAppCursorMode()
			}
		}

		if sm.isModeAppKeypad != isModeAppKeypad {
			sm.isModeAppKeypad = isModeAppKeypad
			if isModeAppKeypad {
				sm.enableAppKeypadMode()
			} else {
				sm.disableAppKeypadMode()
			}
		}
	}
}

func (sm *sessionManager) enableAppCursorMode() {
	fmt.Print("\x1b[?1h")
}

func (sm *sessionManager) disableAppCursorMode() {
	fmt.Print("\x1b[?1l")
}

func (sm *sessionManager) enableAppKeypadMode() {
	fmt.Print("\x1b=")
}

func (sm *sessionManager) disableAppKeypadMode() {
	fmt.Print("\x1b")
}

func (sm *sessionManager) next() {
	if len(sm.sessions) < 1 {
		return
	}

	lastSessionID := len(sm.sessions) - 1
	currentSessionID := 0
	for i := range sm.sessions {
		if sm.sessions[i] == sm.activeSession {
			currentSessionID = i + 1

			if currentSessionID > lastSessionID {
				currentSessionID = 0
			}

			break
		}
	}

	sm.Select(sm.sessions[currentSessionID])
	sm.uiDirty <- sm.activeSession
}

func (sm *sessionManager) parseStdIn(data []byte) {
	input := data[:0]

	for _, b := range data {
		if sm.isHotkey(rune(b)) {
			// If Ctrl-A is detected, switch to command mode
			sm.handleHotkey(rune(b))
		} else {
			// else send to active session
			input = append(input, b)
		}
	}

	if len(input) > 0 {
		sm.mu.Lock()
		sm.activeSession.Write(input)
		sm.mu.Unlock()
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
	sm.mu.Unlock()

	s.Close()
	sm.next()
}

func (sm *sessionManager) Wait() {
	sm.sessionWg.Wait()
}

func (sm *sessionManager) getSize() (int, int) {
	rows, cols, err := pty.Getsize(os.Stdin)
	if err != nil {
		fmt.Printf("Cannot terminal size %v\n", err)
	}

	return rows, cols
}
