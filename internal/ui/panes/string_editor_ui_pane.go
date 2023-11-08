package panes

import (
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/control/edit/views"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
)

// StringEditorPane visualizes the editing of a string (as seen by a StringEditorView).
type StringEditorPane struct {
	ui.LeafPane

	view views.StringEditorView

	cursorController ui.TextCursorController
}

// Draw draws the editor popup.
func (p *StringEditorPane) Draw() {
	if p.IsVisible() {
		x, y, w, h := p.Dims()

		baseBGStyle := p.Stylesheet.Editor
		if p.view.IsActive() {
			baseBGStyle = baseBGStyle.DarkenedBG(10)
		}

		p.Renderer.DrawBox(x, y, w, h, baseBGStyle)
		p.Renderer.DrawText(x+1, y, 10, h, baseBGStyle.Italicized(), p.view.GetName())
		p.Renderer.DrawText(x+12, y, w-13, h, baseBGStyle.DarkenedBG(20), p.view.GetContent())

		if p.view.IsActive() {
			cursorX, cursorY := x+12+(p.view.GetCursorPos()), y
			p.cursorController.ShowCursor(cursorX, cursorY)
			log.Debug().Msgf("drawing cursor at %d, %d", cursorX, cursorY)
		} else {
			p.cursorController.HideCursor()
		}

		// TODO(ja-he): wrap at word boundary; or something...
	}
}

// Undraw ensures that the cursor is hidden.
func (p *StringEditorPane) Undraw() {
	p.cursorController.HideCursor()
}

// GetPositionInfo returns information on a requested position in this pane (nil, for now).
func (p *StringEditorPane) GetPositionInfo(_, _ int) ui.PositionInfo { return nil }

// ProcessInput attempts to process the provided input.
func (p *StringEditorPane) ProcessInput(k input.Key) bool {
	if !p.view.IsActive() {
		log.Warn().Msgf("string editor pane asked to process input despite view reporting not active; likely logic error")
	}
	return p.LeafPane.ProcessInput(k)
}

// NewStringEditorPane creates a new StringEditorPane.
func NewStringEditorPane(
	renderer ui.ConstrainedRenderer,
	visible func() bool,
	inputProcessor input.ModalInputProcessor,
	view views.StringEditorView,
	stylesheet styling.Stylesheet,
	cursorController ui.TextCursorController,
) *StringEditorPane {
	return &StringEditorPane{
		LeafPane: ui.LeafPane{
			BasePane: ui.BasePane{
				ID:             ui.GeneratePaneID(),
				InputProcessor: inputProcessor,
				Visible:        visible,
			},
			Renderer:   renderer,
			Dims:       renderer.Dimensions,
			Stylesheet: stylesheet,
		},
		view:             view,
		cursorController: cursorController,
	}
}
