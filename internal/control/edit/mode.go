package edit

type MouseEditState int

const (
	_ MouseEditState = iota
	MouseEditStateNone
	MouseEditStateMoving
	MouseEditStateResizing
)

func (s MouseEditState) toString() string {
	return "TODO"
}

type EventEditMode = int

const (
	_ EventEditMode = iota
	EventEditModeNormal
	EventEditModeMove
	EventEditModeResize
)
