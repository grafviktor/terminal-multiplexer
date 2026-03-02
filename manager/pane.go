package manager

import "fmt"

type Pane struct {
	ID         int
	Session    Session
	cols       int
	rows       int
	offsetCols int
	offsetRows int
	isFocused  bool
}

func NewPane(id int, session Session, cols, rows, offsetCols, offsetRows int) *Pane {
	// It's required to create offset
	// for left pane because otherwise it will draw border outside of the left border of the screen
	// for right pane because we need margin between panes
	offsetCols += 1
	offsetRows += 1
	// session.SetRect(cols, rows, offsetCols, offsetRows)

	p := &Pane{
		ID:         id,
		Session:    session,
		cols:       cols,
		rows:       rows,
		offsetCols: offsetCols,
		offsetRows: offsetRows,
	}

	p.setSessionSize()
	return p
}

func (p *Pane) SetSize(cols, rows int) {
	p.cols = cols
	p.rows = rows
	p.setSessionSize()
}

func (p *Pane) setSessionSize() {
	// extract borders from cols and rows
	p.Session.SetRect(p.cols-2, p.rows-2, p.offsetCols, p.offsetRows)
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
				fmt.Printf(leftTopCorner, r+p.offsetRows, c+p.offsetCols)
			} else if isRightTopCorner {
				fmt.Printf(rightTopCorner, r+p.offsetRows, c+p.offsetCols)
			} else if isLeftBottomCorner {
				fmt.Printf(leftBottomCorner, r+p.offsetRows, c+p.offsetCols)
			} else if isRightBottomCorner {
				fmt.Printf(rightBottomCorner, r+p.offsetRows, c+p.offsetCols)
			} else if isVerticalBorder {
				fmt.Printf(verticalLine, r+p.offsetRows, c+p.offsetCols)
			} else if isHorizontalBorder {
				fmt.Printf(horizontalLine, r+p.offsetRows, c+p.offsetCols)
			}
		}
	}

	p.Session.Render()
}
