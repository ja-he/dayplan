package hover_state

type HoverState int

const (
	None HoverState = iota
	Move
	Resize
)
