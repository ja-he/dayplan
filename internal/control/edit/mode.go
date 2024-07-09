package edit

type MouseEditState int

const (
	_ MouseEditState = iota
	MouseEditStateNone
	MouseEditStateMoving
	MouseEditStateResizing
)

type EventEditMode = int

const (
	_ EventEditMode = iota
	EventEditModeNormal
	EventEditModeMove
	EventEditModeResize
)
