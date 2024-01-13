package panes

import (
	"fmt"
	"math/rand"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/control/edit"
	"github.com/ja-he/dayplan/internal/control/edit/editors"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
)

// CompositeEditorPane visualizes a composite editor.
type CompositeEditorPane struct {
	ui.LeafPane

	getFocussedIndex func() int
	isInField        func() bool

	e        *editors.Composite
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
		active, focussed := p.e.IsActiveAndFocussed()
		if active {
			style = style.DarkenedBG(20)
		} else if focussed {
			style = style.DarkenedBG(40)
		}
		p.Renderer.DrawBox(x, y, w, h, style)
		p.Renderer.DrawText(x, y, w, 1, p.Stylesheet.Editor.DarkenedFG(20), p.e.GetName())

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
	cursorController ui.CursorLocationRequestHandler,
	visible func() bool,
	inputConfig input.InputConfig,
	stylesheet styling.Stylesheet,
	e *editors.Composite,
) (*CompositeEditorPane, error) {

	subpanes := []ui.Pane{}

	minX, minY, maxWidth, maxHeight := renderer.Dimensions()
	uiBoxModel, err := translateEditorsCompositeToTUI(e, minX, minY, maxWidth, maxHeight)
	if err != nil {
		return nil, fmt.Errorf("error translating editor summary to UI box model (%s)", err.Error())
	}
	log.Debug().Msgf("have UI box model: %s", uiBoxModel.String())

	for _, child := range uiBoxModel.Children {
		childX, childY, childW, childH := child.X, child.Y, child.W, child.H
		subRenderer := ui.NewConstrainedRenderer(renderer, func() (int, int, int, int) { return childX, childY, childW, childH })
		var subeditorPane ui.Pane
		var err error
		switch child := child.Represents.(type) {
		case *editors.StringEditor:
			subeditorPane, err = NewStringEditorPane(subRenderer, cursorController, visible, stylesheet, inputConfig, child)
		case *editors.Composite:
			subeditorPane, err = NewCompositeEditorPane(subRenderer, cursorController, visible, inputConfig, stylesheet, child)
		default:
			err = fmt.Errorf("unhandled subeditor type '%T' (forgot to handle case)", child)
		}
		if err != nil {
			return nil, fmt.Errorf("error constructing subpane of '%s' for subeditor '%s' (%s)", e.GetName(), child.Represents.GetName(), err.Error())
		}
		subpanes = append(subpanes, subeditorPane)
	}

	inputProcessor, err := e.CreateInputProcessor(inputConfig)
	if err != nil {
		return nil, fmt.Errorf("could not construct input processor (%s)", err.Error())
	}

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
		subpanes:         subpanes,
		getFocussedIndex: e.GetActiveFieldIndex,
		isInField:        e.IsInField,
		log:              log.With().Str("source", "composite-pane").Logger(),
		bgoffs:           10 + rand.Intn(20),
		e:                e,
	}, nil
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

func translateEditorsEditorToTUI(e edit.Editor, minX, minY, maxWidth, maxHeight int) (ui.BoxRepresentation[edit.Editor], error) {

	switch e := e.(type) {

	case *editors.Composite:
		return translateEditorsCompositeToTUI(e, minX, minY, maxWidth, maxHeight)

	case *editors.StringEditor:
		return ui.BoxRepresentation[edit.Editor]{
			X:          minX,
			Y:          minY,
			W:          maxWidth,
			H:          1,
			Represents: e,
			Children:   nil,
		}, nil

	default:
		return ui.BoxRepresentation[edit.Editor]{}, fmt.Errorf("unhandled editor type '%T' (forgot to handle case)", e)

	}

}

func translateEditorsCompositeToTUI(e *editors.Composite, minX, minY, maxWidth, maxHeight int) (ui.BoxRepresentation[edit.Editor], error) {

	var children []ui.BoxRepresentation[edit.Editor]
	computedHeight := 1
	rollingY := minY + 1
	for _, child := range e.GetFields() {
		childBoxRepresentation, err := translateEditorsEditorToTUI(child, minX+1, rollingY, maxWidth-2, maxHeight-2)
		if err != nil {
			return ui.BoxRepresentation[edit.Editor]{}, fmt.Errorf("error translating child '%s' (%s)", child.GetName(), err.Error())
		}
		rollingY += childBoxRepresentation.H + 1
		children = append(children, childBoxRepresentation)
		computedHeight += childBoxRepresentation.H + 1
	}
	return ui.BoxRepresentation[edit.Editor]{
		X:          minX,
		Y:          minY,
		W:          maxWidth,
		H:          computedHeight,
		Represents: e,
		Children:   children,
	}, nil

}
