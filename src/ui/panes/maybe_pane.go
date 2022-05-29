package panes

import (
	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/ui"
)

// MaybePane wraps another pane with a condition and exposes its functionality,
// iff the condition holds. Otherwise its operations do or return nothing or,
// in some cases, panic!
//
// One use-case for this type is allowing a fixed number (31) of panes for a
// month, the last few of which are only sometimes used.
type MaybePane struct {
	pane      ui.Pane
	condition func() bool
}

// IsVisible iff condition.
func (p *MaybePane) IsVisible() bool { return p.condition() }

// Draw draws this pane.
func (p *MaybePane) Draw() {
	if p.condition() {
		p.pane.Draw()
	}
}

// Undraw iff condition.
func (p *MaybePane) Undraw() {
	if p.condition() {
		p.pane.Undraw()
	}
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
func (p *MaybePane) Dimensions() (x, y, w, h int) { return p.pane.Dimensions() }

// GetPositionInfo returns information on a requested position in the
// underlying pane, iff condition.
func (p *MaybePane) GetPositionInfo(x, y int) ui.PositionInfo {
	if p.condition() {
		return p.pane.GetPositionInfo(x, y)
	}
	return ui.NewPositionInfo(
		ui.NoPane,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
}

// SetParent iff condition.
func (p *MaybePane) SetParent(parent ui.FocusQueriable) {
	if p.condition() {
		p.pane.SetParent(parent)
	}
}

// PopModalOverlay iff condition.
func (p *MaybePane) PopModalOverlay() error {
	if p.condition() {
		return p.pane.PopModalOverlay()
	} else {
		panic("attempting to pop overlay from maybe-pane with !condition")
	}
}

// PopModalOverlays iff condition.
func (p *MaybePane) PopModalOverlays(index uint) {
	if p.condition() {
		p.pane.PopModalOverlays(index)
	} else {
		panic("attempting to pop overlay from maybe-pane with !condition")
	}
}

// ApplyModalOverlay iff condition.
func (p *MaybePane) ApplyModalOverlay(overlay input.SimpleInputProcessor) uint {
	if p.condition() {
		return p.pane.ApplyModalOverlay(overlay)
	} else {
		panic("attempting to apply input overlay to maybe-pane with !condition")
	}
}

// CapturesInput iff condition.
func (p *MaybePane) CapturesInput() bool {
	if p.condition() {
		return p.pane.CapturesInput()
	} else {
		return false
	}
}

// GetHelp iff condition.
func (p *MaybePane) GetHelp() input.Help {
	if p.condition() {
		return p.pane.GetHelp()
	} else {
		return input.Help{}
	}
}

// Focusses iff condition.
func (p *MaybePane) Focusses() ui.PaneID {
	if p.condition() {
		return p.pane.Identify()
	} else {
		return ui.NonePaneID
	}
}

// HasFocus iff condition.
func (p *MaybePane) HasFocus() bool {
	return p.condition() && p.pane.HasFocus()
}

// Identify by ID, iff condtion.
func (p *MaybePane) Identify() ui.PaneID {
	if p.condition() {
		return p.pane.Identify()
	} else {
		return ui.NonePaneID
	}
}

// ProcessInput iff condition.
func (p *MaybePane) ProcessInput(k input.Key) bool {
	if p.condition() {
		return p.pane.ProcessInput(k)
	} else {
		return false
	}
}

// FocusPrev iff condition.
func (p *MaybePane) FocusPrev() {
	if p.condition() {
		p.pane.FocusPrev()
	}
}

// FocusNext iff condition.
func (p *MaybePane) FocusNext() {
	if p.condition() {
		p.pane.FocusNext()
	}
}

// NewMaybePane constructs and returns a new MaybePane.
func NewMaybePane(
	condition func() bool,
	pane ui.Pane,
) *MaybePane {
	return &MaybePane{
		condition: condition,
		pane:      pane,
	}
}
