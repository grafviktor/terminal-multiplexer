package manager

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

var _ Session = (*PtySession)(nil)

const maxBufferHistory = 1024 * 1024 // 1MB

type PtySession struct {
	Cmd    *exec.Cmd
	Master *os.File
	ID     int
	buf    bytes.Buffer
}

func (s *PtySession) Read(p []byte) (int, error) {
	return s.Master.Read(p)
}

func (s *PtySession) Write(p []byte) (int, error) {
	return s.Master.Write(p)
}

func (s *PtySession) WriteToBuffer(p []byte) (int, error) {
	// This is used in runSessionOutputReader. Once the session becomes active, we write buffer to stdout
	if s.buf.Len()+len(p) > maxBufferHistory {
		// drop oldest
		excess := s.buf.Len() + len(p) - maxBufferHistory
		s.buf.Next(excess)
	}
	return s.buf.Write(p)
}

func (s *PtySession) ClearAndRedraw() {
	pty.InheritSize(os.Stdin, s.Master)
	os.Stdout.Write([]byte("\x1b[2J\x1b[H"))
	os.Stdout.Write(s.buf.Bytes())
}
