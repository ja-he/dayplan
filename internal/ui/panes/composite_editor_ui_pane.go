package panes

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/control/edit"
	"github.com/ja-he/dayplan/internal/control/edit/editors"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
)

// CompositeEditorPane visualizes a composite editor.
type CompositeEditorPane struct {
	ui.LeafPane

	getFocussedEditorID func() editors.EditorID
	isInField           func() bool

	e        *editors.Composite
	subpanes map[editors.EditorID]ui.Pane

	log zerolog.Logger
}

// Draw draws the editor popup.
func (p *CompositeEditorPane) Draw() {
	if p.IsVisible() {
		x, y, w, h := p.Dims()

		// draw background
		style := getAlteredStyleForEditorStatus(p.Stylesheet.Editor, p.e.GetStatus())

		p.Renderer.DrawBox(x, y, w, h, style)
		p.Renderer.DrawText(x+1, y, w-2, 1, style, util.TruncateAt(p.e.GetID(), w-2))
		p.Renderer.DrawText(x, y, 1, 1, style.Bolded(), string(getRuneForEditorStatus(p.e.GetStatus())))

		// draw all subpanes
		fieldOrderSlice := p.e.GetFieldOrder()
		for i, id := range fieldOrderSlice {
			subpane, ok := p.subpanes[id]
			if !ok {
				p.log.Warn().Msgf("subpane '%s' (%d of %d) not found in subpanes (%v)", id, i, len(fieldOrderSlice), p.subpanes)
			} else {
				subpane.Draw()
			}
		}

	}
}

func getRuneForEditorStatus(status edit.EditorStatus) rune {
	switch status {
	case edit.EditorDescendantActive:
		return '.'
	case edit.EditorFocussed:
		return '*'
	case edit.EditorInactive:
		return ' '
	case edit.EditorSelected:
		return '>'
	}
	return '?'
}

func getAlteredStyleForEditorStatus(baseStyle styling.DrawStyling, status edit.EditorStatus) styling.DrawStyling {
	switch status {
	case edit.EditorInactive:
		return baseStyle.LightenedBG(10)
	case edit.EditorSelected:
		return baseStyle
	case edit.EditorDescendantActive:
		return baseStyle.DarkenedBG(10)
	case edit.EditorFocussed:
		return baseStyle.DarkenedBG(20).Bolded()
	}
	return baseStyle
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
			p.log.Warn().Msg("somehow, comosite editor is capturing input despite being in field; likely logic error")
		}
		return p.InputProcessor.ProcessInput(key)
	}

	if p.isInField() {
		editorID := p.getFocussedEditorID()
		focussedSubpane, ok := p.subpanes[editorID]
		if !ok {
			p.log.Error().Msgf("somehow, have an invalid focussed pane '%s' not in (%v)", editorID, p.subpanes)
			return false
		}
		processedBySubpane := focussedSubpane.ProcessInput(key)
		if processedBySubpane {
			return true
		}
		p.log.Warn().
			Str("key", key.ToDebugString()).
			Str("active-subeditor", fmt.Sprint(focussedSubpane.Identify())).
			Msg("input was not processed by active subeditor pane; will not be processed")
		return false
	}

	if p.InputProcessor == nil {
		p.log.Warn().Msg("input processor is nil; will not process input")
		return false
	}

	p.log.Trace().Str("key", key.ToDebugString()).Msg("processing input self")
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

	subpanes := map[editors.EditorID]ui.Pane{}

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
			return nil, fmt.Errorf("error constructing subpane of '%s' for subeditor '%s' (%s)", e.GetID(), child.Represents.GetID(), err.Error())
		}
		subpanes[child.Represents.GetID()] = subeditorPane
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
		subpanes:            subpanes,
		getFocussedEditorID: e.GetActiveFieldID,
		isInField:           e.IsInField,
		log:                 log.With().Str("source", "composite-pane").Logger(),
		e:                   e,
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
			return p.subpanes[p.getFocussedEditorID()].GetHelp()
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
			return ui.BoxRepresentation[edit.Editor]{}, fmt.Errorf("error translating child '%s' (%s)", child.GetID(), err.Error())
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
