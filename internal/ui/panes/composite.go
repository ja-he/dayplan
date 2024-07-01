package panes

import (
	"fmt"
	"math"

	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
)

// Composite is a generic wrapper pane without any rendering logic of its
// own.
type Composite struct {
	ui.BasePane

	drawables   []ui.Pane
	focussables []ui.Pane

	FocussedPane ui.Pane
}

// Draw draws this pane by drawing all its subpanes.
// Absent subpanes this draws nothing.
func (p *Composite) Draw() {
	for _, drawable := range p.drawables {
		drawable.Draw()
	}
}

// Undraw calls undraw on each drawable in the composite.
func (p *Composite) Undraw() {
	for _, drawable := range p.drawables {
		drawable.Undraw()
	}
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
func (p *Composite) Dimensions() (x, y, w, h int) {
	minX, minY := math.MaxInt, math.MaxInt
	maxX, maxY := 0, 0
	for _, drawable := range p.drawables {
		dx, dy, dw, dh := drawable.Dimensions()
		if dx < minX {
			minX = dx
		}
		if dy < minY {
			minY = dy
		}
		if dx+dw > maxX {
			maxX = dx + dw
		}
		if dy+dh > maxY {
			maxY = dy + dh
		}
	}
	return minX, minY, maxX - minX, maxY - minY
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *Composite) GetPositionInfo(x, y int) ui.PositionInfo {
	for _, pane := range p.drawables {
		if pane.IsVisible() && util.NewRect(pane.Dimensions()).Contains(x, y) {
			return pane.GetPositionInfo(x, y)
		}
	}

	panic(fmt.Sprint("none of the current wrapper pane's subpanes contains pos", x, y))
}

// FocusNext focusses the next focussable in the composite.
func (p *Composite) FocusNext() {
	for i := range p.focussables {
		if p.FocussedPane == p.focussables[i] {
			for j := i + 1; j < len(p.focussables); j++ {
				if p.focussables[j].IsVisible() {
					p.FocussedPane = p.focussables[j]
					return
				}
			}
		}
	}
}

// FocusPrev focusses the previous focussable in the composite.
func (p *Composite) FocusPrev() {
	for i := range p.focussables {
		if p.FocussedPane == p.focussables[i] {
			for j := i - 1; j >= 0; j-- {
				if p.focussables[j].IsVisible() {
					p.FocussedPane = p.focussables[j]
					return
				}
			}
		}
	}
}

func (p *Composite) EnsureFocusIsOnVisible() {
	if !p.FocussedPane.IsVisible() {
		for i := range p.focussables {
			if p.focussables[i].IsVisible() {
				p.FocussedPane = p.focussables[i]
			}
		}
	}
}

// CapturesInput returns whether this processor "captures" input, i.E. whether
// it ought to take priority in processing over other processors.
func (p *Composite) CapturesInput() bool {
	childCaptures := p.FocussedPane != nil && p.FocussedPane.CapturesInput()
	selfCaptures := p.InputProcessor != nil && p.InputProcessor.CapturesInput()
	return childCaptures || selfCaptures
}

// ProcessInput attempts to process the provided input.
// Returns whether the provided input "applied", i.E. the processor performed
// an action based on the input.
// Defers to the panes' input processor or its focussed subpanes.
func (p *Composite) ProcessInput(key input.Key) bool {
	if p.InputProcessor != nil && p.InputProcessor.CapturesInput() {
		return p.InputProcessor.ProcessInput(key)
	} else if p.FocussedPane != nil && p.FocussedPane.CapturesInput() {
		return p.FocussedPane.ProcessInput(key)
	} else {
		return (p.FocussedPane != nil && p.FocussedPane.ProcessInput(key)) || (p.InputProcessor != nil && p.InputProcessor.ProcessInput(key))
	}
}

// HasFocus indicates, whether this composite pane has focus.
func (p *Composite) HasFocus() bool {
	return p.Parent.HasFocus() && p.Parent.Focusses() == p.Identify()
}

// Focusses returns the ID of the pane focussed by this composite.
func (p *Composite) Focusses() ui.PaneID { return p.FocussedPane.Identify() }

// SetParent sets the parent of this composite pane.
func (p *Composite) SetParent(parent ui.PaneQuerier) { p.Parent = parent }

// ApplyModalOverlay applies an overlay to this processor.
// It returns the processors index, by which in the future, all overlays down
// to and including this overlay can be removed
func (p *Composite) ApplyModalOverlay(overlay input.SimpleInputProcessor) (index uint) {
	return p.InputProcessor.ApplyModalOverlay(overlay)
}

// PopModalOverlay removes the topmost overlay from this processor.
func (p *Composite) PopModalOverlay() error {
	return p.InputProcessor.PopModalOverlay()
}

// PopModalOverlays pops all overlays down to and including the one at the
// specified index.
func (p *Composite) PopModalOverlays(index uint) {
	p.InputProcessor.PopModalOverlays(index)
}

// GetHelp returns the input help map for this processor.
func (p *Composite) GetHelp() input.Help {
	result := input.Help{}

	for k, v := range p.InputProcessor.GetHelp() {
		result[k] = v
	}
	if p.FocussedPane != nil {
		for k, v := range p.FocussedPane.GetHelp() {
			result[k] = v
		}
	}

	return result
}

// NewWrapperPane constructs and returns a new WrapperPane.
func NewWrapperPane(
	drawables []ui.Pane,
	focussables []ui.Pane,
	inputProcessor input.ModalInputProcessor,
) *Composite {
	p := &Composite{
		focussables: focussables,
		drawables:   drawables,
		BasePane: ui.BasePane{
			InputProcessor: inputProcessor,
			ID:             ui.GeneratePaneID(),
		},
	}
	if len(p.focussables) > 0 {
		p.FocussedPane = p.focussables[0]
	}
	for _, focussable := range p.focussables {
		focussable.SetParent(p)
	}
	return p
}
