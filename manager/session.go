package manager

type Session interface {
	Close()
	Read([]byte) (int, error)
	Render()
	SetSize(cols, rows int) error
	Write([]byte) (int, error)
	WriteBackground([]byte) (int, error)
}
