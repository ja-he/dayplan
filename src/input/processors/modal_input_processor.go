package processors

import "github.com/ja-he/dayplan/src/input"

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
	haveOverlay := (processor != p.base)
	return haveOverlay || p.base.CapturesInput()
}
func (p *ModalInputProcessor) ProcessInput(key input.Key) bool {
	return p.getApplicableProcessor().ProcessInput(key)
}

func (p *ModalInputProcessor) ApplyModalOverlay(overlay input.SimpleInputProcessor) (index int) {
	p.modalOverlays = append(p.modalOverlays, overlay)
	return len(p.modalOverlays) - 1
}
func (p *ModalInputProcessor) PopModalOverlay() {
	if len(p.modalOverlays) < 1 {
		panic("attempt to pop from empty overlay stack")
	}
	p.modalOverlays = p.modalOverlays[:len(p.modalOverlays)-1]
}
func (p *ModalInputProcessor) PopModalOverlays(index int) {
	for i := len(p.modalOverlays) - 1; i >= index; i-- {
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
