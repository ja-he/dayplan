package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/colors"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

type StatusPane struct {
	renderer *TUIScreenHandler

	dimensions func() (x, y, w, h int)

	currentDate *model.Date

	dayWidth           func() int
	totalDaysInPeriod  func() int
	passedDaysInPeriod func() int
	firstDayXOffset    func() int
}

func (p *StatusPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *StatusPane) Draw() {
	_, y, _, h := p.dimensions()

	dateWidth := 10 // 2020-02-12 is 10 wide

	bgStyle := tcell.StyleDefault.Background(colors.ColorFromHexString("#f0f0f0")).Foreground(tcell.ColorBlack)
	bgStyleEmph := colors.DefaultEmphasize(bgStyle)
	dateStyle := bgStyleEmph
	weekdayStyle := colors.LightenFG(dateStyle, 60)

	// header background
	p.renderer.DrawBox(bgStyle, 0, y, p.firstDayXOffset()+p.totalDaysInPeriod()*p.dayWidth(), h)
	// header bar (filled for days until current)
	p.renderer.DrawBox(bgStyleEmph, 0, y, p.firstDayXOffset()+(p.passedDaysInPeriod())*p.dayWidth(), h)
	// date box background
	p.renderer.DrawBox(bgStyleEmph, 0, y, dateWidth, h)
	// date string
	p.renderer.DrawText(0, y, dateWidth, 1, dateStyle, p.currentDate.ToString())
	// weekday string
	p.renderer.DrawText(0, y+1, dateWidth, 1, weekdayStyle, util.TruncateAt(p.currentDate.ToWeekday().String(), dateWidth))
}

func (p *StatusPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}
