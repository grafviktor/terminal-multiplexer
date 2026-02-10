package manager

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

var _ Session = (*PtySession)(nil)

const maxBufferHistory = 1024 * 1024 // 1MB

type PtySession struct {
	Ptmx *os.File
	Term vt10x.Terminal
	ID   int
	buf  bytes.Buffer
}

func (s *PtySession) Read(p []byte) (int, error) {
	return s.Ptmx.Read(p)
}

func (s *PtySession) Write(p []byte) (int, error) {
	// return s.Term.Write(p)
	return s.Ptmx.Write(p)
}

func (s *PtySession) SetSize(cols, rows int) error {
	// That's very important to forward the size of the terminal from the stdin
	// to ps.Master which generates the output. Otherwise ncurses apps
	// will not be rendered correctly.
	// err := pty.InheritSize(t, s.Ptmx)

	err := pty.Setsize(s.Ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return err
	}

	// Then we set size to vt10x to render the terminal correctly.
	// rows, cols, err := pty.Getsize(os.Stdin)
	// if err != nil {
	// 	return err
	// }

	s.Term.Resize(cols, rows) // note: vt10x wants cols, then rows
	return nil
}

// func (s *PtySession) InheritSize(t *os.File) error {
// 	// That's very important to forward the size of the terminal from the stdin
// 	// to ps.Master which generates the output. Otherwise ncurses apps
// 	// will not be rendered correctly.
// 	err := pty.InheritSize(t, s.Ptmx)
// 	if err != nil {
// 		return err
// 	}

// 	// Then we set size to vt10x to render the terminal correctly.
// 	rows, cols, err := pty.Getsize(os.Stdin)
// 	if err != nil {
// 		return err
// 	}

// 	s.Term.Resize(cols, rows) // note: vt10x wants cols, then rows
// 	return nil
// }

func (s *PtySession) Render() {
	// This always clears the screen and moves the cursor to the home position.
	// In other words - it replaces what was previously on the screen.
	// If removed, then ypu'll have every "frame" of the terminal rendered on the screen.
	clearAndHome := "\x1b[2J\x1b[H"
	fmt.Print(clearAndHome)
	s.Term.Lock()
	defer s.Term.Unlock()
	w, h := s.Term.Size()

	sb := strings.Builder{}
	sb.WriteString(clearAndHome)

	// Should diff crrent buffer with previous before rendering.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cell := s.Term.Cell(x, y)
			if cell.Char == 0 {
				sb.WriteString(" ")
			} else {
				sb.WriteString(string(cell.Char))
			}
		}
	}

	// Get the current cursor position and put the cursor in the correct place after rendering the terminal.
	cursorPos := s.Term.Cursor()
	sb.WriteString(fmt.Sprintf("\x1b[%d;%dH", cursorPos.Y+1, cursorPos.X+1))

	fmt.Print(sb.String())
}
