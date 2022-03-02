package tui

import (
	"github.com/ja-he/dayplan/src/category_style"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/ui"
)

type MonthViewMainPane struct {
	renderer *TUIScreenHandler

	dimensions func() (x, y, w, h int)

	status   ui.UIPane
	timeline ui.UIPane

	days []ui.UIPane

	categories *category_style.CategoryStyling
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
	return &TUIPositionInfo{
		paneType: ui.None,
		weather:  ui.WeatherPanelPositionInfo{},
		timeline: ui.TimelinePanelPositionInfo{},
		tools:    ui.ToolsPanelPositionInfo{},
		status:   ui.StatusPanelPositionInfo{},
		events:   ui.EventsPanelPositionInfo{},
	}
}
