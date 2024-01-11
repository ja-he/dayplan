package panes

import (
	"fmt"
	"math/rand"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/control/edit/views"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
)

// CompositeEditorPane visualizes a composite editor.
type CompositeEditorPane struct {
	ui.LeafPane

	getFocussedIndex func() int
	isInField        func() bool

	view     views.CompositeEditorView
	subpanes []ui.Pane

	bgoffs int

	log zerolog.Logger
}

// Draw draws the editor popup.
func (p *CompositeEditorPane) Draw() {
	if p.IsVisible() {
		x, y, w, h := p.Dims()

		// draw background
		style := p.Stylesheet.Editor.DarkenedBG(p.bgoffs)
		active, focussed := p.view.IsActiveAndFocussed()
		if active {
			style = style.DarkenedBG(20)
		} else if focussed {
			style = style.DarkenedBG(40)
		}
		p.Renderer.DrawBox(x, y, w, h, style)
		p.Renderer.DrawText(x, y, w, 1, p.Stylesheet.Editor.DarkenedFG(20), p.view.GetName())

		// draw all subpanes
		for _, subpane := range p.subpanes {
			subpane.Draw()
		}

	}
}

// Undraw ensures that the cursor is hidden.
func (p *CompositeEditorPane) Undraw() {
	for _, subpane := range p.subpanes {
		subpane.Undraw()
	}
}

// ProcessInput attempts to process the provided input.
// Returns whether the provided input "applied", i.E. the processor performed
// an action based on the input.
// Defers to the panes' input processor or its focussed subpanes.
func (p *CompositeEditorPane) ProcessInput(key input.Key) bool {
	if p.InputProcessor != nil && p.InputProcessor.CapturesInput() {
		if p.isInField() {
			p.log.Warn().Msgf("comp: somehow, comosite editor is capturing input despite being in field; likely logic error")
		}
		return p.InputProcessor.ProcessInput(key)
	}

	if p.isInField() {
		focussedIndex := p.getFocussedIndex()
		if focussedIndex < 0 || focussedIndex >= len(p.subpanes) {
			p.log.Error().Msgf("comp: somehow, focussed index for composite editor is out of bounds; %d < 0 || %d >= %d", focussedIndex, focussedIndex, len(p.subpanes))
			return false
		}
		processedBySubpane := p.subpanes[focussedIndex].ProcessInput(key)
		if processedBySubpane {
			return true
		}
		p.log.Warn().Msgf("comp: input '%s' was not processed by active subeditor pane; will not be processed", key.ToDebugString())
		return false
	}

	if p.InputProcessor == nil {
		p.log.Warn().Msg("comp: input processor is nil; will not process input")
		return false
	}

	p.log.Trace().Msgf("comp: processing input '%s' self", key.ToDebugString())
	return p.InputProcessor.ProcessInput(key)
}

// GetPositionInfo returns information on a requested position in this pane (nil, for now).
func (p *CompositeEditorPane) GetPositionInfo(_, _ int) ui.PositionInfo { return nil }

// NewCompositeEditorPane creates a new CompositeEditorPane.
func NewCompositeEditorPane(
	renderer ui.ConstrainedRenderer,
	visible func() bool,
	inputProcessor input.ModalInputProcessor,
	stylesheet styling.Stylesheet,
	subEditors []ui.Pane,
	getFocussedIndex func() int,
	isInField func() bool,
	view views.CompositeEditorView,
) *CompositeEditorPane {
	return &CompositeEditorPane{
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
		subpanes:         subEditors,
		getFocussedIndex: getFocussedIndex,
		isInField:        isInField,
		log:              log.With().Str("source", "composite-pane").Logger(),
		bgoffs:           10 + rand.Intn(20),
		view:             view,
	}
}

// GetHelp returns the input help map for this composite pane.
func (p *CompositeEditorPane) GetHelp() input.Help {
	ownHelp := func() input.Help {
		if p.InputProcessor == nil {
			return input.Help{}
		}
		return p.InputProcessor.GetHelp()
	}()
	activeFieldHelp := func() input.Help {
		if p.isInField() {
			return p.subpanes[p.getFocussedIndex()].GetHelp()
		}
		return input.Help{}
	}()
	result := input.Help{}
	for k, v := range ownHelp {
		result[k] = v
	}
	for k, v := range activeFieldHelp {
		result[k] = v
	}
	return result
}

func (p *CompositeEditorPane) GetDebugInfo() string {
	x, y, w, h := p.Dimensions()
	info := fmt.Sprintf("[ +%d+%d:%dx%d ", x, y, w, h)
	for _, subpane := range p.subpanes {
		switch sp := subpane.(type) {
		case *CompositeEditorPane:
			info += sp.GetDebugInfo()
		case *StringEditorPane:
			x, y, w, h := sp.Dimensions()
			info += fmt.Sprintf("( %d+%d:%dx%d )", x, y, w, h)
		default:
			info += fmt.Sprintf("<unhandled subpane type %T>", subpane)
		}
	}
	info += "]"
	return info
}
