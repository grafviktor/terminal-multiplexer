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
	return s.Ptmx.Write(p)
}

func (s *PtySession) WriteBackground(p []byte) (int, error) {
	return s.Term.Write(p)
}

func (s *PtySession) SetSize(cols, rows int) error {
	// That's very important to forward the size of the terminal from the stdin
	// to ps.Ptmx which generates the output. Otherwise ncurses apps
	// will not be rendered correctly.
	err := pty.Setsize(s.Ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return err
	}

	s.Term.Resize(cols, rows) // note: vt10x wants cols, then rows
	return nil
}

func (s *PtySession) Render() {
	// This always clears the screen and moves the cursor to the home position.
	// In other words - it replaces what was previously on the screen.
	// If removed, then ypu'll have every "frame" of the terminal rendered on the screen.
	clearAndHome := "\x1b[2J\x1b[H"
	resetColors := "\x1b[0m"
	fmt.Print(clearAndHome)
	s.Term.Lock()
	defer s.Term.Unlock()
	w, h := s.Term.Size()

	sb := strings.Builder{}
	sb.WriteString(clearAndHome)

	prevFG := ""
	prevBG := ""

	// Should diff crrent buffer with previous before rendering.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			glyph := s.Term.Cell(x, y)
			newFG := s.makeFG(glyph.FG)
			newBG := s.makeBG(glyph.BG)
			char := s.makeChar(glyph.Char)

			if newFG != prevFG {
				sb.WriteString(newFG)
				prevFG = newFG
			}

			if newBG != prevBG {
				sb.WriteString(newBG)
				prevBG = newBG
			}

			sb.WriteString(char)
		}
	}

	sb.WriteString(resetColors)
	// Get the current cursor position and put the cursor in the correct place after rendering the terminal.
	cursorPos := s.Term.Cursor()
	fmt.Fprintf(&sb, "\x1b[%d;%dH", cursorPos.Y+1, cursorPos.X+1)

	fmt.Print(sb.String())
}

func (s *PtySession) makeChar(char rune) string {
	if char == 0 {
		return " "
	}

	return string(char)
}

/*
Default fg: \x1b[39m
Default bg: \x1b[49m
256-color fg: \x1b[38;5;<n>m
256-color bg: \x1b[48;5;<n>m
*/

func (s *PtySession) makeFG(fg vt10x.Color) string {
	if fg == vt10x.DefaultFG || fg >= (1<<24) {
		return "\x1b[39m"
	} else {
		return fmt.Sprintf("\x1b[38;5;%dm", int(fg))
	}
}

func (s *PtySession) makeBG(bg vt10x.Color) string {
	if bg == vt10x.DefaultBG || bg >= (1<<24) {
		return "\x1b[49m"
	} else {
		return fmt.Sprintf("\x1b[48;5;%dm", int(bg))
	}
}
