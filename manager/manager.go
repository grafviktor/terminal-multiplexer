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

	mu            sync.Mutex
	activeSession Session
	sessions      []Session
	uiDirty       chan struct{}
}

func New() *sessionManager {
	sm := &sessionManager{
		sessions:  []Session{},
		sessionWg: sync.WaitGroup{},
		uiDirty:   make(chan struct{}, 1),
	}

	sm.createServicePane()
	sm.runWindowSizeWatcher()
	sm.runStdInReader()
	sm.runRenderer()

	return sm
}

func (sm *sessionManager) createServicePane() {
	p := &StatusSession{}
	sm.sessions = append(sm.sessions, p)
	sm.Select(p)
}

func (sm *sessionManager) Create(argv []string) (Session, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	session := &PtySession{Ptmx: ptmx, Term: vt10x.New(), ID: sm.nextSessionID}
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
		defer func() {
			ptmx.Close()
			sm.close(session)
		}()
	})

	// Wait session goroutine to start
	<-ready
	return session, nil
}

func (sm *sessionManager) Select(s Session) {
	if ss, ok := s.(*StatusSession); ok {
		if ps, ok := sm.activeSession.(*PtySession); ok {
			w, h := ps.Term.Size()
			sessionInfo := SessionInfo{
				sessionCount: len(sm.sessions) - 1,
				width:        w,
				height:       h,
			}
			ss.Refresh(sessionInfo) // exclude status session
		}
	}

	sm.mu.Lock()
	sm.activeSession = s
	sm.mu.Unlock()

	sm.render()
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
			sm.uiDirty <- struct{}{}
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
			if ps, ok := s.(*PtySession); ok {
				ps.Term.Write(data)
			}
			sm.uiDirty <- struct{}{}
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
	// full channel drain (latest-wins, burst-safe)
	go func() {
		for {
			// Wait for something to be written to the screen
			<-sm.uiDirty

			for {
				select {
				case <-sm.uiDirty:
				default:
					goto RENDER
				}
			}

		RENDER:
			sm.render()
		}
	}()
}

func (sm *sessionManager) render() {
	sm.mu.Lock()
	s := sm.activeSession
	sm.mu.Unlock()
	s.Render()
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
	sm.uiDirty <- struct{}{}
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
