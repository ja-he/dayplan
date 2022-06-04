package input

// CapturingOverlay is a wrapper over a SimpleInputProcessor that always claims to capture input,
// which can be a desirable behavior for modal overlays.
type CapturingOverlay struct {
	Processor SimpleInputProcessor
}

// CapturesInput returns whether this processor "captures" input, i.E. whether
// it ought to take priority in processing over other processors; this is
// always the case for the capturing overlay.
func (o *CapturingOverlay) CapturesInput() bool { return true }

// ProcessInput attempts to process the provided input.
// Returns whether the provided input "applied", i.E. the processor performed
// an action based on the input. This defers to the underlying processor.
func (o *CapturingOverlay) ProcessInput(k Key) bool { return o.Processor.ProcessInput(k) }

// GetHelp returns the input help map for this processor.
func (o *CapturingOverlay) GetHelp() Help { return o.Processor.GetHelp() }

// CapturingOverlayWrap returns a wrapper over the given SimpleInputProcessor that always captures
// all input.
func CapturingOverlayWrap(s SimpleInputProcessor) *CapturingOverlay {
	return &CapturingOverlay{Processor: s}
}
