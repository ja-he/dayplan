package panes

import (
	"fmt"

	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

// DayViewMainPane is the wrapper pane for the day view.
// It contains weather information, a timeline, a single event pane (for the
// current day), a tools pane  and a status bar.
type DayViewMainPane struct {
	Parent ui.FocussablePane

	dimensions func() (x, y, w, h int)

	events ui.FocussablePane
	tools  ui.FocussablePane

	status   ui.Pane
	timeline ui.Pane
	weather  ui.Pane

	inputTree    input.Tree
	focussedPane ui.FocussablePane
}

// Draw draws this pane.
func (p *DayViewMainPane) Draw() {
	p.events.Draw()

	p.weather.Draw()
	p.timeline.Draw()
	p.tools.Draw()
	p.status.Draw()
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
func (p *DayViewMainPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *DayViewMainPane) GetPositionInfo(x, y int) ui.PositionInfo {
	panes := []ui.Pane{p.events, p.tools, p.status, p.timeline, p.weather}

	for _, pane := range panes {
		if util.NewRect(pane.Dimensions()).Contains(x, y) {
			return pane.GetPositionInfo(x, y)
		}
	}

	panic(fmt.Sprint("none of the day view main pane's subpanes contains pos", x, y))
}

func (p *DayViewMainPane) GetFocussed() ui.Pane {
	return p.focussedPane
}

func (p *DayViewMainPane) FocusLeft() {
	switch p.focussedPane {
	case p.events:
		return
	case p.tools:
		p.focussedPane = p.events
	default:
		panic("unknown focussed pane in day view")
	}
}

func (p *DayViewMainPane) FocusRight() {
	switch p.focussedPane {
	case p.events:
		p.focussedPane = p.tools
	case p.tools:
		return
	default:
		panic("unknown focussed pane in day view")
	}
}

func (p *DayViewMainPane) HasPartialInput() bool { return p.focussedPane.HasPartialInput() }
func (p *DayViewMainPane) ProcessInput(key input.Key) bool {
	if p.Focusses().ProcessInput(key) {
		return true
	} else {
		return p.inputTree.Process(key)
	}
}

func (p *DayViewMainPane) HasFocus() bool              { return p.Parent.HasFocus() && p.Parent.Focusses() == p }
func (p *DayViewMainPane) Focusses() ui.FocussablePane { return p.focussedPane }

// NewDayViewMainPane constructs and returns a new DayViewMainPane.
func NewDayViewMainPane(
	dimensions func() (x, y, w, h int),
	events *EventsPane,
	tools *ToolsPane,
	status ui.Pane,
	timeline ui.Pane,
	weather ui.Pane,
	inputTree input.Tree,
) *DayViewMainPane {
	dayViewPane := &DayViewMainPane{
		dimensions:   dimensions,
		events:       events,
		tools:        tools,
		status:       status,
		timeline:     timeline,
		weather:      weather,
		focussedPane: events,
		inputTree:    inputTree,
	}
	events.Parent = dayViewPane
	tools.Parent = dayViewPane
	return dayViewPane
}
