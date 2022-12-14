package panes

import (
	"strings"
	"time"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
)

// TimelinePane can show a timeline, optionally with various embellishments.
// If provided with a sun-time provider, it will display those suntimes in dark
// and light on the timeline. If allowed to get a current time, it will
// highlight the current time.
type TimelinePane struct {
	Leaf

	suntimes    func() *model.SunTimes
	currentTime func() *model.Timestamp

	viewParams ui.TimespanViewParams
}

// Draw draws this pane.
func (p *TimelinePane) Draw() {

	x, y, w, h := p.dimensions()

	p.renderer.DrawBox(x, y, w, h, p.stylesheet.Normal)

	suntimes := p.suntimes()
	currentTime := p.currentTime()

	timestampLength := 5
	timestampLPad := strings.Repeat(" ", w-timestampLength-1)
	timestampRPad := " "
	emptyTimestamp := strings.Repeat(" ", timestampLength)

	if p.viewParams.HeightOfDuration(time.Hour) == 0 {
		panic("RES IS ZERO?!")
	}

	for virtRow := 0; virtRow <= h; virtRow++ {
		timestamp := p.timeAtY(virtRow)

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
			styling = p.stylesheet.TimelineNight
		} else {
			styling = p.stylesheet.TimelineDay
		}

		p.renderer.DrawText(x, virtRow+y, w, 1, styling, timeText)
	}

	if currentTime != nil {
		timeText := timestampLPad + currentTime.ToString() + timestampRPad
		p.renderer.DrawText(x, p.toY(*currentTime)+y, w, 1, p.stylesheet.TimelineNow, timeText)
	}
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *TimelinePane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}

// TODO: remove, this will be part of info returned to controller on query
func (p *TimelinePane) timeAtY(y int) model.Timestamp {
	minutes := y*(60/int(p.viewParams.HeightOfDuration(time.Hour))) + p.viewParams.GetScrollOffset()*(60/int(p.viewParams.HeightOfDuration(time.Hour)))

	ts := model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}

	return ts
}

func (p *TimelinePane) toY(ts model.Timestamp) int {
	return ((ts.Hour*int(p.viewParams.HeightOfDuration(time.Hour)) - p.viewParams.GetScrollOffset()) + (ts.Minute / (60 / int(p.viewParams.HeightOfDuration(time.Hour)))))
}

// NewTimelinePane constructs and returns a new TimelinePane.
func NewTimelinePane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	suntimes func() *model.SunTimes,
	currentTime func() *model.Timestamp,
	viewParams ui.TimespanViewParams,
) *TimelinePane {
	return &TimelinePane{
		Leaf: Leaf{
			Base: Base{
				ID: ui.GeneratePaneID(),
			},
			renderer:   renderer,
			dimensions: dimensions,
			stylesheet: stylesheet,
		},
		suntimes:    suntimes,
		currentTime: currentTime,
		viewParams:  viewParams,
	}
}
