package ui

import (
	"github.com/ja-he/dayplan/internal/model"
)

// PositionInfo describes a position in the user interface.
//
// Retrievers should initially check for the type of pane they are receiving
// information on and can then retrieve the relevant additional information from
// whatever they got.
type PositionInfo interface{}

// NoPanePositionInfo is (no) information about no position. Comprende?
// (example: maybe pane that is none, has to return ~something~)
type NoPanePositionInfo struct{}

// WeatherPanePositionInfo provides information on a position in a weather pane.
type WeatherPanePositionInfo struct{}

// TimelinePanePositionInfo provides information on a position in a timeline
// pane.
type TimelinePanePositionInfo struct{}

// ToolsPanePositionInfo conveys information on a position in a tools pane,
// importantly the possible category displayed at that position.
type ToolsPanePositionInfo struct {
	Category *model.Category
}

// TasksPanePositionInfo provides information on a position in a tasks pane.
type TasksPanePositionInfo struct{}

// StatusPanePositionInfo provides information on a position in a status pane.
type StatusPanePositionInfo struct{}

// EventsPanePositionInfo provides information on a position in an EventsPane.
type EventsPanePositionInfo struct {
	Event        *model.Event
	EventBoxPart EventBoxPart
	Time         model.Timestamp
}
