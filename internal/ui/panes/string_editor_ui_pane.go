package panes

import (
	"github.com/google/uuid"
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

	cursorController ui.CursorLocationRequestHandler

	idStr string
}

// Draw draws the editor popup.
func (p *StringEditorPane) Draw() {
	if p.IsVisible() {
		x, y, w, h := p.Dims()

		baseBGStyle := p.Stylesheet.Editor
		active, focussed := p.view.IsActiveAndFocussed()
		if active {
			baseBGStyle = baseBGStyle.DarkenedBG(10)
		} else if focussed {
			baseBGStyle = baseBGStyle.DarkenedBG(20)
		}

		nameWidth := 8
		modeWidth := 5
		padding := 1

		p.Renderer.DrawBox(x, y, w, h, baseBGStyle)
		p.Renderer.DrawText(x+padding, y, nameWidth, h, baseBGStyle.Italicized(), p.view.GetName())

		if focussed {
			switch p.view.GetMode() {
			case input.TextEditModeInsert:
				p.Renderer.DrawText(x+padding+nameWidth+padding, y, modeWidth, h, baseBGStyle.DarkenedFG(30).Invert(), "(ins)")
			case input.TextEditModeNormal:
				p.Renderer.DrawText(x+padding+nameWidth+padding, y, modeWidth, h, baseBGStyle.DarkenedFG(30), "(nrm)")
			default:
				p.Renderer.DrawText(x+padding+nameWidth+padding, y, modeWidth, h, p.Stylesheet.CategoryFallback, "( ? )")
			}
		}

		contentXOffset := padding + nameWidth + padding + modeWidth + padding
		p.Renderer.DrawText(x+contentXOffset, y, w-contentXOffset+padding, h, baseBGStyle.DarkenedBG(20), p.view.GetContent())

		if focussed {
			cursorX, cursorY := x+contentXOffset+(p.view.GetCursorPos()), y
			p.cursorController.Put(ui.CursorLocation{X: cursorX, Y: cursorY}, p.idStr)
		} else {
			p.cursorController.Delete(p.idStr)
		}

		// TODO(ja-he): wrap at word boundary; or something...
	}
}

// Undraw ensures that the cursor is hidden.
func (p *StringEditorPane) Undraw() {
	p.cursorController.Delete(p.idStr)
}

// GetPositionInfo returns information on a requested position in this pane (nil, for now).
func (p *StringEditorPane) GetPositionInfo(_, _ int) ui.PositionInfo { return nil }

// ProcessInput attempts to process the provided input.
func (p *StringEditorPane) ProcessInput(k input.Key) bool {
	active, _ := p.view.IsActiveAndFocussed()
	if !active {
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
	cursorController ui.CursorLocationRequestHandler,
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
		idStr:            "string-editor-pane-" + uuid.Must(uuid.NewRandom()).String(),
	}
}
