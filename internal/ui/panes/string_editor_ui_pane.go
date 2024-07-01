package panes

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/control/edit"
	"github.com/ja-he/dayplan/internal/control/edit/editors"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
)

// StringEditorPane visualizes the editing of a string (as seen by a StringEditorView).
type StringEditorPane struct {
	ui.LeafPane

	e *editors.StringEditor

	cursorController ui.CursorLocationRequestHandler

	idStr string
}

// Draw draws the editor popup.
func (p *StringEditorPane) Draw() {
	if p.IsVisible() {
		x, y, w, h := p.Dims()

		status := p.e.GetStatus()
		baseStyle := getAlteredStyleForEditorStatus(p.Stylesheet.Editor, status)
		boxStyle := baseStyle
		fieldStyle := baseStyle
		labelStyle := baseStyle

		nameWidth := 8
		modeWidth := 5
		padding := 1

		p.Renderer.DrawBox(x, y, w, h, boxStyle)
		p.Renderer.DrawText(x+padding, y, nameWidth, h, labelStyle, p.e.GetName())
		p.Renderer.DrawText(x, y, 1, 1, boxStyle.Bolded(), string(getRuneForEditorStatus(status)))

		if status == edit.EditorFocussed {
			switch p.e.GetMode() {
			case input.TextEditModeInsert:
				p.Renderer.DrawText(x+padding+nameWidth+padding, y, modeWidth, h, fieldStyle.DarkenedFG(30).Invert(), "(ins)")
			case input.TextEditModeNormal:
				p.Renderer.DrawText(x+padding+nameWidth+padding, y, modeWidth, h, fieldStyle.DarkenedFG(30), "(nrm)")
			default:
				p.Renderer.DrawText(x+padding+nameWidth+padding, y, modeWidth, h, p.Stylesheet.CategoryFallback, "( ? )")
			}
		}

		contentXOffset := padding + nameWidth + padding + modeWidth + padding
		p.Renderer.DrawText(x+contentXOffset, y, w-contentXOffset+padding, h, fieldStyle, p.e.GetContent())

		if status == edit.EditorFocussed {
			cursorX, cursorY := x+contentXOffset+(p.e.GetCursorPos()), y
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
	if p.e.GetStatus() == edit.EditorInactive {
		log.Warn().Msgf("string editor pane asked to process input despite view reporting not active; likely logic error")
	}
	return p.LeafPane.ProcessInput(k)
}

// NewStringEditorPane creates a new StringEditorPane.
func NewStringEditorPane(
	renderer ui.ConstrainedRenderer,
	cursorController ui.CursorLocationRequestHandler,
	visible func() bool,
	stylesheet styling.Stylesheet,
	inputConfig input.InputConfig,
	e *editors.StringEditor,
) (*StringEditorPane, error) {
	inputProcessor, err := e.CreateInputProcessor(inputConfig)
	if err != nil {
		return nil, fmt.Errorf("could not construct normal mode input tree (%s)", err.Error())
	}

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
		e:                e,
		cursorController: cursorController,
		idStr:            "string-editor-pane-" + uuid.Must(uuid.NewRandom()).String(),
	}, nil
}
