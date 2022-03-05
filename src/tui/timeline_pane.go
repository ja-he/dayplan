package tui

import (
	"strings"

	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
)

type TimelinePane struct {
	renderer ui.ConstrainedRenderer

	dimensions func() (x, y, w, h int)
	stylesheet styling.Stylesheet

	suntimes    func() *model.SunTimes
	currentTime func() *model.Timestamp

	viewParams *ViewParams
}

func (p *TimelinePane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *TimelinePane) Draw() {

	x, y, w, h := p.dimensions()
	suntimes := p.suntimes()
	currentTime := p.currentTime()

	timestampLength := 5
	timestampLPad := strings.Repeat(" ", w-timestampLength-1)
	timestampRPad := " "
	emptyTimestamp := strings.Repeat(" ", timestampLength)

	if p.viewParams.NRowsPerHour == 0 {
		panic("RES IS ZERO?!")
	}

	for virtRow := 0; virtRow <= h; virtRow++ {
		timestamp := p.TimeAtY(virtRow)

		if timestamp.Hour >= 24 {
			break
		}

		var timestampString string
		if timestamp.Minute == 0 {
			timestampString = timestamp.ToString()
		} else {
			timestampString = emptyTimestamp
		}
		timeText := timestampLPad + timestampString + timestampRPad

		var styling styling.DrawStyling
		if suntimes != nil && (!(timestamp.IsAfter(suntimes.Rise)) || (timestamp.IsAfter(suntimes.Set))) {
			styling = p.stylesheet.TimelineNight()
		} else {
			styling = p.stylesheet.TimelineDay()
		}

		p.renderer.DrawText(x, virtRow+y, w, 1, styling, timeText)
	}

	if currentTime != nil {
		timeText := timestampLPad + currentTime.ToString() + timestampRPad
		p.renderer.DrawText(x, p.toY(*currentTime)+y, w, 1, p.stylesheet.TimelineNow(), timeText)
	}
}

func (p *TimelinePane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}

// TODO: remove, this will be part of info returned to controller on query
func (t *TimelinePane) TimeAtY(y int) model.Timestamp {
	minutes := y*(60/t.viewParams.NRowsPerHour) + t.viewParams.ScrollOffset*(60/t.viewParams.NRowsPerHour)

	ts := model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}

	return ts
}

func (t *TimelinePane) toY(ts model.Timestamp) int {
	return ((ts.Hour*t.viewParams.NRowsPerHour - t.viewParams.ScrollOffset) + (ts.Minute / (60 / t.viewParams.NRowsPerHour)))
}
