package processors

import (
	"fmt"

	"github.com/ja-he/dayplan/internal/input"
)

// ModalInputProcessor is an input processor that can take any number of input
// overlays over its base input processor.
// It delegates all processing to its processors.
// Implements input.ModalInputProcessor.
type ModalInputProcessor struct {
	base input.SimpleInputProcessor

	modalOverlays []input.SimpleInputProcessor
}

// NewModalInputProcessor returns a pointer to a new ModalInputProcessor with
// the given base processor and no overlays.
func NewModalInputProcessor(base input.SimpleInputProcessor) *ModalInputProcessor {
	return &ModalInputProcessor{
		base:          base,
		modalOverlays: make([]input.SimpleInputProcessor, 0),
	}
}

// CapturesInput returns whether this processor "captures" input, i.E. whether
// it ought to take priority in processing over other processors.
// This is useful, e.g., for prioritizing processors whith partial input
// sequences or for such overlays, that are to take complete priority by
// completely gobbling all input.
func (p *ModalInputProcessor) CapturesInput() bool {
	processor := p.getApplicableProcessor()
	return processor.CapturesInput()
}

// ProcessInput attempts to process the provided input.
// Returns whether the provided input "applied", i.E. the processor performed
// an action based on the input.
// ModalInputProcessor delegates input processing to its topmost overlay
// processor or -- if no overlays are present -- its base processor.
func (p *ModalInputProcessor) ProcessInput(key input.Key) bool {
	return p.getApplicableProcessor().ProcessInput(key)
}

// ApplyModalOverlay applies an overlay to this processor.
// It returns the processors index, by which in the future, all overlays down
// to and including this overlay can be removed
func (p *ModalInputProcessor) ApplyModalOverlay(overlay input.SimpleInputProcessor) (index uint) {
	p.modalOverlays = append(p.modalOverlays, overlay)
	return uint(len(p.modalOverlays) - 1)
}

// PopModalOverlay removes the topmost overlay from this processor.
func (p *ModalInputProcessor) PopModalOverlay() error {
	if len(p.modalOverlays) < 1 {
		return fmt.Errorf("attempt to pop from empty overlay stack")
	}
	p.modalOverlays = p.modalOverlays[:len(p.modalOverlays)-1]
	return nil
}

// PopModalOverlays pops all overlays down to and including the one at the
// specified index.
func (p *ModalInputProcessor) PopModalOverlays(index uint) {
	for i := uint(len(p.modalOverlays)); i > index; i-- {
		p.PopModalOverlay()
	}
}

func (p *ModalInputProcessor) getApplicableProcessor() input.SimpleInputProcessor {
	if len(p.modalOverlays) > 0 {
		return p.modalOverlays[len(p.modalOverlays)-1]
	} else {
		return p.base
	}
}

// GetHelp returns the input help map for this processor.
func (p *ModalInputProcessor) GetHelp() input.Help {
	return p.getApplicableProcessor().GetHelp()
}
