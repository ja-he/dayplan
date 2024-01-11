package ui

import (
	"fmt"

	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/styling"
)

// Pane is a UI pane.
//
// ...
//
// An InputProcessingPane can focus another InputProcessingPane, in fact one of
// any number of "child" InputProcessingPanes.
// Thus they can be structured as a tree and any node in this tree can be asked
// whether it HasFocus, and what it Focusses; generally, to answer wheter a
// pane HasFocus, it would probably consult it's parent whether the parent
// HasFocus and which pane it Focusses.
//
// In this tree of panes, an InputProcessingPane's should generally have a
// parent, which can be set with SetParent; an exception would be the root pane
// of the tree.
type Pane interface {
	Draw()
	Undraw()
	IsVisible() bool
	Dimensions() (x, y, w, h int)
	GetPositionInfo(x, y int) PositionInfo

	input.ModalInputProcessor

	PaneQuerier

	SetParent(PaneQuerier)

	// NOTE: always an option to add/alter to focus{left,right,up,down} or similar
	FocusNext()
	FocusPrev()
}

// PaneQuerier are the querying member functions of a pane.
//
// E.g. letting a child access its parent, this allows limiting the childs
// access.
type PaneQuerier interface {
	HasFocus() bool
	Focusses() PaneID
	IsVisible() bool
	Identify() PaneID
}

// PaneType is the type of the bottommost meaningful UI pane.
//
// It's conceivable that panes could have sub-panes for convenience in rendering
// that aren't meaningfully different from the top pane or their individual
// purpose isn't relevant outside of the pane structure.
type PaneType int

const (
	_ PaneType = iota
	// NoPane describes anything that is not on a meaningful UI Pane, perhaps in
	// padding space.
	NoPane
	// WeatherPaneType represents a pane displaying weather information.
	WeatherPaneType
	// TimelinePaneType represnets a timeline.
	TimelinePaneType
	// EventsPaneType represents an events pane.
	EventsPaneType
	// ToolsPaneType represents a tools pane.
	ToolsPaneType
	// TasksPaneType represents a tasks pane.
	TasksPaneType
	// StatusPaneType represents a status pane (or status bar).
	StatusPaneType
	// EditorPaneType represents an editor (popup/floating) pane.
	EditorPaneType
	// LogPaneType represents a log pane.
	LogPaneType
	// SummaryPaneType represents a summary pane.
	SummaryPaneType
)

// ToString returns the name of this pane type as a string, primarily for
// debugging and logging purposes.
func (t PaneType) ToString() string {
	switch t {
	case NoPane:
		return "NoPane"
	case WeatherPaneType:
		return "WeatherPaneType"
	case TimelinePaneType:
		return "TimelinePaneType"
	case EventsPaneType:
		return "EventsPaneType"
	case ToolsPaneType:
		return "ToolsPaneType"
	case StatusPaneType:
		return "StatusPaneType"
	case EditorPaneType:
		return "EditorPaneType"
	case LogPaneType:
		return "LogPaneType"
	case SummaryPaneType:
		return "SummaryPaneType"
	}
	return "[UNKNOWN]"
}

// ActiveView is the active view of the UI, which could be a single day, a week
// or a full month (or in the future any other stretch of time that's to be
// shown).
type ActiveView int

const (
	_ ActiveView = iota
	// ViewDay represents the view in which a single day is visible.
	ViewDay
	// ViewWeek represents the view in which a full week (Monday to Sunday) is
	// visible.
	ViewWeek
	// ViewMonth represents the view in which a full month (first to last) is
	// visible.
	ViewMonth
)

// PaneID uniquely identifies a pane. No two panes must ever share a PaneID.
type PaneID uint

// NonePaneID represents "no pane" or "invalid pane". Panes guaranteed to be
// assigned different IDs by GeneratePaneID.
const NonePaneID PaneID = 0

var id = NonePaneID

// GeneratePaneID generates a new unique pane ID.
var GeneratePaneID = func() PaneID {
	id++
	return id
}

// EventBoxPart describes the part of an event box (the visual representation
// of an event in the user interface).
// For example this could describe what part of an event the mouse is hovering
// over.
type EventBoxPart int

// NOTE: add values, as they are needed.
const (
	_ EventBoxPart = iota
	// Nowhere. It's not part of the box. It's elsewhere.
	EventBoxNowhere
	// In the bottom right corner of the box.
	EventBoxBottomRight
	// Along the top edge of the box.
	EventBoxTopEdge
	// Inside the box. Anywhere inside the box not described by the above.
	EventBoxInterior
)

// ToString converts an an EventBoxPart to a string.
func (p EventBoxPart) ToString() string {
	switch p {
	case EventBoxNowhere:
		return "EventBoxNowhere"
	case EventBoxInterior:
		return "EventBoxInterior"
	case EventBoxBottomRight:
		return "EventBoxBottomRight"
	case EventBoxTopEdge:
		return "EventBoxTopEdge"
	}
	return "[unknown event box part]"
}

type Renderer interface {
	// Draw a box of the indicated dimensions at the indicated location but
	// limited to the constraint (bounding box) of the renderer.
	// In the case that the box is  not fully contained by the bounding box,
	// it is truncated to fit and drawn at the corrected coordinates with the
	// corrected dimensions.
	DrawBox(x, y, w, h int, style styling.DrawStyling)
	// Draw text within the box described by the given coordinates and dimensions,
	// but limited to the constraint (bounding box) of the renderer.
	// In the case that the box is  not fully contained by the bounding box,
	// it is truncated to fit and drawn at the corrected coordinates with the
	// corrected dimensions.
	DrawText(x, y, w, h int, style styling.DrawStyling, text string)
}

// ConstrainedRenderer is a renderer that is assumed to be constrained to
// certain dimensions, i.E. it does not draw outside of them.
type ConstrainedRenderer interface {
	Renderer

	// Dimensions returns the dimensions of the renderer.
	Dimensions() (x, y, w, h int)
}

// RenderOrchestratorControl is the set of functions of a renderer (e.g.,
// tcell.Screen) that the root pane needs to use to have full control over a
// render cycle. Other panes should not need this access to the renderer.
type RenderOrchestratorControl interface {
	Clear()
	Show()
}

// MouseCursorPos represents the position of a mouse cursor on the UI's
// x-y-plane, which has its origin 0,0 in the top left.
type MouseCursorPos struct {
	X, Y int
}

// TextCursorController offers control of a text cursor, such as for a terminal.
type TextCursorController interface {
	HideCursor()
	ShowCursor(CursorLocation)
}

type CursorLocation struct {
	X int
	Y int
}

func (l CursorLocation) String() string {
	return fmt.Sprintf("%d:%d", l.X, l.Y)
}

type BoxRepresentation[T any] struct {
	X int
	Y int
	W int
	H int

	Represents T

	Children []BoxRepresentation[T]
}

func (r *BoxRepresentation[T]) String() string {
	str := fmt.Sprintf("+%d+%d %dx%d [ ", r.X, r.Y, r.W, r.H)
	for _, child := range r.Children {
		str += child.String()
	}
	str += "]"
	return str
}
