package processors

import (
	"fmt"

	"github.com/ja-he/dayplan/src/input"
)

type ModalInputProcessor struct {
	base input.SimpleInputProcessor

	modalOverlays []input.SimpleInputProcessor
}

func NewModalInputProcessor(base input.SimpleInputProcessor) *ModalInputProcessor {
	return &ModalInputProcessor{
		base:          base,
		modalOverlays: make([]input.SimpleInputProcessor, 0),
	}
}

func (p *ModalInputProcessor) CapturesInput() bool {
	processor := p.getApplicableProcessor()
	return processor.CapturesInput()
}
func (p *ModalInputProcessor) ProcessInput(key input.Key) bool {
	return p.getApplicableProcessor().ProcessInput(key)
}

func (p *ModalInputProcessor) ApplyModalOverlay(overlay input.SimpleInputProcessor) (index uint) {
	p.modalOverlays = append(p.modalOverlays, overlay)
	return uint(len(p.modalOverlays) - 1)
}
func (p *ModalInputProcessor) PopModalOverlay() error {
	if len(p.modalOverlays) < 1 {
		return fmt.Errorf("attempt to pop from empty overlay stack")
	}
	p.modalOverlays = p.modalOverlays[:len(p.modalOverlays)-1]
	return nil
}
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

func (p *ModalInputProcessor) GetHelp() input.Help {
	return p.getApplicableProcessor().GetHelp()
}
