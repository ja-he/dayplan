package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/ui"
)

type EditorPane struct {
	renderer         ConstrainedRenderer
	cursorController CursorController
	dimensions       func() (x, y, w, h int)
	condition        func() bool

	name      func() string
	cursorPos func() int
}

func (p *EditorPane) Condition() bool { return p.condition() }

func (p *EditorPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *EditorPane) GetPositionInfo(x, y int) ui.PositionInfo { return nil }

// Draw the help popup.
func (p *EditorPane) Draw() {
	style := tcell.StyleDefault.Background(tcell.ColorLightGrey).Foreground(tcell.ColorBlack)
	if p.condition() {
		x, y, w, h := p.Dimensions()

		p.renderer.DrawBox(x, y, w, h, style)
		p.renderer.DrawText(x+1, y+1, w-2, h-2, style, p.name())
		p.cursorController.ShowCursor(x+1+(p.cursorPos()%(w-2)), y+1+(p.cursorPos()/(w-2)))
		// TODO(ja-he): wrap at word boundary
	} else {
		p.cursorController.HideCursor()
	}
}
