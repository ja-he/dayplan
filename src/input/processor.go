package input

type SimpleInputProcessor interface {
	CapturesInput() bool
	ProcessInput(key Key) bool

	GetHelp() Help
}

type ModalInputProcessor interface {
	SimpleInputProcessor

	ApplyModalOverlay(SimpleInputProcessor) (index uint)
	PopModalOverlay() error
	PopModalOverlays(index uint)
}
