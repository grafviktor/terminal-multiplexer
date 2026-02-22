package manager

type Session interface {
	Close()
	Invalidate()
	Read([]byte) (int, error)
	Render()
	SetRect(cols, rows, x0, y0 int) error
	Write([]byte) (int, error)
	WriteBackground([]byte) (int, error)
}
