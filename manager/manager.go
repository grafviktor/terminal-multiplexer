package manager

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

const hotKey = 0x01 // Ctrl-A

type sessionManager struct {
	// nextSessionID   int
	nextWindowID    int
	sessionWg       sync.WaitGroup
	uiDirty         chan struct{}
	panePositionMap map[int]PanePosition

	mu sync.Mutex
	// focusedPane   *Pane
	// panes         []*Pane
	focusedWindow *Window
	windows       map[int]*Window
}

func NewManager() *sessionManager {
	sm := &sessionManager{
		// panes:     []*Pane{},
		sessionWg: sync.WaitGroup{},
		uiDirty:   make(chan struct{}, 1),
		windows:   make(map[int]*Window),
	}

	// sm.createServicePane()
	sm.createServiceWindow()
	sm.runWindowSizeWatcher()
	sm.runStdInReader()
	sm.runRenderer()

	return sm
}

func (sm *sessionManager) createServiceWindow() {
	// rows, cols := sm.getSize(PanePositionEnum.FullScreen)
	rows, cols, _ := pty.Getsize(os.Stdin)
	s := &StatusSession{cols: cols, rows: rows}
	p := &Pane{ID: sm.nextWindowID, Session: s}
	// sm.panes = append(sm.panes, p)
	w := NewWindow(&sm.sessionWg, 0)
	w.AddPane(p)
	w.Select(p)
	sm.windows[0] = w
	sm.nextWindowID++
	sm.Select(w)
}

// func (sm *sessionManager) createServicePane() {
// 	// rows, cols := sm.getSize(PanePositionEnum.FullScreen)
// 	rows, cols, _ := pty.Getsize(os.Stdin)
// 	s := &StatusSession{cols: cols, rows: rows}
// 	p := &Pane{ID: sm.nextSessionID, Session: s}
// 	sm.panes = append(sm.panes, p)
// 	sm.Select(p)
// }

func (sm *sessionManager) Create(windowId int, position PanePosition, argv []string) (*Window, error) {
	// TODO: Move pane creation logic to window.go
	// p, err := NewPane(&sm.sessionWg, sm.nextSessionID, argv)
	// if err != nil {
	// 	return nil, err
	// }

	// rows, cols := sm.getSize(position)
	// offsetCols := 0
	// if position == PanePositionEnum.Right {
	// 	offsetCols += cols
	// }
	// offsetRows := 0
	// p.SetRect(cols, rows, offsetCols, offsetRows)

	// sm.panes = append(sm.panes, p)
	// sm.panePositionMap[p.ID] = position
	// sm.nextSessionID++
	// sm.runStdOutReader(*p)

	w, ok := sm.windows[windowId]
	if !ok {
		w = NewWindow(&sm.sessionWg, windowId)
		sm.windows[windowId] = w
	}
	p, _ := w.CreatePane(position, argv)
	w.AddPane(p)
	w.Select(p)

	return w, nil
}

func (sm *sessionManager) Select(w *Window) {
	if ss, ok := w.focusedPane.Session.(*StatusSession); ok {
		sessionInfo := SessionInfo{
			sessionCount: len(sm.windows) - 1,
		}
		ss.Refresh(sessionInfo) // exclude status session
	}

	sm.mu.Lock()
	sm.focusedWindow = w
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

// func (sm *sessionManager) runStdOutReader(p Pane) {
// 	go func() {
// 		tmp := make([]byte, 4096)

// 		for {
// 			n, err := p.Session.Read(tmp)
// 			if err != nil {
// 				return
// 			}

// 			data := bytes.Clone(tmp[:n])
// 			p.Session.WriteBackground(data)
// 			sm.uiDirty <- struct{}{}
// 		}
// 	}()
// }

func (sm *sessionManager) runWindowSizeWatcher() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			// for _, p := range sm.panes {
			// 	// TODO: Notify windows about size change
			// 	rows, cols := sm.getSize(sm.panePositionMap[p.ID])
			// 	p.SetSize(cols, rows)
			// }
			for _, w := range sm.windows {
				w.RequestResize()
			}

			sm.mu.Lock()
			focused := sm.focusedWindow
			sm.mu.Unlock()
			// focused.focusedPane.Session.Invalidate()
			focused.Render()
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
	w := sm.focusedWindow
	sm.mu.Unlock()
	w.Render()
	// // Clears the previous screen of the session.
	// if shouldInvalidate {
	// 	w.focusedPane.Session.Invalidate()
	// }

	// w.focusedPane.Render()
}

func (sm *sessionManager) next() {
	if len(sm.windows) < 1 {
		return
	}

	sm.mu.Lock()
	focusedWindow := sm.focusedWindow
	sm.mu.Unlock()
	lastWindowID := len(sm.windows) - 1
	currentWindowID := 0
	for i := range sm.windows {
		if sm.windows[i] == focusedWindow {
			currentWindowID = i + 1

			if currentWindowID > lastWindowID {
				currentWindowID = 0
			}

			break
		}
	}

	window := sm.windows[currentWindowID]
	sm.Select(window)
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
		sm.focusedWindow.focusedPane.Session.Write(input)
		// sm.focusedPane.Session.Write(input)
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

func (sm *sessionManager) Wait() {
	sm.sessionWg.Wait()
}
