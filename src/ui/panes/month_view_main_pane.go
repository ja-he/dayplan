package panes

import (
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/ui"
)

// MonthViewMainPane is the wrapper pane for the month view.
// It contains a timeline, 31 vertically stacked event panes (one for each
// possible day of the month, but if the month only has 28 days, 3 days will
// essentially be invisible and do nothing (see MaybeEventsPane)) and a status
// bar.
type MonthViewMainPane struct {
	dimensions func() (x, y, w, h int)

	status   ui.Pane
	timeline ui.Pane

	days []ui.Pane

	logReader  potatolog.LogReader
	logWriter  potatolog.LogWriter
	viewParams *ui.ViewParams
}

// Draw draws this pane.
func (p *MonthViewMainPane) Draw() {
	for i := range p.days {
		p.days[i].Draw()
	}

	p.timeline.Draw()
	p.status.Draw()
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *MonthViewMainPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *MonthViewMainPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}

// NewMonthViewMainPane constructs and returns a new MonthViewMainPane.
func NewMonthViewMainPane(
	dimensions func() (x, y, w, h int),
	status ui.Pane,
	timeline ui.Pane,
	days []ui.Pane,
	logReader potatolog.LogReader,
	logWriter potatolog.LogWriter,
	viewParams *ui.ViewParams,
) *MonthViewMainPane {
	return &MonthViewMainPane{
		dimensions: dimensions,
		status:     status,
		timeline:   timeline,
		days:       days,
		logReader:  logReader,
		logWriter:  logWriter,
		viewParams: viewParams,
	}
}
