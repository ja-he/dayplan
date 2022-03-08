package panes

import (
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
)

// WeekViewMainPane is the wrapper pane for the week view.
// It contains a timeline, seven vertically stacked event panes (one for each
// week day) and a status bar.
type WeekViewMainPane struct {
	dimensions func() (x, y, w, h int)

	timeline ui.Pane
	status   ui.Pane
	days     []ui.Pane

	categories *styling.CategoryStyling
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

// NewWeekViewMainPane constructs and returns a new WeekViewMainPane.
func NewWeekViewMainPane(
	dimensions func() (x, y, w, h int),
	status ui.Pane,
	timeline ui.Pane,
	days []ui.Pane,
	categories *styling.CategoryStyling,
	logReader potatolog.LogReader,
	logWriter potatolog.LogWriter,
	viewParams *ui.ViewParams,
) *WeekViewMainPane {
	return &WeekViewMainPane{
		dimensions: dimensions,
		status:     status,
		timeline:   timeline,
		days:       days,
		categories: categories,
		logReader:  logReader,
		logWriter:  logWriter,
		viewParams: viewParams,
	}
}
