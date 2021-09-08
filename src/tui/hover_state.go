package tui

type HoverState int

const (
	HoverStateNone HoverState = iota
	HoverStateMove
	HoverStateResize
)
