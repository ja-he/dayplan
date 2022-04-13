package panes

import (
	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
)

// EditorPane visualizes the detailed editing of an event.
type EditorPane struct {
	renderer         ui.ConstrainedRenderer
	cursorController ui.TextCursorController
	dimensions       func() (x, y, w, h int)
	stylesheet       styling.Stylesheet
	condition        func() bool

	getMode func() input.TextEditMode

	name      func() string
	cursorPos func() int
}

// EnsureHidden informs the pane that it is not being shown so that it can take
// potential actions to ensure that, e.g., hide the terminal cursor, if
// necessary.
func (p *EditorPane) EnsureHidden() { p.cursorController.HideCursor() }

// Condition returns whether this pane should be visible.
func (p *EditorPane) Condition() bool { return p.condition() }

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
func (p *EditorPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *EditorPane) GetPositionInfo(x, y int) ui.PositionInfo { return nil }

// Draw draws the editor popup.
func (p *EditorPane) Draw() {
	if p.condition() {
		x, y, w, h := p.Dimensions()

		p.renderer.DrawBox(x, y, w, h, p.stylesheet.Editor)
		p.renderer.DrawText(x+1, y+1, w-2, h-2, p.stylesheet.Editor, p.name())
		p.cursorController.ShowCursor(x+1+(p.cursorPos()%(w-2)), y+1+(p.cursorPos()/(w-2)))
		// TODO(ja-he): wrap at word boundary

		mode := p.getMode()
		var modeStr string
		var style styling.DrawStyling
		switch mode {
		case input.TextEditModeNormal:
			modeStr = "-- NORMAL --"
			style = p.stylesheet.Editor.Italicized()
		case input.TextEditModeInsert:
			modeStr = "-- INSERT --"
			style = p.stylesheet.Editor.DefaultEmphasized().Italicized().Bolded()
		default:
			panic("unknown text edit mode")
		}
		p.renderer.DrawText(x+4, y+h-2, len(modeStr), 1, style, modeStr)
	}
}

// NewEditorPane constructs and returns a new EditorPane.
func NewEditorPane(
	renderer ui.ConstrainedRenderer,
	cursorController ui.TextCursorController,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	condition func() bool,
	name func() string,
	getMode func() input.TextEditMode,
	cursorPos func() int,
) *EditorPane {
	return &EditorPane{
		renderer:         renderer,
		cursorController: cursorController,
		dimensions:       dimensions,
		stylesheet:       stylesheet,
		condition:        condition,
		name:             name,
		getMode:          getMode,
		cursorPos:        cursorPos,
	}
}
