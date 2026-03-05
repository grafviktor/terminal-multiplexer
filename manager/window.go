package manager

import (
	"fmt"
	"os"

	"github.com/creack/pty"
)

type Window struct {
	ID    int
	panes []*Pane
}

func (w *Window) AddPane(p *Pane) {
	w.panes = append(w.panes, p)
}

func (w *Window) Render() {
	for _, p := range w.panes {
		p.Render()
	}
}

func (w *Window) SetSize(cols, rows int) {
	for _, p := range w.panes {
		p.SetSize(cols, rows)
	}
}

func (w *Window) calculatePaneRect(id int) (cols, rows int) {
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
