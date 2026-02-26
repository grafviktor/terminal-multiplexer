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
)

const hotKey = 0x01 // Ctrl-A

type sessionManager struct {
	nextSessionID int
	sessionWg     sync.WaitGroup
	uiDirty       chan struct{}

	mu            sync.Mutex
	activePane    *Pane
	panes         []*Pane
	panePosionMap map[int]PanePosition
}

func New() *sessionManager {
	sm := &sessionManager{
		panes:         []*Pane{},
		sessionWg:     sync.WaitGroup{},
		uiDirty:       make(chan struct{}, 1),
		panePosionMap: make(map[int]PanePosition),
	}

	sm.createServicePane()
	sm.runWindowSizeWatcher()
	sm.runStdInReader()
	sm.runRenderer()

	return sm
}

func (sm *sessionManager) createServicePane() {
	rows, cols := sm.getSize(PanePositionEnum.FullScreen)
	s := &StatusSession{cols: cols, rows: rows}
	p := &Pane{ID: sm.nextSessionID, Session: s}
	sm.panes = append(sm.panes, p)
	sm.Select(p)
}

func (sm *sessionManager) Create(position PanePosition, argv []string) (*Pane, error) {
	rows, cols := sm.getSize(position)
	cmd := exec.Command(argv[0], argv[1:]...)
	session, err := NewPtySession(sm.nextSessionID, cmd)
	if err != nil {
		return nil, err
	}

	xOffset := 0
	if position == PanePositionEnum.Right {
		xOffset += cols
	}

	// session.SetSize(cols, rows)
	p := NewPane(session.ID, session, cols, rows, xOffset, 0)
	sm.panes = append(sm.panes, p)
	sm.panePosionMap[session.ID] = position
	sm.runStdOutReader(*p)

	ready := make(chan any, 1)
	sm.sessionWg.Go(func() {
		ready <- struct{}{}
		waitErr := cmd.Wait()
		if waitErr != nil {
			log.Printf("session %d ended with error: %v", session.ID, waitErr)
		}
		defer sm.close(*p)
	})

	// Wait session goroutine to start
	<-ready
	sm.nextSessionID++
	return p, nil
}

func (sm *sessionManager) Select(p *Pane) {
	if ss, ok := p.Session.(*StatusSession); ok {
		sessionInfo := SessionInfo{
			sessionCount: len(sm.panes) - 1,
		}
		ss.Refresh(sessionInfo) // exclude status session
	}

	sm.mu.Lock()
	sm.activePane = p
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
			sm.uiDirty <- struct{}{}
		}
	}()
}

func (sm *sessionManager) runStdOutReader(p Pane) {
	go func() {
		tmp := make([]byte, 4096)

		for {
			n, err := p.Session.Read(tmp)
			if err != nil {
				return
			}

			data := bytes.Clone(tmp[:n])
			p.Session.WriteBackground(data)
			sm.uiDirty <- struct{}{}
		}
	}()
}

func (sm *sessionManager) runWindowSizeWatcher() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			for _, p := range sm.panes {
				rows, cols := sm.getSize(sm.panePosionMap[p.ID])
				p.SetSize(cols, rows)
			}

			sm.mu.Lock()
			activePane := sm.activePane
			sm.mu.Unlock()
			activePane.Session.Invalidate()
			activePane.Render()
		}
	}()
}

func (sm *sessionManager) runRenderer() {
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
			sm.render(false)
		}
	}()
}

func (sm *sessionManager) render(shouldInvalidate bool) {
	sm.mu.Lock()
	p := sm.activePane
	sm.mu.Unlock()
	// Clears the previous screen of the session.
	if shouldInvalidate {
		p.Session.Invalidate()
	}

	p.Render()
}

func (sm *sessionManager) next() {
	if len(sm.panes) < 1 {
		return
	}

	sm.mu.Lock()
	activeSession := sm.activePane.Session
	sm.mu.Unlock()
	lastSessionID := len(sm.panes) - 1
	currentSessionID := 0
	for i := range sm.panes {
		if sm.panes[i].Session == activeSession {
			currentSessionID = i + 1

			if currentSessionID > lastSessionID {
				currentSessionID = 0
			}

			break
		}
	}

	pane := sm.panes[currentSessionID]
	sm.Select(pane)
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
		sm.activePane.Session.Write(input)
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

func (sm *sessionManager) close(p Pane) {
	sm.mu.Lock()
	for i, pane := range sm.panes {
		if pane == &p {
			sm.panes = append(sm.panes[:i], sm.panes[i+1:]...)
			break
		}
	}
	sm.mu.Unlock()

	p.Session.Close()
	sm.next()
}

func (sm *sessionManager) Wait() {
	sm.sessionWg.Wait()
}

func (sm *sessionManager) getSize(position PanePosition) (int, int) {
	if position != PanePositionEnum.FullScreen {
		return sm.getSizeSplit()
	}

	return sm.getSizeFull()
}

func (sm *sessionManager) getSizeSplit() (int, int) {
	rows, cols := sm.getSizeFull()
	return rows, cols/2 - 1
}

func (sm *sessionManager) getSizeFull() (int, int) {
	rows, cols, err := pty.Getsize(os.Stdin)
	if err != nil {
		// FIXME: This should be saved in logs rather than printed to the console.
		fmt.Printf("Cannot terminal size %v\n", err)
	}

	return rows, cols
}
