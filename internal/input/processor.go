package input

// SimpleInputProcessor can process the input it is configured for and provide
// help information for that configuration. It can also "capture" input to
// ensure its precedence over other processors, e.g. when it has partial input.
type SimpleInputProcessor interface {

	// CapturesInput returns whether this processor "captures" input, i.E. whether
	// it ought to take priority in processing over other processors.
	// This is useful, e.g., for prioritizing processors with partial input
	// sequences or for such overlays, that are to take complete priority by
	// completely gobbling all input.
	CapturesInput() bool

	// ProcessInput attempts to process the provided input.
	// Returns whether the provided input "applied", i.E. the processor performed
	// an action based on the input.
	ProcessInput(key Key) bool

	// GetHelp returns the input help map for this processor.
	GetHelp() Help
}

// ModalInputProcessor is an input processor that (additionally to
// SimpleInputProcessor) can be temporarily overlaid with any number of
// additional input processors, which can be removed one-by-one of the top or
// by their indices.
type ModalInputProcessor interface {
	SimpleInputProcessor

	// ApplyModalOverlay applies an overlay to this processor.
	// It returns the processors index, by which in the future, all overlays down
	// to and including this overlay can be removed
	ApplyModalOverlay(SimpleInputProcessor) (index uint)

	// PopModalOverlay removes the topmost overlay from this processor.
	PopModalOverlay() error

	// PopModalOverlays pops all overlays down to and including the one at the
	// specified index.
	PopModalOverlays(index uint)
}
