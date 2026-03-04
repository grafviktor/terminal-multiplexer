package manager

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
)

type Pane struct {
	ID           int
	Session      Session
	cols         int
	rows         int
	offsetCols   int
	offsetRows   int
	isFocused    bool
	IsTerminated bool
}

// func NewPane(wg *sync.WaitGroup, id int, argv []string) (*Pane, error) {
// 	p := &Pane{
// 		ID: id,
// 	}

// 	return p, nil
// }

func NewPane(wg *sync.WaitGroup, id int, argv []string) (*Pane, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	session, err := NewPtySession(cmd)
	if err != nil {
		return nil, err
	}

	p := &Pane{
		ID:      id,
		Session: session,
	}

	ready := make(chan any, 1)
	wg.Go(func() {
		ready <- struct{}{}
		waitErr := cmd.Wait()
		if waitErr != nil {
			log.Printf("session %d ended with error: %v", p.ID, waitErr)
		}
		session.Close()
		p.IsTerminated = true
		p.Render()
		// fmt.Printf("\x1b[%d;%dH Exited", p.offsetRows, p.offsetCols)
	})

	// Wait session goroutine to start
	<-ready
	return p, nil
}

func (p *Pane) SetRect(cols, rows, offsetCols, offsetRows int) {
	// It's required to create offset
	// for left pane because otherwise it will draw border outside of the left border of the screen
	// for right pane because we need margin between panes
	offsetCols += 1
	offsetRows += 1

	p.offsetCols = offsetCols
	p.offsetRows = offsetRows
	p.SetSize(cols, rows)
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
	if p.IsTerminated {
		fmt.Printf("\x1b[?25l")
		fmt.Printf("\x1b[31;1m")
	} else {
		fmt.Printf("\x1b[?25h")
	}

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

	if p.IsTerminated {
		fmt.Printf("\x1b[%d;%dH Exited", p.offsetRows+p.rows/2, p.offsetCols+p.cols/2-3)
		fmt.Printf("\x1b[0m")
	}

	p.Session.Render()
}
