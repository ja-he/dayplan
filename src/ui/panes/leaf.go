package panes

import (
	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
)

// Leaf is a simple set of data and implementation of a "leaf pane", i.E. a
// pane that does not have subpanes but instead makes actual draw calls.
type Leaf struct {
	Base
	renderer   ui.ConstrainedRenderer
	dimensions func() (x, y, w, h int)
	stylesheet styling.Stylesheet
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
func (p *Leaf) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draw panics. It MUST be overridden if it is to be called.
func (p *Leaf) Draw() {
	// TODO: draw fill with warning message
	panic("unimplemented draw")
}

// Undraw does nothing. Override this, if necessary.
func (p *Leaf) Undraw() {}

// HasFocus returns whether the pane has focus.
func (p *Leaf) HasFocus() bool {
	return p.Parent != nil && p.Parent.HasFocus() && p.Parent.Focusses() == p.Identify()
}

// Focusses returns the "none pane", as a leaf does not focus another pane.
func (p *Leaf) Focusses() ui.PaneID { return ui.NonePaneID }

// CapturesInput returns whether this processor "captures" input, i.E. whether
// it ought to take priority in processing over other processors.
func (p *Leaf) CapturesInput() bool {
	return p.InputProcessor != nil && p.InputProcessor.CapturesInput()
}

// ProcessInput attempts to process the provided input.
// Returns whether the provided input "applied", i.E. the processor performed
// an action based on the input.
// Defers to the panes' input processor.
func (p *Leaf) ProcessInput(key input.Key) bool {
	return p.InputProcessor != nil && p.InputProcessor.ProcessInput(key)
}

// ApplyModalOverlay applies an overlay to this processor.
// It returns the processors index, by which in the future, all overlays down
// to and including this overlay can be removed
func (p *Leaf) ApplyModalOverlay(overlay input.SimpleInputProcessor) (index uint) {
	if p.InputProcessor == nil {
		panic("ApplyModalOverlay on nil InputProcessor")
	}
	return p.InputProcessor.ApplyModalOverlay(overlay)
}

// PopModalOverlay removes the topmost overlay from this processor.
func (p *Leaf) PopModalOverlay() error {
	if p.InputProcessor == nil {
		panic("PopModalOverlay on nil InputProcessor")
	}
	return p.InputProcessor.PopModalOverlay()
}

// PopModalOverlays pops all overlays down to and including the one at the
// specified index.
func (p *Leaf) PopModalOverlays(index uint) {
	if p.InputProcessor == nil {
		panic("PopModalOverlays on nil InputProcessor")
	}
	p.InputProcessor.PopModalOverlays(index)
}

// GetHelp returns the input help map for this processor.
func (p *Leaf) GetHelp() input.Help {
	if p.InputProcessor == nil {
		return input.Help{}
	}
	return p.InputProcessor.GetHelp()
}

// FocusPrev does nothing, as this implements a leaf, which does not focus
// anything.
func (p *Leaf) FocusPrev() {}

// FocusNext does nothing, as this implements a leaf, which does not focus
// anything.
func (p *Leaf) FocusNext() {}
