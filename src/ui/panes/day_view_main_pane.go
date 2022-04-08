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
	dimensions func() (x, y, w, h int)

	events   ui.Pane
	tools    ui.Pane
	status   ui.Pane
	timeline ui.Pane
	weather  ui.Pane

	inputTree input.Tree
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

func (p *DayViewMainPane) HasPartialInput() bool           { return p.inputTree.Active() }
func (p *DayViewMainPane) ProcessInput(key input.Key) bool { return p.inputTree.Process(key) }

// NewDayViewMainPane constructs and returns a new DayViewMainPane.
func NewDayViewMainPane(
	dimensions func() (x, y, w, h int),
	events ui.Pane,
	tools ui.Pane,
	status ui.Pane,
	timeline ui.Pane,
	weather ui.Pane,
	inputTree input.Tree,
) *DayViewMainPane {
	return &DayViewMainPane{
		dimensions: dimensions,
		events:     events,
		tools:      tools,
		status:     status,
		timeline:   timeline,
		weather:    weather,
		inputTree:  inputTree,
	}
}
