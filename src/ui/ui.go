package ui

import "github.com/ja-he/dayplan/src/model"

type UIPaneType int

const (
	_ UIPaneType = iota
	None
	WeatherUIPanelType
	TimelineUIPanelType
	EventsUIPanelType
	ToolsUIPanelType
	StatusUIPanelType
)

// The active view of the day(s), which could be a single day, a
// week or a full month (or in the future any other stretch of time
// that's to be shown).
type ActiveView int

const (
	_ ActiveView = iota
	ViewDay
	ViewWeek
	ViewMonth
)

type UIPane interface {
	Draw()

	Dimensions() (x, y, w, h int)

	GetPositionInfo(x, y int) PositionInfo
}

type MainUIPanel interface {
	UIPane
	Close()
	NeedsSync()
}

type EventHoverState int

const (
	EventHoverStateNone EventHoverState = iota
	EventHoverStateMove
	EventHoverStateResize
	EventHoverStateEdit
)

type PositionInfo interface {
	PaneType() UIPaneType

	GetCursorTimestampGuess() (*model.Timestamp, error)

	GetExtraWeatherInfo() *WeatherPanelPositionInfo
	GetExtraTimelineInfo() *TimelinePanelPositionInfo
	GetExtraEventsInfo() *EventsPanelPositionInfo
	GetExtraToolsInfo() *ToolsPanelPositionInfo
	GetExtraStatusInfo() *StatusPanelPositionInfo
}

type WeatherPanelPositionInfo struct{}
type TimelinePanelPositionInfo struct{}
type ToolsPanelPositionInfo struct {
	Category *model.Category
}
type StatusPanelPositionInfo struct{}
type EventsPanelPositionInfo struct {
	Event           model.EventID
	HoverState      EventHoverState
	TimeUnderCursor model.Timestamp
}
