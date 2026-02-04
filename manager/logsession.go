package manager

import (
	"strings"
	"sync"
)

var _ Session = (*LogSession)(nil)

type LogSession struct {
	id     int
	mu     sync.Mutex
	lines  []string
	scroll int
}

func (l *LogSession) Read(p []byte) (int, error) {
	return 0, nil // no PTY, nothing to read
}

func (l *LogSession) WriteToBuffer(p []byte) (int, error) {
	return 0, nil
}

func (l *LogSession) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.lines = append(l.lines, strings.TrimRight(string(p), "\n"))
	return len(p), nil
}
