package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/colors"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

type StatusPanel struct {
	renderer *TUIRenderer

	dimensions func() (x, y, w, h int)

	currentDate *model.Date
	activeView  *ui.ActiveView
}

func (p *StatusPanel) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *StatusPanel) Draw() {
	_, y, w, h := p.dimensions()

	var firstDay, lastDay model.Date
	switch *p.activeView {
	case ui.ViewDay:
		firstDay, lastDay = *p.currentDate, *p.currentDate
	case ui.ViewWeek:
		firstDay, lastDay = p.currentDate.Week()
	case ui.ViewMonth:
		firstDay, lastDay = p.currentDate.MonthBounds()
	}

	// TODO: should be field, maybe closure or just ptr?
	firstDayXOffset := 10

	nDaysInPeriod := firstDay.DaysUntil(lastDay) + 1
	nDaysTilCurrent := firstDay.DaysUntil(*p.currentDate)
	dateWidth := 10 // 2020-02-12 is 10 wide
	dayWidth := (w - firstDayXOffset) / nDaysInPeriod

	bgStyle := tcell.StyleDefault.Background(colors.ColorFromHexString("#f0f0f0")).Foreground(tcell.ColorBlack)
	bgStyleEmph := colors.DefaultEmphasize(bgStyle)
	dateStyle := bgStyleEmph
	weekdayStyle := colors.LightenFG(dateStyle, 60)

	// header background
	p.renderer.DrawBox(bgStyle, 0, y, firstDayXOffset+nDaysInPeriod*dayWidth, h)
	// header bar (filled for days until current)
	p.renderer.DrawBox(bgStyleEmph, 0, y, firstDayXOffset+(nDaysTilCurrent+1)*dayWidth, h)
	// date box background
	p.renderer.DrawBox(bgStyleEmph, 0, y, dateWidth, h)
	// date string
	p.renderer.DrawText(0, y, dateWidth, 0, dateStyle, p.currentDate.ToString())
	// weekday string
	p.renderer.DrawText(0, y+1, dateWidth, 0, weekdayStyle, util.TruncateAt(p.currentDate.ToWeekday().String(), dateWidth))
}

func (p *StatusPanel) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}
