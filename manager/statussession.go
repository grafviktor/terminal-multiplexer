package manager

import (
	"bytes"
	"fmt"
	"os"
	"sync"
)

var _ Session = (*StatusSession)(nil)

type StatusSession struct {
	ID  int
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *StatusSession) Refresh(sessionInfo SessionInfo) {
	s.buf.Reset()
	// Explicit cariage return is required to move cursor to the beginning of the line after clearing the screen
	fmt.Fprintf(&s.buf, "Control Pane\n")
	fmt.Fprintf(&s.buf, "\rNumber of sessions: %d\n\r", sessionInfo.sessionCount)
	fmt.Fprintf(&s.buf, "\rHeight: %d\n\r", sessionInfo.height)
	fmt.Fprintf(&s.buf, "\rWidth: %d\n\r", sessionInfo.width)
	fmt.Fprintf(&s.buf, "\r\n")
	s.Write(s.buf.Bytes())
}

func (s *StatusSession) Read(p []byte) (int, error) {
	return 0, nil // no PTY, nothing to read
}

func (s *StatusSession) WriteToBuffer(p []byte) (int, error) {
	return 0, nil
}

func (s *StatusSession) Write(p []byte) (int, error) {
	// os.Stdout.Write([]byte("\x1b[2J\x1b[H"))
	os.Stdout.Write([]byte("\x1b[2J\x1b[H"))
	return os.Stdout.Write(s.buf.Bytes())
}

func (s *StatusSession) Render() {}

func (s *StatusSession) InheritSize(t *os.File) error {
	return nil
}