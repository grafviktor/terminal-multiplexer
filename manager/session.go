package manager

type Session interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Render()
	SetSize(cols, rows int) error
}
