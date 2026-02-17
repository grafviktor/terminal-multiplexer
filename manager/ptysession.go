package manager

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

var _ Session = (*PtySession)(nil)

type PtySession struct {
	ID        int
	Term      vt10x.Terminal
	ptmx      *os.File
	buf       bytes.Buffer
	prevFrame map[int]string
	prevX     int
	prevY     int
}

func NewPtySession(id int, cmd *exec.Cmd) (*PtySession, error) {
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	return &PtySession{
		ID:        id,
		Term:      vt10x.New(),
		ptmx:      ptmx,
		prevFrame: make(map[int]string),
	}, nil
}

func (s *PtySession) resetPrevFrame() {
	s.prevFrame = make(map[int]string)
	clearAndHome := "\x1b[2J\x1b[H"
	fmt.Print(clearAndHome)
}

func (s *PtySession) Read(p []byte) (int, error) {
	return s.ptmx.Read(p)
}

func (s *PtySession) Write(p []byte) (int, error) {
	return s.ptmx.Write(p)
}

func (s *PtySession) WriteBackground(p []byte) (int, error) {
	return s.Term.Write(p)
}

func (s *PtySession) SetSize(cols, rows int) error {
	// That's very important to forward the size of the terminal from the stdin
	// to ps.Ptmx which generates the output. Otherwise ncurses apps
	// will not be rendered correctly.
	err := pty.Setsize(s.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return err
	}

	s.Term.Resize(cols, rows) // note: vt10x wants cols, then rows
	// Finally clear the screen.
	s.resetPrevFrame()
	return nil
}

func (s *PtySession) Render() {
	// This always clears the screen and moves the cursor to the home position.
	// In other words - it replaces what was previously on the screen.
	// If removed, then ypu'll have every "frame" of the terminal rendered on the screen.
	resetColors := "\x1b[0m"
	sb := strings.Builder{}
	s.Term.Lock()
	defer s.Term.Unlock()

	// FIXME: Need to clear screen when switch from another session.
	cols, rows := s.Term.Size()
	// if s.prevX != cols || s.prevY != rows {
	// 	s.clearScreen()
	// 	// Full re-render if the size of the terminal has changed.
	// 	s.prevX = cols
	// 	s.prevY = rows
	// }

	prevFG := ""
	prevBG := ""

	// Should diff crrent buffer with previous before rendering.
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
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

			sb.WriteRune(char)
		}

		prevLine, lineFound := s.prevFrame[y]
		currentLine := sb.String()
		if !lineFound || prevLine != currentLine {
			s.prevFrame[y] = currentLine
			fmt.Fprintf(os.Stdout, "\x1b[%d;1H\x1b[2K%s", y+1, currentLine)
		}

		sb.Reset()
	}

	sb.WriteString(resetColors)
	// Get the current cursor position and put the cursor in the correct place after rendering the terminal.
	cursorPos := s.Term.Cursor()
	fmt.Fprintf(&sb, "\x1b[%d;%dH", cursorPos.Y+1, cursorPos.X+1)
	fmt.Print(sb.String())
}

func (s *PtySession) makeChar(char rune) rune {
	if char == 0 {
		return ' '
	}

	return char
}

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

func (s *PtySession) Close() {
	s.ptmx.Close()
}
