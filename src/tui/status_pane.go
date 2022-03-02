package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

type StatusPane struct {
	renderer ui.ConstrainedRenderer

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

	bgStyle := NewStyling((tcell.ColorBlack), styling.ColorFromHexString("#f0f0f0"))
	bgStyleEmph := bgStyle.DefaultEmphasized()
	dateStyle := bgStyleEmph
	weekdayStyle := dateStyle.LightenedFG(60)

	// header background
	p.renderer.DrawBox(0, y, p.firstDayXOffset()+p.totalDaysInPeriod()*p.dayWidth(), h, bgStyle)
	// header bar (filled for days until current)
	p.renderer.DrawBox(0, y, p.firstDayXOffset()+(p.passedDaysInPeriod())*p.dayWidth(), h, bgStyleEmph)
	// date box background
	p.renderer.DrawBox(0, y, dateWidth, h, bgStyleEmph)
	// date string
	p.renderer.DrawText(0, y, dateWidth, 1, dateStyle, p.currentDate.ToString())
	// weekday string
	p.renderer.DrawText(0, y+1, dateWidth, 1, weekdayStyle, util.TruncateAt(p.currentDate.ToWeekday().String(), dateWidth))
}

func (p *StatusPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}
