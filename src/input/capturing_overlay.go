package input

// CapturingOverlay is a wrapper over a SimpleInputProcessor that always claims to capture input,
// which can be a desirable behavior for modal overlays.
type CapturingOverlay struct {
	Processor SimpleInputProcessor
}

func (o *CapturingOverlay) CapturesInput() bool     { return true }
func (o *CapturingOverlay) ProcessInput(k Key) bool { return o.Processor.ProcessInput(k) }
func (o *CapturingOverlay) GetHelp() Help           { return o.Processor.GetHelp() }

// CapturingOverlayWrap returns a wrapper over the given SimpleInputProcessor that always captures
// all input.
func CapturingOverlayWrap(s SimpleInputProcessor) *CapturingOverlay {
	return &CapturingOverlay{Processor: s}
}
