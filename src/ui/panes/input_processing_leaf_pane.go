package panes

import (
	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/ui"
)

// TODO: naming?
// TODO: nil checks?
type InputProcessingLeafPane struct {
	InputProcessingPaneBaseData
}

func (p *InputProcessingLeafPane) HasFocus() bool {
	return p.Parent != nil && p.Parent.HasFocus() && p.Parent.Focusses() == p.Identify()
}

func (p *InputProcessingLeafPane) Focusses() ui.PaneID { return 0 }

// CapturesInput returns whether this processor "captures" input, i.E. whether
// it ought to take priority in processing over other processors.
func (p *InputProcessingLeafPane) CapturesInput() bool {
	return p.InputProcessor != nil && p.InputProcessor.CapturesInput()
}

// ProcessInput attempts to process the provided input.
// Returns whether the provided input "applied", i.E. the processor performed
// an action based on the input.
// Defers to the panes' input processor.
func (p *InputProcessingLeafPane) ProcessInput(key input.Key) bool {
	return p.InputProcessor != nil && p.InputProcessor.ProcessInput(key)
}

// ApplyModalOverlay applies an overlay to this processor.
// It returns the processors index, by which in the future, all overlays down
// to and including this overlay can be removed
func (p *InputProcessingLeafPane) ApplyModalOverlay(overlay input.SimpleInputProcessor) (index uint) {
	if p.InputProcessor == nil {
		panic("ApplyModalOverlay on nil InputProcessor")
	}
	return p.InputProcessor.ApplyModalOverlay(overlay)
}

// PopModalOverlay removes the topmost overlay from this processor.
func (p *InputProcessingLeafPane) PopModalOverlay() error {
	if p.InputProcessor == nil {
		panic("PopModalOverlay on nil InputProcessor")
	}
	return p.InputProcessor.PopModalOverlay()
}

// PopModalOverlays pops all overlays down to and including the one at the
// specified index.
func (p *InputProcessingLeafPane) PopModalOverlays(index uint) {
	if p.InputProcessor == nil {
		panic("PopModalOverlays on nil InputProcessor")
	}
	p.InputProcessor.PopModalOverlays(index)
}

// GetHelp returns the input help map for this processor.
func (p *InputProcessingLeafPane) GetHelp() input.Help {
	if p.InputProcessor == nil {
		return input.Help{}
	}
	return p.InputProcessor.GetHelp()
}

func NewLeafPaneBase(
	InputProcessor input.ModalInputProcessor,
) *InputProcessingLeafPane {
	return &InputProcessingLeafPane{InputProcessingPaneBaseData: InputProcessingPaneBaseData{
		ID:             ui.GeneratePaneID(),
		InputProcessor: InputProcessor,
	}}
}
