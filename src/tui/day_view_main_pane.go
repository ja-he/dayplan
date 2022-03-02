package tui

import (
	"fmt"

	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

type DayViewMainPane struct {
	renderer *TUIScreenHandler

	dimensions func() (x, y, w, h int)

	events   ui.UIPane
	tools    ui.UIPane
	status   ui.UIPane
	timeline ui.UIPane
	weather  ui.UIPane
}

func (p *DayViewMainPane) Draw() {
	p.events.Draw()

	p.weather.Draw()
	p.timeline.Draw()
	p.tools.Draw()
	p.status.Draw()
}

func (p *DayViewMainPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *DayViewMainPane) GetPositionInfo(x, y int) ui.PositionInfo {
	panes := []ui.UIPane{p.events, p.tools, p.status, p.timeline, p.weather}

	for _, pane := range panes {
		if util.NewRect(pane.Dimensions()).Contains(x, y) {
			return pane.GetPositionInfo(x, y)
		}
	}

	panic(fmt.Sprint("none of the day view main pane's subpanes contains pos", x, y))
}
