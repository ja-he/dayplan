package tui

import (
	"math"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/category_style"
	"github.com/ja-he/dayplan/src/colors"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

type MonthViewMainPane struct {
	renderer *TUIRenderer

	dimensions func() (x, y, w, h int)

	// TODO timeline
	status ui.UIPane

	days        *DaysData
	currentDate *model.Date
	categories  *category_style.CategoryStyling
	logReader   potatolog.LogReader
	logWriter   potatolog.LogWriter
	viewParams  *ViewParams

	// TODO: get rid of this
	positions map[model.EventID]util.Rect
}

func (p *MonthViewMainPane) Draw() {
	// TODO timeline
	p.drawEvents()

	p.status.Draw()
}

func (p *MonthViewMainPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *MonthViewMainPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return &TUIPositionInfo{
		paneType:       ui.None,
		weather:        ui.WeatherPanelPositionInfo{},
		timeline:       ui.TimelinePanelPositionInfo{},
		tools:          ui.ToolsPanelPositionInfo{},
		status:         ui.StatusPanelPositionInfo{},
		events:         ui.EventsPanelPositionInfo{},
		timestampGuess: *model.NewTimestamp("00:00"),
	}
}

// TODO: remove, this will be part of info returned to controller on query
func (t *MonthViewMainPane) TimeAtY(y int) model.Timestamp {
	minutes := y*(60/t.viewParams.NRowsPerHour) + t.viewParams.ScrollOffset*(60/t.viewParams.NRowsPerHour)

	ts := model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}

	return ts
}

func (t *MonthViewMainPane) toY(ts model.Timestamp) int {
	return ((ts.Hour*t.viewParams.NRowsPerHour - t.viewParams.ScrollOffset) + (ts.Minute / (60 / t.viewParams.NRowsPerHour)))
}

func (t *MonthViewMainPane) drawEvents() {
	_, _, w, h := t.Dimensions()
	fakeStatusHeight := 2 // TODO: hardcoded just for now

	// TODO: define all styles here (prep to probably move out further)
	headerBG := tcell.StyleDefault.Background(colors.ColorFromHexString("#f0f0f0")).Foreground(tcell.ColorBlack)
	headerBGEmph := colors.DefaultEmphasize(headerBG)
	dayBG := tcell.StyleDefault
	dayBGEmph := headerBGEmph
	loadingStyle := dayBG.Foreground(tcell.ColorLightSeaGreen)

	// TODO
	start, end := t.currentDate.MonthBounds()
	nDays := start.DaysUntil(end) + 1
	if nDays > w {
		t.renderer.DrawText(0, 0, w, h,
			tcell.StyleDefault.Foreground(tcell.ColorRebeccaPurple),
			"refusing to render month on screen with fewer columns than days")
		return
	}

	{
		firstDayXOffset := 10
		x := firstDayXOffset
		dayWidth := (w - firstDayXOffset) / nDays

		t.drawTimelineTmp(0, 0, firstDayXOffset, h-fakeStatusHeight, make([]timestampStyle, 0), nil)

		for drawDate := start; drawDate != end.Next(); drawDate = drawDate.Next() {
			if drawDate == *t.currentDate {
				t.renderer.DrawBox(dayBGEmph, x, 0, dayWidth, h)
			} else {
				t.renderer.DrawBox(dayBG, x, 0, dayWidth, h)
			}
			day := t.days.GetDay(drawDate)
			if day != nil {
				positions := t.ComputeRects(day, x, 0, dayWidth, h-fakeStatusHeight)
				for _, e := range day.Events {
					p := positions[e.ID]
					style, err := t.categories.GetStyle(e.Cat)
					if err != nil {
						panic(err)
					}
					if drawDate != *t.currentDate {
						style = colors.DefaultDim(style)
					}
					t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H)
				}
			} else {
				loadingText := "⋮"
				t.renderer.DrawText(x, h/2-len([]rune(loadingText)), 1, len([]rune(loadingText)),
					loadingStyle,
					loadingText)
			}
			x += dayWidth
		}
	}
}

func (t *MonthViewMainPane) ComputeRects(day *model.Day, offsetX, offsetY, width, height int) map[model.EventID]util.Rect {
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

// TODO: remove
func (t *MonthViewMainPane) drawTimelineTmp(
	x, y, w, h int,
	highlightTimes []timestampStyle,
	suntimes *model.SunTimes) {

	timestampLength := 5
	timestampLPad := strings.Repeat(" ", w-timestampLength-1)
	timestampRPad := " "
	emptyTimestamp := strings.Repeat(" ", timestampLength)
	defaultStyle := tcell.StyleDefault.Foreground(tcell.ColorLightGray)

	if t.viewParams.NRowsPerHour == 0 {
		panic("RES IS ZERO?!")
	}

	for virtRow := 0; virtRow <= h; virtRow++ {
		timestamp := t.TimeAtY(virtRow)

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

		var style tcell.Style
		if suntimes != nil && (!(timestamp.IsAfter(suntimes.Rise)) || (timestamp.IsAfter(suntimes.Set))) {
			style = defaultStyle.Background(tcell.ColorBlack)
		} else {
			style = defaultStyle
		}

		t.renderer.DrawText(x, virtRow+y, w, 1, style, timeText)
	}
	for _, timestampStyle := range highlightTimes {
		timestamp := timestampStyle.timestamp
		style := timestampStyle.style
		timeText := timestampLPad + timestamp.ToString() + timestampRPad
		t.renderer.DrawText(x, t.toY(timestamp)+y, w, 1, style, timeText)
	}
}
