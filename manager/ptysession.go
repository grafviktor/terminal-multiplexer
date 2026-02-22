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
	colOffset int
	rowOffset int
	isModeAppCursor bool
	isModeAppKeypad bool
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

func (s *PtySession) Invalidate() {
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

func (s *PtySession) SetRect(cols, rows, collOffset, rowOffset int) error {
	// It's important to forward the size of the terminal from the stdin
	// to ps.Ptmx which generates the output. Otherwise ncurses apps
	// will not be rendered correctly.
	cols = cols - collOffset - 1
	rows = rows - rowOffset - 1
	err := pty.Setsize(s.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return err
	}

	s.Term.Resize(cols, rows) // note: vt10x wants cols, then rows
	s.colOffset = collOffset
	s.rowOffset = rowOffset
	return nil
}

func (s *PtySession) Render() {
	s.setTerminalMode()

	sb := strings.Builder{}
	s.Term.Lock()
	defer s.Term.Unlock()

	cols, rows := s.Term.Size()
	for y := 0; y < rows; y++ {
		// Reset previous foreground and background for every line.
		prevFG := ""
		prevBG := ""

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
			// fmt.Fprintf(os.Stdout, "\x1b[?7l") // Disable line wrapping
			fmt.Fprintf(os.Stdout, "\x1b[%d;%dH\x1b[2K%s", y+s.rowOffset+1, s.colOffset+1, currentLine)
			// fmt.Fprintf(os.Stdout, "\x1b[?7h") // Enable line wrapping
		}

		sb = strings.Builder{}
	}

	// Without this line, when switch between sessions, you can see the previous session's colors.
	resetColors := "\x1b[0m"
	sb.WriteString(resetColors)
	// Get the current cursor position and put the cursor in the correct place after rendering the terminal.
	cursorPos := s.Term.Cursor()
	fmt.Fprintf(&sb, "\x1b[%d;%dH", cursorPos.Y+s.rowOffset+1, cursorPos.X+s.colOffset+1)
	fmt.Print(sb.String())
}

func (s *PtySession) setTerminalMode() {
	s.Term.Lock()
	mode := s.Term.Mode()
	s.Term.Unlock()
	isModeAppCursor := mode & vt10x.ModeAppCursor != 0
	isModeAppKeypad := mode & vt10x.ModeAppKeypad != 0

	if s.isModeAppCursor != isModeAppCursor {
		s.isModeAppCursor = isModeAppCursor
		if isModeAppCursor {
			s.enableAppCursorMode()
		} else {
			s.disableAppCursorMode()
		}
	}

	if s.isModeAppKeypad != isModeAppKeypad {
		s.isModeAppKeypad = isModeAppKeypad
		if isModeAppKeypad {
			s.enableAppKeypadMode()
		} else {
			s.disableAppKeypadMode()
		}
	}
}

func (s *PtySession) enableAppCursorMode() {
	fmt.Print("\x1b[?1h")
}

func (s *PtySession) disableAppCursorMode() {
	fmt.Print("\x1b[?1l")
}

func (s *PtySession) enableAppKeypadMode() {
	fmt.Print("\x1b=")
}

func (s *PtySession) disableAppKeypadMode() {
	fmt.Print("\x1b>")
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
