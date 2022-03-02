package tui

import (
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/ui"
)

type MonthViewMainPane struct {
	dimensions func() (x, y, w, h int)

	status   ui.UIPane
	timeline ui.UIPane

	days []ui.UIPane

	categories *CategoryStyling
	logReader  potatolog.LogReader
	logWriter  potatolog.LogWriter
	viewParams *ViewParams
}

func (p *MonthViewMainPane) Draw() {
	for i := range p.days {
		p.days[i].Draw()
	}

	p.timeline.Draw()
	p.status.Draw()
}

func (p *MonthViewMainPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *MonthViewMainPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}
