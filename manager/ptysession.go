package manager

import (
	"bytes"
	"fmt"
	"os"

	"github.com/hinshun/vt10x"
)

var _ Session = (*PtySession)(nil)

const maxBufferHistory = 1024 * 1024 // 1MB

type PtySession struct {
	Master *os.File
	Term   vt10x.Terminal
	ID  int
	buf bytes.Buffer
}

func (s *PtySession) Read(p []byte) (int, error) {
	return s.Master.Read(p)
}

func (s *PtySession) Write(p []byte) (int, error) {
	// return s.Term.Write(p)
	return s.Master.Write(p)
}

func (s *PtySession) Render() {
	w, h := s.Term.Size()

	// This always clears the screen and moves the cursor to the home position.
	// In other words - it replaces what was previously on the screen.
	// If removed, then ypu'll have every "frame" of the terminal rendered on the screen.
	clearAndHome := "\x1b[2J\x1b[H"
	fmt.Print(clearAndHome)
	s.Term.Lock()
	defer s.Term.Unlock()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cell := s.Term.Cell(x, y)
			if cell.Char == 0 {
				fmt.Print(" ")
			} else {
				fmt.Print(string(cell.Char))
			}
		}
	}
}
