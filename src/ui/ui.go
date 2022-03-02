package ui

import "github.com/ja-he/dayplan/src/model"

type UIPaneType int

const (
	_ UIPaneType = iota
	None
	WeatherUIPaneType
	TimelineUIPaneType
	EventsUIPaneType
	ToolsUIPaneType
	StatusUIPaneType
	EditorUIPaneType
	LogUIPaneType
	SummaryUIPaneType
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

type MainUIPane interface {
	UIPane
	Close()
	NeedsSync()
}

type ConditionalOverlayPane interface {
	UIPane
	Condition() bool
}

type EventBoxPart int

const (
	_ EventBoxPart = iota
	EventBoxNowhere
	EventBoxInterior
	EventBoxBottomRight
	EventBoxTopEdge
)

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

type PositionInfo interface {
	PaneType() UIPaneType

	GetExtraWeatherInfo() WeatherPanePositionInfo
	GetExtraTimelineInfo() TimelinePanePositionInfo
	GetExtraEventsInfo() EventsPanePositionInfo
	GetExtraToolsInfo() ToolsPanePositionInfo
	GetExtraStatusInfo() StatusPanePositionInfo
}

type WeatherPanePositionInfo interface{}
type TimelinePanePositionInfo interface{}
type ToolsPanePositionInfo interface {
	Category() *model.Category
}
type StatusPanePositionInfo interface{}
type EventsPanePositionInfo interface {
	Event() model.EventID
	EventBoxPart() EventBoxPart
	Time() model.Timestamp
}
