package manager

type Session interface {
	Read([]byte) (int, error)
	Render()
	SetSize(cols, rows int) error
	Write([]byte) (int, error)
	WriteBackground([]byte) (int, error)
}
