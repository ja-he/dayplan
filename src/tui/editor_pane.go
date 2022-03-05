package tui

import (
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
)

type EditorPane struct {
	renderer         ui.ConstrainedRenderer
	cursorController CursorController
	dimensions       func() (x, y, w, h int)
	stylesheet       styling.Stylesheet
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
	if p.condition() {
		x, y, w, h := p.Dimensions()

		p.renderer.DrawBox(x, y, w, h, p.stylesheet.Editor())
		p.renderer.DrawText(x+1, y+1, w-2, h-2, p.stylesheet.Editor(), p.name())
		p.cursorController.ShowCursor(x+1+(p.cursorPos()%(w-2)), y+1+(p.cursorPos()/(w-2)))
		// TODO(ja-he): wrap at word boundary
	} else {
		p.cursorController.HideCursor()
	}
}
