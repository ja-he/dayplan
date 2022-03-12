package ui

import (
	"github.com/ja-he/dayplan/src/model"
)

// PositionInfo describes a position in the user interface.
//
// Retrievers should initially check for the type of pane they are receiving
// information on and can then retreive the relevant additional information from
// the relevant `GetExtra...` function.
// Note that that information will likely be invalid, if it doesn't correspond
// to the pane type indicated.
type PositionInfo interface {
	// TODO: rename PositionInformer (maybe?)

	// The type of pane to which the information pertains.
	PaneType() PaneType

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

// WeatherPanePositionInfo provides information on a position in a weather pane.
type WeatherPanePositionInfo interface{}

// TimelinePanePositionInfo provides information on a position in a timeline
// pane.
type TimelinePanePositionInfo interface{}

// ToolsPanePositionInfo provides information on a position in a tools pane.
type ToolsPanePositionInfo interface {
	Category() *model.Category
}

// StatusPanePositionInfo provides information on a position in a status pane.
type StatusPanePositionInfo interface{}

// EventsPanePositionInfo provides information on a position in a events pane.
type EventsPanePositionInfo interface {
	Event() model.EventID
	EventBoxPart() EventBoxPart
	Time() model.Timestamp
}

// TUIPositionInfo provides information on a position in a TUI, implementing
// the PositionInfo interface.
type TUIPositionInfo struct {
	paneType PaneType
	weather  WeatherPanePositionInfo
	timeline TimelinePanePositionInfo
	tools    ToolsPanePositionInfo
	status   StatusPanePositionInfo
	events   EventsPanePositionInfo
}

// GetExtraWeatherInfo provides additional information for a position in a
// weather pane.
func (t *TUIPositionInfo) GetExtraWeatherInfo() WeatherPanePositionInfo {
	return nil
}

// GetExtraTimelineInfo provides additional information for a position in a
// timeline pane.
func (t *TUIPositionInfo) GetExtraTimelineInfo() TimelinePanePositionInfo {
	return nil
}

// GetExtraToolsInfo provides additional information for a position in a tools
// pane.
func (t *TUIPositionInfo) GetExtraToolsInfo() ToolsPanePositionInfo {
	return t.tools
}

// GetExtraStatusInfo provides additional information for a position in a
// status pane.
func (t *TUIPositionInfo) GetExtraStatusInfo() StatusPanePositionInfo {
	return nil
}

// GetExtraEventsInfo provides additional information for a position in a
// events pane.
func (t *TUIPositionInfo) GetExtraEventsInfo() EventsPanePositionInfo {
	return t.events
}

// PaneType provides additional information for a position in a
func (t *TUIPositionInfo) PaneType() PaneType {
	return t.paneType
}

// NewPositionInfo constructs and returns a PositionInfo from the given
// parameters.
func NewPositionInfo(
	paneType PaneType,
	weather WeatherPanePositionInfo,
	timeline TimelinePanePositionInfo,
	tools ToolsPanePositionInfo,
	status StatusPanePositionInfo,
	events EventsPanePositionInfo,
) PositionInfo {
	return &TUIPositionInfo{
		paneType: paneType,
		weather:  weather,
		timeline: timeline,
		tools:    tools,
		status:   status,
		events:   events,
	}
}
