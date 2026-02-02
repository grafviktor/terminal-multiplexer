package manager

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type Session interface {
	// ID() int
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	// Draw()
}

var _ Session = (*PtySession)(nil)

type PtySession struct {
	Cmd    *exec.Cmd
	Master *os.File
	ID     int
	buf    bytes.Buffer
}

// func (s *PtySession) ID() int { return s.ID }
func (s *PtySession) Read(p []byte) (int, error) {
	return s.Master.Read(p)
}

const maxBufferHistory = 1024 * 1024 // 1MB
func (s *PtySession) Write(p []byte) (int, error) {
	// return s.Master.Write(p)

	if s.buf.Len()+len(p) > maxBufferHistory {
		// drop oldest
		excess := s.buf.Len() + len(p) - maxBufferHistory
		s.buf.Next(excess)
	}
	return s.buf.Write(p)
}

func (s *PtySession) MasterFD() *os.File {
	return s.Master
}

func (s *PtySession) Buffer() *bytes.Buffer {
	return &s.buf
}

func (s *PtySession) ClearAndRedraw() {
	pty.InheritSize(os.Stdin, s.Master)
	os.Stdout.Write([]byte("\x1b[2J\x1b[H"))
	os.Stdout.Write(s.buf.Bytes())
}

// type Session struct {
// 	Cmd    *exec.Cmd
// 	Master *os.File
// 	ID     int
// 	buf    bytes.Buffer
// }

// const maxBufferHistory = 1024 * 1024 // 1MB

// func (s *Session) append(data []byte) {
// 	if s.buf.Len()+len(data) > maxBufferHistory {
// 		// drop oldest
// 		excess := s.buf.Len() + len(data) - maxBufferHistory
// 		s.buf.Next(excess)
// 	}
// 	s.buf.Write(data)
// }
