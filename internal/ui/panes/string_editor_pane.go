package panes

import (
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/control/editor"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
)

// StringEditorPane visualizes the editing of a string (as seen by a StringEditorView).
type StringEditorPane struct {
	Leaf

	view editor.StringEditorView

	cursorController ui.TextCursorController
}

// Draw draws the editor popup.
func (p *StringEditorPane) Draw() {
	log.Debug().Msgf("drawing string editor")
	if p.IsVisible() {
		log.Debug().Msgf("string editor visible")
		x, y, w, h := p.Dimensions()

		p.renderer.DrawBox(x, y, w, h, p.stylesheet.Editor.DarkenedBG(10))
		p.renderer.DrawText(x+1, y, 10, h, p.stylesheet.CategoryFallback, p.view.GetName())
		p.renderer.DrawText(x+12, y, w-13, h, p.stylesheet.Normal, p.view.GetContent())
		cursorX, cursorY := x+12+(p.view.GetCursorPos()), y
		p.cursorController.ShowCursor(cursorX, cursorY)
		log.Debug().Msgf("drawing cursor at %d, %d", cursorX, cursorY)
		// TODO(ja-he): wrap at word boundary
	}
}

// Undraw ensures that the cursor is hidden.
func (p *StringEditorPane) Undraw() {
	p.cursorController.HideCursor()
}

// GetPositionInfo returns information on a requested position in this pane (nil, for now).
func (p *StringEditorPane) GetPositionInfo(_, _ int) ui.PositionInfo { return nil }

// NewStringEditorPane creates a new StringEditorPane.
func NewStringEditorPane(
	dimensions func() (int, int, int, int),
	visible func() bool,
	inputProcessor input.ModalInputProcessor,
	renderer ui.ConstrainedRenderer,
	view editor.StringEditorView,
	stylesheet styling.Stylesheet,
	cursorController ui.TextCursorController,
) *StringEditorPane {
	return &StringEditorPane{
		Leaf: Leaf{
			Base: Base{
				ID:             ui.GeneratePaneID(),
				InputProcessor: inputProcessor,
				Visible:        visible,
			},
			renderer:   renderer,
			dimensions: dimensions,
			stylesheet: stylesheet,
		},
		view:             view,
		cursorController: cursorController,
	}
}
