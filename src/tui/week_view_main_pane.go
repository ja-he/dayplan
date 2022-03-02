package tui

import (
	"math"

	"github.com/ja-he/dayplan/src/category_style"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

type WeekViewMainPane struct {
	dimensions func() (x, y, w, h int)

	timeline ui.UIPane
	status   ui.UIPane
	days     []ui.UIPane

	categories *category_style.CategoryStyling
	logReader  potatolog.LogReader
	logWriter  potatolog.LogWriter
	viewParams *ViewParams

	// TODO: get rid of this
	positions map[model.EventID]util.Rect
}

func (p *WeekViewMainPane) Draw() {
	for i := range p.days {
		p.days[i].Draw()
	}

	p.timeline.Draw()
	p.status.Draw()
}

func (p *WeekViewMainPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *WeekViewMainPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}

// TODO: remove, this will be part of info returned to controller on query
func (t *WeekViewMainPane) TimeAtY(y int) model.Timestamp {
	minutes := y*(60/t.viewParams.NRowsPerHour) + t.viewParams.ScrollOffset*(60/t.viewParams.NRowsPerHour)

	ts := model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}

	return ts
}

func (t *WeekViewMainPane) toY(ts model.Timestamp) int {
	return ((ts.Hour*t.viewParams.NRowsPerHour - t.viewParams.ScrollOffset) + (ts.Minute / (60 / t.viewParams.NRowsPerHour)))
}

func (t *WeekViewMainPane) ComputeRects(day *model.Day, offsetX, offsetY, width, height int) map[model.EventID]util.Rect {
	active_stack := make([]model.Event, 0)
	positions := make(map[model.EventID]util.Rect)
	for _, e := range day.Events {
		// remove all stacked elements that have finished
		for i := len(active_stack) - 1; i >= 0; i-- {
			if e.Start.IsAfter(active_stack[i].End) || e.Start == active_stack[i].End {
				active_stack = active_stack[:i]
			} else {
				break
			}
		}
		active_stack = append(active_stack, e)
		// based on event state, draw a box or maybe a smaller one, or ...
		y := t.toY(e.Start) + offsetY
		x := offsetX
		h := t.toY(e.End) + offsetY - y
		w := width

		// scale the width by 3/4 for every extra item on the stack, so for one
		// item stacked underneath the current items width will be (3/4) ** 1 = 75%
		// of the original width, for four it would be (3/4) ** 4 = (3**4)/(4**4)
		// or 31.5 % of the width, etc.
		widthFactor := 0.75
		w = int(float64(w) * math.Pow(widthFactor, float64(len(active_stack)-1)))
		x += (width - w)

		positions[e.ID] = util.Rect{X: x, Y: y, W: w, H: h}
	}
	return positions
}
