package manager

type PanePosition int

var PanePositionEnum = struct {
	FullScreen PanePosition
	Left       PanePosition
	Right      PanePosition
}{
	FullScreen: 0,
	Left:       1,
	Right:      2,
}
