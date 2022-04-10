package panes

import (
	"github.com/ja-he/dayplan/src/input"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/ui"
)

// WeekViewMainPane is the wrapper pane for the week view.
// It contains a timeline, seven vertically stacked event panes (one for each
// week day) and a status bar.
type WeekViewMainPane struct {
	Parent ui.FocussablePane

	dimensions func() (x, y, w, h int)

	timeline ui.Pane
	status   ui.Pane
	days     []ui.Pane

	logReader  potatolog.LogReader
	logWriter  potatolog.LogWriter
	viewParams *ui.ViewParams
}

// Draw draws this pane.
func (p *WeekViewMainPane) Draw() {
	for i := range p.days {
		p.days[i].Draw()
	}

	p.timeline.Draw()
	p.status.Draw()
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *WeekViewMainPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *WeekViewMainPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}

func (p *WeekViewMainPane) HasPartialInput() bool           { return false } // TODO
func (p *WeekViewMainPane) ProcessInput(key input.Key) bool { return false } // TODO

func (p *WeekViewMainPane) HasFocus() bool              { return p.Parent.HasFocus() && p.Parent.Focusses() == p }
func (p *WeekViewMainPane) Focusses() ui.FocussablePane { return nil }

// NewWeekViewMainPane constructs and returns a new WeekViewMainPane.
func NewWeekViewMainPane(
	dimensions func() (x, y, w, h int),
	status ui.Pane,
	timeline ui.Pane,
	days []*EventsPane,
	logReader potatolog.LogReader,
	logWriter potatolog.LogWriter,
	viewParams *ui.ViewParams,
) *WeekViewMainPane {
	weekViewPane := &WeekViewMainPane{
		dimensions: dimensions,
		status:     status,
		timeline:   timeline,
		days:       make([]ui.Pane, 0),
		logReader:  logReader,
		logWriter:  logWriter,
		viewParams: viewParams,
	}
	for i := range days {
		days[i].Parent = weekViewPane
		weekViewPane.days = append(weekViewPane.days, days[i])
	}
	return weekViewPane
}
