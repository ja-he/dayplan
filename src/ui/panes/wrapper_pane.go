package panes

import (
	"fmt"
	"math"

	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

// WrapperPane is a generic wrapper pane whithout any rendering logic of its
// own.
type WrapperPane struct {
	Parent ui.FocusQueriable

	drawables   []ui.Pane
	focussables []ui.FocussablePane

	inputProcessor input.ModalInputProcessor
	focussedPane   ui.FocussablePane
}

// Draw draws this pane by drawing all its subpanes.
// Absent subpanes this draws nothing.
func (p *WrapperPane) Draw() {
	for _, drawable := range p.drawables {
		drawable.Draw()
	}
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
func (p *WrapperPane) Dimensions() (x, y, w, h int) {
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
func (p *WrapperPane) GetPositionInfo(x, y int) ui.PositionInfo {
	for _, pane := range p.drawables {
		if util.NewRect(pane.Dimensions()).Contains(x, y) {
			return pane.GetPositionInfo(x, y)
		}
	}

	panic(fmt.Sprint("none of the current wrapper pane's subpanes contains pos", x, y))
}

func (p *WrapperPane) FocusNext() {
	for i := range p.focussables {
		if p.focussedPane == p.focussables[i] {
			if i < len(p.focussables)-1 {
				p.focussedPane = p.focussables[i+1]
			}
			return
		}
	}
}

func (p *WrapperPane) FocusPrev() {
	for i := range p.focussables {
		if p.focussedPane == p.focussables[i] {
			if i > 0 {
				p.focussedPane = p.focussables[i-1]
			}
			return
		}
	}
}

// CapturesInput returns whether this processor "captures" input, i.E. whether
// it ought to take priority in processing over other processors.
func (p *WrapperPane) CapturesInput() bool {
	childCaptures := p.focussedPane != nil && p.focussedPane.CapturesInput()
	selfCaptures := p.inputProcessor != nil && p.inputProcessor.CapturesInput()
	return childCaptures || selfCaptures
}

// ProcessInput attempts to process the provided input.
// Returns whether the provided input "applied", i.E. the processor performed
// an action based on the input.
// Defers to the panes' input processor or its focussed subpanes.
func (p *WrapperPane) ProcessInput(key input.Key) bool {
	if p.inputProcessor != nil && p.inputProcessor.CapturesInput() {
		return p.inputProcessor.ProcessInput(key)
	} else if p.Focusses() != nil && p.Focusses().CapturesInput() {
		return p.Focusses().ProcessInput(key)
	} else {
		return (p.Focusses() != nil && p.Focusses().ProcessInput(key)) || (p.inputProcessor != nil && p.inputProcessor.ProcessInput(key))
	}
}

func (p *WrapperPane) HasFocus() bool              { return p.Parent.HasFocus() && p.Parent.Focusses() == p }
func (p *WrapperPane) Focusses() ui.FocussablePane { return p.focussedPane }

func (p *WrapperPane) SetParent(parent ui.FocusQueriable) {
	p.Parent = parent
}

// ApplyModalOverlay applies an overlay to this processor.
// It returns the processors index, by which in the future, all overlays down
// to and including this overlay can be removed
func (p *WrapperPane) ApplyModalOverlay(overlay input.SimpleInputProcessor) (index uint) {
	return p.inputProcessor.ApplyModalOverlay(overlay)
}

// PopModalOverlay removes the topmost overlay from this processor.
func (p *WrapperPane) PopModalOverlay() error {
	return p.inputProcessor.PopModalOverlay()
}

// PopModalOverlays pops all overlays down to and including the one at the
// specified index.
func (p *WrapperPane) PopModalOverlays(index uint) {
	p.inputProcessor.PopModalOverlays(index)
}

// GetHelp returns the input help map for this processor.
func (p *WrapperPane) GetHelp() input.Help {
	result := input.Help{}

	for k, v := range p.inputProcessor.GetHelp() {
		result[k] = v
	}
	if p.Focusses() != nil {
		for k, v := range p.Focusses().GetHelp() {
			result[k] = v
		}
	}

	return result
}

// NewWrapperPane constructs and returns a new WrapperPane.
func NewWrapperPane(
	drawables []ui.Pane,
	focussables []ui.FocussablePane,
	inputProcessor input.ModalInputProcessor,
) *WrapperPane {
	p := &WrapperPane{
		focussables:    focussables,
		drawables:      drawables,
		inputProcessor: inputProcessor,
	}
	if len(p.focussables) > 0 {
		p.focussedPane = p.focussables[0]
	}
	for _, focussable := range p.focussables {
		focussable.SetParent(p)
	}
	return p
}
