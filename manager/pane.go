package manager

import "fmt"

type Pane struct {
	ID         int
	Session    Session
	cols int
	rows int
	x0 int
	y0 int
	isFocused bool
}

func NewPane(id int, session Session, cols, rows int) *Pane {
	offsetCols := 1
	offsetRows := 1
	session.SetRect(cols, rows, offsetCols, offsetRows)

	return &Pane{
		ID: id,
		Session: session,
		cols: cols,
		rows: rows,
		x0: offsetCols,
		y0: offsetRows,
	}
}

func (p *Pane) SetSize(cols, rows int) {
	p.cols = cols
	p.rows = rows
	p.Session.SetRect(cols, rows, p.x0, p.y0)
}

func (p *Pane) Render() {
	leftTopCorner := "\x1b[%d;%dH┌"
	rightTopCorner := "\x1b[%d;%dH┐"
	leftBottomCorner := "\x1b[%d;%dH└"
	rightBottomCorner := "\x1b[%d;%dH┘"
	verticalLine := "\x1b[%d;%dH│"
	horizontalLine := "\x1b[%d;%dH─"

	for c := range p.cols {
		for r := range p.rows {
			isLeftTopCorner := c == 0 && r == 0
			isRightTopCorner := c == p.cols-1 && r == 0
			isLeftBottomCorner := c == 0 && r == p.rows-1
			isRightBottomCorner := c == p.cols-1 && r == p.rows-1
			isVerticalBorder := c == 0 || c == p.cols-1
			isHorizontalBorder := r == 0 || r == p.rows-1

			if isLeftTopCorner {
				fmt.Printf(leftTopCorner, r+1, c+1)
			} else if isRightTopCorner {
				fmt.Printf(rightTopCorner, r+1, c+1)
			} else if isLeftBottomCorner {
				fmt.Printf(leftBottomCorner, r+1, c+1)
			} else if isRightBottomCorner {
				fmt.Printf(rightBottomCorner, r+1, c+1)
			} else if isVerticalBorder {
				fmt.Printf(verticalLine, r+1, c+1)
			} else if isHorizontalBorder {
				fmt.Printf(horizontalLine, r+1, c+1)
			}
		}
	}

	p.Session.Render()
}
