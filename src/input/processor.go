package input

type SimpleInputProcessor interface {
	CapturesInput() bool
	ProcessInput(key Key) bool

	GetHelp() Help
}

type ModalInputProcessor interface {
	SimpleInputProcessor

	ApplyModalOverlay(SimpleInputProcessor) (index int)
	PopModalOverlay()
	PopModalOverlays(index int)
}
