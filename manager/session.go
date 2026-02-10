package manager

import "os"

type Session interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Render()
	InheritSize(*os.File) error
}
