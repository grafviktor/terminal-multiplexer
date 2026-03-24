package manager

import (
	"bytes"
	"fmt"
	"os"
	"sync"

	"github.com/creack/pty"
)

type Window struct {
	ID    int
	panes []*Pane
	// panePositionMap maps pane ID to its position in the window (e.g., left, right, full-screen).
	panePositionMap map[int]PanePosition
	nextPaneID      int
	sessionWg       *sync.WaitGroup
	uiDirty         chan struct{}
}

func NewWindow(sessionWg *sync.WaitGroup, windowId int) *Window {
	return &Window{
		sessionWg:       sessionWg,
		ID:              windowId,
		panePositionMap: make(map[int]PanePosition),
		uiDirty:         make(chan struct{}, 1),
	}
}

func (w *Window) CreatePane(position PanePosition, argv []string) (*Pane, error) {
	p, err := NewPane(w.sessionWg, w.nextPaneID, argv)
	if err != nil {
		return nil, err
	}

	rows, cols := w.getSize()
	offsetCols := 0
	if position == PanePositionEnum.Right {
		offsetCols += cols
	}
	offsetRows := 0
	p.SetRect(cols, rows, offsetCols, offsetRows)

	w.panes = append(w.panes, p)
	w.panePositionMap[p.ID] = position
	w.nextPaneID++
	w.runStdOutReader()

	return p, nil
}

func (w *Window) AddPane(p *Pane) {
	w.panes = append(w.panes, p)
}

func (w *Window) Render() {
	for _, p := range w.panes {
		p.Render()
	}
}

func (w *Window) RequestResize() {
	rows, cols := w.getSize()
	for _, p := range w.panes {
		p.SetSize(cols, rows)
	}
}

func (w *Window) SetSize(cols, rows int) {
	for _, p := range w.panes {
		p.SetSize(cols, rows)
	}
}

func (w *Window) getSize() (cols, rows int) {
	if len(w.panes) == 0 {
		return w.getSizeSplit()

	}
	return w.getSizeFull()

}

func (w *Window) getSizeSplit() (int, int) {
	rows, cols := w.getSizeFull()
	return rows, cols / 2
}

func (w *Window) getSizeFull() (int, int) {
	rows, cols, err := pty.Getsize(os.Stdin)
	if err != nil {
		// FIXME: This should be saved in logs rather than printed to the console.
		fmt.Printf("Cannot terminal size %v\n", err)
	}

	return rows, cols
}

func (w *Window) runStdOutReader() {
	go func() {
		tmp := make([]byte, 4096)

		for {
			for _, p := range w.panes {
				n, err := p.Session.Read(tmp)
				if err != nil {
					return
				}

				data := bytes.Clone(tmp[:n])
				p.Session.WriteBackground(data)
				w.uiDirty <- struct{}{}
			}
		}
	}()
}
