package ui

import (
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
)

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

// Pane is a rectangular pane in a user interface.
// It can be drawn or queried for information.
type Pane interface {
	// Draw draws this pane.
	// Renders whatever contents the pane is supposed to display to whatever UI is
	// currently in use.
	Draw()

	// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
	// height) for this pane.
	Dimensions() (x, y, w, h int)

	// GetPositionInfo returns information on a requested position in this pane.
	GetPositionInfo(x, y int) PositionInfo
}

// RootPane is the root pane in a UI.
// TODO:
//  - probably doesn't make much sense actually
//    - 'close' roughly means do necessary teardown to cleanly end UI
//      - but UI was constructed in controller, why leave closing it to this?
//    - needs sync is also sort of TUI specific? probably? and this one i've
//      definitely wanted the controller to handle, maybe by directly putting
//      a call to the renderer into a callback triggered on resize?
type RootPane interface {
	Pane
	Close()
	NeedsSync()
}

// ConditionalOverlayPane is a UI pane that is only visible given some
// condition holds true.
//
// The condition can be queried by whatever parent holds the pane.
// It's probably best that the parent conditionally draws, but internal
// verification of the condition before executing draw calls is also an option.
type ConditionalOverlayPane interface {
	// Implements the Pane interface.
	Pane
	// The condition which determines, whether this pane should be visible.
	Condition() bool
	// Inform the pane that it is not being shown so that it can take potential
	// actions to ensure that, e.g., hide the terminal cursor, if necessary.
	EnsureHidden()
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

// ConstrainedRenderer is a renderer that is assumed to be constrained to
// certain dimensions, i.E. it does not draw outside of them.
type ConstrainedRenderer interface {
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

// RootPaneRendererControl is the set of functions of a renderer (e.g.,
// tcell.Screen) that the root pane needs to use to have full control over a
// render cycle. Other panes should not need this access to the renderer.
type RootPaneRendererControl interface {
	NeedsSync()
	Fini()
	Clear()
	Show()
}

// ViewParams represents the zoom and scroll of a timeline  in the UI.
type ViewParams struct {
	// NRowsPerHour is the number of rows in the UI that represent an hour in the
	// timeline.
	NRowsPerHour int
	// ScrollOffset is the offset in rows by which the UI is scrolled.
	// (An unscrolled UI would have 00:00 at the very top.)
	ScrollOffset int
}

// MouseCursorPos represents the position of a mouse cursor on the UI's
// x-y-plane, which has its origin 0,0 in the top left.
type MouseCursorPos struct {
	X, Y int
}

// TimeAtY is the time that corresponds to a given y-position.
func (p *ViewParams) TimeAtY(y int) model.Timestamp {
	minutes := y*(60/p.NRowsPerHour) + p.ScrollOffset*(60/p.NRowsPerHour)
	ts := model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}
	return ts
}

// TextCursorController offers control of a text cursor, such as for a terminal.
type TextCursorController interface {
	HideCursor()
	ShowCursor(x, y int)
}
