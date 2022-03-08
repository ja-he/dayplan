package ui

import (
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
)

// UIPaneType is the type of the bottommost meaningful UI pane.
//
// It's conceivable that panes could have sub-panes for convenience in rendering
// that aren't meaningfully different from the top pane or their individual
// purpose isn't relevant outside of the pane structure.
type UIPaneType int

const (
	_ UIPaneType = iota
	// NoUIPane describes anything that is not on a meaningful UI Pane, perhaps
	// in padding space.
	NoUIPane
	// WeatherUIPaneType represents a pane displaying weather information.
	WeatherUIPaneType
	// TimelineUIPaneType represnets a timeline.
	TimelineUIPaneType
	// EventsUIPaneType represents an events pane.
	EventsUIPaneType
	// ToolsUIPaneType represents a tools pane.
	ToolsUIPaneType
	// StatusUIPaneType represents a status pane (or status bar).
	StatusUIPaneType
	// EditorUIPaneType represents an editor (popup/floating) pane.
	EditorUIPaneType
	// LogUIPaneType represents a log pane.
	LogUIPaneType
	// SummaryUIPaneType represents a summary pane.
	SummaryUIPaneType
)

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

// A rectangular pane in a user interface.
// It can be drawn or queried for information.
type UIPane interface {
	// Draw this pane.
	// Renders whatever contents the pane is supposed to display to whatever UI is
	// currently in use.
	Draw()

	// Get the dimensions (x-axis offset, y-axis offset, width, height) for this
	// pane.
	Dimensions() (x, y, w, h int)

	// Get the dimensions (x-axis offset, y-axis offset, width, height) for this
	// pane.
	GetPositionInfo(x, y int) PositionInfo
}

// TODO:
//  - probably doesn't make much sense actually
//    - 'close' roughly means do necessary teardown to cleanly end UI
//      - but UI was constructed in controller, why leave closing it to this?
//    - needs sync is also sort of TUI specific? probably? and this one i've
//      definitely wanted the controller to handle, maybe by directly putting
//      a call to the renderer into a callback triggered on resize?
type MainUIPane interface {
	UIPane
	Close()
	NeedsSync()
}

// A UI pane that is only visible given some condition holds true.
//
// The condition can be queried by whatever parent holds the pane.
// It's probably best that the parent conditionally draws, but internal
// verification of the condition before executing draw calls is also an option.
type ConditionalOverlayPane interface {
	// Implements the `UIPane` interface.
	UIPane
	// The condition which determines, whether this pane should be visible.
	Condition() bool
	// Inform the pane that it is not being shown so that it can take potential
	// actions to ensure that, e.g., hide the terminal cursor, if necessary.
	EnsureHidden()
}

// The type of an event box (the visual representation of an event in the user
// interface).
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

// Convert an an `EventBoxPart` to a string.
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

// Describes a position in the user interface.
//
// Retrievers should initially check for the type of pane they are receiving
// information on and can then retreive the relevant additional information from
// the relevant `GetExtra...` function.
// Note that that information will likely be invalid, if it doesn't correspond
// to the pane type indicated.
type PositionInfo interface {
	// The type of pane to which the information pertains.
	PaneType() UIPaneType

	// Additional information on a position in a weather pane.
	GetExtraWeatherInfo() WeatherPanePositionInfo
	// Additional information on a position in a timeline pane.
	GetExtraTimelineInfo() TimelinePanePositionInfo
	// Additional information on a position in a events pane.
	GetExtraEventsInfo() EventsPanePositionInfo
	// Additional information on a position in a tools pane.
	GetExtraToolsInfo() ToolsPanePositionInfo
	// Additional information on a position in a status pane.
	GetExtraStatusInfo() StatusPanePositionInfo

	// NOTE: additional functions to be expected, corresponding to pane types.
}

// Information on a position in a weather pane.
type WeatherPanePositionInfo interface{}

// Information on a position in a timeline pane.
type TimelinePanePositionInfo interface{}

// Information on a position in a tools pane.
type ToolsPanePositionInfo interface {
	Category() *model.Category
}

// Information on a position in a status pane.
type StatusPanePositionInfo interface{}

// Information on a position in a events pane.
type EventsPanePositionInfo interface {
	Event() model.EventID
	EventBoxPart() EventBoxPart
	Time() model.Timestamp
}

// A renderer that is assumed to be constrained to certain dimensions, i.E. it
// does not draw outside of them.
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
