package manager

type PanePosition string

var PanePositionEnum = struct {
	FullScreen PanePosition
	Left       PanePosition
	Right      PanePosition
}{
	FullScreen: "FULL_SCREEN",
	Left:       "LEFT",
	Right:      "RIGHT",
}
