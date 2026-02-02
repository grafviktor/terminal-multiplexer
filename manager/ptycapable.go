package manager

import (
	"bytes"
	"os"
)

type PtyCapable interface {
	Session
	MasterFD() *os.File
	Buffer() *bytes.Buffer
}
