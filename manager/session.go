package manager

type Session interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
}
